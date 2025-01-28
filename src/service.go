package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Service struct {
    ID          uint    `json:"id"`
    Name       string    `json:"service_name"`
    Address    string    `json:"address"`  //maybe should eventual be its own IP/FQDN type
    Link       string    `json:"link"`     //used for onclick functionality must provide http:// or https:// if left blank, no link
    Protocol   Protocol  `json:"protocol"` //interface to allow for Strategy Pattern and future expansion
    Start      time.Time //`json:"start_time`
    LastUpdate time.Time  `json:"update_time"`
    Status     bool      `json:"status"`
    Uptime  float64     `json:"uptime"`
    timer      int  // how often to check service in seconds
    timeWindow int  //time window for calculating uptime in days
    itersPerWindow int  //number of timer iterations time window 
    Icon    string // name of icon to use from /assets/icons
    Failed     []response
    History     []response `json:"uptime_history"`
}

/*****************************************
* Strategy pattern for different protocols
*****************************************/
const SECONDS_PER_DAY = 86400

type Protocol interface {
	Connect(address string, timeout time.Duration) (net.Conn, error)
}

type TCPProtocol struct{}

// create TCP connection
func (t *TCPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, timeout)
}

type UDPProtocol struct{}

// create UDP connection
func (u *UDPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("udp", address, timeout)
}

// Generic unimplemented type to allow for future growth
// Use for implementing things like Githubs health check
type HTTPProtocol struct{}

func (h *HTTPProtocol) Connect(address string, timout time.Duration) (net.Conn, error) {
	//stubbed interface not yet implemented
	return nil, nil
}

/**************************************
* Repsonse type for getStatus
* Should return good/bad and timestamp
***************************************/
type response struct {
	Status    bool
	timestamp time.Time
}

func (service *Service) getStatus() response {

	var resp response
	resp.timestamp = time.Now()

	conn, err := service.Protocol.Connect(service.Address, 2*time.Second)
	if err != nil {
		resp.Status = false
	} else {
		resp.Status = true
		//defer in here because conn.Close on an error will segfault
		defer conn.Close()
	}

	return resp
}


/******************************************************
* Returns float64 that represents uptime of the service
* as a percentage of either window selected or if less
* than 1 full window, the total time running
*******************************************************/
func (service *Service) getUptime() float64 {
    log.Printf("%s uptime function %q", service.Name, service)
    var uptime float64

    if service.itersPerWindow == 0 {
        service.itersPerWindow = (SECONDS_PER_DAY * service.timeWindow) / service.timer
    }

    current := time.Now()
    totalTime := current.Sub(service.Start)
    totalIters := int( totalTime.Seconds() / float64(service.timer))

    if totalIters == 0 {
        if service.Status{
            service.Uptime = 100
        }else{
            service.Uptime = 0
        }
    }else if totalIters < service.itersPerWindow {
        uptime = 100 - ( float64(len(service.Failed) / totalIters) )
    }else {
        uptime = 100 - ( float64(len(service.Failed) / service.itersPerWindow) )
    }

    service.Uptime = uptime
    log.Printf("%s uptime %f after %d iterations.", service.Name, service.Uptime, totalIters)

    return uptime

}

/*********************************************************
* Used to return string that should be used as html content
***********************************************************/
func (service *Service) String() string {
	element := fmt.Sprintf("id: %d\nevent: %s\ndata: <div>%s</div>\n\n", 1, service.Name, service.Name)
	return element
}
