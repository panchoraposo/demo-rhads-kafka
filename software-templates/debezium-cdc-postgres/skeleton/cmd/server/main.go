package main

import (
	"log"
	"net/http"
	"os"

	"${{values.module_path}}/internal/api"
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
	port := env("PORT", "${{values.port}}")
	bootstrap := env("KAFKA_BOOTSTRAP_SERVERS", "rhads-kafka-kafka-bootstrap.kafka.svc:9092")
	topic := env("KAFKA_TOPIC", "${{values.kafka_topic}}")
	webRoot := env("WEB_ROOT", "web")
	apicurio := env("APICURIO_REGISTRY_URL", "http://rhads-registry-service.apicurio.svc:8080")

	st := store.New()
	go kafka.Consume(bootstrap, topic, st)

	mux := http.NewServeMux()
	api.Register(mux, st, apicurio, topic)
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))

	log.Printf("CDC consumer listening on :%s topic=%s apicurio=%s", port, topic, apicurio)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
