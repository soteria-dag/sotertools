// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"fmt"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterwallet/waddrmgr"
	"github.com/soteria-dag/soterwallet/wallet"
	"github.com/soteria-dag/soterwallet/walletdb"
	// This is imported, so that the wallet driver is included in this program when compiled
	_ "github.com/soteria-dag/soterwallet/walletdb/bdb"
	"os"
	"path/filepath"
	"time"
)

const (
	walletDbType = "bdb"

	// The recoveryWindow is used when attempting to find unspent outputs that pay to any of our wallet's addresses.
	// We won't use this, but we need to specify a value when calling wallet.Open().
	// Here we use the github.com/soteria-dag/soterwallet/walletsetup.go createWallet function's default of 250
	recoveryWindow = 250
)

// newWalletAddress creates and returns a new address for an account in a wallet.
// It was taken from github.com/soteria-dag/soterwallet/wallet/wallet.go, because it was an unexported method.
func newWalletAddress(w *wallet.Wallet, addrmgrNs walletdb.ReadWriteBucket, account uint32,
	scope waddrmgr.KeyScope) (soterutil.Address, *waddrmgr.AccountProperties, error) {

	manager, err := w.Manager.FetchScopedKeyManager(scope)
	if err != nil {
		return nil, nil, err
	}

	// Get next address from wallet.
	addrs, err := manager.NextExternalAddresses(addrmgrNs, account, 1)
	if err != nil {
		return nil, nil, err
	}

	props, err := manager.AccountProperties(addrmgrNs, account)
	if err != nil {
		errMsg := fmt.Errorf("Cannot fetch account properties for notification "+
			"after deriving next external address: %v", err)
		return nil, nil, errMsg
	}

	return addrs[0].Address(), props, nil
}

// CreateWallet creates a wallet
// NOTE(cedric): Based on github.com/soteria-dag/soterwallet/walletsetup.go createSimulationWallet function
func CreateWallet(name, privPass, pubPass string, netParams *chaincfg.Params) error {
	priv := []byte(privPass)
	pub := []byte(pubPass)

	walletDir, _ := filepath.Split(name)

	if len(walletDir) > 0 {
		err := os.MkdirAll(walletDir, 0750)
		if err != nil {
			return err
		}
	}

	// Create the wallet database backed by bolt db.
	db, err := walletdb.Create(walletDbType, name)
	if err != nil {
		return err
	}
	defer db.Close()

	// Initialize wallet db, creating the wallet
	err = wallet.Create(db, pub, priv, nil, netParams, time.Now())
	if err != nil {
		return err
	}

	return nil
}

// OpenWallet opens a wallet db, then wallet from it
func OpenWallet(name, pubPass string, params *chaincfg.Params) (*wallet.Wallet, error) {
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

// NewAddress creates and returns a new address for an account in a wallet.
// It was taken from github.com/soteria-dag/soterwallet/wallet/wallet.go, because the w.NewAddress was meant to be run from
// the context of a running soterwallet process.
func NewAddress(w *wallet.Wallet, account uint32, scope waddrmgr.KeyScope) (soterutil.Address, error) {
	var (
		addr  soterutil.Address
	)
	err := walletdb.Update(w.Database(), func(tx walletdb.ReadWriteTx) error {
		addrmgrNs := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		var err error
		addr, _, err = newWalletAddress(w, addrmgrNs, account, scope)
		return err
	})
	if err != nil {
		return nil, err
	}

	return addr, nil
}

// WalletAddresses returns a slice of addresses found in the wallet
func WalletAddresses(w *wallet.Wallet) ([]soterutil.Address, error) {
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