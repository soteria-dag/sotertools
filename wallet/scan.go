// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/txscript"
	"github.com/soteria-dag/soterd/wire"
)

var zeroHash chainhash.Hash

// TxMatch represents a match when finding a transaction that matches an address from a wallet.
type TxMatch struct {
	// Which address matched
	Address string
	Amount  soterutil.Amount

	// The index of the matching input/output in the transaction
	VIndex int

	// A pointer to the transaction info for the match
	Info *TxInfo
}

// TxInfo stores some extra context about a transaction, making coinbase maturity checks easier
type TxInfo struct {
	Tx          *wire.MsgTx
	Block       *wire.MsgBlock
	Index       int
	BlockHeight int32
}

// AllTransactions returns a slice of all transactions
func AllTransactions(client *rpcclient.Client) ([]TxInfo, error) {
	var transactions = make([]TxInfo, 0)

	tips, err := client.GetDAGTips()
	if err != nil {
		return transactions, err
	}

	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := client.GetBlockHash(int64(height))
		if err != nil {
			return transactions, err
		}

		for _, hash := range hashes {
			block, err := client.GetBlock(hash)
			if err != nil {
				return transactions, err
			}

			for i, tx := range block.Transactions {
				info := TxInfo{
					Tx:          tx,
					Block:       block,
					Index:       i,
					BlockHeight: height,
				}

				transactions = append(transactions, info)
			}
		}
	}

	return transactions, nil
}

// IsAddressIn returns true if the given address is in the set of addresses
func IsAddressIn(address soterutil.Address, set []soterutil.Address) bool {
	for _, member := range set {
		if address.EncodeAddress() == member.EncodeAddress() {
			return true
		}
	}

	return false
}

// IsSpendable returns true if a transaction and its input meet the coinbase maturity requirements
func IsSpendable(info, prev TxInfo, params *chaincfg.Params) bool {
	// The coin is spendable if the input for the transaction is >= coinbase maturity height away from the output.
	//
	// TODO(cedric): Update the definition of 'spendable' to be:
	// If the shortest distance between input and output along bluest blocks between the two transactions
	// is >= coinbase maturity.
	if info.BlockHeight > prev.BlockHeight+ int32(params.CoinbaseMaturity) {
		return true
	}

	return false
}

// GetBalance returns the balance and spendable balance of coin for the given addresses, based on matching output transactions in the dag
func GetBalance(client *rpcclient.Client, addresses []soterutil.Address, params *chaincfg.Params) (soterutil.Amount, soterutil.Amount, error) {
	var balance = soterutil.Amount(0)
	var spendableBalance = soterutil.Amount(0)
	var transactions []TxInfo
	var txIndex = make(map[chainhash.Hash]TxInfo)

	transactions, err := AllTransactions(client)
	if err != nil {
		return balance, spendableBalance, err
	}

	for _, info := range transactions {
		txIndex[info.Tx.TxHash()] = info
	}

	for _, info := range transactions {
		// Deduct matching inputs from the balance
		for i, txIn := range info.Tx.TxIn {
			if txIn.PreviousOutPoint.Hash.IsEqual(&zeroHash) {
				// We don't attempt to find the previous output for the input of the genesis transactions,
				// because there won't be any.
				continue
			}

			prev, ok := txIndex[txIn.PreviousOutPoint.Hash]
			if !ok {
				err := fmt.Errorf("missing previous transaction %s for transaction %s input %d",
					txIn.PreviousOutPoint.Hash, info.Tx.TxHash(), i)
				return balance, spendableBalance, err
			}

			prevOut := prev.Tx
			prevValue := prevOut.TxOut[txIn.PreviousOutPoint.Index].Value

			prevPkScript := prevOut.TxOut[txIn.PreviousOutPoint.Index].PkScript
			_, outAddrs, _, err := txscript.ExtractPkScriptAddrs(prevPkScript, params)
			if err != nil {
				return balance, spendableBalance, err
			}

			for _, prevAddress := range outAddrs {
				if !IsAddressIn(prevAddress, addresses) {
					continue
				}

				prevAmount := soterutil.Amount(prevValue)
				// Deduct the input amount from the balance
				balance -= prevAmount

				if IsSpendable(info, prev, params) {
					spendableBalance -= prevAmount
				}
			}
		}

		// Add matching outputs to the balance
		for _, txOut := range info.Tx.TxOut {
			// Extract output addresses from the script in the output
			_, outAddresses, _, err := txscript.ExtractPkScriptAddrs(txOut.PkScript, params)
			if err != nil {
				return balance, spendableBalance, err
			}

			for _, address := range outAddresses {
				if !IsAddressIn(address, addresses) {
					continue
				}

				amount := soterutil.Amount(txOut.Value)
				balance += amount

				// TODO(cedric): Base spendability off of the highest transaction input, not the first
				prev := txIndex[info.Tx.TxIn[0].PreviousOutPoint.Hash]
				if IsSpendable(info, prev, params) {
					spendableBalance += amount
				}
			}
		}
	}

	return balance, spendableBalance, nil
}

// SpendableTxOuts returns a slice of transactions from the dag, where
// * output addresses match the given addresses, and
// * the coin in the transaction is spendable.
func SpendableTxOuts(client *rpcclient.Client, addresses []soterutil.Address, params *chaincfg.Params) ([]TxMatch, error) {
	var matches = make([]TxMatch, 0)
	var transactions []TxInfo
	var txIndex = make(map[chainhash.Hash]TxInfo)

	tips, err := client.GetDAGTips()
	if err != nil {
		return nil, err
	}

	transactions, err = AllTransactions(client)
	if err != nil {
		return nil, err
	}

	for _, info := range transactions {
		txIndex[info.Tx.TxHash()] = info
	}

	for _, info := range transactions {
		for i, txOut := range info.Tx.TxOut {
			_, outAddresses, _, err := txscript.ExtractPkScriptAddrs(txOut.PkScript, params)
			if err != nil {
				return nil, err
			}

			for _, address := range outAddresses {
				if !IsAddressIn(address, addresses) {
					continue
				}

				// TODO(cedric): Update the definition of 'spendable' to be:
				// If the shortest distance between input and output along bluest blocks between the two transactions
				// is >= coinbase maturity.
				if tips.MaxHeight > info.BlockHeight + int32(params.CoinbaseMaturity) {
					// Locally bind info to a local variable, to keep the Info field pointing at the correct
					// TxInfo struct as the loop continues.
					matchInfo := info
					m := TxMatch{
						Address: address.EncodeAddress(),
						Amount:  soterutil.Amount(txOut.Value),
						VIndex:  i,
						Info:    &matchInfo,
					}

					matches = append(matches, m)
				}
			}
		}
	}

	return matches, nil
}