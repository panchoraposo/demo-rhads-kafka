package store

import (
	"sync"
	"time"

	"${{values.module_path}}/internal/model"
)

// Use uuid - add to go.mod. Fallback simple ids if needed.

type Store struct {
	mu      sync.RWMutex
	byReq   map[string]*model.TripPlanStatus
	byInst  map[string]*model.TripPlanStatus
	latest  *model.TripPlanStatus
}

func New() *Store {
	return &Store{
		byReq:  map[string]*model.TripPlanStatus{},
		byInst: map[string]*model.TripPlanStatus{},
	}
}

func (s *Store) Register(req model.TripRequest) *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := &model.TripPlanStatus{
		RequestID:  newID(),
		InstanceID: "",
		Request:    req,
		Status:     "planning",
	}
	s.byReq[st.RequestID] = st
	s.latest = st
	return st
}

func (s *Store) Bind(requestID, instanceID string) *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.byReq[requestID]
	if st == nil {
		return nil
	}
	st.InstanceID = instanceID
	s.byInst[instanceID] = st
	return st
}

func (s *Store) ByInstance(id string) *model.TripPlanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.byInst[id]; ok {
		cp := *st
		return &cp
	}
	return nil
}

func (s *Store) ByRequest(id string) *model.TripPlanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.byReq[id]; ok {
		cp := *st
		return &cp
	}
	return nil
}

func (s *Store) Latest() *model.TripPlanStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latest == nil {
		return nil
	}
	cp := *s.latest
	return &cp
}

func (s *Store) Update(st *model.TripPlanStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byReq[st.RequestID] = st
	if st.InstanceID != "" {
		s.byInst[st.InstanceID] = st
	}
	s.latest = st
}

func (s *Store) MatchesDecision(instanceID string, a model.TripApproval) bool {
	st := s.ByInstance(instanceID)
	if st == nil || st.Status != "awaiting_approval" {
		return false
	}
	return a.InstanceID == instanceID && (a.Status == "approved" || a.Status == "rejected")
}

func newID() string {
	return time.Now().UTC().Format("20060102T150405.000000000")
}
