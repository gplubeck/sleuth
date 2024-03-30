package main

import (
	"fmt"
	"net"
	"time"
)

type Service struct {
    Name string `json:"service_name"`
    Address string `json:"address"`//maybe should eventual be its own IP/FQDN type
    Protocol Protocol `json:"protocol"`//interface to allow for Strategy Pattern and future expansion
    Start time.Time //`json:"start_time`
    lastUpdate time.Time // `json:"update_time"`
    Status bool `json:"status"`
    timer int
    Failed []response 
}


/*****************************************
* Strategy pattern for different protocols
*****************************************/

type Protocol interface {
    Connect(address string, timeout time.Duration) (net.Conn, error)
}

type TCPProtocol struct{}

//create TCP connection
func (t *TCPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error){
    return net.DialTimeout("tcp", address, timeout)
}

type UDPProtocol struct{}
//create UDP connection
func (u *UDPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error){
    return net.DialTimeout("udp", address, timeout)
}

// Generic unimplemented type to allow for future growth
type HTTPProtocol struct{}

func (h *HTTPProtocol) Connect(address string, timout time.Duration)(net.Conn, error){
    //stubbed interface not yet implemented
    return nil, nil
}


/**************************************
* Repsonse type for getStatus
* Should return good/bad and timestamp
***************************************/
type response struct {
    Status bool
    timestamp time.Time
}


func  (service *Service) getStatus() response{

    var resp response 
    resp.timestamp = time.Now()

    conn, err := service.Protocol.Connect(service.Address, 2*time.Second)
    if err != nil {
        resp.Status = false
    }else{
        resp.Status = true 
        //defer in here because conn.Close on an error will segfault
        defer conn.Close()
    }

    return resp
}

/*********************************************************
* Used to return string that should be used as html content
***********************************************************/
func  (service *Service) String() string{
    element := fmt.Sprintf("id: %d\nevent: %s\ndata: <div>%s</div>\n\n", 1, service.Name, service.Name)
    return element
}
/*
func  (service *Service) String() string{
    element := fmt.Sprintf(`event: %s
    data: <tr> <td>%s</td>
        <td>
           %t 
        </td>
        <td><a href='%s'>%s</a></td>
        </tr>
`, service.Name, service.Name, service.Status, service.Address, service.Address)

    return element

}
*/
