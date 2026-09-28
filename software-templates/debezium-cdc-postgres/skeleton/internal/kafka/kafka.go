package kafka

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"

	"${{values.module_path}}/internal/db"
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

// Publisher emits Debezium-shaped events (used when LOCAL_CDC_FANOUT=true).
type Publisher struct {
	prod  sarama.SyncProducer
	topic string
}

func NewPublisher(bootstrap, topic string) (*Publisher, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Version = sarama.V3_6_0_0
	prod, err := sarama.NewSyncProducer(strings.Split(bootstrap, ","), cfg)
	if err != nil {
		return nil, err
	}
	return &Publisher{prod: prod, topic: topic}, nil
}

func (p *Publisher) Close() {
	if p != nil && p.prod != nil {
		_ = p.prod.Close()
	}
}

func (p *Publisher) Emit(op string, before, after *db.Order) error {
	if p == nil {
		return nil
	}
	payload := map[string]any{
		"before": beforeMap(before),
		"after":  afterMap(after),
		"op":     op,
		"ts_ms":  time.Now().UnixMilli(),
	}
	body, _ := json.Marshal(map[string]any{"payload": payload})
	key := ""
	if after != nil {
		key = fmt.Sprintf("%d", after.ID)
	} else if before != nil {
		key = fmt.Sprintf("%d", before.ID)
	}
	_, _, err := p.prod.SendMessage(&sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(body),
	})
	if err != nil {
		log.Printf("fanout publish failed: %v", err)
	} else {
		log.Printf("fanout published op=%s key=%s topic=%s", op, key, p.topic)
	}
	return err
}

func beforeMap(o *db.Order) any {
	if o == nil {
		return nil
	}
	return o.AsMap()
}

func afterMap(o *db.Order) any {
	if o == nil {
		return nil
	}
	return o.AsMap()
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
	if before, ok := payload["before"].(map[string]any); ok && before != nil {
		e.Before = before
	}
	if after, ok := payload["after"].(map[string]any); ok && after != nil {
		e.After = after
	}
	row := e.After
	if row == nil {
		row = e.Before
	}
	if row != nil {
		switch id := row["id"].(type) {
		case float64:
			e.ID = fmt.Sprintf("%.0f", id)
		case string:
			e.ID = id
		case int64:
			e.ID = fmt.Sprintf("%d", id)
		case json.Number:
			e.ID = id.String()
		}
		if c, ok := row["customer"].(string); ok {
			e.Customer = c
		}
		switch a := row["amount"].(type) {
		case float64:
			e.Amount = a
		case json.Number:
			f, _ := a.Float64()
			e.Amount = f
		}
		if s, ok := row["status"].(string); ok {
			e.Status = s
		}
	}
	return e
}
