package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"

	"${{values.module_path}}/internal/store"
)

func Consume(bootstrap, topic string, st *store.Store) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Return.Errors = true
	cfg.Version = sarama.V3_6_0_0

	for {
		client, err := sarama.NewConsumer(strings.Split(bootstrap, ","), cfg)
		if err != nil {
			log.Printf("kafka connect retry: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		pc, err := client.ConsumePartition(topic, 0, sarama.OffsetNewest)
		if err != nil {
			log.Printf("kafka partition retry: %v", err)
			_ = client.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("consuming CDC topic %s", topic)
		for msg := range pc.Messages() {
			st.Add(parse(msg.Value))
		}
		_ = pc.Close()
		_ = client.Close()
		time.Sleep(2 * time.Second)
	}
}

func parse(b []byte) store.Event {
	e := store.Event{Raw: string(b), Op: "?", ID: "n/a"}
	var envelope map[string]any
	if err := json.Unmarshal(b, &envelope); err != nil {
		return e
	}
	payload, _ := envelope["payload"].(map[string]any)
	if payload == nil {
		payload = envelope
	}
	if op, ok := payload["op"].(string); ok {
		e.Op = op
	}
	after, _ := payload["after"].(map[string]any)
	if after == nil {
		after, _ = payload["before"].(map[string]any)
	}
	if after != nil {
		switch id := after["id"].(type) {
		case float64:
			e.ID = fmt.Sprintf("%.0f", id)
		case string:
			e.ID = id
		}
		if c, ok := after["customer"].(string); ok {
			e.Customer = c
		}
		if a, ok := after["amount"].(float64); ok {
			e.Amount = a
		}
		if s, ok := after["status"].(string); ok {
			e.Status = s
		}
	}
	return e
}
