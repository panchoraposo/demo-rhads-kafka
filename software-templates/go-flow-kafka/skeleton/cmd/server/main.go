package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"${{values.module_path}}/internal/api"
	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/planner"
	"${{values.module_path}}/internal/store"
)

func main() {
	bootstrap := env("KAFKA_BOOTSTRAP_SERVERS", "rhads-kafka-kafka-bootstrap.kafka.svc:9092")
	port := env("PORT", "${{values.port}}")
	webRoot := env("WEB_ROOT", "web")

	st := store.New()
	pl := planner.New(env("MAAS_BASE_URL", "${{values.maas_base_url}}"), env("MAAS_API_KEY", ""), env("MAAS_MODEL", "${{values.maas_model}}"))

	bus, err := kafka.NewBus(bootstrap, st)
	var engine *kafka.Engine
	if err != nil {
		log.Printf("WARN: Kafka unavailable (%v); in-process bus", err)
		bus = kafka.NewInProcess(st)
	} else {
		engine, err = kafka.StartEngine(bootstrap, st, pl)
		if err != nil {
			log.Printf("WARN: flow engine not started: %v", err)
		} else {
			defer engine.Close()
		}
	}
	defer bus.Close()

	h := api.New(st, bus, pl)
	mux := http.NewServeMux()
	h.Register(mux)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/q/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})
	mux.HandleFunc("/q/health/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))

	log.Printf("${{values.component_id}} listening on :%s (kafka=%s)", port, bootstrap)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
