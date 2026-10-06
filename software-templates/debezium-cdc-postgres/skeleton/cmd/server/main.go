package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"${{values.module_path}}/internal/api"
	"${{values.module_path}}/internal/config"
	_ "${{values.module_path}}/internal/cve"
	"${{values.module_path}}/internal/db"
	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/store"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	config.ApplyFile("application.properties")
	port := env("PORT", "${{values.port}}")
	bootstrap := env("KAFKA_BOOTSTRAP_SERVERS", "rhads-kafka-kafka-bootstrap.kafka.svc:9092")
	topic := env("KAFKA_TOPIC", "${{values.kafka_topic}}")
	webRoot := env("WEB_ROOT", "web")
	apicurio := env("APICURIO_REGISTRY_URL", "http://rhads-registry-service.apicurio.svc:8080")
	component := env("COMPONENT_ID", "${{values.component_id}}")

	st := store.New()
	go kafka.Consume(bootstrap, topic, st)

	database := openDB()
	if database != nil {
		defer database.Close()
	}

	var fanout *kafka.Publisher
	if strings.EqualFold(os.Getenv("LOCAL_CDC_FANOUT"), "true") {
		p, err := kafka.NewPublisher(bootstrap, topic)
		if err != nil {
			log.Printf("WARN: LOCAL_CDC_FANOUT set but publisher failed: %v", err)
		} else {
			fanout = p
			defer fanout.Close()
			log.Printf("LOCAL_CDC_FANOUT enabled — app will emit Debezium-shaped events after writes")
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	h := api.Register(mux, st, database, fanout, apicurio, topic, component)
	if database == nil && dbConfigured() {
		go reconnectDB(h)
	}
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))

	log.Printf("CDC Order Desk listening on :%s topic=%s apicurio=%s", port, topic, apicurio)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func dbConfigured() bool {
	return os.Getenv("DATABASE_URL") != "" || os.Getenv("DATABASE_HOST") != ""
}

func openDB() *db.DB {
	if !dbConfigured() {
		log.Printf("DATABASE_* not set — Order Desk writes disabled (consumer-only)")
		return nil
	}
	d, err := db.OpenFromEnv()
	if err != nil {
		log.Printf("WARN: database unavailable (%v); Order Desk writes disabled until reconnect", err)
		return nil
	}
	log.Printf("database connected (Order Desk writes enabled)")
	return d
}

func reconnectDB(h *api.Handler) {
	for {
		time.Sleep(5 * time.Second)
		d := openDB()
		if d == nil {
			continue
		}
		h.SetDB(d)
		return
	}
}
