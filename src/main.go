package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
    configPath := "config.toml"
    log.Printf("Parsing config file: %s", configPath)
    config := parseConfigs(configPath)

    log.Printf("Setting log level: %s", config.Server.LogLevel)
    slog.SetLogLoggerLevel(getLogLevel(config.Server.LogLevel))

	//mock memory store
	store := NewInMemoryStore()

    for _, service := range config.Services {
        store.AddService(service)
    }

    // event bus from scheduler publisher
	updateChannel := make(chan []byte)

	//start scheduler
	go Scheduler(store, updateChannel)

	//start server
	server := NewServer(store, updateChannel)
    server.Cert_key = config.Server.Cert_key
    server.Cert_file = config.Server.Cert_file
    server.Port = config.Server.Port
    mux := http.NewServeMux()
    server.addRoutes(mux)
    log.Printf("Starting Service Sleuth Server version: %s", Version)
    log.Printf("Build Time: %s", BuildTime)
    log.Printf("Listening on port: %d", server.Port)

    go func() {
        port := fmt.Sprintf(":%d", server.Port)
        err := http.ListenAndServe(port, mux)
        if err != nil {
            log.Fatalf("Failed server state.  Error: %s", err)
        }
    }()

    //set up control flow of program via signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
    go func(){
        for {
            select {
            case sig := <- sigChan:
                switch sig{
                case syscall.SIGINT, syscall.SIGTERM:
                    log.Printf("Received signal %s. Cleaning up....", sig)
                    os.Exit(0)
                default:
                    log.Printf("%s caught, but not yet implemented.", sig)
                }
            }
        }
    }()

    //block until goroutines are cleaned up
    select{}

}

