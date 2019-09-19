// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/soteria-dag/sotertools/wallet"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
)

// Represents address balance info that we're interested in rendering
type balanceInfo struct {
	Address string
	Balance soterutil.Amount
	Spendable soterutil.Amount
}

// getBalance returns a balanceInfo
func getBalance(c *rpcclient.Client, address string) (balanceInfo, error) {
	info := balanceInfo{
		Address:   address,
	}

	addr, err := soterutil.DecodeAddress(address, activeNetParams)
	if err != nil {
		return info, err
	}

	balance, spendable, err := wallet.GetBalance(c, []soterutil.Address{addr}, activeNetParams)
	if err != nil {
		return info, err
	}

	info.Balance = balance
	info.Spendable = spendable

	return info, nil
}