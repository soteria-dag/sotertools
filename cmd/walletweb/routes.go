// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/soteria-dag/sotertools/cmd/walletweb/static"
	"github.com/soteria-dag/sotertools/wallet"
	"github.com/soteria-dag/soterd/soterutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// beforeBody renders common HTML document sections including the opening <body> element
func beforeBody(w http.ResponseWriter, title string) {
	renderHTMLOpen(w)
	renderHTMLHeader(w, title)
	renderHTMLBodyOpen(w)
	renderHTMLNavbar(w)
}

// afterBody renders common HTML document sections starting from the closing </body> element
// to the end of the HTML document.
func afterBody(w http.ResponseWriter) {
	renderHTMLFooter(w)
	renderHTMLScript(w)
	renderHTMLBodyClose(w)
	renderHTMLClose(w)
}

// handleRoot responds to requests for root url /
func handleRoot(w http.ResponseWriter, r *http.Request) {
	// By default we'll print out balance information
	handleBalance(w, r)
}

// handleBalance responds to requests for /balance/<address> or /balance?address=<address>
// It renders known balance of the address in the dag
func handleBalance(w http.ResponseWriter, r *http.Request) {
	title := "walletweb - balance"
	beforeBody(w, title)
	defer afterBody(w)
	// For r.URL.Path of /balance/Sh7EBro, parts will be: ["", "balance", "Sh7EBro"]
	parts := strings.Split(r.URL.Path, "/")

	var address string
	address = r.URL.Query().Get("address")

	if len(address) == 0 && len(parts) == 3 {
		address = parts[2]
	}

	balanceForm := `<form action="/balance" method="get">
  <div class="form-group">
    <label for="address">Get balance of coin address</label>
    <input type="text" class="form-control" id="address" name="address">
  </div>
  <button type="submit" class="btn btn-primary">Submit</button>
</form>`

	if len(address) == 0 {
		// Render a search box
		renderHTML(w, balanceForm, nil)
		renderHTML(w, "<br>", nil)
		return
	}

	info, err := getBalance(client, address)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to get balance info for %s: %s", address, err))
		return
	}
	info.RenderHTML(w)
}

// handleSendCoin responds to requests for /sendcoin
// It renders a page to send coin, or generates a transaction with the given inputs and submits it to the soter network.
func handleSendCoin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleSendCoinGet(w, r)
	case "POST":
		handleSendCoinPost(w, r)
	default:
		handleSendCoinGet(w, r)
	}
}

// handleSendCoinGet responds to GET requests for /sendcoin
func handleSendCoinGet(w http.ResponseWriter, r *http.Request) {
	title := "walletweb - sendcoin"
	// Render the different HTML sections for the response
	beforeBody(w, title)
	defer afterBody(w)

	addresses, err := wallet.WalletAddresses(myWallet)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to get wallet address info: %s", err))
		return
	}

	renderHTML(w, "<h2>Wallet addresses</h2>", nil)
	infos := make([]balanceInfo, len(addresses))
	for i, address := range addresses {
		info, err := getBalance(client, address.EncodeAddress())
		if err != nil {
			renderHTMLErr(w, fmt.Errorf("failed to get balance info for %s: %s", address, err))
			return
		}

		infos[i] = info
		info.RenderHTML(w)
		renderHTML(w, "<br>", nil)
	}

	sendForm := `<form action="/sendcoin" method="post">
  <div class="form-group">
    <label for="source">Wallet address to send coin from</label>
    <select class="form-control" id="source" name="source">
      {{- range . }}
      <option>{{ .Address }}</option>
      {{- end}}
    </select>
  </div>
  <div class="form-group">
    <label for="dest">Send coin to address</label>
    <input type="text" class="form-control" id="dest" name="dest">
  </div>
  <div class="form-group">
    <label for="amount">Amount of SOTER to send</label>
    <input type="number" class="form-control" id="amount" name="amount">
  </div>
  <div class="form-group">
    <label for="fee">Fee for transfer (in SOTER)</label>
    <input type="number" class="form-control" id="fee" name="fee">
  </div>
  <button type="submit" class="btn btn-primary">Send</button>
</form>`

	renderHTML(w, sendForm, infos)
	renderHTML(w, "<br>", nil)
}

// handleSendCoinPost responds to POST requests for /sendcoin
func handleSendCoinPost(w http.ResponseWriter, r *http.Request) {
	var source, dest soterutil.Address
	var amount, fee soterutil.Amount

	title := "walletweb - sendcoin"
	// Render the different HTML sections for the response
	beforeBody(w, title)
	defer afterBody(w)

	err := r.ParseForm()
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to parse POST form: %s", err))
		return
	}

	// Validate form
	src := r.Form.Get("source")
	source, err = soterutil.DecodeAddress(src, activeNetParams)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to parse source address %s: %s", src, err))
		return
	}

	dst := r.Form.Get("dest")
	dest, err = soterutil.DecodeAddress(dst, activeNetParams)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to parse destination address %s: %s", dst, err))
		return
	}

	a := r.Form.Get("amount")
	if len(a) == 0 {
		renderHTMLErr(w, fmt.Errorf("no coin amount specified"))
		return
	}
	amt, err := strconv.ParseFloat(a, 64)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to parse coin amount %s: %s", a, err))
		return
	}
	amount, err = soterutil.NewAmount(amt)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to cast coin amount %f: %s", amt, err))
		return
	}

	f := r.Form.Get("fee")
	if len(f) == 0 {
		renderHTMLErr(w, fmt.Errorf("no fee specified"))
		return
	}
	feeNum, err := strconv.ParseFloat(f, 64)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to parse transaction fee %s: %s", f, err))
		return
	}
	fee, err = soterutil.NewAmount(feeNum)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to cast transaction fee %f: %s", feeNum, err))
		return
	}

	// Look for transactions with spendable outputs
	matches, err := wallet.SpendableTxOuts(client, []soterutil.Address{source}, activeNetParams)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to find matching transactions in dag for address %s: %s", source, err))
		return
	}

	if len(matches) == 0 {
		renderHTMLErr(w, fmt.Errorf("no matching transactions for source address %s found in dag", source))
		return
	}

	spendable := soterutil.Amount(0)
	for _, m := range matches {
		spendable += m.Amount
	}

	if amount + fee > spendable {
		renderHTMLErr(w, fmt.Errorf("not enough coin found to satisfy amount requested for transaction; %s requested + %s fee, %s spendable",
			amount, fee, spendable))
		return
	}

	txHash, err := wallet.Send(client, myWallet, privPass, matches, dest, amount, fee)
	if err != nil {
		renderHTMLErr(w, fmt.Errorf("failed to send coin: %s", err))
		return
	}

	renderHTML(w, fmt.Sprintf("<p>Sent %s to %s in transaction %s</p>", amount, dest, txHash), nil)
	renderHTML(w, "<br>", nil)
}

// handleFavicon responds to requests for /favicon.ico
func handleFavicon(w http.ResponseWriter, r *http.Request) {
	setContentType(w, "image/vnd.microsoft.icon")
	_, err := w.Write(static.Favicon)
	if err != nil {
		log.Printf("Failed to respond to /favicon.ico: %s", err)
	}
}

// handleLogo responds to requests for /static/soteria_logo.jpg
func handleLogo(w http.ResponseWriter, r *http.Request) {
	setContentType(w, "image/jpeg")
	_, err := w.Write(static.SoteriaLogo)
	if err != nil {
		log.Printf("Failed to respond to /static/soteria_logo.jpg: %s", err)
	}
}