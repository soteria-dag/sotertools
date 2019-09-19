// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/soteria-dag/sotertools/wallet"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
	"io/ioutil"
	"os"
	"path/filepath"
)

// abort prints the message and exits with code 1
func abort(msg string) {
	fmt.Println(msg)
	os.Exit(1)
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

	cfg := rpcclient.ConnConfig{
		Host: host,
		Endpoint: "ws",
		User: user,
		Pass: pass,
		Certificates: certs,
		DisableAutoReconnect: true,
	}

	client, err := rpcclient.New(&cfg, nil)
	if err != nil {
		return client, err
	}

	return client, nil
}

func main() {
	var mainnet, testnet, simnet bool
	var walletName, privPass, pubPass, srcAddr, destAddr, rpcSrv, rpcUser, rpcPass, rpcCert string
	var amt, fee float64
	// Converted values from parameters
	var sendAmount, feeAmount soterutil.Amount
	var source, dest soterutil.Address

	// Parse cli parameters
	flag.BoolVar(&mainnet, "mainnet", false, "Use mainnet params for wallet")
	flag.BoolVar(&testnet, "testnet", false, "Use testnet params for wallet")
	flag.BoolVar(&simnet, "simnet", false, "Use simnet params for wallet")
	flag.StringVar(&walletName, "w", "", "Source wallet file name")
	flag.StringVar(&privPass, "priv", "", "Password to use, for unlocking address manager (for private keys and info)")
	flag.StringVar(&pubPass, "pub", "", "Password to use, for opening address manager")
	flag.StringVar(&srcAddr, "source", "", "Source address of funds")
	flag.StringVar(&destAddr, "dest", "", "Destination address of funds")
	flag.Float64Var(&amt, "amt", float64(0), "Amount of coin to transfer (SOTER)")
	flag.Float64Var(&fee, "fee", float64(0), "Fee for transfer (SOTER)")
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
		abort("You can only specify one net param (-mainnet, -testnet, -simnet)")
	}
	if len(srcAddr) == 0 {
		abort("No source address specified (-source)")
	}
	if len(destAddr) == 0 {
		abort("No destination address specified (-dest)")
	}
	if len(privPass) == 0 {
		fmt.Println("WARNING: -priv (private password) is not set!")
	}
	if len(pubPass) == 0 {
		fmt.Println("WARNING: -pub (pub password) is not set!")
	}
	if amt == 0 {
		fmt.Printf("WARNING: SOTER amount to transfer is %f\n", amt)
	}
	if fee == 0 {
		fmt.Printf("WARNING: SOTER fee for transfer is %f\n", fee)
	}

	// Convert cli params
	sendAmount, err := soterutil.NewAmount(amt)
	if err != nil {
		abort(fmt.Sprintf("failed to convert amount %f", amt))
	}
	feeAmount, err = soterutil.NewAmount(fee)
	if err != nil {
		abort(fmt.Sprintf("failed to convert amount %f", amt))
	}

	source, err = soterutil.DecodeAddress(srcAddr, activeNetParams)
	if err != nil {
		abort(err.Error())
	}
	dest, err = soterutil.DecodeAddress(destAddr, activeNetParams)
	if err != nil {
		abort(err.Error())
	}

	// Open wallet
	w, err := wallet.OpenWallet(walletName, pubPass, activeNetParams)
	if err != nil {
		abort(err.Error())
	}
	defer func() {
		_ = w.Database().Close()
	}()

	fmt.Printf("Opened wallet %s\n", walletName)

	// Connect to soterd node
	client, err := connectRPC(rpcSrv, rpcUser, rpcPass, rpcCert)
	if err != nil {
		abort(fmt.Sprintf("RPC connection to %s failed: %s", rpcSrv, err))
	}

	addresses := []soterutil.Address{source}

	// Look for transactions with spendable outputs
	matches, err := wallet.SpendableTxOuts(client, addresses, activeNetParams)
	if err != nil {
		abort(fmt.Sprintf("Failed to find matching transactions in dag: %s", err))
	}

	if len(matches) == 0 {
		abort(fmt.Sprintf("No matching transactions for source address found in dag"))
	}

	fmt.Println("Matching transactions:")
	txTotalAmt := soterutil.Amount(0)
	for _, m := range matches {
		txTotalAmt += m.Amount
		fmt.Printf("block %s\theight %d\ttx %s\toutputNum %d\tvalue %s\tmatching wallet addr %s\n",
			m.Info.Block.BlockHash(), m.Info.BlockHeight, m.Info.Tx.TxHash(), m.VIndex, m.Amount, m.Address)
	}

	// Confirm that there's enough spendable coin
	if sendAmount + feeAmount > txTotalAmt {
		abort(fmt.Sprintf("Not enough coin found to satisfy amount requested for transaction; %s requested + %s fee, %s spendable",
			sendAmount, feeAmount, txTotalAmt))
	}

	fmt.Println()

	fmt.Printf("Creating a transaction for %s to %s\n", sendAmount, destAddr)
	txHash, err := wallet.Send(client, w, privPass, matches, dest, sendAmount, feeAmount)
	if err != nil {
		abort(err.Error())
	}

	fmt.Printf("Sent transaction with hash %s\n", txHash)
}