package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"sleuth/internal/ringbuffer"
	"strings"
	"time"

	// external
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   Server    `toml:"server"`
	Services []Service `toml:"service"`
}

// validateConfig checks all services for required fields and returns the first
// error found. Keeping validation separate from file I/O makes it testable.
func validateConfig(config *Config) error {
	seenIDs := make(map[uint]bool)
	for i, service := range config.Services {
		if service.ID == 0 {
			return fmt.Errorf("service %q (index %d): id must be set and non-zero", service.Name, i)
		}
		if seenIDs[service.ID] {
			return fmt.Errorf("duplicate service id: %d", service.ID)
		}
		seenIDs[service.ID] = true
		if service.Name == "" {
			return fmt.Errorf("service at index %d: service_name must not be empty", i)
		}
		if service.Timer <= 0 {
			return fmt.Errorf("service %q: timer must be greater than 0", service.Name)
		}
		if service.DegradedMs < 0 {
			return fmt.Errorf("service %q: degraded_ms must not be negative", service.Name)
		}
		if NewProtocol(service) == nil {
			return fmt.Errorf("service %q: unknown protocol %q", service.Name, service.ProtocolString)
		}
		if err := validateHTTPService(service, i); err != nil {
			return err
		}
	}
	return nil
}

func validateHTTPService(service Service, i int) error {
	if service.ProtocolString != "HTTP" {
		return nil
	}
	u, err := url.Parse(service.Address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("service %q (index %d): HTTP protocol requires a full URL (e.g. https://host/path), got %q",
			service.Name, i, service.Address)
	}
	if service.HTTPExpectedStatus != 0 && service.HTTPExpectedCategory != 0 {
		return fmt.Errorf("service %q: set either http_expected_status or http_expected_category, not both",
			service.Name)
	}
	if service.HTTPExpectedCategory != 0 && (service.HTTPExpectedCategory < 1 || service.HTTPExpectedCategory > 5) {
		return fmt.Errorf("service %q: http_expected_category must be 1–5, got %d",
			service.Name, service.HTTPExpectedCategory)
	}
	return nil
}

func parseConfigs(configFile string) (Config, error) {
	var config Config

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		return Config{}, fmt.Errorf("error loading TOML config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	for i, service := range config.Services {
		config.Services[i].protocol = NewProtocol(service)
		config.Services[i].Start = time.Now()
		slog.Debug("Parsed service.", "service", service)
		maxSize := config.Services[i].MaxHistorySize
		if maxSize == 0 {
			maxSize = 100
		}
		config.Services[i].History = ringbuffer.NewRingBuffer[EventData](maxSize)
	}

	return config, nil
}

func getLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}
