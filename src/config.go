package main

import (
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

func parseConfigs(configFile string) Config {
	var config Config

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Fatalf("Error loading TOML config. Error: %s", err)
	}

	seenIDs := make(map[uint]bool)
	for i, service := range config.Services {
		if service.ID == 0 {
			log.Fatalf("Service %q (index %d): id must be set and non-zero", service.Name, i)
		}
		if seenIDs[service.ID] {
			log.Fatalf("Duplicate service id: %d", service.ID)
		}
		seenIDs[service.ID] = true

		if service.Name == "" {
			log.Fatalf("Service at index %d: service_name must not be empty", i)
		}
		if service.Timer <= 0 {
			log.Fatalf("Service %q: timer must be greater than 0", service.Name)
		}

		config.Services[i].protocol = NewProtocol(service.ProtocolString)
		if config.Services[i].protocol == nil {
			log.Fatalf("Service %q: unknown protocol %q", service.Name, service.ProtocolString)
		}

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
