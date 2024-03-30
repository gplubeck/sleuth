package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

/*
func Scheduler(services ServiceStore){
    for {
        servicesSlice := services.GetServices()

        for _, service := range servicesSlice{
            //item := i
            fmt.Println(service.Name)
            fmt.Println(service.Address)
                resp := service.getStatus()
                if (resp.Status){
                    fmt.Println(service.Name+" is up!")
                }else{
                    fmt.Println(service.Name+" currently down.")
                    //service.Failed = append(s.Failed, resp)
                }
        time.Sleep(2*time.Second)
     }
    }
}
*/


type EventData struct {
    Counter int `json:"counter"`
}

func Scheduler(services ServiceStore, channel chan<- []byte){
   
    var wg sync.WaitGroup
    wg.Add(len(*services.GetServices()))

    fmt.Println("Starting go routines.")
    servicesSlice := services.GetServices()
    for _, service := range *servicesSlice{ 

        go monitorService(service, channel)

    }

    wg.Wait()
    fmt.Println("all cleaned up.")
}

func monitorService(service Service, channel chan<- []byte){

    for {
        response := service.getStatus()
        service.Status = response.Status
        service.lastUpdate= response.timestamp
        update, err := json.Marshal(service)
        if err != nil {
            //NEED TO ADD LOG MESSAGE
            fmt.Println("Error marshalling JSON: ", err)
            continue
        }
        channel <- update 
        time.Sleep(time.Duration(service.timer) * time.Second)
    }
}

/*
    for {
        servicesSlice := services.GetServices()

        for _, service := range *servicesSlice{ 
            fmt.Println("test1")
            data, err := json.Marshal(service)
            if err != nil {
                fmt.Println("Error marshalling JSON: ", err)
                continue
            }
            channel <- data
        }
        */
