package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"sync"
	"time"
)

type EventData struct {
	ServiceID uint      `json:"serviceID"`
	Status    bool      `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func Scheduler(store ServiceStore, channel chan<- []byte) {

	var wg sync.WaitGroup
	wg.Add(len(*store.GetServices()))

	log.Println("Starting go routines.")

	eventBus := make(chan []byte)

	servicesSlice := store.GetServices()
	for _, service := range *servicesSlice {
		go monitorService(service, eventBus)
	}

	for {
		eventData := <-eventBus
		var event EventData
		err := json.Unmarshal(eventData, &event)
		if err != nil {
			slog.Error("Unable to Unmarshal event update.", "Error", err)
		}

		// update server.store
		err = store.EventUpdate(event)
		if err != nil {
			slog.Error("Unable to handle event update.", "Error", err)
		}

		//send event to server to distribute to active connections
		channel <- eventData
	}

	//wg.Wait()
	//log.Printf("All %d services cleaned up.", len(*store.GetServices()))
}

func monitorService(service Service, eventBus chan<- []byte) {

	for {
		var event EventData
		response := service.getStatus()
		event.Status = response.Status
		event.Timestamp = response.timestamp
		event.ServiceID = service.ID

		update, err := json.Marshal(event)
		if err != nil {
			slog.Error("Error marshalling JSON. ", "error", err)
			continue
		}
		eventBus <- update
		time.Sleep(time.Duration(service.Timer) * time.Second)
	}
}
