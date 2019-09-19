// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"github.com/soteria-dag/sotertools/wallet"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/rpcclient"
	"github.com/soteria-dag/soterd/soterutil"
	soterwallet "github.com/soteria-dag/soterwallet/wallet"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
)

var (
	// Clients are held in a global variable, to make them available to other packages like routes
	client *rpcclient.Client
	activeNetParams *chaincfg.Params
	myWallet *soterwallet.Wallet
	privPass string
)

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
	var addr, walletName, pubPass, rpcSrv, rpcUser, rpcPass, rpcCert string

	// Parse cli parameters
	flag.StringVar(&addr, "l", ":5077", "Which [ip]:port to listen on")
	flag.BoolVar(&mainnet, "mainnet", false, "Use mainnet params for rpc connections")
	flag.BoolVar(&testnet, "testnet", false, "Use testnet params for rpc connections")
	flag.BoolVar(&simnet, "simnet", false, "Use simnet params for rpc connections")
	flag.StringVar(&walletName, "w", "", "Wallet file name (for sending coin)")
	flag.StringVar(&privPass, "priv", "", "Password to use, for unlocking address manager (for private keys and info)")
	flag.StringVar(&pubPass, "pub", "", "Password to use, for opening address manager")
	flag.StringVar(&rpcSrv, "rpcserver", "", "Soterd RPC server connect to (ip:port)")
	flag.StringVar(&rpcUser, "rpcuser", "", "Soterd RPC server username to use")
	flag.StringVar(&rpcPass, "rpcpass", "", "Soterd RPC server password to use")
	flag.StringVar(&rpcCert, "rpccert", "", "Soterd RPC server cert chain")

	flag.Parse()

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
		log.Fatal("You can only specify one net param (-mainnet, -testnet, -simnet)")
	}
	if len(privPass) == 0 {
		log.Println("WARNING: -priv (private password) is not set!")
	}
	if len(pubPass) == 0 {
		log.Println("WARNING: -pub (pub password) is not set!")
	}

	// Open wallet
	w, err := wallet.OpenWallet(walletName, pubPass, activeNetParams)
	if err != nil {
		log.Fatalf("Failed to open wallet: %s", err)
	}
	myWallet = w
	defer func() {
		_ = myWallet.Database().Close()
	}()

	// Connect to soterd node
	client, err = connectRPC(rpcSrv, rpcUser, rpcPass, rpcCert)
	if err != nil {
		log.Fatalf("RPC connection to %s failed: %s", rpcSrv, err)
	}

	// Route requests for / (or anything that doesn't match another pattern) to handleRoot, in DefaultServeMux.
	// https://golang.org/pkg/net/http/#ServeMux
	http.HandleFunc("/", handleRoot)
	// Show coin balance details of an address
	// The trailing / allows us to route requests for URLs
	// like /balance/Sh7EBrov7iZqbMiYe6kPn3ebaBevB7DcH3 to handleBalance
	http.HandleFunc("/balance", handleBalance)
	http.HandleFunc("/balance/", handleBalance)
	// Send coin to an address
	http.HandleFunc("/sendcoin", handleSendCoin)
	// Serve favicon from hard-coded bytes
	http.HandleFunc("/favicon.ico", handleFavicon)
	// Serve the soteria logo from hard-coded bytes
	http.HandleFunc("/static/soteria_logo.jpg", handleLogo)

	// Start http server in a goroutine, so that it doesn't block other background activities we may want to start.
	// We'll use a channel to let us know if there was a problem encountered.
	httpSrvResult := make(chan error)
	startHttp := func() {
		// Use DefaultServeMux as the handler
		err := http.ListenAndServe(addr, nil)
		httpSrvResult <- err
	}

	go startHttp()

	// NOTE(cedric): We can start and manage other background processes here

	// Listen for signals telling us to shut down, or for http server to stop
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case err := <-httpSrvResult:
		if err != nil {
			log.Printf("Failed to ListenAndServe for addr %s: %s", addr, err)
		}
	case s := <-c:
		log.Println("Shutting down due to signal:", s)
	}
}