package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

/*************************************************
* Interface type that will serve
* as way to maintain all services being monitored
*************************************************/

type ServiceStore interface {
	AddService(Service)
	GetServices() *[]Service
	GetServiceByID(uint) (*Service, error)
    EventUpdate(EventData) error
}

type Server struct {
    Port int `toml:"port"`
    Cert_key string `toml:"cert_key"`
    Cert_file string `toml:"cert_file"`
    LogLevel string `toml:"log_level"`

	store ServiceStore
	http.Handler

	//channel for json updates
	channel <-chan []byte
}

func(s* Server) addRoutes(mux *http.ServeMux){
    mux.HandleFunc("/", s.statusHandler)
    mux.HandleFunc("/updates", s.updateHandler)
    //resources
    mux.HandleFunc("/static/{type}/{file}", s.static)
}

func NewServer(store ServiceStore, ch <-chan []byte) *Server {
	server:= new(Server)
	server.store = store
	server.channel = ch

	return server
}

func (server *Server) statusHandler(w http.ResponseWriter, r *http.Request) {

    switch r.Method {
    case http.MethodGet:
        w.Header().Set("Content-Type", "text/html")
        tmpl, err := template.New("homepage.gohtml").Funcs(template.FuncMap{
            "formatTime": formatTime,
            "getAllHistory": getAllHistory,
        }).ParseFiles("static/templates/layout.gohtml",
                "static/templates/header.gohtml",
                "static/templates/homepage.gohtml")

            if err != nil {
                log.Printf("Failed to parse homepage template: %s", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

            services := server.store.GetServices()
            //log.Printf("Services sent: %+q", services)
            err = tmpl.ExecuteTemplate(w, "layout", services)
            if err != nil {
                log.Printf("Failed to parse homepage template: %s", err)
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

    default:
        http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
    }

}

func (server *Server) static(w http.ResponseWriter, r *http.Request) {
    filetype := r.PathValue("type")
    asset := r.PathValue("file")
    slog.Info("Serving file static asset.", filetype, "filetype", "asset", asset)

    file, err := os.Open("static/" + filetype + "/" + asset)
    if err != nil {
        log.Printf("Failed to serve static file %s. Error: %s ", asset, err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer file.Close()

    switch filetype {
    case "javascript":
	    w.Header().Set("Content-Type", "application/javascript")
    
    case "css":
	    w.Header().Set("Content-Type", "text/css")

    default:
	    w.Header().Set("Content-Type", "text/html")
    }

    _, err = io.Copy(w, file)
    if err != nil {
        log.Printf("Failed to copy static file %s. Error: %s", asset, err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

}

func (server *Server) updateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.(http.Flusher).Flush()

	//create context for handling client disconnect
	_, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Infinite loop to send events every second
	for {
		// pop event
        slog.Debug("Preparing to receive from channel")
		eventData := <-server.channel
        slog.Debug("Update Received Server Side.", "event", eventData)
        var event EventData
        err := json.Unmarshal(eventData, &event)
        if err != nil {
            slog.Error("Unable to Unmarshal event update.", "Error", err)
        }

        // update server.store
        err = server.store.EventUpdate(event)
        if err != nil {
            slog.Error("Unable to handle event update.", "Error", err)
        }

        // send to html elements to subscribers 
        service, err := server.store.GetServiceByID(event.ServiceID)
        fmt.Fprintf(w, "event: service-%d\n", service.ID)
		fmt.Fprintf(w, "data: %s\n\n", service.toHTML())
		w.(http.Flusher).Flush() // Flush the response writer to send the event immediately
        slog.Debug("Flushed Event from bus to subscribers.", "serviceID", service.ID, "html", service.toHTML())
	}

}

func formatTime(t time.Time) string {
    return t.Format("2006-01-02 15:04:05")
}
