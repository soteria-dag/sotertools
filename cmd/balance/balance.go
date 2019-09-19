// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/soteria-dag/sotertools/wallet"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
	"io/ioutil"
	"syscall"
)

// Define the structure of json output for this program.
type Output struct {
	Balance float64	`json:"balance"`
	SpendableBalance float64	`json:"spendableBalance"`
	HadError bool	`json:"hadError"`
	ErrorMsg string	`json:"errorMsg"`
}

// abort prints the message and exits with code 1
func abort(msg string, doJson bool) {
	if doJson {
		out := Output{
			Balance: -1,
			SpendableBalance: -1,
			HadError: true,
			ErrorMsg: msg,
		}

		js, err := json.MarshalIndent(&out, "", "\t")
		if err != nil {
			fmt.Printf("{\"balance\":%f,\"spendableBalance\":%f,\"hadError\":%v,\"errorMsg\":\"%s\"}\n",
				out.Balance,
				out.SpendableBalance,
				out.HadError,
				out.ErrorMsg)
		} else {
			fmt.Println(string(js))
		}
	} else {
		fmt.Println(msg)
	}
	syscall.Exit(1)
}

func main() {
	var mainnet, testnet, simnet, jsonOutput bool
	var inputAddress, rpcSrv, rpcUser, rpcPass, rpcCert string

	// Parse cli parameters
	flag.BoolVar(&mainnet, "mainnet", false, "Use mainnet params for rpc calls")
	flag.BoolVar(&testnet, "testnet", false, "Use testnet params for rpc calls")
	flag.BoolVar(&simnet, "simnet", false, "Use simnet params for rpc calls")
	flag.StringVar(&rpcSrv, "rpcserver", "", "Soterd RPC server to scan for transactions")
	flag.StringVar(&rpcUser, "rpcuser", "", "Soterd RPC server username to use")
	flag.StringVar(&rpcPass, "rpcpass", "", "Soterd RPC server password to use")
	flag.StringVar(&rpcCert, "rpccert", "", "Soterd RPC server cert chain")
	// TODO(cedric): Support multiple addresses?
	flag.StringVar(&inputAddress, "address", "", "Address to check balance of")
	flag.BoolVar(&jsonOutput, "json", false, "Output in JSON format")

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
	if selectedNets == 0 {
		abort("You must specify one net param (-mainnet, -testnet, -simnet)", jsonOutput)
	}
	if selectedNets > 1 {
		abort("You can only specify one net param (-mainnet, -testnet, -simnet)", jsonOutput)
	}
	if len(rpcSrv) == 0 {
		fmt.Println("WARNING: -rpcserver is not set!")
	}
	if len(rpcUser) == 0 {
		fmt.Println("WARNING: -rpcuser is not set!")
	}
	if len(rpcPass) == 0 {
		fmt.Println("WARNING: -rpcpass is not set!")
	}
	if len(rpcCert) == 0 {
		fmt.Println("WARNING: -rpccert is not set!")
	}
	if len(inputAddress) == 0 {
		abort("You must specify an address to check the balance of (-address)", jsonOutput)
	}

	// Read RPC cert
	var cert []byte
	cert, err := ioutil.ReadFile(rpcCert)
	if err != nil {
		abort(fmt.Sprintf("failed to read rpc certificate: %s", err), jsonOutput)
	}

	// Decode input address
	var address soterutil.Address
	address, err = soterutil.DecodeAddress(inputAddress, activeNetParams)
	if err != nil {
		abort(fmt.Sprintf("failed to decode address from %s: %s", inputAddress, err), jsonOutput)
	}

	var addresses = []soterutil.Address{address}

	cfg := rpcclient.ConnConfig{
		Host:                 rpcSrv,
		Endpoint: "ws",
		User:                 rpcUser,
		Pass:                 rpcPass,
		Certificates: cert,
		DisableAutoReconnect: true,
	}

	client, err := rpcclient.New(&cfg, nil)
	if err != nil {
		abort(fmt.Sprintf("failed to create soterd rpc client: %s", err), jsonOutput)
	}

	balance, spendable, err := wallet.GetBalance(client, addresses, activeNetParams)
	if err != nil {
		abort(fmt.Sprintf("failed to get balance of address %s: %s", address, err), jsonOutput)
	}

	if jsonOutput {
		out := Output{
			Balance: float64(balance),
			SpendableBalance: float64(spendable),
			HadError: false,
			ErrorMsg: "",
		}
		js, err := json.MarshalIndent(&out, "", "\t")
		if err != nil {
			abort(err.Error(), jsonOutput)
		}
		fmt.Println(string(js))
	} else {
		fmt.Printf("balance of %s: %s\n", address, balance)
		fmt.Printf("spendable balance of %s: %s\n", address, spendable)
	}

}
