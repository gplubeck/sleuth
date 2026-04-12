package main

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to build a minimal valid service
func validService(id uint, name string) Service {
	return Service{
		ID:             id,
		Name:           name,
		Address:        "localhost:8080",
		ProtocolString: "TCP",
		Timer:          30,
	}
}

// ---- validateConfig tests ----

func TestValidateConfig_Valid(t *testing.T) {
	config := &Config{
		Services: []Service{
			validService(1, "Alpha"),
			validService(2, "Beta"),
		},
	}
	if err := validateConfig(config); err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

func TestValidateConfig_ZeroID(t *testing.T) {
	config := &Config{
		Services: []Service{
			validService(0, "Bad Service"),
		},
	}
	if err := validateConfig(config); err == nil {
		t.Error("expected error for id=0, got nil")
	}
}

func TestValidateConfig_DuplicateID(t *testing.T) {
	config := &Config{
		Services: []Service{
			validService(1, "First"),
			validService(1, "Duplicate"),
		},
	}
	if err := validateConfig(config); err == nil {
		t.Error("expected error for duplicate id, got nil")
	}
}

func TestValidateConfig_EmptyName(t *testing.T) {
	config := &Config{
		Services: []Service{
			validService(1, ""),
		},
	}
	if err := validateConfig(config); err == nil {
		t.Error("expected error for empty service_name, got nil")
	}
}

func TestValidateConfig_ZeroTimer(t *testing.T) {
	s := validService(1, "No Timer")
	s.Timer = 0
	if err := validateConfig(&Config{Services: []Service{s}}); err == nil {
		t.Error("expected error for timer=0, got nil")
	}
}

func TestValidateConfig_NegativeTimer(t *testing.T) {
	s := validService(1, "Negative Timer")
	s.Timer = -5
	if err := validateConfig(&Config{Services: []Service{s}}); err == nil {
		t.Error("expected error for negative timer, got nil")
	}
}

func TestValidateConfig_UnknownProtocol(t *testing.T) {
	s := validService(1, "Bad Protocol")
	s.ProtocolString = "ICMP"
	if err := validateConfig(&Config{Services: []Service{s}}); err == nil {
		t.Error("expected error for unknown protocol, got nil")
	}
}

func TestValidateConfig_AllProtocols(t *testing.T) {
	tests := []struct {
		proto   string
		address string
	}{
		{"TCP", "localhost:8080"},
		{"UDP", "localhost:8080"},
		{"Test", "localhost:8080"},
		{"HTTP", "https://localhost:8080/health"},
	}
	for i, tt := range tests {
		s := validService(uint(i+1), tt.proto+" Service")
		s.ProtocolString = tt.proto
		s.Address = tt.address
		if err := validateConfig(&Config{Services: []Service{s}}); err != nil {
			t.Errorf("expected protocol %q to be valid, got: %v", tt.proto, err)
		}
	}
}

// ---- validateHTTPService tests ----

func validHTTPService(id uint, name string) Service {
	return Service{
		ID:             id,
		Name:           name,
		Address:        "https://localhost:8080/health",
		ProtocolString: "HTTP",
		Timer:          30,
	}
}

func TestValidateHTTPService_ValidURL(t *testing.T) {
	s := validHTTPService(1, "API")
	if err := validateConfig(&Config{Services: []Service{s}}); err != nil {
		t.Errorf("expected valid HTTP service, got: %v", err)
	}
}

func TestValidateHTTPService_HTTPScheme(t *testing.T) {
	s := validHTTPService(1, "API")
	s.Address = "http://localhost:8080/health"
	if err := validateConfig(&Config{Services: []Service{s}}); err != nil {
		t.Errorf("expected http:// to be valid, got: %v", err)
	}
}

func TestValidateHTTPService_BareHostPort(t *testing.T) {
	s := validHTTPService(1, "API")
	s.Address = "localhost:8080"
	if err := validateConfig(&Config{Services: []Service{s}}); err == nil {
		t.Error("expected error for bare host:port with HTTP protocol, got nil")
	}
}

func TestValidateHTTPService_BothStatusFields(t *testing.T) {
	s := validHTTPService(1, "API")
	s.HTTPExpectedStatus = 200
	s.HTTPExpectedCategory = 2
	if err := validateConfig(&Config{Services: []Service{s}}); err == nil {
		t.Error("expected error when both http_expected_status and http_expected_category are set")
	}
}

func TestValidateHTTPService_BadCategory(t *testing.T) {
	s := validHTTPService(1, "API")
	s.HTTPExpectedCategory = 7
	if err := validateConfig(&Config{Services: []Service{s}}); err == nil {
		t.Error("expected error for http_expected_category=7, got nil")
	}
}

func TestValidateHTTPService_ValidCategory(t *testing.T) {
	for _, cat := range []int{1, 2, 3, 4, 5} {
		s := validHTTPService(uint(cat), "API")
		s.HTTPExpectedCategory = cat
		if err := validateConfig(&Config{Services: []Service{s}}); err != nil {
			t.Errorf("expected category %d to be valid, got: %v", cat, err)
		}
	}
}

func TestValidateHTTPService_ExactStatus(t *testing.T) {
	s := validHTTPService(1, "API")
	s.HTTPExpectedStatus = 204
	if err := validateConfig(&Config{Services: []Service{s}}); err != nil {
		t.Errorf("expected http_expected_status=204 to be valid, got: %v", err)
	}
}

func TestParseConfigs_HTTPService(t *testing.T) {
	toml := `
[server]
port = 5000
storage_type = "memory"

[[service]]
id = 1
service_name = "API Health"
address = "https://api.example.com/health"
protocol_str = "HTTP"
timer = 30
http_expected_status = 200
http_skip_tls_verify = true
`
	path := writeTempConfig(t, toml)
	config := parseConfigs(path)

	if len(config.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(config.Services))
	}
	s := config.Services[0]
	if s.protocol == nil {
		t.Error("expected protocol to be initialized")
	}
	if s.protocol.String() != "HTTP" {
		t.Errorf("expected protocol=HTTP, got %q", s.protocol.String())
	}
	if s.HTTPExpectedStatus != 200 {
		t.Errorf("expected HTTPExpectedStatus=200, got %d", s.HTTPExpectedStatus)
	}
	if !s.HTTPSkipTLSVerify {
		t.Error("expected HTTPSkipTLSVerify=true")
	}
}

func TestValidateConfig_Empty(t *testing.T) {
	// No services is technically valid — caller decides if that's useful
	if err := validateConfig(&Config{}); err != nil {
		t.Errorf("expected no error for empty service list, got: %v", err)
	}
}

// ---- parseConfigs tests ----

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestParseConfigs_ValidFile(t *testing.T) {
	toml := `
[server]
port = 5000
log_level = "warn"
storage_type = "memory"

[[service]]
id = 1
service_name = "Test Service"
address = "localhost:8080"
protocol_str = "TCP"
timer = 30
`
	path := writeTempConfig(t, toml)
	config := parseConfigs(path)

	if config.Server.Port != 5000 {
		t.Errorf("expected port 5000, got %d", config.Server.Port)
	}
	if len(config.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(config.Services))
	}
	if config.Services[0].protocol == nil {
		t.Error("expected protocol to be initialized, got nil")
	}
	if config.Services[0].History.MaxSize() == 0 {
		t.Error("expected history ring buffer to be initialized")
	}
}

func TestParseConfigs_DefaultMaxHistory(t *testing.T) {
	toml := `
[server]
port = 5000
storage_type = "memory"

[[service]]
id = 1
service_name = "No MaxHistory"
address = "localhost:8080"
protocol_str = "TCP"
timer = 10
`
	path := writeTempConfig(t, toml)
	config := parseConfigs(path)

	if config.Services[0].History.MaxSize() != 100 {
		t.Errorf("expected default MaxHistory=100, got %d", config.Services[0].History.MaxSize())
	}
}

func TestParseConfigs_CustomMaxHistory(t *testing.T) {
	toml := `
[server]
port = 5000
storage_type = "memory"

[[service]]
id = 1
service_name = "Custom History"
address = "localhost:8080"
protocol_str = "TCP"
timer = 10
MaxHistory = 50
`
	path := writeTempConfig(t, toml)
	config := parseConfigs(path)

	if config.Services[0].History.MaxSize() != 50 {
		t.Errorf("expected MaxHistory=50, got %d", config.Services[0].History.MaxSize())
	}
}

func TestParseConfigs_MultipleServices(t *testing.T) {
	toml := `
[server]
port = 8080
storage_type = "memory"

[[service]]
id = 1
service_name = "Alpha"
address = "localhost:80"
protocol_str = "TCP"
timer = 10

[[service]]
id = 2
service_name = "Beta"
address = "localhost:443"
protocol_str = "UDP"
timer = 60
`
	path := writeTempConfig(t, toml)
	config := parseConfigs(path)

	if len(config.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(config.Services))
	}
	for _, s := range config.Services {
		if s.protocol == nil {
			t.Errorf("service %q: protocol not initialized", s.Name)
		}
		if s.Start.IsZero() {
			t.Errorf("service %q: Start time not set", s.Name)
		}
	}
}

// ---- getLogLevel tests ----

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "DEBUG"},
		{"DEBUG", "DEBUG"},
		{"info", "INFO"},
		{"INFO", "INFO"},
		{"error", "ERROR"},
		{"ERROR", "ERROR"},
		{"warn", "WARN"},
		{"unknown", "WARN"},
		{"", "WARN"},
	}

	for _, tt := range tests {
		result := getLogLevel(tt.input)
		if result.String() != tt.expected {
			t.Errorf("getLogLevel(%q) = %q, want %q", tt.input, result.String(), tt.expected)
		}
	}
}
