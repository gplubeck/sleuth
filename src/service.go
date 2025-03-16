package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

type Service struct {
    ID          uint    `json:"id"`
    Name       string    `json:"service_name"`
    Address    string    `json:"address"`  //maybe should eventual be its own IP/FQDN type
    Link       string    `json:"link"`     //used for onclick functionality must provide http:// or https:// if left blank, no link
    Protocol   Protocol  `json:"protocol"` //interface to allow for Strategy Pattern and future expansion
    ProtocolString string `toml:"protocol_str"`
    Start      time.Time `json:"start_time"`
    LastUpdate time.Time  `json:"update_time"`
    Status     bool      `json:"status"`
    Uptime  float64     `json:"uptime"`
    Timer      int  // how often to check service in seconds
    Icon    string // name of icon to use from /assets/icons
    History     []EventData `json:"uptime_history"`
}

/*****************************************
* Strategy pattern for different protocols
*****************************************/
const SECONDS_PER_DAY = 86400

type Protocol interface {
    String() string
	Connect(address string, timeout time.Duration) (net.Conn, error)
}

func NewProtocol(protocol string) Protocol {
    switch protocol {
    case "TCP":
        return &TCPProtocol{Protocol: "TCP"}
    case "UDP":
        return &UDPProtocol{Protocol: "UDP"}
    case "Test":
        return &TestProtocol{Protocol: "Test"}
    default:
        return nil
    }
}


type TCPProtocol struct{
    Protocol string 
}

func (t *TCPProtocol) String() string{
    return t.Protocol
}

// create TCP connection
func (t *TCPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, timeout)
}

type UDPProtocol struct{
    Protocol string
}

func (u *UDPProtocol) String() string{
    return u.Protocol
}

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

type TestProtocol struct {
    Protocol string 
}

func (t *TestProtocol) String() string{
    return t.Protocol
}

func (t *TestProtocol) Connect(address string, timeout time.Duration) (net.Conn, error) {
    
    test := rand.Intn(2) == 0
    if test {
        return nil, nil
    }
    return nil, errors.New("Random failure")

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
        if conn != nil {
            defer conn.Close()
        }
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
    var success float64


    for _, event := range service.History {
        if event.Status{
            success++
        }
    }

    uptime = (success/float64(len(service.History)) * 100)
    
    return uptime

}

/*********************************************************
* Used to return string that should be used as html content
***********************************************************/
func (service *Service) String() string {
	element := fmt.Sprintf("id: %d\nevent: %s\ndata: <div>%s</div>\n\n", 1, service.Name, service.Name)
	return element
}
