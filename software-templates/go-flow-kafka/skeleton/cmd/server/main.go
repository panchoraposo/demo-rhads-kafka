package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"${{values.module_path}}/internal/api"
	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/planner"
	"${{values.module_path}}/internal/store"
)

func main() {
	bootstrap := env("KAFKA_BOOTSTRAP_SERVERS", "rhads-kafka-kafka-bootstrap.kafka.svc:9092")
	port := env("PORT", "8082")
	webRoot := env("WEB_ROOT", "web")
	skillsDir := env("SKILLS_DIR", "skills")
	if !filepath.IsAbs(skillsDir) {
		if abs, err := filepath.Abs(skillsDir); err == nil {
			skillsDir = abs
		}
	}

	st := store.New()
	pl := planner.New(
		env("MAAS_BASE_URL", "https://maas-rhdp.apps.maas.redhatworkshops.io/v1/"),
		env("MAAS_API_KEY", ""),
		env("MAAS_MODEL", "qwen3-14b"),
		skillsDir,
	)
	if env("MAAS_API_KEY", "") == "" {
		log.Printf("WARN: MAAS_API_KEY is empty — planning will return maas_api_key_missing")
	}

	bus, err := kafka.NewBus(bootstrap, st)
	var engine *kafka.Engine
	if err != nil {
		log.Printf("WARN: Kafka unavailable (%v); using in-process bus", err)
		bus = kafka.NewInProcess(st, pl)
	} else {
		engine, err = kafka.StartEngine(bootstrap, st, pl)
		if err != nil {
			log.Printf("WARN: kafka engine not started: %v", err)
		} else {
			defer engine.Close()
		}
	}
	defer bus.Close()

	h := api.New(st, bus)
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

	log.Printf("trip-go listening on :%s (kafka=%s skills=%s)", port, bootstrap, skillsDir)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
