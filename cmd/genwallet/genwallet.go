// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/soteria-dag/sotertools/wallet"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/soterutil"
	"github.com/soteria-dag/soterwallet/waddrmgr"
	// This is imported, so that the wallet driver is included in this program when compiled
	_ "github.com/soteria-dag/soterwallet/walletdb/bdb"
)

const (
	defaultWalletName = "wallet.db"
)

// fileExists returns true if a file with the name exists
func fileExists(name string) bool {
	_, err := os.Stat(name)
	if err != nil {
		return false
	}

	return true
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
		err = wallet.CreateWallet(walletName, privPass, pubPass, activeNetParams)
		if err != nil {
			fmt.Printf("Failed to create wallet: %s", err)
			os.Exit(1)
		}
		fmt.Printf("Created wallet: %s\n", walletName)
	}

	// Open wallet
	w, err := wallet.OpenWallet(walletName, pubPass, activeNetParams)
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
			newAddr, err := wallet.NewAddress(w, a.AccountNumber, waddrmgr.KeyScopeBIP0044)
			if err != nil {
				fmt.Printf("Failed to create new address for account %s (%d): %s\n", a.AccountName, a.AccountNumber, err)
				return
			}

			fmt.Printf("\t\taddress: %s\n", newAddr.EncodeAddress())
		}
	}
}
