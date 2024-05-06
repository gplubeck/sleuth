package main

import (
    "net/http"
    "log"
    "time"
)


func main () {
    port := ":5000"

    //read in config file stuff
    git := Service{
        Name: "Gitea",
        Address: "git.grant:3000",
        Protocol: &TCPProtocol{},
        Start: time.Now(),
        Status: false,
        timer: 5,
    }

    //must have colon address! need to check for this.
    notes := Service{
        Name: "Notes Page",
        Address: "notes.gplubeck.com:443",
        Protocol: &UDPProtocol{},
        Start: time.Now(),
        Status: false,
        timer: 3,
    }
    
    //must have colon address! need to check for this.
    homepage := Service{
        Name: "Homepage",
        Address: "gplubeck.com:443",
        Protocol: &UDPProtocol{},
        Start: time.Now(),
        Status: false,
        timer: 3,
    }
    
    google := Service{
        Name: "Google",
        Address: "8.8.8.8",
        Protocol: &TCPProtocol{},
        Start: time.Now(),
        Status: false,
        timer: 3,
    }
    
    fake := Service{
        Name: "Down Service",
        Address: "8.8.8.8",
        Protocol: &TCPProtocol{},
        Start: time.Now(),
        Status: false,
        timer: 3,
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
    log.Fatal(http.ListenAndServe(port, server))

}
