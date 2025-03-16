package main

import (
	//	"encoding/json"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"sync"
)

type InMemoryStore struct {
	sync.Mutex //embedded Mutex for appending slice
	store      []Service
}

func NewInMemoryStore() *InMemoryStore {
	i := new(InMemoryStore)
	i.store = []Service{}
	return i
}

func (i *InMemoryStore) GetServices() *[]Service {
	return &i.store
}
func (i *InMemoryStore) AddService(service Service) {
	//embedded struct mutex
	i.Lock()
	i.store = append(i.store, service)
	i.Unlock()
}

func (i *InMemoryStore) GetServiceByID(id uint) (*Service, error) {
    i.Lock()         
    defer i.Unlock()

    for _, service := range i.store {
        if service.ID == id {
            return &service, nil
        }
    }
    return nil, errors.New("Unable to get service.")
}

func (i *InMemoryStore) EventUpdate(eventData []byte) error {

    var event EventData
    err := json.Unmarshal(eventData, &event)

    if err != nil {
        slog.Error("Unable to handle event update.", "Error", err)
    }

    _, err = i.GetServiceByID(event.ServiceID)
    if err != nil {
        return err
    }

    i.Lock()
    defer i.Unlock()
    for idx, service := range i.store {
        if service.ID == event.ServiceID { 
            i.store[idx].LastUpdate = event.Timestamp
            i.store[idx].Status = event.Status
            i.store[idx].History = append(i.store[idx].History, event)
            i.store[idx].Uptime = i.store[idx].getUptime()
            log.Printf("updated: %v", i.store[idx])
            break
        }
    }

    /*
    service.LastUpdate = event.Timestamp
    service.Status = event.Status
    service.History = append(service.History, event)
    service.Uptime = service.getUptime()
    log.Printf("ServiceID: %d\n uptime: %f+", service.ID, service.Uptime)
    */

    return nil

}
