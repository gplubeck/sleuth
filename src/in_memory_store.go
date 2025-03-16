package main

import (
	//	"encoding/json"
	"errors"
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

func (i *InMemoryStore) EventUpdate(event EventData) error {

    _, err := i.GetServiceByID(event.ServiceID)
    if err != nil {
        return err
    }

    i.Lock()
    defer i.Unlock()
    for idx, service := range i.store {
        if service.ID == event.ServiceID { 
            i.store[idx].LastUpdate = event.Timestamp
            i.store[idx].Status = event.Status
            i.store[idx].History.Push(event)
            i.store[idx].Uptime = i.store[idx].getUptime()
            slog.Debug("Updating service", "service",  i.store[idx])
            break
        }
    }

    return nil

}
