package api

import (
	"encoding/json"
	"net/http"
	"time"

	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/model"
	"${{values.module_path}}/internal/planner"
	"${{values.module_path}}/internal/store"
)

type Handler struct {
	store   *store.Store
	bus     kafka.Bus
	planner *planner.Planner
}

func New(st *store.Store, bus kafka.Bus, pl *planner.Planner) *Handler {
	return &Handler{store: st, bus: bus, planner: pl}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/trip/plan", h.plan)
	mux.HandleFunc("/trip/approve", h.approve)
	mux.HandleFunc("/trip/plan/status", h.status)
	mux.HandleFunc("/trip/plan/latest", h.latest)
}

func (h *Handler) plan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req model.TripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	st := h.store.Register(req)
	_ = h.bus.PublishRequest(st.RequestID, req)

	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		cur := h.store.ByRequest(st.RequestID)
		if cur != nil && (cur.Status == "awaiting_approval" || cur.Status == "failed") {
			writeJSON(w, http.StatusOK, cur)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	writeJSON(w, http.StatusGatewayTimeout, map[string]any{"status": "planning_timeout", "requestId": st.RequestID})
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var a model.TripApproval
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil || a.InstanceID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if a.Status != "approved" && a.Status != "rejected" {
		http.Error(w, "status must be approved or rejected", http.StatusBadRequest)
		return
	}
	cur := h.store.ByInstance(a.InstanceID)
	if cur == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if cur.Status != "awaiting_approval" {
		http.Error(w, "conflict", http.StatusConflict)
		return
	}
	if err := h.bus.PublishDecision(a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "decision_submitted", "instanceId": a.InstanceID})
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	inst := r.URL.Query().Get("instanceId")
	reqID := r.URL.Query().Get("requestId")
	var st *model.TripPlanStatus
	if inst != "" {
		st = h.store.ByInstance(inst)
	} else if reqID != "" {
		st = h.store.ByRequest(reqID)
	} else {
		http.Error(w, "instanceId or requestId required", http.StatusBadRequest)
		return
	}
	if st == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (h *Handler) latest(w http.ResponseWriter, r *http.Request) {
	st := h.store.Latest()
	if st == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
