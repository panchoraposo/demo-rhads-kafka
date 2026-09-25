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
	"${{values.module_path}}/internal/planner"
	"${{values.module_path}}/internal/store"
)

type Bus interface {
	PublishRequest(requestID string, req model.TripRequest) error
	PublishDecision(a model.TripApproval) error
	Close()
}

type kafkaBus struct {
	prod   sarama.SyncProducer
	cons   sarama.ConsumerGroup
	store  *store.Store
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type inProcess struct {
	store   *store.Store
	planner *planner.Planner
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
	group, err := sarama.NewConsumerGroup(brokers, "trip-go-store", cfg)
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
				log.Printf("[kafka] store consume: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	log.Printf("[kafka] store listening on topic=%s group=trip-go-store", topicOut)
	return b, nil
}

func NewInProcess(st *store.Store, pl *planner.Planner) Bus {
	return &inProcess{store: st, planner: pl}
}

func (b *kafkaBus) PublishRequest(requestID string, req model.TripRequest) error {
	log.Printf("[kafka] publish trip.requested requestId=%s", requestID)
	payload := map[string]any{"requestId": requestID, "request": req}
	return b.emit(topicIn, "com.tripplanner.trip.requested", requestID, "", payload)
}

func (b *kafkaBus) PublishDecision(a model.TripApproval) error {
	log.Printf("[kafka] publish approval.done instanceId=%s status=%s", a.InstanceID, a.Status)
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
	if err != nil {
		log.Printf("[kafka] publish %s failed: %v", ceType, err)
	}
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
	switch ev.Type() {
	case "com.tripplanner.trip.approval.requested",
		"com.tripplanner.booking.finalized",
		"com.tripplanner.trip.rejected",
		"com.tripplanner.trip.failed":
	default:
		return
	}
	var status model.TripPlanStatus
	if err := json.Unmarshal(ev.Data(), &status); err != nil {
		log.Printf("[kafka] bad outcome payload type=%s: %v", ev.Type(), err)
		return
	}
	log.Printf("[kafka] store accept type=%s requestId=%s status=%s", ev.Type(), status.RequestID, status.Status)
	h.store.AcceptOutcome(&status)
}

func (p *inProcess) PublishRequest(requestID string, req model.TripRequest) error {
	instanceID := "go-" + requestID
	st := p.store.Bind(requestID, instanceID)
	if st == nil {
		return fmt.Errorf("unknown request")
	}
	log.Printf("[kafka] in-process planning requestId=%s", requestID)
	plan, err := p.planner.Plan(req)
	if err != nil {
		code, message := planner.MapError(err)
		failed := *st
		failed.Status = "failed"
		failed.Error = code
		failed.Message = message
		p.store.AcceptOutcome(&failed)
		return nil
	}
	awaiting := *st
	awaiting.Status = "awaiting_approval"
	awaiting.Plan = plan
	p.store.AcceptOutcome(&awaiting)
	return nil
}

func (p *inProcess) PublishDecision(a model.TripApproval) error {
	if !p.store.MatchesDecision(a.InstanceID, a) {
		return fmt.Errorf("unmatched decision")
	}
	st := p.store.ByInstance(a.InstanceID)
	if st == nil {
		return fmt.Errorf("unknown instance")
	}
	if a.Status == "rejected" {
		rejected := *st
		rejected.Status = "rejected"
		p.store.AcceptOutcome(&rejected)
		return nil
	}
	confirmed := *st
	confirmed.Status = "confirmed"
	confirmed.Confirmation = &model.BookingConfirmation{
		BookingReference: fmt.Sprintf("MOS-%d", time.Now().Unix()%100000000),
		Message:          "Simulated booking confirmed. No vehicle has been reserved.",
	}
	p.store.AcceptOutcome(&confirmed)
	return nil
}

func (p *inProcess) Close() {}
