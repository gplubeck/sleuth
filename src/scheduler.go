package main

import (
	"encoding/json"
    "log"
	"sync"
	"time"
)

type EventData struct {
	ServiceID uint `json:"serviceID"`
    Status bool `json:"status"`
    Timestamp time.Time `json:"timestamp"`
}

func Scheduler(services ServiceStore, channel chan<- []byte) {

	var wg sync.WaitGroup
	wg.Add(len(*services.GetServices()))

	log.Println("Starting go routines.")
	servicesSlice := services.GetServices()
	for _, service := range *servicesSlice {
		go monitorService(service, channel)
	}

	wg.Wait()
	log.Printf("All %d services cleaned up.", len(*services.GetServices()))
}

func monitorService(service Service, channel chan<- []byte) {

	for {
        var event EventData
        response := service.getStatus()
		event.Status = response.Status
		event.Timestamp = response.timestamp
        event.ServiceID = service.ID

		update, err := json.Marshal(event)
		if err != nil {
			log.Println("Error marshalling JSON: ", err)
			continue
		}
		channel <- update
		time.Sleep(time.Duration(service.Timer) * time.Second)
	}
}
