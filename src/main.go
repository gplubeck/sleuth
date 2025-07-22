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

	//mock memory store
	store, err := NewServiceStore(config.Server.Storage, *noHistory)
	if err != nil {
		log.Fatalf("Unable to intialize storage. Error: %s", err)
	}

	for _, service := range config.Services {
        //test if service is already in store
        _, err := store.GetServiceByID(service.ID)
        // err means could not find service
        if err != nil {
            store.AddService(service)
        }
	}

	// event bus from scheduler publisher
	updateChannel := make(chan []byte)


	//start server
	server := config.Server
	server.channel = updateChannel
    server.publisher = NewPublisher()
	server.store = store
	mux := http.NewServeMux()
	server.addRoutes(mux)
	log.Printf("Starting Service Sleuth Server version: %s", Version)
	log.Printf("Build Time: %s", BuildTime)
	log.Printf("Listening on port: %d", server.Port)
	slog.Debug("Server cert files", "key", server.Cert_key, "cert", server.Cert_file)

    //complete start up send them goroutines 
    go server.publisher.Start()
	//go Scheduler(store, updateChannel)
	go Scheduler(store, server.publisher.publish)

	go func() {
		var err error
		port := fmt.Sprintf(":%d", server.Port)
		if server.Cert_file != "" && server.Cert_key != "" {
			slog.Info("Starting TLS Server.")
			err = http.ListenAndServeTLS(port, server.Cert_file, server.Cert_key, mux)
		} else {
			slog.Warn("Starting Plaintext Server.")
			err = http.ListenAndServe(port, mux)
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
					log.Printf("Received signal %s. Reload configs.", sig)
					log.Printf("Oh wait... Reminder to implement this.")
				default:
					log.Printf("%s caught, but not yet implemented.", sig)
				}
			}
		}
	}()


    //block until goroutines are cleaned up
    select{}

}
