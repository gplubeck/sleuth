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
    }

    //must have colon address! need to check for this.
    notes := Service{
        Name: "Notes Page",
        Address: "notes.gplubeck.com:443",
        Protocol: &UDPProtocol{},
        Start: time.Now(),
        Status: false,
    }


    //mock memory store
    store := NewInMemoryStore()
    store.AddService(git)
    store.AddService(notes)

    updateChannel := make(chan string)

    //start scheduler
    go Scheduler(store, updateChannel)

    //start server
    server := NewServiceServer(store, updateChannel)
    log.Fatal(http.ListenAndServe(port, server))

}
