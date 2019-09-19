// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"github.com/soteria-dag/soterd/blockdag"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/soteria-dag/soterd/soterutil"
	"math/rand"
	"testing"
	"time"
)

func TestGetBalance(t *testing.T) {
	var activeNet = &chaincfg.SimNetParams
	var miners []*rpctest.Harness
	var minerAddresses = make(map[int][]soterutil.Address)

	// Number of miners to spawn
	// (when using SetUp to initialize the test dag, you don't need another miner to propagate the blocks to)
	minerCount := 1
	// Set to true, to retain logs from miners
	keepLogs := false
	// Set to debug or trace to produce more logging output from miners.
	extraArgs := []string{
		//"--debuglevel=debug",
	}
	// How many spendable outputs there should be
	rand.Seed(time.Now().Unix())
	numMatureOutputs := uint32(rand.Intn(10))

	for i := 0; i < minerCount; i++ {
		miner, err := rpctest.New(activeNet, nil, extraArgs, keepLogs)
		if err != nil {
			t.Fatalf("unable to create mining node %v: %v", i, err)
		}

		if err := miner.SetUp(true, numMatureOutputs); err != nil {
			t.Fatalf("unable to complete mining node %v setup: %v", i, err)
		}

		if keepLogs {
			t.Logf("miner %d log dir: %s", i, miner.LogDir())
		}

		miners = append(miners, miner)

		minerAddresses[i] = miner.Addresses()
	}
	// NOTE(cedric): We'll call defer on a single anonymous function instead of minerCount times in the above loop
	defer func() {
		for _, miner := range miners {
			_ = (*miner).TearDown()
		}
	}()

	for minerNum, addresses := range minerAddresses {
		miner := miners[minerNum]
		balance, spendable, err := GetBalance(miner.Node, addresses, activeNet)
		if err != nil {
			t.Fatalf("failed to get balance of addresses %s: %s", addresses, err)
		}

		tips, err := miner.Node.GetDAGTips()
		if err != nil {
			t.Fatalf("failed to get dag tips: %s", err)
		}

		// We don't count the genesis block, when calculating how much coin should have been produced
		blockCount := tips.BlkCount - 1
		blockSubsidy := blockdag.CalcBlockSubsidy(tips.MaxHeight, activeNet)
		totalSubsidy := float64(blockSubsidy * int64(blockCount))
		expectedBalance := soterutil.Amount(totalSubsidy)

		if balance != expectedBalance {
			t.Errorf("wrong SOTER balance; got %s, want %s", balance, expectedBalance)
		}

		totalMatureSubsidy := float64(blockSubsidy * int64(numMatureOutputs))
		expectedSpendable := soterutil.Amount(totalMatureSubsidy)

		if spendable != expectedSpendable {
			t.Errorf("wrong SOTER spendable balance; got %s, want %s", spendable, expectedSpendable)
		}

		// The spendable balance should match what the miner's in-memory wallet is tracking
		memWalletSpendable := miner.ConfirmedBalance()
		if spendable != memWalletSpendable {
			t.Errorf("wrong SOTER spendable balance compared to memwallet; got %s, want %s", spendable, memWalletSpendable)
		}

		t.Logf("numMatureOutputs %d, balance: %s, spendable: %s", numMatureOutputs, balance, spendable)
	}
}


