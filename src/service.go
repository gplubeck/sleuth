package main

import (
	"bytes"
	"errors"
	"html/template"
	"log/slog"
	"math/rand"
	"net"
	"sleuth/internal/ringbuffer"
	"strings"
	"time"
)

type Service struct {
	ID             uint                             `toml:"id"`
	Name           string                           `toml:"service_name"`
	Address        string                           `toml:"address"`  //maybe should eventual be its own IP/FQDN type
	Link           string                           `toml:"link"`     //used for onclick functionality must provide http:// or https:// if left blank, no link
	protocol       Protocol                         `toml:"protocol"` //interface to allow for Strategy Pattern and future expansion
	ProtocolString string                           `toml:"protocol_str"`
	Start          time.Time                        `toml:"start_time"`
	LastUpdate     time.Time                        `toml:"update_time"`
	Status         bool                             `toml:"status"`
	Uptime         float64                          `toml:"uptime"`
	Timer          int                              //how often to check service in seconds
	Icon           string                           //name of icon to use from /assets/icons
	History        ringbuffer.RingBuffer[EventData] `toml:"uptime_history"`
    MaxHistorySize int                              `toml:"maxHistory"` //number of Events to hold
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

type TCPProtocol struct {
	Protocol string
}

func (t *TCPProtocol) String() string {
	return t.Protocol
}

// create TCP connection
func (t *TCPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", address, timeout)
}

type UDPProtocol struct {
	Protocol string
}

func (u *UDPProtocol) String() string {
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

func (t *TestProtocol) String() string {
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

	conn, err := service.protocol.Connect(service.Address, 2*time.Second)
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
	var uptime float64
	var success float64

	for _, event := range service.History.GetAll() {
		if event.Status {
			success++
		}
	}

	uptime = (success / float64(service.History.GetSize()) * 100)
	return uptime

}

/**********************************************
* Changes service start time if old start time 
* is overwritten in RingBuffer
************************************************/
func (service *Service) updateStart() time.Time{

    // tail will be oldest in ringbuff
    tail, isEmpty := service.History.Peek()
    //should never happedn
    if isEmpty {
        return time.Time{}
    }

	return tail.Timestamp

}
/*************************************************************
* Used to return string that will be used to swap html content
*************************************************************/
func (service *Service) toHTML() string {

	templateStr := service.templateStr()
	tmpl, err := template.New("serviceElement").Funcs(template.FuncMap{
		"getAllHistory": getAllHistory,
	}).Parse(templateStr)
	if err != nil {
		slog.Error("Unable to create parse template.", "service", service.ID, "error", err.Error())
	}

	var tmplOutput bytes.Buffer
	err = tmpl.Execute(&tmplOutput, service)

	return tmplOutput.String()
}

func (service *Service) templateStr() string {
	tmpl, err := template.New("service-card").Funcs(template.FuncMap{
		"getAllHistory": getAllHistory,
		"formatTime":    formatTime,
	}).ParseFiles("static/templates/service_card.gohtml",
		"static/templates/service_header.gohtml",
		"static/templates/service_body.gohtml")

	if err != nil {
		slog.Error("Failed to parse serviceElement template.", "Error", err.Error())
	}

	var tmplOutput bytes.Buffer
	err = tmpl.Execute(&tmplOutput, service)
	if err != nil {
		slog.Error("Failed execute serviceElement template.", "Error", err.Error())
	}

	templateStr := tmplOutput.String()
	// remove newlines so we don't mess up server side messages
	templateStr = strings.ReplaceAll(templateStr, "\n", "")

	return templateStr
}

// func for templates to range
func getAllHistory(buffer ringbuffer.RingBuffer[EventData]) []EventData {
	return buffer.GetAll()
}
