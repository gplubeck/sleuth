package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

type EventData struct {
	ServiceID uint      `json:"serviceID"`
	Status    bool      `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type Scheduler struct {
	store    ServiceStore
	channel  chan<- []byte
	eventBus chan []byte
	cancels  map[uint]context.CancelFunc
	mu       sync.Mutex
}

func NewScheduler(store ServiceStore, channel chan<- []byte) *Scheduler {
	return &Scheduler{
		store:    store,
		channel:  channel,
		eventBus: make(chan []byte),
		cancels:  make(map[uint]context.CancelFunc),
	}
}

// Start launches monitor goroutines for all services currently in the store
// and begins the event loop. Call once at startup.
func (s *Scheduler) Start() {
	slog.Debug("Starting Scheduler goroutines.")
	for _, service := range *s.store.GetServices() {
		s.startMonitor(service)
	}
	go s.run()
}

// AddService starts a monitor goroutine for a newly added service.
func (s *Scheduler) AddService(service Service) {
	s.startMonitor(service)
}

// RemoveService cancels the monitor goroutine for a service.
func (s *Scheduler) RemoveService(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.cancels[id]; ok {
		cancel()
		delete(s.cancels, id)
		slog.Info("Stopped monitoring service.", "id", id)
	}
}

func (s *Scheduler) startMonitor(service Service) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[service.ID] = cancel
	s.mu.Unlock()
	slog.Info("Started monitoring service.", "id", service.ID, "name", service.Name)
	go monitorService(ctx, service, s.eventBus)
}

func (s *Scheduler) run() {
	saveTicker := time.NewTicker(30 * time.Second)
	defer saveTicker.Stop()

	for {
		select {
		case eventData := <-s.eventBus:
			var event EventData
			if err := json.Unmarshal(eventData, &event); err != nil {
				slog.Error("Unable to unmarshal event update.", "error", err)
				continue
			}
			if err := s.store.EventUpdate(event); err != nil {
				slog.Error("Unable to handle event update.", "error", err)
			}
			s.channel <- eventData

		case <-saveTicker.C:
			if err := s.store.Save(); err != nil {
				slog.Error("Failed to persist store.", "error", err)
			}
		}
	}
}

func monitorService(ctx context.Context, service Service, eventBus chan<- []byte) {
	for {
		var event EventData
		response := service.getStatus()
		event.Status = response.Status
		event.Timestamp = response.timestamp
		event.ServiceID = service.ID

		update, err := json.Marshal(event)
		if err != nil {
			slog.Error("Error marshalling JSON.", "error", err)
		} else {
			slog.Debug("Sending update to scheduler.", "service", service.ID)
			select {
			case eventBus <- update:
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(service.Timer) * time.Second):
		}
	}
}
