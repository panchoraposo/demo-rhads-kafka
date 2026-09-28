package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"${{values.module_path}}/internal/api"
	"${{values.module_path}}/internal/db"
	"${{values.module_path}}/internal/kafka"
	"${{values.module_path}}/internal/store"
	_ "${{values.module_path}}/internal/cve"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	port := env("PORT", "${{values.port}}")
	bootstrap := env("KAFKA_BOOTSTRAP_SERVERS", "rhads-kafka-kafka-bootstrap.kafka.svc:9092")
	topic := env("KAFKA_TOPIC", "${{values.kafka_topic}}")
	webRoot := env("WEB_ROOT", "web")
	apicurio := env("APICURIO_REGISTRY_URL", "http://rhads-registry-service.apicurio.svc:8080")
	component := env("COMPONENT_ID", "${{values.component_id}}")

	st := store.New()
	go kafka.Consume(bootstrap, topic, st)

	var database *db.DB
	if os.Getenv("DATABASE_URL") != "" || os.Getenv("DATABASE_HOST") != "" {
		d, err := db.OpenFromEnv()
		if err != nil {
			log.Printf("WARN: database unavailable (%v); Order Desk writes disabled", err)
		} else {
			database = d
			defer database.Close()
			log.Printf("database connected (Order Desk writes enabled)")
		}
	} else {
		log.Printf("DATABASE_* not set — Order Desk writes disabled (consumer-only)")
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
	api.Register(mux, st, database, fanout, apicurio, topic, component)
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))

	log.Printf("CDC Order Desk listening on :%s topic=%s apicurio=%s", port, topic, apicurio)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
