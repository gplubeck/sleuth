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
    <link rel = "stylesheet" href="/themes/classic.css">
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
    
    <body>
    <h1>Services</h1>
    <div class = "container">
    {{range .}}
    <div class = "card"> 
        <div class =
            {{ if .Status}}
                "status online"
            {{else}}
                "status offline"
            {{end}}
            ></div>
        <div class = "service">{{.Name}}</div>
    </div>
    {{end}}
    </div>
    
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


