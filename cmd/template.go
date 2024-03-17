package main

import (
	"html/template"
	"net/http"
)

func StatusTemplate(store ServiceStore, w http.ResponseWriter) {

    services := store.GetServices()

    tmpl := `
    <!DOCTYPE html>
    <html>
    <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Status Page</title>
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>

    </head>
    <body>
    <h1>Services</h1>
    <table>
        <tr style='text-align: left'>
        <th> Service</th>
        <th> Status</th>
        <th> Address</th>
        </tr>
    {{range .}}
    <tr>
        <td>{{.Name}}</td>
        <td>
            {{ if .Status}}
                "UP"
            {{else}}
                "DOWN"
            {{end}}
        </td>
        <td><a href='{{.Address}}'>{{.Address}}</a></td>
    {{end}}
    </tr>
    </table>
    
<!-- Display real-time updates here -->
<div id="card1" hx-get="/update-card1">Initial content for Card 1</div>
<div id="card2" hx-get="/update-card2">Initial content for Card 2</div>

    `
    t, err := template.New("services").Parse(tmpl)
    if err != nil {
        panic (err)
    }

    err = t.Execute(w, services)
    if err != nil {
        panic(err)
    }

}


