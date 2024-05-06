package main

import (
    "html/template"
    "net/http"
)

func StatusTemplate(store ServiceStore, w http.ResponseWriter) {

    services := store.GetServices()

    tmpl := `
    <!DOCTYPE html>
    <html lang="en">
    <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel = "stylesheet" href="/themes/css/gruvbox.css">
    <script src ="themes/javascript/original.js"></script>
    <title>Service Status</title>
    </head>
    <body>
    <h1>Service Status</h1>
    <div class= "services"> 
    {{range .}}
    <div class = "card" id={{.Name}}> 
        <div class  =
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
    </div>

    </body>
    </html>
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


