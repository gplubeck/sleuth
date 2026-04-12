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
	i.Lock()
	defer i.Unlock()
	return &i.store
}
// Add service to slice
// Todo add autogenerating IDs that figure out if it is a new service or just updated
func (i *InMemoryStore) AddService(service Service) {
	//embedded struct mutex
	i.Lock()
	defer i.Unlock()
	i.store = append(i.store, service)
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

// ReconcileServices syncs the store against the provided config slice:
//   - Services in config but not store are added fresh.
//   - Services in store but not config are removed.
//   - Services in both keep their runtime state (history, uptime, status)
//     but have their config fields updated (name, address, timer, protocol, icon, link).
func (i *InMemoryStore) ReconcileServices(services []Service) {
	i.Lock()
	defer i.Unlock()

	// Index config services by ID for O(1) lookups
	configMap := make(map[uint]Service, len(services))
	for _, s := range services {
		configMap[s.ID] = s
	}

	// Drop store entries that are no longer in config
	kept := make([]Service, 0, len(i.store))
	for _, s := range i.store {
		if _, inConfig := configMap[s.ID]; inConfig {
			kept = append(kept, s)
		} else {
			slog.Info("Removing service no longer in config.", "id", s.ID, "name", s.Name)
		}
	}
	i.store = kept

	// Update config-driven fields on services that survived, preserving runtime state
	for idx := range i.store {
		cfg := configMap[i.store[idx].ID]
		i.store[idx].Name = cfg.Name
		i.store[idx].Address = cfg.Address
		i.store[idx].Timer = cfg.Timer
		i.store[idx].ProtocolString = cfg.ProtocolString
		i.store[idx].protocol = cfg.protocol
		i.store[idx].Icon = cfg.Icon
		i.store[idx].Link = cfg.Link
	}

	// Add services that are in config but not yet in the store
	storeIDs := make(map[uint]bool, len(i.store))
	for _, s := range i.store {
		storeIDs[s.ID] = true
	}
	for _, s := range services {
		if !storeIDs[s.ID] {
			slog.Info("Adding new service from config.", "id", s.ID, "name", s.Name)
			i.store = append(i.store, s)
		}
	}
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
            i.store[idx].Start = i.store[idx].updateStart()
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

