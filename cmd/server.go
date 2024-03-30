package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
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
    channel <-chan []byte 
}

func NewServiceServer(store ServiceStore, ch <-chan []byte) *ServiceServer {
    service := new(ServiceServer)
    service.store = store

    service.channel = ch

    router := http.NewServeMux()
    router.Handle("/", http.HandlerFunc(service.statusHandler))
    service.Handler = router
    router.Handle("/updates", http.HandlerFunc(service.updateHandler))
    //define themes file server
    fs := http.FileServer(http.Dir("themes"))
    router.Handle("/themes/", http.StripPrefix("/themes/", fs))

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
        
    // Initialize counter
        // Infinite loop to send events every second
for{
                // Send event
                eventData:= <-server.channel

                fmt.Fprintf(w, "data: %s\n\n", eventData)
                w.(http.Flusher).Flush() // Flush the response writer to send the event immediately

                // Delay for one second before sending the next event
                time.Sleep(1 * time.Second)
            }

    //send data
    /*go func() {
        //for  serviceUpdate := range server.channel{
            //serviceUpdate := <-server.channel
            //channel is receiveing event: service.Name
            // data: new div
            serviceUpdate := `{"counter": "1"}`
            
            //fmt.Fprintf(w, `data: {"service_name":"help"}\n\n`)
            fmt.Fprintf(w, "data: %s\n\n", serviceUpdate)
            w.(http.Flusher).Flush()
            fmt.Printf("Update: %s", serviceUpdate)
        //}
    }()
*/
    /*simulate sending data
    for {
        updates <- time.Now().Format(time.RFC3339)
        time.Sleep(1 * time.Second)
    }
    */


}
