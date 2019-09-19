// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package templates

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

var (
	// templateCache holds parsed templates
	templateCache = make(map[string]*template.Template)
	// Maps the template to its generating function
	templates = map[string]func()(*template.Template, error){
		"header": header,
		"footer": footer,
		"navbar": navbar,
		"script": script,
		"balance": balance,
	}
)

// header generates a header template
func header() (*template.Template, error) {
	tpl := `<head>
    <title>{{ . }}</title>
    <!-- Bootstrap CSS -->
    <link href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">
</head>`

	t := template.New("header")
	return t.Parse(tpl)
}

// footer generates a footer template
func footer() (*template.Template, error) {
	tpl := `<footer class="pt-2 border-top">
    <div class="d-flex justify-content-center">
        <h5><small class="text-muted">&copy; 2018-%d The Soteria DAG developers</small></h5>
    </div>
</footer>`

	t := template.New("footer")
	return t.Parse(fmt.Sprintf(tpl, time.Now().Year()))
}

func navbar() (*template.Template, error) {
	tpl := `<nav class="navbar navbar-expand-lg navbar-light bg-light">
    <a class="navbar-brand" href="/">
        <img src="/static/soteria_logo.jpg" width="30" height="30" class="d-inline-block align-top" alt="soteria logo">
        {{ .Brand }}</a>
    <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbarSupportedContent" aria-controls="navbarSupportedContent" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
    </button>

    <div class="collapse navbar-collapse" id="navbarSupportedContent">
        <ul class="navbar-nav mr-auto">
            <li class="nav-item">
                <a class="nav-link" href="/balance">balance</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/sendcoin">send coin</a>
            </li>
        </ul>
    </div>
</nav>`

	t := template.New("navbar")
	return t.Parse(tpl)
}

func script() (*template.Template, error) {
	tpl := `<!-- Bootstrap JS -->
<script src="https://code.jquery.com/jquery-3.3.1.slim.min.js" integrity="sha384-q8i/X+965DzO0rT7abK41JStQIAqVgRVzpbzo5smXKp4YfRvH+8abtTE1Pi6jizo" crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.14.7/umd/popper.min.js" integrity="sha384-UO2eT0CpHqdSJQ6hJty5KVphtPhzWj9WO1clHTMGa3JDZwrnQq4sF86dIHNDz0W1" crossorigin="anonymous"></script>
<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/js/bootstrap.min.js" integrity="sha384-JjSmVgyd0p3pXB1rRibZUAYoIIy6OrQ6VrjIEaFf/nJGzIxFDsf4x0xIM+B07jRM" crossorigin="anonymous"></script>`

	t := template.New("script")
	return t.Parse(tpl)
}

func balance() (*template.Template, error) {
	tpl := `<div class="card-group">
    <div class="card">
        <div class="card-body">
            <ul class="list-unstyled">
                <li>Address: {{ .Address }}</li>
				<li>Balance: {{ .Balance }}</li>
				<li>Spendable: {{ .Spendable }}</li>
            </ul>
        </div>
    </div>
</div>`

	t := template.New("balance")
	return t.Parse(tpl)
}

func init() {
	// Pre-parse templates
	for name, tplGen := range templates {
		t, err := tplGen()
		if err != nil {
			panic(err)
		}

		templateCache[name] = t
	}
}

// ExecuteTemplate looks for the given template name, and if available executes it with the given data against the writer
func ExecuteTemplate(w http.ResponseWriter, name string, data interface{}) error {
	t, ok := templateCache[name]
	if ok {
		return t.Execute(w, data)
	}

	tplGen, ok := templates[name]
	if !ok {
		return fmt.Errorf("No template with name %s", name)
	}

	t, err := tplGen()
	if err != nil {
		return fmt.Errorf("Failed to generate template %s: %s", name, err)
	}

	templateCache[name] = t

	return t.Execute(w, data)
}