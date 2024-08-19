package main

import (
	"encoding/json"
    "log"
	"sync"
	"time"
)

type EventData struct {
	Counter int `json:"counter"`
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
		response := service.getStatus()
		service.Status = response.Status
		service.LastUpdate = response.timestamp
        if !response.Status {
            service.Failed = append(service.Failed, response)
        }

        service.getUptime()

		update, err := json.Marshal(service)
		if err != nil {
			log.Println("Error marshalling JSON: ", err)
			continue
		}
		channel <- update
		time.Sleep(time.Duration(service.timer) * time.Second)
	}
}
