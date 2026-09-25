package api

import (
	"encoding/json"
	"net/http"

	"${{values.module_path}}/internal/store"
)

func Register(mux *http.ServeMux, st *store.Store, apicurioURL, topic string) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st.List())
	})
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"topic":    topic,
			"apicurio": apicurioURL,
			"artifact": "order-change",
			"group":    "${{values.component_id}}",
		})
	})
}
