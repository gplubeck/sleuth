package main

import (
    "fmt"
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

func Scheduler(services ServiceStore){
    for {
        servicesSlice := services.GetServices()

        for i, service := range *servicesSlice{ 
            s := service

            go func (s Service){
                resp := s.getStatus()
                if (resp.Status){
                    fmt.Println(s.Name+" is up!")
                    (*servicesSlice)[i].Status = true
                }else{
                    fmt.Println(s.Name+" currently down.")
                    (*servicesSlice)[i].Status = false 
                }
            }(s)
        time.Sleep(2*time.Second)
     }
    }
}

