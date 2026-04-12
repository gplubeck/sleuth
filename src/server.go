package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	homepageTmpl    *template.Template
	serviceCardTmpl *template.Template
)

func initTemplates() error {
	var err error
	funcs := template.FuncMap{
		"formatTime":      formatTime,
		"formatTimestamp": formatTimestamp,
		"getAllHistory":    getAllHistory,
	}

	homepageTmpl, err = template.New("homepage.gohtml").Funcs(funcs).ParseFiles(
		"static/templates/layout.gohtml",
		"static/templates/header.gohtml",
		"static/templates/homepage.gohtml",
		"static/templates/service_card.gohtml",
		"static/templates/service_header.gohtml",
		"static/templates/service_body.gohtml")
	if err != nil {
		return err
	}

	serviceCardTmpl, err = template.New("service-card").Funcs(funcs).ParseFiles(
		"static/templates/service_card.gohtml",
		"static/templates/service_header.gohtml",
		"static/templates/service_body.gohtml")
	return err
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.RequestURI,
			"remote", r.RemoteAddr, "duration_ms", time.Since(start).Milliseconds())
	})
}

/*************************************************
* Interface type that will serve
* as way to maintain all services being monitored
*************************************************/

type ServiceStore interface {
	AddService(Service)
	GetServices() *[]Service
	GetServiceByID(uint) (*Service, error)
	EventUpdate(EventData) error
	ReconcileServices([]Service)
	Load() error
	Save() error
}


func NewServiceStore(storageType string, noHistory bool) (ServiceStore, error) {
	switch storageType {
	case "memory":
		return NewInMemoryStore(noHistory)
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
	Title string `toml:"title"`
	Subtitle string `toml:"subtitle"`

	store ServiceStore
    publisher *Publisher
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
		services := server.store.GetServices()
		templateData := struct {
			Server   *Server
			Services *[]Service
		}{Server: server, Services: services}

		err := homepageTmpl.ExecuteTemplate(w, "layout", templateData)
		if err != nil {
			log.Printf("Failed to execute homepage template: %s", err)
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

	// Whitelist valid asset types to prevent path traversal
	switch filetype {
	case "javascript":
		w.Header().Set("Content-Type", "application/javascript")
	case "css":
		w.Header().Set("Content-Type", "text/css")
	default:
		http.Error(w, "Invalid asset type", http.StatusBadRequest)
		return
	}

	// Reject any path that tries to escape the static directory
	clean := filepath.Clean(asset)
	if strings.Contains(clean, "..") || strings.HasPrefix(clean, "/") {
		http.Error(w, "Invalid asset path", http.StatusBadRequest)
		return
	}

	file, err := os.Open(filepath.Join("static", filetype, clean))
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

    flusher, okay := w.(http.Flusher)
    if !okay {
        http.Error(w, "Streaming unsupported! What browser?", http.StatusInternalServerError)
        return
    }

    // make channel for new subscriber
    clientChan := make(chan []byte, 10)
    server.publisher.addClient <- clientChan
    defer func() {
        server.publisher.removeClient <- clientChan
    }()

    flusher.Flush()

	//create context for handling client disconnect
	ctx := r.Context()

    // Heartbeat ticker
    heartbeatTicker := time.NewTicker(30 * time.Second)
    defer heartbeatTicker.Stop()

	// Infinite loop to send events
	for {
        select {
        case <-ctx.Done():
            slog.Debug("Client disconnect.")
            return
            // pop event
        case eventData := <-clientChan:
            slog.Debug("Update Received in updateHandler.", "event", eventData)
            var event EventData
            if err := json.Unmarshal(eventData, &event); err != nil {
                slog.Error("Unable to Unmarshal event update.", "Error", err)
                continue
            }

            // send to html elements to subscribers
            service, err := server.store.GetServiceByID(event.ServiceID)
            if err != nil {
                slog.Error("Unable to get service by ID.", "ServiceID", event.ServiceID, "Error", err)
                continue
            }
            _, err = fmt.Fprintf(w, "event: service-%d\ndata:%s\n\n", service.ID, service.toHTML())
            if err != nil {
                slog.Error("Error writing event to update stream.", "ServiceID", event.ServiceID, "Error", err)
            }
            flusher.Flush() // Flush the response writer to send the event immediately

        case <-heartbeatTicker.C:
            // heatbeat to client for good connection reasons
            _, err := fmt.Fprintf(w, ":heartbeat\n\n")
            if err != nil {
                slog.Error("Error writing ehartbeat to client.", "Error", err)
                // assume disconnect and kill
                return
            }
            slog.Debug("Sending heartbeat to active clients.")
            flusher.Flush()
        }
    }

}

// formatTime is used for the time-range labels under the uptime graph.
// Shows time-only when the timestamp is from today (local time), otherwise
// includes the date so the label makes sense across day boundaries.
func formatTime(t time.Time) string {
	now := time.Now()
	y, m, d := now.Date()
	localMidnight := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	ty, tm, td := t.Date()
	tMidnight := time.Date(ty, tm, td, 0, 0, 0, 0, t.Location())
	if tMidnight.Equal(localMidnight) {
		return t.Format("15:04:05")
	}
	return t.Format("01-02 15:04:05")
}

// formatTimestamp is used for segment tooltips and always returns the full
// date+time so every tooltip in a card uses the same format.
func formatTimestamp(t time.Time) string {
	return t.Format("01-02 15:04:05")
}
