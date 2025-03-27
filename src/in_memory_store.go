package main

import (
	//	"encoding/json"
	"bytes"
	"encoding/gob"
	"errors"
	"log/slog"
	"os"
	"sync"
)

type InMemoryStore struct {
	sync.Mutex //embedded Mutex for appending slice
	store      []Service
}

func NewInMemoryStore(noHistory bool) (*InMemoryStore, error) {
	i := new(InMemoryStore)
	i.store = []Service{}
    // check for gob storage
    isFile, _ := os.Stat(".sleuth.bin")
    
    // if gob storage exists, load
    if isFile != nil  && !noHistory {
        i.Load()
    }

	return i, nil
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
			slog.Debug("Updating service", "service", i.store[idx].Name)
			break
		}
	}

	return nil
}

func (i *InMemoryStore) Save() error {
    buffer := new(bytes.Buffer)
    encoder := gob.NewEncoder(buffer)
    err := encoder.Encode(i.store)
    if err != nil {
        slog.Error("Failed to gob encode data.", "error", err)
        return err
    }

    err = os.WriteFile(".sleuth.bin", buffer.Bytes(), 0600)
    if err != nil {
        slog.Error("Failed to write gob data.", "error", err)
        return err
    }

    return nil
}

func (i *InMemoryStore) Load() error {
    raw, err := os.ReadFile(".sleuth.bin")
    if err != nil {
        slog.Error("Failed to read gob data.", "error", err)
        return err
    }

    var tmpStore *[]Service

    buffer := bytes.NewBuffer(raw)
    dec := gob.NewDecoder(buffer)
    err = dec.Decode(&tmpStore)
    if err != nil {
        slog.Error("Failed to decode gob data.", "error", err)
        return err
    }

    if tmpStore != nil {
        for _, service := range *tmpStore {
            // make the interface
            if service.protocol == nil {
                service.protocol = NewProtocol(service.ProtocolString)
            }
            i.store = append(i.store, service)
        }
        slog.Info("Loading previous storage into memory.")
    }

    return nil
}

