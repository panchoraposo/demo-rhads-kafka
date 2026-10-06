package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// envFromProperty maps application.properties keys to process env.
// A variable already set in the environment wins (cluster Deployment, Dev Spaces).
var envFromProperty = map[string]string{
	"port":                     "PORT",
	"kafka.bootstrap.servers":  "KAFKA_BOOTSTRAP_SERVERS",
	"kafka.topic":              "KAFKA_TOPIC",
	"apicurio.registry.url":    "APICURIO_REGISTRY_URL",
	"component.id":             "COMPONENT_ID",
	"web.root":                 "WEB_ROOT",
	"database.host":            "DATABASE_HOST",
	"database.port":            "DATABASE_PORT",
	"database.user":            "DATABASE_USER",
	"database.password":        "DATABASE_PASSWORD",
	"database.name":            "DATABASE_NAME",
	"database.sslmode":         "DATABASE_SSLMODE",
	"local.cdc.fanout":         "LOCAL_CDC_FANOUT",
}

// ApplyFile loads path and sets env vars that are still empty.
func ApplyFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("no %s (%v); using environment only", path, err)
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	applied := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		envName, mapped := envFromProperty[strings.TrimSpace(key)]
		if !mapped {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" || os.Getenv(envName) != "" {
			continue
		}
		if err := os.Setenv(envName, val); err != nil {
			log.Printf("set %s: %v", envName, err)
			continue
		}
		applied++
	}
	log.Printf("applied %d settings from %s", applied, path)
}
