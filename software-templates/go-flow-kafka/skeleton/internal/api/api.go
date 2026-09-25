package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/model"
	"${{values.module_path}}/internal/store"
)

type Handler struct {
	store *store.Store
	bus   kafka.Bus
}

func New(st *store.Store, bus kafka.Bus) *Handler {
	return &Handler{store: st, bus: bus}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Destination == "" {
		writeJSON(w, http.StatusBadRequest, model.TripError{Error: "invalid_request", Message: "Trip details are required."})
		return
	}
	st := h.store.Register(req)
	log.Printf("[api] POST /trip/plan requestId=%s destination=%q", st.RequestID, req.Destination)
	if err := h.bus.PublishRequest(st.RequestID, req); err != nil {
		h.store.MarkFailed(st.RequestID, "planning_failed", "Could not publish the planning request to Kafka.")
		writeJSON(w, http.StatusInternalServerError, h.store.ByRequest(st.RequestID))
		return
	}

	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		cur := h.store.ByRequest(st.RequestID)
		if cur != nil && cur.Status != "planning" {
			code := http.StatusOK
			if cur.Status == "failed" {
				code = http.StatusInternalServerError
				if cur.Error == "quality_not_met" || cur.Error == "guardrail_violation" {
					code = http.StatusUnprocessableEntity
				}
			}
			log.Printf("[api] plan complete requestId=%s status=%s", cur.RequestID, cur.Status)
			writeJSON(w, code, cur)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	pending := h.store.ByRequest(st.RequestID)
	log.Printf("[api] plan timeout requestId=%s", st.RequestID)
	writeJSON(w, http.StatusGatewayTimeout, pending)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var a model.TripApproval
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil || a.InstanceID == "" {
		writeJSON(w, http.StatusBadRequest, model.TripError{Error: "invalid_decision", Message: "instanceId is required."})
		return
	}
	if a.Status != "approved" && a.Status != "rejected" {
		writeJSON(w, http.StatusBadRequest, model.TripError{Error: "invalid_decision", Message: "status must be approved or rejected."})
		return
	}
	submitted, terr := h.store.SubmitDecision(a)
	if terr != nil {
		code := http.StatusConflict
		if terr.Error == "unknown_trip" {
			code = http.StatusNotFound
		}
		writeJSON(w, code, terr)
		return
	}
	log.Printf("[api] PUT /trip/approve instanceId=%s status=%s", a.InstanceID, a.Status)
	if err := h.bus.PublishDecision(a); err != nil {
		h.store.MarkFailed(submitted.RequestID, "finalization_failed", "Could not publish the decision to Kafka.")
		writeJSON(w, http.StatusInternalServerError, h.store.ByInstance(a.InstanceID))
		return
	}
	writeJSON(w, http.StatusAccepted, submitted)
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
		writeJSON(w, http.StatusBadRequest, model.TripError{Error: "invalid_request", Message: "instanceId or requestId is required."})
		return
	}
	if st == nil {
		writeJSON(w, http.StatusNotFound, model.TripError{Error: "unknown_trip", Message: "The requested trip was not found."})
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
