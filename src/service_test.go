package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---- helpers ----

func httpServiceWith(fields Service) Service {
	fields.ProtocolString = "HTTP"
	fields.protocol = NewProtocol(fields)
	return fields
}

// ---- String ----

func TestHTTPProtocol_String(t *testing.T) {
	p := NewProtocol(Service{ProtocolString: "HTTP"})
	if p.String() != "HTTP" {
		t.Errorf("expected String()=%q, got %q", "HTTP", p.String())
	}
}

// ---- 2xx default ----

func TestHTTPProtocol_AcceptsDefaultOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := httpServiceWith(Service{})
	conn, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err != nil {
		t.Errorf("expected success for 200, got: %v", err)
	}
	if conn != nil {
		t.Error("expected nil conn from HTTP protocol")
	}
}

func TestHTTPProtocol_RejectsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := httpServiceWith(Service{})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err == nil {
		t.Error("expected error for 500, got nil")
	}
}

// ---- exact status code ----

func TestHTTPProtocol_ExactStatusMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPExpectedStatus: http.StatusNoContent})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err != nil {
		t.Errorf("expected success for exact status 204, got: %v", err)
	}
}

func TestHTTPProtocol_ExactStatusMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPExpectedStatus: http.StatusNoContent})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err == nil {
		t.Error("expected error when status 200 != expected 204, got nil")
	}
}

// ---- category check ----

func TestHTTPProtocol_CategoryCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // 503 = 5xx
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPExpectedCategory: 5})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err != nil {
		t.Errorf("expected success for 503 with category=5, got: %v", err)
	}
}

func TestHTTPProtocol_CategoryCheckFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // 200 = 2xx, not 5xx
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPExpectedCategory: 5})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err == nil {
		t.Error("expected error for 200 when category=5, got nil")
	}
}

// ---- connection failures ----

func TestHTTPProtocol_ConnectionRefused(t *testing.T) {
	// Use a server we immediately close so the port is definitely not listening
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	s := httpServiceWith(Service{})
	_, err := s.protocol.Connect(url, 2*time.Second)
	if err == nil {
		t.Error("expected error for refused connection, got nil")
	}
}

func TestHTTPProtocol_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // longer than the timeout we'll set
	}))
	defer srv.Close()

	s := httpServiceWith(Service{})
	_, err := s.protocol.Connect(srv.URL, 100*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

// ---- TLS ----

func TestHTTPProtocol_SkipTLSVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPSkipTLSVerify: true})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err != nil {
		t.Errorf("expected success with skip_tls_verify=true, got: %v", err)
	}
}

func TestHTTPProtocol_TLSVerifyFails(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// skip_tls_verify=false (default) — self-signed cert should fail
	s := httpServiceWith(Service{HTTPSkipTLSVerify: false})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err == nil {
		t.Error("expected TLS verification error for self-signed cert, got nil")
	}
}

// ---- body content check ----

func TestHTTPProtocol_BodyContainsMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","db":"connected"}`))
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPExpectedBodyContains: `"status":"ok"`})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err != nil {
		t.Errorf("expected success when body contains expected substring, got: %v", err)
	}
}

func TestHTTPProtocol_BodyContainsMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"degraded"}`))
	}))
	defer srv.Close()

	s := httpServiceWith(Service{HTTPExpectedBodyContains: `"status":"ok"`})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err == nil {
		t.Error("expected error when body does not contain expected substring, got nil")
	}
}

func TestHTTPProtocol_BodyContainsSkippedWhenUnset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`anything at all`))
	}))
	defer srv.Close()

	s := httpServiceWith(Service{})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err != nil {
		t.Errorf("expected success when body check is unset, got: %v", err)
	}
}

func TestHTTPProtocol_BodyContainsCombinedWithStatus(t *testing.T) {
	// status matches but body doesn't — should still fail
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"degraded"}`))
	}))
	defer srv.Close()

	s := httpServiceWith(Service{
		HTTPExpectedStatus:       http.StatusOK,
		HTTPExpectedBodyContains: `"status":"ok"`,
	})
	_, err := s.protocol.Connect(srv.URL, 2*time.Second)
	if err == nil {
		t.Error("expected error when status matches but body does not, got nil")
	}
}

// ---- NewProtocol ----

func TestNewProtocol_HTTP(t *testing.T) {
	p := NewProtocol(Service{ProtocolString: "HTTP"})
	if p == nil {
		t.Fatal("expected non-nil protocol for HTTP")
	}
	if p.String() != "HTTP" {
		t.Errorf("expected String()=%q, got %q", "HTTP", p.String())
	}
}

func TestNewProtocol_HTTPFieldsPropagated(t *testing.T) {
	p := NewProtocol(Service{
		ProtocolString:           "HTTP",
		HTTPExpectedStatus:       201,
		HTTPExpectedCategory:     0,
		HTTPSkipTLSVerify:        true,
		HTTPExpectedBodyContains: `"status":"ok"`,
	})
	hp, ok := p.(*HTTPProtocol)
	if !ok {
		t.Fatal("expected *HTTPProtocol")
	}
	if hp.expectedStatus != 201 {
		t.Errorf("expectedStatus: got %d, want 201", hp.expectedStatus)
	}
	if hp.expectedBodyContains != `"status":"ok"` {
		t.Errorf("expectedBodyContains: got %q, want %q", hp.expectedBodyContains, `"status":"ok"`)
	}
	if hp.client == nil {
		t.Error("client should be initialized")
	}
}
