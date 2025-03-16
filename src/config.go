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
    Server Server  `toml:"server"`
    Services []Service `toml:"service"`
}


func parseConfigs(configFile string) Config {
    var config Config

    _, err := toml.DecodeFile(configFile, &config)
    if err != nil {
        log.Fatalf("Error loading TOML config. Error: %s", err)
    }

    for i, service := range config.Services {
        config.Services[i].Protocol = NewProtocol(service.ProtocolString)
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

func getLogLevel( level string) slog.Level {
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
