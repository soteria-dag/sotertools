// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterwallet/waddrmgr"
	"github.com/soteria-dag/soterwallet/wallet"
	"github.com/soteria-dag/soterwallet/walletdb"
	// This is imported, so that the wallet driver is included in this program when compiled
	_ "github.com/soteria-dag/soterwallet/walletdb/bdb"
)

const (
	defaultWalletName = "wallet.db"

	walletDbType = "bdb"

	// The recoveryWindow is used when attempting to find unspent outputs that pay to any of our wallet's addresses.
	// We won't use this, but we need to specify a value when calling wallet.Open().
	// Here we use the github.com/soteria-dag/soterwallet/walletsetup.go createWallet function's default of 250
	recoveryWindow = 250
)

var (
	waddrmgrNamespaceKey = []byte("waddrmgr")
)

// fileExists returns true if a file with the name exists
func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err != nil {
		return false
	}

	return true
}

// createWallet creates a wallet
// NOTE(cedric): Based on github.com/soteria-dag/soterwallet/walletsetup.go createSimulationWallet function
func createWallet(name, privPass, pubPass string, netParams *chaincfg.Params) error {
	priv := []byte(privPass)
	pub := []byte(pubPass)

	walletDir, _ := filepath.Split(name)
	err := os.MkdirAll(walletDir, 0750)
	if err != nil {
		return err
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

// newAddress creates and returns a new address for an account in a wallet.
// It was taken from github.com/soteria-dag/soterwallet/wallet/wallet.go, because the w.NewAddress was meant to be run from
// the context of a running soterwallet process.
func newAddress(w *wallet.Wallet, account uint32, scope waddrmgr.KeyScope) (soterutil.Address, error) {
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

// soterwalletPath returns a default soterwallet file name
func soterwalletPath(name string, params *chaincfg.Params) (string, error) {
	base := soterutil.AppDataDir("soterwallet", false)
	return filepath.Join(base, params.Name, name), nil
}

func main() {
	var mainnet, testnet, simnet bool
	var walletName, privPass, pubPass string

	// Parse cli parameters
	flag.BoolVar(&mainnet, "mainnet", false, "Use mainnet params for wallet")
	flag.BoolVar(&testnet, "testnet", false, "Use testnet params for wallet")
	flag.BoolVar(&simnet, "simnet", false, "Use simnet params for wallet")
	flag.StringVar(&walletName, "w", "", "Wallet file name")
	flag.StringVar(&privPass, "priv", "", "Password to use, for unlocking address manager (for private keys and info)")
	flag.StringVar(&pubPass, "pub", "", "Password to use, for opening address manager")

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
	if len(privPass) == 0 {
		fmt.Println("WARNING: -priv (private password) is not set!")
	}
	if len(pubPass) == 0 {
		fmt.Println("WARNING: -pub (pub password) is not set!")
	}

	var err error
	exists := fileExists(walletName)
	if !exists && activeNetParams == nil {
		fmt.Println("Need to specify which net params to use, when creating a new wallet (-mainnet, -testnet, -simnet)")
		os.Exit(1)
	}

	if len(walletName) == 0 {
		// Use a default wallet path
		walletName, err = soterwalletPath(defaultWalletName, activeNetParams)
		if err != nil {
			fmt.Printf("Failed to determine default wallet path: %s\n", err)
			os.Exit(1)
		}
	}

	if !exists {
		// Create wallet
		err = createWallet(walletName, privPass, pubPass, activeNetParams)
		if err != nil {
			fmt.Printf("Failed to create wallet: %s", err)
			os.Exit(1)
		}
		fmt.Printf("Created wallet: %s\n", walletName)
	}

	// Open wallet
	w, err := openWallet(walletName, pubPass, activeNetParams)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer w.Database().Close()
	fmt.Printf("Opened wallet %s\n", walletName)

	// List accounts
	resp, err := w.Accounts(waddrmgr.KeyScopeBIP0044)
	if err != nil {
		fmt.Printf("Failed to retrieve accounts from wallet: %s", err)
		os.Exit(1)
	}
	fmt.Println("Accounts:")
	for _, a := range resp.Accounts {
		fmt.Printf("\tname: %s\tnumber: %d\tbalance: %s\n", a.AccountName, a.AccountNumber, a.TotalBalance)

		// List addresses for each account
		addresses, err := w.AccountAddresses(a.AccountNumber)
		if err != nil {
			fmt.Printf("Failed to retrieve account addresses for account %s (%d): %s\n", a.AccountName, a.AccountNumber, err)
			return
		}

		for _, addr := range addresses {
			fmt.Printf("\t\taddress: %s\n", addr)
		}

		if a.AccountName == "default" && len(addresses) == 0 {
			// Create an address that could be used for transactions
			newAddr, err := newAddress(w, a.AccountNumber, waddrmgr.KeyScopeBIP0044)
			if err != nil {
				fmt.Printf("Failed to create new address for account %s (%d): %s\n", a.AccountName, a.AccountNumber, err)
				return
			}

			fmt.Printf("\t\taddress: %s\n", newAddr.EncodeAddress())
		}
	}
}
