package store

import (
	"sync"
	"time"
)

type Event struct {
	ID        string         `json:"id"`
	Op        string         `json:"op"`
	Customer  string         `json:"customer"`
	Amount    float64        `json:"amount"`
	Status    string         `json:"status"`
	Before    map[string]any `json:"before,omitempty"`
	After     map[string]any `json:"after,omitempty"`
	Raw       string         `json:"raw"`
	Received  time.Time      `json:"received"`
	LatencyMs *int64         `json:"latencyMs,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	events   []Event
	subs     map[chan Event]struct{}
	writes   map[string]time.Time
}

func New() *Store {
	return &Store{
		events: make([]Event, 0, 64),
		subs:   map[chan Event]struct{}{},
		writes: map[string]time.Time{},
	}
}

// NoteWrite records when the Order Desk mutated a row (for capture latency).
func (s *Store) NoteWrite(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes[id] = time.Now().UTC()
}

func (s *Store) Add(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e.Received = time.Now().UTC()
	if t, ok := s.writes[e.ID]; ok {
		ms := e.Received.Sub(t).Milliseconds()
		if ms < 0 {
			ms = 0
		}
		e.LatencyMs = &ms
		delete(s.writes, e.ID)
	}
	s.events = append([]Event{e}, s.events...)
	if len(s.events) > 100 {
		s.events = s.events[:100]
	}
	for ch := range s.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (s *Store) List() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *Store) Subscribe() chan Event {
	ch := make(chan Event, 16)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

func (s *Store) Unsubscribe(ch chan Event) {
	s.mu.Lock()
	delete(s.subs, ch)
	s.mu.Unlock()
	close(ch)
}
