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
    <title>Status Page</title>
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


