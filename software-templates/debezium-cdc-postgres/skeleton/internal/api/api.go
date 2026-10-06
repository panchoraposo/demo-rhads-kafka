package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"${{values.module_path}}/internal/db"
	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/store"
)

type Handler struct {
	store     *store.Store
	mu        sync.RWMutex
	db        *db.DB
	fanout    *kafka.Publisher
	apicurio  string
	topic     string
	component string
}

func Register(mux *http.ServeMux, st *store.Store, database *db.DB, fanout *kafka.Publisher, apicurioURL, topic, component string) *Handler {
	h := &Handler{store: st, db: database, fanout: fanout, apicurio: apicurioURL, topic: topic, component: component}
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/events", h.listEvents)
	mux.HandleFunc("/api/events/stream", h.streamEvents)
	mux.HandleFunc("/api/info", h.info)
	mux.HandleFunc("/api/orders", h.orders)
	mux.HandleFunc("/api/orders/", h.orderByID)
	return h
}

func (h *Handler) SetDB(database *db.DB) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.db = database
}

func (h *Handler) DB() *db.DB {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.db
}

func (h *Handler) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"topic":    h.topic,
		"apicurio": h.apicurio,
		"artifact": "order-change",
		"group":    h.component,
		"db":       h.DB() != nil,
		"fanout":   h.fanout != nil,
	})
}

func (h *Handler) listEvents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

func (h *Handler) streamEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := h.store.Subscribe()
	defer h.store.Unsubscribe(ch)
	flusher.Flush()
	ctx := r.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case e, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, e)
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, e store.Event) {
	b, _ := json.Marshal(e)
	_, _ = fmt.Fprintf(w, "event: cdc\ndata: %s\n\n", b)
}

func (h *Handler) orders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	database := h.DB()
	if database == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "database_unavailable",
			"message": "Order Desk requires DATABASE_* env (dev namespace with Postgres).",
		})
		return
	}
	var body struct {
		Customer string  `json:"customer"`
		Amount   float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Customer) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "customer is required"})
		return
	}
	if body.Amount <= 0 {
		body.Amount = 19.99
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	o, err := database.Create(ctx, strings.TrimSpace(body.Customer), body.Amount)
	if err != nil {
		log.Printf("create order: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db_error", "message": err.Error()})
		return
	}
	id := fmt.Sprintf("%d", o.ID)
	h.store.NoteWrite(id)
	if h.fanout != nil {
		_ = h.fanout.Emit("c", nil, o)
	}
	log.Printf("[orders] CREATE id=%d after=%s", o.ID, o.LogLine())
	writeJSON(w, http.StatusCreated, o)
}

func (h *Handler) orderByID(w http.ResponseWriter, r *http.Request) {
	database := h.DB()
	if database == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "database_unavailable",
			"message": "Order Desk requires DATABASE_* env (dev namespace with Postgres).",
		})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/orders/")
	path = strings.Trim(path, "/")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "order id required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodPatch:
		var body struct {
			Status *string  `json:"status"`
			Amount *float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
			return
		}
		var statusPtr *string
		if body.Status != nil {
			status := strings.ToLower(strings.TrimSpace(*body.Status))
			if status != "paid" && status != "cancelled" && status != "shipped" && status != "new" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "status must be paid, cancelled, shipped, or new"})
				return
			}
			statusPtr = &status
		}
		var amountPtr *float64
		if body.Amount != nil {
			if *body.Amount <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "amount must be > 0"})
				return
			}
			amountPtr = body.Amount
		}
		if statusPtr == nil && amountPtr == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "message": "status and/or amount required"})
			return
		}
		before, after, err := database.Update(ctx, id, statusPtr, amountPtr)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": err.Error()})
			return
		}
		h.store.NoteWrite(fmt.Sprintf("%d", id))
		if h.fanout != nil {
			_ = h.fanout.Emit("u", before, after)
		}
		log.Printf("[orders] UPDATE id=%d before=%s after=%s", id, before.LogLine(), after.LogLine())
		writeJSON(w, http.StatusOK, after)
	case http.MethodDelete:
		before, err := database.Delete(ctx, id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": err.Error()})
			return
		}
		h.store.NoteWrite(fmt.Sprintf("%d", id))
		if h.fanout != nil {
			_ = h.fanout.Emit("d", before, nil)
		}
		log.Printf("[orders] DELETE id=%d before=%s after=∅", id, before.LogLine())
		writeJSON(w, http.StatusOK, before)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
