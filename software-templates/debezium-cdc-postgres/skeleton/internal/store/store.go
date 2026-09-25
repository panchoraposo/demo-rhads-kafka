package store

import (
	"sync"
	"time"
)

type Event struct {
	ID        string    `json:"id"`
	Op        string    `json:"op"`
	Customer  string    `json:"customer"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	Raw       string    `json:"raw"`
	Received  time.Time `json:"received"`
}

type Store struct {
	mu     sync.RWMutex
	events []Event
}

func New() *Store {
	return &Store{events: make([]Event, 0, 64)}
}

func (s *Store) Add(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e.Received = time.Now().UTC()
	s.events = append([]Event{e}, s.events...)
	if len(s.events) > 100 {
		s.events = s.events[:100]
	}
}

func (s *Store) List() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}
