package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
    noHistory := flag.Bool("no-history", false, "Start without loading any history from .sleuth.bin")
    flag.Parse()
	configPath := "config.toml"
	log.Printf("Parsing config file: %s", configPath)
	config := parseConfigs(configPath)

	log.Printf("Setting log level: %s", config.Server.LogLevel)
	slog.SetLogLoggerLevel(getLogLevel(config.Server.LogLevel))

	if err := initTemplates(); err != nil {
		log.Fatalf("Failed to parse templates: %s", err)
	}

	//mock memory store
	store, err := NewServiceStore(config.Server.Storage, *noHistory)
	if err != nil {
		log.Fatalf("Unable to intialize storage. Error: %s", err)
	}

	store.ReconcileServices(config.Services)

	// event bus from scheduler publisher
	updateChannel := make(chan []byte)


	//start server
	server := config.Server
	server.channel = updateChannel
    server.publisher = NewPublisher()
	server.store = store
	mux := http.NewServeMux()
	server.addRoutes(mux)
	loggedMux := loggingMiddleware(mux)
	log.Printf("Starting Service Sleuth Server version: %s", Version)
	log.Printf("Build Time: %s", BuildTime)
	log.Printf("Listening on port: %d", server.Port)
	slog.Debug("Server cert files", "key", server.Cert_key, "cert", server.Cert_file)

	//complete start up send them goroutines
	go server.publisher.Start()
	scheduler := NewScheduler(store, server.publisher.publish)
	scheduler.Start()

	go func() {
		var err error
		port := fmt.Sprintf(":%d", server.Port)
		if server.Cert_file != "" && server.Cert_key != "" {
			slog.Info("Starting TLS Server.")
			err = http.ListenAndServeTLS(port, server.Cert_file, server.Cert_key, loggedMux)
		} else {
			slog.Warn("Starting Plaintext Server.")
			err = http.ListenAndServe(port, loggedMux)
		}
		if err != nil {
			log.Fatalf("Failed server state.  Error: %s", err)
		}
	}()

	//set up control flow of program via signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for {
			select {
			case sig := <-sigChan:
				switch sig {
				case syscall.SIGINT, syscall.SIGTERM:
					log.Printf("Received signal %s. Cleaning up....", sig)
                    store.Save()
                    os.Exit(0)
				case syscall.SIGHUP:
					log.Printf("Received signal %s. Reloading config.", sig)
					newConfig := parseConfigs(configPath)

					// Snapshot current IDs before touching the store
					current := *store.GetServices()
					currentIDs := make(map[uint]bool, len(current))
					for _, s := range current {
						currentIDs[s.ID] = true
					}
					newIDs := make(map[uint]bool, len(newConfig.Services))
					for _, s := range newConfig.Services {
						newIDs[s.ID] = true
					}

					// Cancel goroutines for services being removed
					for _, s := range current {
						if !newIDs[s.ID] {
							scheduler.RemoveService(s.ID)
						}
					}

					// Sync the store (remove stale, update existing, add new)
					store.ReconcileServices(newConfig.Services)

					// Start goroutines for newly added services
					for _, s := range newConfig.Services {
						if !currentIDs[s.ID] {
							scheduler.AddService(s)
						}
					}
					log.Printf("Config reloaded successfully.")
				default:
					log.Printf("%s caught, but not yet implemented.", sig)
				}
			}
		}
	}()


    //block until goroutines are cleaned up
    select{}

}
