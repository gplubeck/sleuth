package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := ":5000"

	//read in config file stuff
	git := Service{
        ID: 0,
		Name:     "Gitea",
		Address:  "git.grant:3000",
		Link:     "http://git.grant:3000",
		Protocol: &TCPProtocol{},
		Start:    time.Now(),
		Status:   false,
		timer:    5,
        timeWindow: 1,

	}

	//must have colon address! need to check for this.
	notes := Service{
        ID: 1,
		Name:     "NotesPage",
		Address:  "notes.gplubeck.com:443",
		Link:     "https://notes.gplubeck.com",
		Protocol: &UDPProtocol{},
		Start:    time.Now(),
		Status:   false,
		timer:    30,
        timeWindow: 10,
	}

	//must have colon address! need to check for this.
	homepage := Service{
        ID: 2,
		Name:     "Homepage",
		Address:  "gplubeck.com:443",
		Link:     "https://gplubeck.com:443",
		Protocol: &UDPProtocol{},
		Start:    time.Now(),
		Status:   false,
		timer:    30,
        timeWindow: 5,
	}

	google := Service{
        ID: 3,
		Name:     "Google",
		Address:  "8.8.8.8",
		Link:     "https://google.com",
		Protocol: &TCPProtocol{},
		Start:    time.Now(),
		Status:   false,
		timer:    30,
        timeWindow: 1,
	}

	fake := Service{
        ID: 4,
		Name:     "DownService",
		Address:  "8.8.8.8",
		Protocol: &TCPProtocol{},
		Start:    time.Now(),
		Status:   false,
		timer:    30,
        timeWindow: 1,
	}

	//mock memory store
	store := NewInMemoryStore()
	store.AddService(git)
	store.AddService(notes)
	store.AddService(homepage)
	store.AddService(google)
	store.AddService(fake)

	updateChannel := make(chan []byte)

	//start scheduler
	go Scheduler(store, updateChannel)

	//start server
	server := NewServiceServer(store, updateChannel)
    mux := http.NewServeMux()
    server.addRoutes(mux)
    log.Printf("Starting Service Sleuth Server version: %s", Version)
    log.Printf("Build Time: %s", BuildTime)
    log.Printf("Listening on port%s", port)

    go func() {
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
