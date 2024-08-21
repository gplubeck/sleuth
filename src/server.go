package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

/*************************************************
* Interface type that will serve
* as way to maintain all services being monitored
*************************************************/

type ServiceStore interface {
	AddService(service Service)
	GetServices() *[]Service
}

type ServiceServer struct {
	store ServiceStore
	//store InMemoryStore
	http.Handler

	//channel for json updates
	channel <-chan []byte
}

func(s* ServiceServer) addRoutes(mux *http.ServeMux){
    mux.HandleFunc("/", s.statusHandler)
    mux.HandleFunc("/updates", s.updateHandler)
    mux.HandleFunc("/static/{type}/{file}", s.static)
}

func NewServiceServer(store ServiceStore, ch <-chan []byte) *ServiceServer {
	server:= new(ServiceServer)
	server.store = store
	server.channel = ch

	return server
}

func (server *ServiceServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

    switch r.Method {
    case http.MethodGet:
        log.Printf("Inside get for template")
        tmpl, err := template.New("homepage.gohtml").Funcs(template.FuncMap{
            "formatTime": formatTime}).ParseFiles("static/templates/homepage.gohtml")

            if err != nil {
                log.Printf("Failed to parse homepage template: %s", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            services := server.store.GetServices()
            err = tmpl.Execute(w, services)
    default:
        http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
    }

}

func (server *ServiceServer) static(w http.ResponseWriter, r *http.Request) {
    filetype := r.PathValue("type")
    asset := r.PathValue("file")
    log.Printf("Serving file %s/%s", filetype, asset)

    file, err := os.Open("static/" + filetype + "/" + asset)
    if err != nil {
        log.Printf("Failed to serve static file %s. Error: ", asset, err.Error)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer file.Close()

	w.Header().Set("Content-Type", "text/html")

    _, err = io.Copy(w, file)
    if err != nil {
        log.Printf("Failed to copy static file %s. Error: ", asset, err.Error)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

}


func (server *ServiceServer) updateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	//create conect for handling client disconnect
	_, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Initialize counter
	// Infinite loop to send events every second
	for {
		// Send event
		eventData := <-server.channel

		fmt.Fprintf(w, "data: %s\n\n", eventData)
		w.(http.Flusher).Flush() // Flush the response writer to send the event immediately

		// Delay for one second before sending the next event
		//time.Sleep(1 * time.Second)
	}

}


func formatTime(t time.Time) string {
    return t.Format("2006-01-02 15:04:05")
}
