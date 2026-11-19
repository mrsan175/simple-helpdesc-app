package realtime

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Audience membatasi penerima event. Audience kosong = semua client.
type Audience struct {
	Roles   []string
	UserIDs []string
}

type client struct {
	ch     chan []byte
	role   string
	userID string
}

type Broker struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

var DefaultBroker = NewBroker()

func NewBroker() *Broker {
	return &Broker{
		clients: make(map[*client]struct{}),
	}
}

func (b *Broker) Subscribe(role, userID string) (chan []byte, func()) {
	c := &client{ch: make(chan []byte, 32), role: role, userID: userID}
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		delete(b.clients, c)
		close(c.ch)
		b.mu.Unlock()
	}
	return c.ch, unsubscribe
}

func (b *Broker) BroadcastTo(audience Audience, eventType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data))

	b.mu.RLock()
	defer b.mu.RUnlock()
	for c := range b.clients {
		if !audience.matches(c) {
			continue
		}
		select {
		case c.ch <- msg:
		default:
			// Client slow or buffer full; drop to avoid blocking other clients
		}
	}
}

func (a Audience) matches(c *client) bool {
	if len(a.Roles) == 0 && len(a.UserIDs) == 0 {
		return true
	}
	for _, r := range a.Roles {
		if r == c.role {
			return true
		}
	}
	for _, u := range a.UserIDs {
		if u == c.userID {
			return true
		}
	}
	return false
}
