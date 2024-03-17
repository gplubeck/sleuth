package main

import (
	"context"
	"fmt"
	"net/http"
)

/*************************************************
* Interface type that will serve
* as way to maintain all services being monitored
*************************************************/

type ServiceStore interface {
    AddService(service Service)
    GetServices() *[]Service
}

type ServiceServer struct {
    store ServiceStore
    //store InMemoryStore 
    http.Handler
    
    //channel for json updates
    channel <-chan string 
}

func NewServiceServer(store ServiceStore, ch <-chan string) *ServiceServer {
    service := new(ServiceServer)
    service.store = store

    service.channel = ch

    router := http.NewServeMux()
    router.Handle("/", http.HandlerFunc(service.statusHandler))
    service.Handler = router
    router.Handle("/updates", http.HandlerFunc(service.updateHandler))

    return service
}


func (server *ServiceServer) statusHandler(w http.ResponseWriter, r *http.Request){
    w.Header().Set("Content-Type", "text/html")

    StatusTemplate(server.store, w)
}

func (server *ServiceServer) updateHandler(w http.ResponseWriter, r *http.Request){
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    //create channel
    //TODO move to global so scheduler can use
    //updates := make(chan string)

    //create conect for handling client disconnect
    _, cancel := context.WithCancel(r.Context())
    defer cancel()

    //send data
    go func() {
        //for data := range server.channel{
        serviceUpdate := <-server.channel
            //channel is receiveing event: service.Name
            // data: new div
            fmt.Fprintf(w, "%s", serviceUpdate)
            w.(http.Flusher).Flush()
            fmt.Printf("%s", serviceUpdate)
    }()

    /*simulate sending data
    for {
        updates <- time.Now().Format(time.RFC3339)
        time.Sleep(1 * time.Second)
    }
    */

}

