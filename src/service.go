package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"sleuth/internal/ringbuffer"
	"strings"
	"time"
)

type Service struct {
	ID             uint                             `toml:"id"`
	Name           string                           `toml:"service_name"`
	Address        string                           `toml:"address"`  // host:port for TCP/UDP; full URL for HTTP
	Link           string                           `toml:"link"`     // used for onclick functionality; must include http:// or https://
	protocol       Protocol                         `toml:"protocol"` // interface to allow for Strategy Pattern and future expansion
	ProtocolString string                           `toml:"protocol_str"`
	Start          time.Time                        `toml:"start_time"`
	LastUpdate     time.Time                        `toml:"update_time"`
	Status         bool                             `toml:"status"`
	Uptime         float64                          `toml:"uptime"`
	Timer          int                              // how often to check service in seconds
	Icon           string                           // name of icon to use from /assets/icons
	History        ringbuffer.RingBuffer[EventData] `toml:"uptime_history"`
	MaxHistorySize int                              `toml:"maxHistory"` // number of Events to hold
	// HTTP-specific fields (ignored for TCP/UDP)
	HTTPExpectedStatus   int  `toml:"http_expected_status"`   // exact status code; 0 = use category
	HTTPExpectedCategory int  `toml:"http_expected_category"` // 1–5 (first digit); 0 = default to 2 (any 2xx)
	HTTPSkipTLSVerify    bool `toml:"http_skip_tls_verify"`   // skip TLS cert verification (self-signed certs)
}

/*****************************************
* Strategy pattern for different protocols
*****************************************/
const SECONDS_PER_DAY = 86400

type Protocol interface {
	String() string
	Connect(address string, timeout time.Duration) (net.Conn, error)
}

func NewProtocol(service Service) Protocol {
	switch service.ProtocolString {
	case "TCP":
		return &TCPProtocol{Protocol: "TCP"}
	case "UDP":
		return &UDPProtocol{Protocol: "UDP"}
	case "HTTP":
		return &HTTPProtocol{
			expectedStatus:   service.HTTPExpectedStatus,
			expectedCategory: service.HTTPExpectedCategory,
			client: &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: service.HTTPSkipTLSVerify},
				},
			},
		}
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

type HTTPProtocol struct {
	expectedStatus   int // 0 = unset, use category
	expectedCategory int // 0 = unset, default to 2 (any 2xx)
	client           *http.Client
}

func (h *HTTPProtocol) String() string {
	return "HTTP"
}

func (h *HTTPProtocol) Connect(address string, timeout time.Duration) (net.Conn, error) {
	h.client.Timeout = timeout

	resp, err := h.client.Get(address)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain so connection can be reused

	if h.expectedStatus != 0 {
		if resp.StatusCode != h.expectedStatus {
			return nil, fmt.Errorf("HTTP check failed: got %d, want %d", resp.StatusCode, h.expectedStatus)
		}
		return nil, nil
	}

	category := h.expectedCategory
	if category == 0 {
		category = 2 // default: any 2xx
	}
	if resp.StatusCode/100 != category {
		return nil, fmt.Errorf("HTTP check failed: got %d, want %dxx", resp.StatusCode, category)
	}
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
	var tmplOutput bytes.Buffer
	err := serviceCardTmpl.ExecuteTemplate(&tmplOutput, "service-card", service)
	if err != nil {
		slog.Error("Failed to execute service card template.", "service", service.ID, "error", err.Error())
		return ""
	}
	// remove newlines so we don't mess up server side events
	return strings.ReplaceAll(tmplOutput.String(), "\n", "")
}

// func for templates to range
func getAllHistory(buffer ringbuffer.RingBuffer[EventData]) []EventData {
	return buffer.GetAll()
}
