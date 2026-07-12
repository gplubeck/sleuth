package main

import (
	"flag"
	"fmt"
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
	slog.Info("Parsing config file.", "path", configPath)
	config, err := parseConfigs(configPath)
	if err != nil {
		slog.Error("Failed to parse config.", "error", err)
		os.Exit(1)
	}

	slog.Info("Setting log level.", "level", config.Server.LogLevel)
	slog.SetLogLoggerLevel(getLogLevel(config.Server.LogLevel))

	if err := initTemplates(); err != nil {
		slog.Error("Failed to parse templates.", "error", err)
		os.Exit(1)
	}

	//mock memory store
	store, storeErr := NewServiceStore(config.Server.Storage, *noHistory)
	if storeErr != nil {
		slog.Error("Unable to initialize storage.", "error", storeErr)
		os.Exit(1)
	}

	store.ReconcileServices(config.Services)

	//start server
	server := config.Server
	server.publisher = NewPublisher()
	server.store = store
	mux := http.NewServeMux()
	server.addRoutes(mux)
	loggedMux := loggingMiddleware(mux)
	slog.Info("Starting Service Sleuth Server.", "version", Version, "build_time", BuildTime, "port", server.Port)
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
			slog.Error("Failed server state.", "error", err)
			os.Exit(1)
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
					slog.Info("Received shutdown signal. Cleaning up.", "signal", sig.String())
					if err := store.Save(); err != nil {
						slog.Error("Failed to save store during shutdown.", "error", err)
					}
					os.Exit(0)
				case syscall.SIGHUP:
					slog.Info("Received signal. Reloading config.", "signal", sig.String())
					newConfig, err := parseConfigs(configPath)
					if err != nil {
						slog.Error("Config reload failed, keeping previous config.", "error", err)
						continue
					}

					// Stop every monitor: running goroutines hold a stale copy
					// of their Service, so surviving services must be restarted
					// to pick up config changes (timer, address, protocol, ...).
					for _, s := range store.GetServices() {
						scheduler.RemoveService(s.ID)
					}

					// Sync the store (remove stale, update existing, add new)
					store.ReconcileServices(newConfig.Services)

					// Restart monitors from the reconciled store copies so they
					// carry both updated config and preserved runtime state.
					for _, s := range store.GetServices() {
						scheduler.AddService(s)
					}
					slog.Info("Config reloaded successfully.")
				default:
					slog.Warn("Signal caught, but not yet implemented.", "signal", sig.String())
				}
			}
		}
	}()

	//block until goroutines are cleaned up
	select {}

}
