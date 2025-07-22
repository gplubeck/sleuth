package main

import "log/slog"

type Publisher struct {
    clients map[chan []byte]bool
    addClient chan chan []byte
    removeClient chan chan []byte
    publish chan []byte
}

func NewPublisher() *Publisher {
    return &Publisher{
        clients:      make(map[chan []byte]bool),
        addClient:    make(chan chan []byte),
        removeClient: make(chan chan []byte),
        publish:    make(chan []byte),
    }
}

func (p *Publisher) Start() {
    for {
        select {
        case client := <-p.addClient:
            slog.Debug("Attempting to add client.", "client", client)
            p.clients[client] = true
        case client := <-p.removeClient:
            slog.Debug("Attempting to remove client.", "client", client)
            if _, ok := p.clients[client]; ok {
                delete(p.clients, client)
                close(client)
            }
        case msg := <-p.publish:
            slog.Debug("Message received in publisher.")
            for client := range p.clients {
            slog.Debug("Publishing to client.", "client", client)
                // Non-blocking send to avoid slow clients blocking the broker
                select {
                case client <- msg:
                default:
                }
            }
        }
    }
}
