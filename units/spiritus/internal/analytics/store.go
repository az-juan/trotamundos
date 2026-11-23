// Package analytics mantiene el read-model de eventos del prototipo (ADR-107).
package analytics

import (
	"sync"

	"trotamundos/shared/event"
)

type Memory struct {
	mu     sync.RWMutex
	events []event.Event
}

func New() *Memory { return &Memory{events: []event.Event{}} }

func (s *Memory) Add(e event.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *Memory) List(limit int) []event.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	out := make([]event.Event, limit)
	for i := 0; i < limit; i++ {
		out[i] = s.events[len(s.events)-1-i]
	}
	return out
}

type Summary struct {
	Total          int            `json:"total"`
	ByType         map[string]int `json:"by_type"`
	Confirmations  int            `json:"confirmations"`
	Disruptions    int            `json:"disruptions"`
	LastEventAt    string         `json:"last_event_at,omitempty"`
	ResolutionRate float64        `json:"resolution_rate,omitempty"`
}

func (s *Memory) Summary() Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := Summary{Total: len(s.events), ByType: map[string]int{}}
	var handoffs, resolved int
	for _, e := range s.events {
		out.ByType[e.Type]++
		switch e.Type {
		case event.BookingConfirmed:
			out.Confirmations++
		case event.TripDisrupted:
			out.Disruptions++
		case event.BookingDiscovered:
			if e.Payload["tier1_resolved"] == true {
				resolved++
			}
		}
		if e.Type == "assistance.resolved" {
			handoffs++
			if e.Payload["tier1_resolved"] == true {
				resolved++
			}
		}
		out.LastEventAt = e.OccurredAt
	}
	if handoffs > 0 {
		out.ResolutionRate = float64(resolved) / float64(handoffs)
	}
	return out
}
