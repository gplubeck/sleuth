package main

import (
	"fmt"
	"log"
	"log/slog"
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
		if NewProtocol(service.ProtocolString) == nil {
			return fmt.Errorf("service %q: unknown protocol %q", service.Name, service.ProtocolString)
		}
	}
	return nil
}

func parseConfigs(configFile string) Config {
	var config Config

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Fatalf("Error loading TOML config. Error: %s", err)
	}

	if err := validateConfig(&config); err != nil {
		log.Fatalf("Invalid config: %s", err)
	}

	for i, service := range config.Services {
		config.Services[i].protocol = NewProtocol(service.ProtocolString)
		config.Services[i].Start = time.Now()
		slog.Debug("Parsed service.", "service", service)
		maxSize := config.Services[i].MaxHistorySize
		if maxSize == 0 {
			maxSize = 100
		}
		config.Services[i].History = ringbuffer.NewRingBuffer[EventData](maxSize)
	}

	return config
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
