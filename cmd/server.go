package main

import (
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
}

func NewServiceServer(store ServiceStore) *ServiceServer {
    service := new(ServiceServer)
    service.store = store

    router := http.NewServeMux()
    router.Handle("/", http.HandlerFunc(service.statusHandler))
    service.Handler = router

    return service
}


func (server *ServiceServer) statusHandler(w http.ResponseWriter, r *http.Request){
    w.Header().Set("content-type", "text/html")
    StatusTemplate(server.store, w)
}

