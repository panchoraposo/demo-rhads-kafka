package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	"${{values.module_path}}/internal/model"
	"${{values.module_path}}/internal/store"
)

const (
	topicIn  = "flow-in"
	topicOut = "flow-out"
)

type Bus interface {
	PublishRequest(requestID string, req model.TripRequest) error
	PublishDecision(a model.TripApproval) error
	Close()
}

type kafkaBus struct {
	prod  sarama.SyncProducer
	cons  sarama.ConsumerGroup
	store *store.Store
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type inProcess struct {
	store *store.Store
	mu    sync.Mutex
	wait  map[string]chan model.TripApproval
}

func NewBus(bootstrap string, st *store.Store) (Bus, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRange()
	cfg.Version = sarama.V3_6_0_0
	brokers := strings.Split(bootstrap, ",")
	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	group, err := sarama.NewConsumerGroup(brokers, "trip-go-flow-out", cfg)
	if err != nil {
		_ = prod.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := &kafkaBus{prod: prod, cons: group, store: st, cancel: cancel}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		h := &outHandler{store: st}
		for {
			if err := group.Consume(ctx, []string{topicOut}, h); err != nil {
				log.Printf("kafka consume: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	return b, nil
}

func NewInProcess(st *store.Store) Bus {
	return &inProcess{store: st, wait: map[string]chan model.TripApproval{}}
}

func (b *kafkaBus) PublishRequest(requestID string, req model.TripRequest) error {
	payload := map[string]any{"requestId": requestID, "request": req}
	return b.emit(topicIn, "com.tripplanner.trip.requested", requestID, "", payload)
}

func (b *kafkaBus) PublishDecision(a model.TripApproval) error {
	return b.emit(topicIn, "com.tripplanner.trip.approval.done", a.InstanceID, a.InstanceID, a)
}

func (b *kafkaBus) emit(topic, ceType, key, instanceID string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	event := cloudevents.NewEvent()
	event.SetID(fmt.Sprintf("%d", time.Now().UnixNano()))
	event.SetType(ceType)
	event.SetSource("trip-go")
	event.SetDataContentType("application/json")
	_ = event.SetData(cloudevents.ApplicationJSON, json.RawMessage(raw))
	if instanceID != "" {
		event.SetExtension("flowinstanceid", instanceID)
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder(key), Value: sarama.ByteEncoder(body)}
	_, _, err = b.prod.SendMessage(msg)
	return err
}

func (b *kafkaBus) Close() {
	b.cancel()
	b.wg.Wait()
	_ = b.cons.Close()
	_ = b.prod.Close()
}

type outHandler struct{ store *store.Store }

func (h *outHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *outHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *outHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var ev cloudevents.Event
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			sess.MarkMessage(msg, "")
			continue
		}
		h.apply(ev)
		sess.MarkMessage(msg, "")
	}
	return nil
}

func (h *outHandler) apply(ev cloudevents.Event) {
	var status model.TripPlanStatus
	if err := json.Unmarshal(ev.Data(), &status); err != nil {
		return
	}
	if status.RequestID == "" && status.InstanceID == "" {
		return
	}
	h.store.Update(&status)
}

func (p *inProcess) PublishRequest(requestID string, req model.TripRequest) error {
	instanceID := "local-" + requestID
	st := p.store.Bind(requestID, instanceID)
	if st == nil {
		return fmt.Errorf("unknown request")
	}
	st.Status = "awaiting_approval"
	st.Plan = &model.TripPlan{
		Summary:   fmt.Sprintf("In-process plan for %s", req.Destination),
		Itinerary: []string{"Day 1: arrive", "Day 2: explore", "Day 3: return"},
		Vehicle:   "Compact SUV",
		Estimate:  "stub (Kafka offline)",
	}
	p.store.Update(st)
	return nil
}

func (p *inProcess) PublishDecision(a model.TripApproval) error {
	st := p.store.ByInstance(a.InstanceID)
	if st == nil {
		return fmt.Errorf("unknown instance")
	}
	if a.Status == "rejected" {
		st.Status = "rejected"
	} else {
		st.Status = "confirmed"
		st.Confirmation = &model.BookingConfirmation{Reference: "MOS-GODEMO", Message: "Simulated booking confirmed. No vehicle has been reserved."}
	}
	p.store.Update(st)
	return nil
}

func (p *inProcess) Close() {}
