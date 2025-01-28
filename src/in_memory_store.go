package main

import (
//	"encoding/json"
//	"log"
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

func (i *InMemoryStore) EventUpdate(eventData []byte) {


}
