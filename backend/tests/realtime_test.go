package tests

import (
	"testing"
	"time"

	"backend/internal/realtime"
)

func TestBrokerAudienceScoping(t *testing.T) {
	b := realtime.NewBroker()

	agentCh, unsubAgent := b.Subscribe("agent", "agent-1")
	defer unsubAgent()
	reqCh, unsubReq := b.Subscribe("requester", "req-1")
	defer unsubReq()

	// ticket_created hanya untuk agent & admin
	b.BroadcastTo(realtime.Audience{Roles: []string{"agent", "admin"}}, "ticket_created", map[string]string{"id": "t1"})

	select {
	case <-agentCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("agent should receive ticket_created")
	}

	select {
	case msg := <-reqCh:
		t.Fatalf("requester must not receive ticket_created, got %s", msg)
	case <-time.After(100 * time.Millisecond):
	}

	// Requester tetap menerima event untuk tiket miliknya
	b.BroadcastTo(
		realtime.Audience{Roles: []string{"agent", "admin"}, UserIDs: []string{"req-1"}},
		"comment_added",
		map[string]string{"ticket_id": "t1"},
	)
	select {
	case <-reqCh:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("requester should receive event for own ticket")
	}
}
