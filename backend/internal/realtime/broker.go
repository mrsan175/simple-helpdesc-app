package realtime

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Broker struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

var DefaultBroker = NewBroker()

func NewBroker() *Broker {
	return &Broker{
		clients: make(map[chan []byte]struct{}),
	}
}

func (b *Broker) Subscribe() (chan []byte, func()) {
	ch := make(chan []byte, 32)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		delete(b.clients, ch)
		close(ch)
		b.mu.Unlock()
	}
	return ch, unsubscribe
}

func (b *Broker) Broadcast(eventType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data))

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
			// Client slow or buffer full; drop to avoid blocking other clients
		}
	}
}
