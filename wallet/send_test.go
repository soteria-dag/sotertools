// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/chaincfg/chainhash"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterd/wire"
	"github.com/soteria-dag/soterwallet/waddrmgr"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

// waitForTx waits for the given transaction hash to appear in a block. It returns:
// - Whether the transaction was found within wait time (bool)
// - The block the transaction was found in
// - The height of the first block the transaction was found in
// - An error, if an error was encountered during the search
func waitForTx(c *rpcclient.Client, txHash *chainhash.Hash, startHeight int32, wait time.Duration) (bool, *wire.MsgBlock, int32, error) {
	pollInterval := time.Duration(time.Second * 4)
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
		for height := tips.MaxHeight; height >= 0; height-- {
			hashes, err := c.GetBlockHash(int64(height))
			if err != nil {
				continue AFTERNAP
			}

			for _, hash := range hashes {
				block, err := c.GetBlock(hash)
				if err != nil {
					continue AFTERNAP
				}

				txs := make([]string, 0)
				for _, tx := range block.Transactions {
					txs = append(txs, tx.TxHash().String())
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

func TestSend(t *testing.T) {
	var activeNet = &chaincfg.SimNetParams
	// Number of miners to spawn
	var minerCount = 2
	var blockCount = uint32(500 + rand.Intn(500))
	// Max time to wait for blocks to sync between nodes, once all are generated
	wait := time.Second * 30
	// Set to true, to retain logs from miners
	var keepLogs = false
	// extraArgs contents will be defined later, after a wallet is generated
	var extraArgs []string
	var miners []*rpctest.Harness
	var minerAddresses = make([]soterutil.Address, 0)
	var destAddr = "Sh7EBrov7iZqbMiYe6kPn3ebaBevB7DcH3"
	var dest soterutil.Address
	var privPass = "priv"
	var pubPass = "pub"

	dest, err := soterutil.DecodeAddress(destAddr, activeNet)
	if err != nil {
		t.Fatalf("failed to decode address %s: %s", destAddr, err)
	}

	tmpfile, err := ioutil.TempFile("", "TestSend_wallet-*.db")
	if err != nil {
		t.Fatalf("failed to create wallet file: %s", err)
	}
	var walletName = tmpfile.Name()
	_ = tmpfile.Close()
	_ = os.Remove(walletName)

	err = CreateWallet(walletName, privPass, pubPass, activeNet)
	if err != nil {
		t.Fatalf("failed to create wallet %s: %s", walletName, err)
	}

	w, err := OpenWallet(walletName, pubPass, activeNet)
	if err != nil {
		t.Fatalf("failed to open wallet %s: %s", walletName, err)
	}
	defer func() {
		_ = w.Database().Close()
		_ = os.Remove(walletName)
	}()

	resp, err := w.Accounts(waddrmgr.KeyScopeBIP0044)
	if err != nil {
		t.Fatalf("failed to retrieve accounts from wallet: %s", err)
	}

	for _, a := range resp.Accounts {
		addresses, err := w.AccountAddresses(a.AccountNumber)
		if err != nil {
			t.Fatalf("failed to retrieve addresses for account %s, number %d: %s",
				a.AccountName, a.AccountNumber, err)
		}

		for _, address := range addresses {
			minerAddresses = append(minerAddresses, address)
		}

		if a.AccountName == "default" && len(minerAddresses) == 0 {
			// Create an address that could be used for transactions
			newAddr, err := NewAddress(w, a.AccountNumber, waddrmgr.KeyScopeBIP0044)
			if err != nil {
				fmt.Printf("Failed to create new address for account %s (%d): %s\n", a.AccountName, a.AccountNumber, err)
				return
			}

			minerAddresses = append(minerAddresses, newAddr)
		}
	}

	// Set to debug or trace to produce more logging output from miners.
	extraArgs = []string{
		//"--debuglevel=debug",
	}

	for _, a := range minerAddresses {
		extraArgs = append(extraArgs, "--miningaddr=" + a.EncodeAddress())
	}

	for i := 0; i < minerCount; i++ {
		miner, err := rpctest.New(activeNet, nil, extraArgs, keepLogs)
		if err != nil {
			t.Fatalf("unable to create mining node %v: %v", i, err)
		}

		if err := miner.SetUp(false, 0); err != nil {
			t.Fatalf("unable to complete mining node %v setup: %v", i, err)
		}

		if keepLogs {
			t.Logf("miner %d log dir: %s", i, miner.LogDir())
		}

		miners = append(miners, miner)
	}
	// NOTE(cedric): We'll call defer on a single anonymous function instead of minerCount times in the above loop
	defer func() {
		for _, miner := range miners {
			_ = (*miner).TearDown()
		}
	}()

	// Connect the nodes to one another
	err = rpctest.ConnectNodes(miners)
	if err != nil {
		t.Fatalf("unable to connect nodes: %v", err)
	}

	// Mine some blocks on the first miner
	_, err = miners[0].Node.Generate(blockCount)
	if err != nil {
		t.Fatalf("failed to generate blocks: %s", err)
	}

	err = rpctest.WaitForDAG(miners, wait)
	if err != nil {
		t.Fatalf("block sync failed: %s", err)
	}

	matches, err := SpendableTxOuts(miners[0].Node, minerAddresses, activeNet)
	if err != nil {
		t.Fatalf("failed to find matching transactions in dag: %s", err)
	}

	if len(matches) == 0 {
		t.Fatalf("no matching transactions for source address found in dag")
	}

	spendable := soterutil.Amount(0)
	for _, m := range matches {
		spendable += m.Amount
	}
	t.Logf("spendable of %s: %s", minerAddresses, spendable)

	// Send some coin to the destination
	smallest, err := soterutil.NewAmount(1)
	if err != nil {
		t.Fatalf("failed to convert into SOTER: %s", err)
	}
	if spendable < smallest {
		t.Fatalf("there's not enough spendable coin")
	}

	feeAmount := smallest
	sending := float64(int(smallest.ToSOTER()) + rand.Intn(int((spendable - feeAmount).ToSOTER())))
	sendAmount, err := soterutil.NewAmount(sending)
	if err != nil {
		t.Fatalf("failed to convert into SOTER: %s", err)
	}

	if sendAmount + feeAmount > spendable {
		t.Fatalf("not enough coin found to satisfy amount requested for transaction; %s requested + %s fee, %s spendable",
			sendAmount, feeAmount, spendable)
	}

	txHash, err := Send(miners[0].Node, w, privPass, matches, dest, sendAmount, feeAmount)
	if err != nil {
		t.Fatalf("failed to send coin: %s", err)
	}

	// Generate some more blocks, to have the transaction included in a block on the network.
	_, err = miners[0].Node.Generate(10)
	if err != nil {
		t.Fatalf("failed to generate blocks: %s", err)
	}

	found, _, _, err := waitForTx(miners[0].Node, txHash, 0, wait)
	if err != nil {
		t.Fatalf("failed to find transaction %s in dag: %s", txHash, err)
	} else if !found {
		t.Fatalf("failed to find transaction %s in dag", txHash)
	}

	// Check balance of dest address
	destAddresses := []soterutil.Address{dest}
	balance, _, err := GetBalance(miners[0].Node, destAddresses, activeNet)
	if err != nil {
		t.Fatalf("failed to get balance of addresses %s: %s", destAddresses, err)
	}

	if balance != sendAmount {
		t.Fatalf("sent wrong amount of coin; got %s, want %s", balance, sendAmount)
	}

	t.Logf("sent %s coin (fee %s) to %s", balance, feeAmount, dest)
}