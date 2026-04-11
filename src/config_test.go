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
	protocols := []string{"TCP", "UDP", "Test"}
	for i, proto := range protocols {
		s := validService(uint(i+1), proto+" Service")
		s.ProtocolString = proto
		if err := validateConfig(&Config{Services: []Service{s}}); err != nil {
			t.Errorf("expected protocol %q to be valid, got: %v", proto, err)
		}
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
