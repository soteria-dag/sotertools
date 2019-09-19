// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/soteria-dag/sotertools/cmd/walletweb/templates"
	"html/template"
	"net/http"
)

// setContentType sets the Content-Type HTTP header of a response
func setContentType(w http.ResponseWriter, cType string) {
	w.Header().Set("Content-Type", cType)
}

// renderHTML renders the given template text into the response, with optional template data
func renderHTML(w http.ResponseWriter, tmpl string, data interface{}) {
	t := template.New("htmlElem")
	t, err := t.Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderHTMLErr renders the error in the response
func renderHTMLErr(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderHTMLTmpl renders the template from the file in the response
func renderHTMLTmpl(w http.ResponseWriter, name string, data interface{}) {
	err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderHTMLOpen renders the <html> element's opening tag in the response
func renderHTMLOpen(w http.ResponseWriter) {
	setContentType(w, "text/html")
	renderHTML(w, "<!DOCTYPE html>", nil)
	renderHTML(w, "<html>", nil)
}

// renderHTMLClose renders the </html> element's closing tag in the response
func renderHTMLClose(w http.ResponseWriter) {
	renderHTML(w, "</html>", nil)
}

// renderHTMLBodyOpen renders the <body> element's opening tag in the response
func renderHTMLBodyOpen(w http.ResponseWriter) {
	renderHTML(w, "<body>", nil)
}

// renderHTMLBodyClose renders the <body> element's closing tag in the response
func renderHTMLBodyClose(w http.ResponseWriter) {
	renderHTML(w, "</body>", nil)
}

// renderHTMLHeader renders the header.tmpl template in the response
func renderHTMLHeader(w http.ResponseWriter, title string) {
	renderHTMLTmpl(w, "header", title)
}

// renderHTMLNavbar renders the navbar.tmpl template in the response
func renderHTMLNavbar(w http.ResponseWriter) {
	type navbar struct {
		// The 'brand' name used in the navbar
		Brand string
	}

	n := navbar{
		Brand: "walletweb",
	}

	renderHTMLTmpl(w, "navbar", n)
}

// renderHTMLFooter renders the footer.tmpl template in the response
func renderHTMLFooter(w http.ResponseWriter) {
	renderHTMLTmpl(w, "footer", nil)
}

// renderHTMLScript renders the script.tmpl template in the response
func renderHTMLScript(w http.ResponseWriter) {
	renderHTMLTmpl(w, "script", nil)
}

// RenderHTML renders the balanceInfo as a bootstrap card in the response
func (info *balanceInfo) RenderHTML(w http.ResponseWriter) {
	renderHTMLTmpl(w, "balance", info)
}