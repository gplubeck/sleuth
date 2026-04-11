package main

import (
	"encoding/json"
	"log/slog"
	"time"
)

type EventData struct {
	ServiceID uint      `json:"serviceID"`
	Status    bool      `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func Scheduler(store ServiceStore, channel chan<- []byte) {
	slog.Debug("Starting Scheduler go routines.")

	eventBus := make(chan []byte)

	servicesSlice := store.GetServices()
	for _, service := range *servicesSlice {
		go monitorService(service, eventBus)
	}

	saveTicker := time.NewTicker(30 * time.Second)
	defer saveTicker.Stop()

	for {
		select {
		case eventData := <-eventBus:
			var event EventData
			if err := json.Unmarshal(eventData, &event); err != nil {
				slog.Error("Unable to Unmarshal event update.", "Error", err)
				continue
			}

			// update server.store
			if err := store.EventUpdate(event); err != nil {
				slog.Error("Unable to handle event update.", "Error", err)
			}

			// send event to server to distribute to active connections
			channel <- eventData

		case <-saveTicker.C:
			if err := store.Save(); err != nil {
				slog.Error("Failed to persist store.", "error", err)
			}
		}
	}
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
        slog.Debug("Sending update to scheduluer.", "service", service.ID)
		eventBus <- update
		time.Sleep(time.Duration(service.Timer) * time.Second)
	}
}
