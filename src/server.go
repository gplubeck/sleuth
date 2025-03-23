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

func NewServiceStore(storageType string) (ServiceStore, error) {
	switch storageType {
	case "memory":
		return NewInMemoryStore()
	case "sqlite":
		return nil, fmt.Errorf("sqlite memory not yet implemented.")
	default:
		return nil, fmt.Errorf("Unkown storage type \"%s\".", storageType)
	}
}

type Server struct {
	Port      int    `toml:"port"`
	Cert_key  string `toml:"cert_key"`
	Cert_file string `toml:"cert_file"`
	LogLevel  string `toml:"log_level"`
	Storage   string `toml:"storage_type"`

	Theme string `toml:"theme"`

	store ServiceStore
	http.Handler

	//channel for json updates
	channel <-chan []byte
}

func (s *Server) addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.statusHandler)
	mux.HandleFunc("/updates", s.updateHandler)
	//resources
	mux.HandleFunc("/static/{type}/{file}", s.static)
}

func (server *Server) statusHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html")
		tmpl, err := template.New("homepage.gohtml").Funcs(template.FuncMap{
			"formatTime":    formatTime,
			"getAllHistory": getAllHistory,
		}).ParseFiles("static/templates/layout.gohtml",
			"static/templates/header.gohtml",
			"static/templates/homepage.gohtml",
			"static/templates/service_card.gohtml",
			"static/templates/service_header.gohtml",
			"static/templates/service_body.gohtml")

		if err != nil {
			slog.Error("Failed to parse homepage template.", "Error", err.Error())
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

	switch filetype {
	case "javascript":
		w.Header().Set("Content-Type", "application/javascript")

	case "css":
		w.Header().Set("Content-Type", "text/css")
		// Must take care of switching for theme
		if asset == "theme.css" {
			asset = server.Theme
		}

	default:
		w.Header().Set("Content-Type", "text/html")
	}

	file, err := os.Open("static/" + filetype + "/" + asset)
	if err != nil {
		log.Printf("Failed to serve static file %s. Error: %s ", asset, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

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
		eventData := <-server.channel
		slog.Debug("Update Received Server Side.", "event", eventData)
		var event EventData
		err := json.Unmarshal(eventData, &event)
		if err != nil {
			slog.Error("Unable to Unmarshal event update.", "Error", err)
		}

		// send to html elements to subscribers
		service, err := server.store.GetServiceByID(event.ServiceID)
		fmt.Fprintf(w, "event: service-%d\n", service.ID)
		fmt.Fprintf(w, "data: %s\n\n", service.toHTML())
		w.(http.Flusher).Flush() // Flush the response writer to send the event immediately
	}

}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
