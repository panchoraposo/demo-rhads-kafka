package store

import (
	"crypto/rand"
	"fmt"
	"sync"

	"${{values.module_path}}/internal/model"
)

type Store struct {
	mu        sync.Mutex
	byReq     map[string]*model.TripPlanStatus
	byInst    map[string]string
	decisions map[string]model.TripApproval
	latestID  string
}

func New() *Store {
	return &Store{
		byReq:     map[string]*model.TripPlanStatus{},
		byInst:    map[string]string{},
		decisions: map[string]model.TripApproval{},
	}
}

func (s *Store) Register(req model.TripRequest) *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	st := &model.TripPlanStatus{
		RequestID: id,
		Request:   req,
		Status:    "planning",
	}
	s.byReq[id] = st
	s.latestID = id
	cp := *st
	return &cp
}

func (s *Store) Bind(requestID, instanceID string) *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.byReq[requestID]
	if st == nil || st.Status != "planning" || st.InstanceID != "" {
		return nil
	}
	st.InstanceID = instanceID
	s.byInst[instanceID] = requestID
	cp := *st
	return &cp
}

func (s *Store) ByInstance(id string) *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	reqID := s.byInst[id]
	if reqID == "" {
		return nil
	}
	st := s.byReq[reqID]
	if st == nil {
		return nil
	}
	cp := *st
	return &cp
}

func (s *Store) ByRequest(id string) *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.byReq[id]
	if st == nil {
		return nil
	}
	cp := *st
	return &cp
}

func (s *Store) Latest() *model.TripPlanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.byReq[s.latestID]
	if st == nil {
		return nil
	}
	cp := *st
	return &cp
}

func (s *Store) AcceptOutcome(st *model.TripPlanStatus) {
	if st == nil || st.RequestID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.byReq[st.RequestID]
	if prev == nil {
		return
	}
	planning := prev.Status == "planning"
	if planning && st.Status != "awaiting_approval" && st.Status != "failed" {
		return
	}
	if !planning && (prev.Status != "decision_submitted" || st.Status == "awaiting_approval") {
		return
	}
	if st.Status == "awaiting_approval" && st.Plan == nil {
		return
	}
	if st.Status == "confirmed" && st.Confirmation == nil {
		return
	}
	next := *st
	if !planning {
		next.Plan = prev.Plan
		delete(s.decisions, prev.InstanceID)
	}
	s.byReq[st.RequestID] = &next
	if next.InstanceID != "" {
		s.byInst[next.InstanceID] = next.RequestID
	}
	s.latestID = next.RequestID
}

func (s *Store) SubmitDecision(a model.TripApproval) (*model.TripPlanStatus, *model.TripError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	reqID := s.byInst[a.InstanceID]
	prev := s.byReq[reqID]
	if prev == nil {
		return nil, &model.TripError{Error: "unknown_trip", Message: "The requested trip was not found."}
	}
	if prev.Status != "awaiting_approval" {
		return nil, &model.TripError{Error: "decision_not_pending", Message: "This trip is not awaiting a decision."}
	}
	next := *prev
	next.Status = "decision_submitted"
	s.byReq[reqID] = &next
	s.decisions[a.InstanceID] = a
	cp := next
	return &cp, nil
}

func (s *Store) MatchesDecision(instanceID string, a model.TripApproval) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	reqID := s.byInst[instanceID]
	st := s.byReq[reqID]
	if st == nil || st.Status != "decision_submitted" {
		return false
	}
	prev, ok := s.decisions[instanceID]
	return ok && prev == a
}

func (s *Store) MarkFailed(requestID, code, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.byReq[requestID]
	if st == nil {
		return
	}
	if st.Status != "planning" && st.Status != "awaiting_approval" && st.Status != "decision_submitted" {
		return
	}
	st.Status = "failed"
	st.Error = code
	st.Message = message
	if st.InstanceID != "" {
		delete(s.decisions, st.InstanceID)
	}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
