// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterjson"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/txscript"
	"github.com/soteria-dag/soterd/wire"
	"github.com/soteria-dag/soterwallet/waddrmgr"
	"github.com/soteria-dag/soterwallet/wallet"
	"github.com/soteria-dag/soterwallet/walletdb"
	// This is imported, so that the wallet driver is included in this program when compiled
	_ "github.com/soteria-dag/soterwallet/walletdb/bdb"
)

const (
	walletDbType = "bdb"

	// The recoveryWindow is used when attempting to find unspent outputs that pay to any of our wallet's addresses.
	// We won't use this, but we need to specify a value when calling wallet.Open().
	// Here we use the github.com/soteria-dag/soterwallet/walletsetup.go createWallet function's default of 250
	recoveryWindow = 250
)

var (
	maxTxWait = time.Second * 30

	// This variable is used when unlocking a wallet
	waddrmgrNamespaceKey = []byte("waddrmgr")
)

// MockAddr is an implementation of soterutil.Address that provides enough functionality to satisfy the
// createrawtransaction RPC call.
type MockAddr struct {
	value string
}

// txMatch represents a match when finding a transaction that matches an address in a wallet.
// It's used to help generate new transactions using coins from a wallet.
type txMatch struct {
	// Which wallet address matched
	walletAddr string
	// The number of the matching output in the transaction
	vOut int
	// A pointer to the matching transaction. We'll use this to access values like:
	// tx.TxHash (used as txid field in createrawtransaction call)
	// tx.TxOut[vOut].Value, via the Amount() method
	tx *wire.MsgTx
	// A pointer to the block containing the matching transaction
	block *wire.MsgBlock
	blockHeight int32
}

// String returns a string value of the MockAddr soterutil.Address interface implementation
func (ma *MockAddr) String() string {
	return ma.value
}

// EncodeAddress returns an empty value, because it's not meant to be used but is required to meet the soterutil.Address
// interface implementation.
func (ma *MockAddr) EncodeAddress() string {
	return ""
}

// ScriptAddress returns an empty byte value, because it's not meant to be used but is required to meet the
// soterutil.Address interface implementation.
func (ma *MockAddr) ScriptAddress() []byte {
	return []byte{}
}

// IsForNet always returns true, because it's not meant ot be used but is required to meet the soterutil.Address
// interface implementation.
func (ma *MockAddr) IsForNet(p *chaincfg.Params) bool {
	return true
}

// Amount returns the coin value of the matching transaction
func (m *txMatch) Amount() soterutil.Amount {
	return soterutil.Amount(m.tx.TxOut[m.vOut].Value)
}

func (m *txMatch) PKScript() []byte {
	return m.tx.TxOut[m.vOut].PkScript
}

// absAmount returns the absolute value of the amount
func absAmount(amt soterutil.Amount) soterutil.Amount {
	if amt < soterutil.Amount(0) {
		return -amt
	} else {
		return amt
	}
}

// connectRPC returns an RPC client connection
func connectRPC(host, user, pass, certPath string) (*rpcclient.Client, error) {
	// Attempt to read certs
	certs := []byte{}
	var readCerts []byte
	var err error
	if len(certPath) > 0 {
		readCerts, err = ioutil.ReadFile(certPath)
	} else {
		// Try a default cert path
		soterdDir := soterutil.AppDataDir("soterd", false)
		readCerts, err = ioutil.ReadFile(filepath.Join(soterdDir, "rpc.cert"))
	}
	if err == nil {
		certs = readCerts
	}

	rpcConf := rpcclient.ConnConfig{
		Host: host,
		Endpoint: "ws",
		User: user,
		Pass: pass,
		Certificates: certs,
		DisableAutoReconnect: true,
	}

	client, err := rpcclient.New(&rpcConf, nil)
	if err != nil {
		return client, err
	}

	return client, nil
}

// openWallet opens a wallet db, then wallet from it
func openWallet(name, pubPass string, params *chaincfg.Params) (*wallet.Wallet, error) {
	// Open wallet db
	db, err := walletdb.Open(walletDbType, name)
	if err != nil {
		return nil, fmt.Errorf("Failed to open wallet db %s: %s", name, err)
	}

	// Open wallet
	w, err := wallet.Open(db, []byte(pubPass), nil, params, recoveryWindow)
	if err != nil {
		return nil, fmt.Errorf("Failed to open wallet: %s", err)
	}

	return w, nil
}

// walletAddresses returns a slice of addresses found in the wallet
func walletAddresses(w *wallet.Wallet) ([]soterutil.Address, error) {
	addresses := make([]soterutil.Address, 0)

	resp, err := w.Accounts(waddrmgr.KeyScopeBIP0044)
	if err != nil {
		return addresses, err
	}

	for _, a := range resp.Accounts {
		addrs, err := w.AccountAddresses(a.AccountNumber)
		if err != nil {
			return addresses, err
		}

		for _, addr := range addrs {
			addresses = append(addresses, addr)
		}
	}

	return addresses, nil
}

// matchTx returns a slice of transactions that match the addresses in the wallet
func matchTxOut(walletAddrs []soterutil.Address, transactions []*wire.MsgTx, params *chaincfg.Params) ([]txMatch, error) {
	matches := make([]txMatch, 0)

	for _, tx := range transactions {
		// Look for matches in the outputs of the transaction
		for i, out := range tx.TxOut {
			// Extract output addresses from the script in the output
			_, outAddrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, params)
			if err != nil {
				return matches, err
			}

			for _, oa := range outAddrs {
				if !oa.IsForNet(params) {
					continue
				}

				// Match the output against addresses in our wallet
				for _, wAddr := range walletAddrs {
					if wAddr.EncodeAddress() == oa.EncodeAddress() {
						m := txMatch{
							walletAddr: wAddr.EncodeAddress(),
							vOut: i,
							tx: tx,
						}
						matches = append(matches, m)
						break
					}
				}
			}
		}
	}

	return matches, nil
}

// matchingTxs returns a slice of transactions from the dag, that match the addresses from the wallet
func matchingTxs(w *wallet.Wallet, c *rpcclient.Client, params *chaincfg.Params) ([]txMatch, error) {
	txs := make([]txMatch, 0)

	addresses, err := walletAddresses(w)
	if err != nil {
		return txs, err
	}

	tips, err := c.GetDAGTips()
	if err != nil {
		return txs, err
	}

	// Look at blocks from lowest height before genesis to highest.
	// We'll start low because there is a required number of blocks that need to exist after the transaction,
	// before coins can be spent (params.CoinbaseMaturity).
	// When spending the coins we'll default to using transactions from lowest-to-highest height.
	for height := int32(0); height <= tips.MaxHeight; height++ {
		hashes, err := c.GetBlockHash(int64(height))
		if err != nil {
			return txs, err
		}

		for _, hash := range hashes {
			block, err := c.GetBlock(hash)
			if err != nil {
				return txs, err
			}

			// Find matching transactions in the block's transactions' outputs
			matches, err := matchTxOut(addresses, block.Transactions, params)
			if err != nil {
				return txs, err
			}

			for _, m := range matches {
				// Add block info to match
				m.block = block
				m.blockHeight = height
				txs = append(txs, m)
			}
		}
	}

	return txs, nil
}

// makeTxInputs creates some transaction inputs that can be used in the createrawtransaction RPC call
func makeTxInputs(matches []txMatch, desiredAmt soterutil.Amount) []soterjson.TransactionInput {
	txIns := make([]soterjson.TransactionInput, 0)

	remainingAmt := desiredAmt
	for _, m := range matches {
		t := soterjson.TransactionInput{
			Txid: m.tx.TxHash().String(),
			Vout: uint32(m.vOut),
		}
		txIns = append(txIns, t)
		remainingAmt -= m.Amount()
		if remainingAmt <= soterutil.Amount(0) {
			break
		}
	}

	return txIns
}

// makeTxAmts creates a map of which addresses will recieve what amount of coin in a transaction.
// It can be used in the createrawtransaction RPC call.
func makeTxAmts(matches []txMatch, destAddr string, desiredAmt soterutil.Amount) map[soterutil.Address]soterutil.Amount {
	amounts := make(map[soterutil.Address]soterutil.Amount)
	// We need to declare that dest is of soterutil.Address type, in order for it to work as a key in the amounts map.
	var dest soterutil.Address
	dest = &MockAddr{value: destAddr}
	amounts[dest] = soterutil.Amount(0)

	remainingAmt := desiredAmt
	for _, m := range matches {
		if remainingAmt - m.Amount() < soterutil.Amount(0) {
			// The abs difference between the remainingAmt below zero is sent back to the owner of the coin, and
			// the remainingAmt to zero is sent to the destAddr.
			extraAmt := absAmount(remainingAmt - m.Amount())

			var owner soterutil.Address
			owner = &MockAddr{value: m.walletAddr}
			_, exists := amounts[owner]
			if exists {
				amounts[owner] += extraAmt
			} else {
				amounts[owner] = extraAmt
			}

			amounts[dest] += (m.Amount() - extraAmt)
		} else {
			amounts[dest] += m.Amount()
		}

		remainingAmt -= m.Amount()
		if remainingAmt <= soterutil.Amount(0) {
			break
		}
	}

	return amounts
}

// makePrevScripts returns a mapping of scripts from outputs that could be used in a transaction. This is meant for
// preventing soterwallet functions from attempting to look up info internally that it wouldn't have, due to us not
// running a full soterwallet node.
func makePrevScripts(matches []txMatch) map[wire.OutPoint][]byte {
	prevScripts := make(map[wire.OutPoint][]byte)

	for _, m := range matches {
		txHash := m.tx.TxHash()
		op := wire.NewOutPoint(&txHash, uint32(m.vOut))
		prevScripts[*op] = m.PKScript()
	}

	return prevScripts
}

// waitForTx waits for the given transaction hash to appear in a block. It returns:
// - Whether the transaction was found within wait time (bool)
// - The block the transaction was found in
// - The height of the first block the transaction was found in
// - An error, if an error was encountered during the search
func waitForTx(c *rpcclient.Client, txHash *chainhash.Hash, startHeight int32, wait time.Duration) (bool, *wire.MsgBlock, int32, error) {
	pollInterval := time.Duration(time.Second)
	waitThreshold := time.Now().Add(wait)

	pollCount := 0
	for {
		if pollCount > 0 {
			if time.Now().Before(waitThreshold) {
				time.Sleep(pollInterval)
			} else {
				timeout := fmt.Errorf("Timeout while waiting for transaction")
				return false, nil, -1, timeout
			}
		}
		pollCount++

		tips, err := c.GetDAGTips()
		if err != nil {
			continue
		}

		// Look at blocks from starting height to tips for transaction
		AFTERNAP:
		for height := startHeight; height <= tips.MaxHeight; height++ {
			hashes, err := c.GetBlockHash(int64(height))
			if err != nil {
				continue AFTERNAP
			}

			for _, hash := range hashes {
				block, err := c.GetBlock(hash)
				if err != nil {
					continue AFTERNAP
				}

				for _, tx := range block.Transactions {
					h := tx.TxHash()
					if h.IsEqual(txHash) {
						return true, block, height, nil
					}
				}
			}
		}
	}
}

func main() {
	var mainnet, testnet, simnet bool
	var walletName, privPass, pubPass, destAddr, rpcSrv, rpcUser, rpcPass, rpcCert string
	var amt float64

	// Parse cli parameters
	flag.BoolVar(&mainnet, "mainnet", false, "Use mainnet params for wallet")
	flag.BoolVar(&testnet, "testnet", false, "Use testnet params for wallet")
	flag.BoolVar(&simnet, "simnet", false, "Use simnet params for wallet")
	flag.StringVar(&walletName, "w", "", "Source wallet file name")
	flag.StringVar(&privPass, "priv", "", "Password to use, for unlocking address manager (for private keys and info)")
	flag.StringVar(&pubPass, "pub", "", "Password to use, for opening address manager")
	flag.StringVar(&destAddr, "dest", "", "Destination address of funds")
	flag.Float64Var(&amt, "amt", float64(0), "Amount of coin to transfer (SOTO)")
	flag.StringVar(&rpcSrv, "rpcserver", "", "Soterd RPC server to send transaction to (ip:port)")
	flag.StringVar(&rpcUser, "rpcuser", "", "Soterd RPC server username to use")
	flag.StringVar(&rpcPass, "rpcpass", "", "Soterd RPC server password to use")
	flag.StringVar(&rpcCert, "rpccert", "", "Soterd RPC server cert chain")

	flag.Parse()

	var activeNetParams *chaincfg.Params
	selectedNets := 0
	if mainnet {
		selectedNets++
		activeNetParams = &chaincfg.MainNetParams
	}
	if testnet {
		selectedNets++
		activeNetParams = &chaincfg.TestNet1Params
	}
	if simnet {
		selectedNets++
		activeNetParams = &chaincfg.SimNetParams
	}

	// Validate cli parameters
	if selectedNets > 1 {
		fmt.Println("You can only specify one net param (-mainnet, -testnet, -simnet)")
		os.Exit(1)
	}
	if len(destAddr) == 0 {
		fmt.Println("No destination address specified (-dest)")
		os.Exit(1)
	}
	if len(privPass) == 0 {
		fmt.Println("WARNING: -priv (private password) is not set!")
	}
	if len(pubPass) == 0 {
		fmt.Println("WARNING: -pub (pub password) is not set!")
	}
	if amt == 0 {
		fmt.Printf("WARNING: SOTO amount to transfer is %f\n", amt)
	}

	// Open wallet
	w, err := openWallet(walletName, pubPass, activeNetParams)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer w.Database().Close()
	fmt.Printf("Opened wallet %s\n", walletName)

	fmt.Println()

	// Connect to soterd node
	client, err := connectRPC(rpcSrv, rpcUser, rpcPass, rpcCert)
	if err != nil {
		fmt.Printf("RPC connection to %s failed: %s\n", rpcSrv, err)
		os.Exit(1)
	}

	// Scan through blocks for transactions with outputs addrs matching our wallet addrs
	matches, err := matchingTxs(w, client, activeNetParams)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(matches) == 0 {
		fmt.Println("No transactions matching wallet addresses found in dag, so I have no coins to spend in a new transaction")
		os.Exit(2)
	}

	fmt.Println("Transactions matching wallet addresses:")
	txTotalAmt := soterutil.Amount(0)
	for _, m := range matches {
		txTotalAmt += m.Amount()
		fmt.Printf("block %s\theight %d\ttx %s\toutputNum %d\tvalue %s\tmatching wallet addr %s\n", m.block.BlockHash(), m.blockHeight, m.tx.TxHash(), m.vOut, m.Amount(), m.walletAddr)
	}

	// Confirm that there's enough coin available for the transaction
	destAmt, err := soterutil.NewAmount(amt)
	if err != nil {
		fmt.Printf("Can't convert dest amount %f to a soter amount: %s\n", amt, err)
		os.Exit(1)
	}

	if destAmt > txTotalAmt {
		fmt.Printf("Not enough coin found to satisfy the amount requested for the transaction: %s requested > %s available\n", destAmt, txTotalAmt)
		os.Exit(3)
	}

	fmt.Println()

	// Create new transaction
	fmt.Printf("Creating a transaction for %s, to %s\n", destAmt, destAddr)
	txIns := makeTxInputs(matches, destAmt)
	txAmts := makeTxAmts(matches, destAddr, destAmt)
	fmt.Println("Wallet output transactions being used as inputs for this new transaction:")
	for _, txi := range txIns {
		fmt.Printf("tx %s\toutputNum %d\n", txi.Txid, txi.Vout)
	}
	fmt.Println()
	fmt.Println("Output amounts for this new transaction")
	for addr, v := range txAmts {
		fmt.Printf("addr %s\tvalue %s\n", addr, v)
	}

	fmt.Println()

	// Have the soterd node translate our inputs and amounts into a raw transaction
	tx, err := client.CreateRawTransaction(txIns, txAmts, nil)
	if err != nil {
		fmt.Printf("createrawtransaction RPC call failed: %s\n", err)
		os.Exit(1)
	}

	// Build a map of scripts from the outputs that are used as inputs in the new transaction.
	// This is so that the w.SignTransaction method won't attempt to look up this information in its own records,
	// which it won't have because we aren't running a full soterwallet node.
	prevScripts := makePrevScripts(matches)

	fmt.Println("Unlocking wallet")
	err = walletdb.View(w.Database(), func(tx walletdb.ReadTx) error {
		addrmgrNs := tx.ReadBucket(waddrmgrNamespaceKey)
		return w.Manager.Unlock(addrmgrNs, []byte(privPass))
	})
	if err != nil {
		fmt.Printf("Failed to unlock wallet: %s\n", err)
		os.Exit(1)
	}

	// Sign the transaction
	fmt.Println("Signing transaction")
	invalidSigs, err := w.SignTransaction(tx, txscript.SigHashAll, prevScripts, nil, nil)
	if err != nil {
		fmt.Printf("Failed to sign transaction: %s\n", err)
		os.Exit(1)
	}
	for _, e := range invalidSigs {
		fmt.Printf("Unsigned input at index: %d\n", e.InputIndex)
	}

	// Get the current tips height, to help us determine how far back in dag we should look for our transaction once sent.
	tips, err := client.GetDAGTips()
	if err != nil {
		fmt.Printf("Failed to getdagtips: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Sending transaction")

	// Send the transaction to the network
	txHash, err := client.SendRawTransaction(tx, false)
	if err != nil {
		fmt.Printf("Failed to send transaction to network: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sent transaction with hash %s\n", txHash)

	// Confirm that coins have been transferred, by looking for a new block containing the transaction.
	// NOTE(cedric): At this point you may need to trigger block-generation, if the network isn't mining blocks on its own.
	fmt.Println("Waiting for transaction to appear in block. If the network isn't mining blocks already, please trigger mining now :)")
	found, block, height, err := waitForTx(client, txHash, tips.MaxHeight, maxTxWait)
	if err != nil {
		fmt.Printf("Failed to wait for transaction: %s\n", err)
		os.Exit(1)
	}
	if found {
		fmt.Println("Transaction found!")
		fmt.Printf("block %s\theight %d\ttx %s\n", block.BlockHash(), height, txHash)
		for _, t := range block.Transactions {
			for i, out := range t.TxOut {
				_, outAddrs, _, err := txscript.ExtractPkScriptAddrs(out.PkScript, activeNetParams)
				if err != nil {
					continue
				}
				for _, oa := range outAddrs {
					fmt.Printf("\toutput %d\thash %s\tvalue %s\n", i, oa.EncodeAddress(), soterutil.Amount(out.Value))
				}
			}
		}
	} else {
		fmt.Println("Transaction not found!")
	}

	// TODO(cedric): Update wallet with sent transaction info, or look for spent transactions when looking for matching
	// transactions in dag. This would allow us to run this program multiple times without triggering an error for
	// double-spending coins.
}