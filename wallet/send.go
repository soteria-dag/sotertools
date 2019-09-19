// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterjson"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/txscript"
	"github.com/soteria-dag/soterd/wire"
	"github.com/soteria-dag/soterwallet/wallet"
	"github.com/soteria-dag/soterwallet/walletdb"
)

var (
	// This variable is used when unlocking a wallet
	waddrmgrNamespaceKey = []byte("waddrmgr")
)

// absAmount returns the absolute value of the amount
func absAmount(amt soterutil.Amount) soterutil.Amount {
	if amt < soterutil.Amount(0) {
		return -amt
	} else {
		return amt
	}
}

// makeTxInputs creates some transaction inputs that can be used in the createrawtransaction RPC call
func makeTxInputs(matches []TxMatch, desired, fee soterutil.Amount) []soterjson.TransactionInput {
	txIns := make([]soterjson.TransactionInput, 0)

	total := soterutil.Amount(0)
	remaining := desired + fee
	for _, m := range matches {
		t := soterjson.TransactionInput{
			Txid: m.Info.Tx.TxHash().String(),
			Vout: uint32(m.VIndex),
		}
		txIns = append(txIns, t)
		remaining -= m.Amount
		total += m.Amount
		if remaining <= soterutil.Amount(0) {
			break
		}
	}

	return txIns
}

// makeTxAmts creates a map of which addresses will receive what amount of coin in a transaction.
// It can be used in the createrawtransaction RPC call.
func makeTxAmts(matches []TxMatch, destAddr soterutil.Address, desired, fee soterutil.Amount) map[soterutil.Address]soterutil.Amount {
	var amounts = make(map[soterutil.Address]soterutil.Amount)
	// We need to declare that dest is of soterutil.Address type, in order for it to work as a key in the amounts map.
	var dest soterutil.Address
	dest = NewMockAddr(destAddr.EncodeAddress())
	amounts[dest] = soterutil.Amount(0)

	for _, m := range matches {
		if amounts[dest] + m.Amount > desired {
			needed := desired - amounts[dest]
			change := m.Amount - needed - fee
			amounts[dest] += needed

			if change > soterutil.Amount(0) {
				var owner soterutil.Address
				owner = NewMockAddr(m.Address)
				_, exists := amounts[owner]
				if exists {
					amounts[owner] += change
				} else {
					amounts[owner] = change
				}
			}
		} else {
			amounts[dest] += m.Amount
		}

		if amounts[dest] >= desired {
			break
		}
	}

	return amounts
}

// makePrevScripts returns a mapping of scripts from outputs that could be used in a transaction. This is meant for
// preventing soterwallet functions from attempting to look up info internally that it wouldn't have, due to us not
// running a full soterwallet node.
func makePrevScripts(matches []TxMatch) map[wire.OutPoint][]byte {
	prevScripts := make(map[wire.OutPoint][]byte)

	for _, m := range matches {
		txHash := m.Info.Tx.TxHash()
		op := wire.NewOutPoint(&txHash, uint32(m.VIndex))
		prevScripts[*op] = m.Info.Tx.TxOut[m.VIndex].PkScript
	}

	return prevScripts
}

// newTransaction returns a raw transaction that can be signed and sent to the soter network
func newTransaction(client *rpcclient.Client, matches []TxMatch, dest soterutil.Address, amount, fee soterutil.Amount) (*wire.MsgTx, error) {
	txIns := makeTxInputs(matches, amount, fee)
	txAmts := makeTxAmts(matches, dest, amount, fee)
	// Have the soterd node translate our inputs and amounts into a raw transaction
	return client.CreateRawTransaction(txIns, txAmts, nil)
}

// Send creates a new transaction to send coin to the given address, signs it, and sends it to the network via the rpc client.
func Send(client *rpcclient.Client, w *wallet.Wallet, privPass string, matches []TxMatch, dest soterutil.Address, amount, fee soterutil.Amount) (*chainhash.Hash, error) {
	// Create a new transaction
	tx, err := newTransaction(client, matches, dest, amount, fee)
	if err != nil {
		return nil, fmt.Errorf("createrawtransaction RPC call failed: %s", err)
	}

	// Build a map of scripts from the outputs that are used as inputs in the new transaction.
	// This is so that the w.SignTransaction method won't attempt to look up this information in its own records,
	// which it won't have because we aren't running a full soterwallet node.
	prevScripts := makePrevScripts(matches)

	err = walletdb.View(w.Database(), func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		return w.Manager.Unlock(addrmgrNs, []byte(privPass))
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to unlock wallet: %s", err)
	}

	// Sign the transaction
	invalidSigs, err := w.SignTransaction(tx, txscript.SigHashAll, prevScripts, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to sign transaction: %s", err)
	}

	for _, e := range invalidSigs {
		fmt.Printf("Unsigned input at index: %d\n", e.InputIndex)
	}

	txHash, err := client.SendRawTransaction(tx, false)
	if err != nil {
		return nil, fmt.Errorf("Failed to send transaction to network: %s", err)
	}

	return txHash, nil
}