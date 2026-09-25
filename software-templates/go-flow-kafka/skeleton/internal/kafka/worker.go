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

const (
	topicIn  = "flow-in"
	topicOut = "flow-out"
)

// Engine consumes trip.requested / approval.done on flow-in and emits outcomes on flow-out
// (same CloudEvents contract as the Quarkus LangChain4j trip planner).
type Engine struct {
	store   *store.Store
	planner *planner.Planner
	prod    sarama.SyncProducer
	cons    sarama.ConsumerGroup
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func StartEngine(bootstrap string, st *store.Store, pl *planner.Planner) (*Engine, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Version = sarama.V3_6_0_0
	brokers := strings.Split(bootstrap, ",")
	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	group, err := sarama.NewConsumerGroup(brokers, "trip-go-engine", cfg)
	if err != nil {
		_ = prod.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{store: st, planner: pl, prod: prod, cons: group, cancel: cancel}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		h := &inHandler{e: e}
		for {
			if err := group.Consume(ctx, []string{topicIn}, h); err != nil {
				log.Printf("[kafka] engine consume: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
	log.Printf("[kafka] engine listening on topic=%s group=trip-go-engine", topicIn)
	return e, nil
}

func (e *Engine) Close() {
	e.cancel()
	e.wg.Wait()
	_ = e.cons.Close()
	_ = e.prod.Close()
}

type inHandler struct{ e *Engine }

func (h *inHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *inHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *inHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var ev cloudevents.Event
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			log.Printf("[kafka] ignore non-cloudevent on %s: %v", topicIn, err)
		} else {
			h.e.handle(ev)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func (e *Engine) handle(ev cloudevents.Event) {
	log.Printf("[kafka] received type=%s id=%s", ev.Type(), ev.ID())
	switch ev.Type() {
	case "com.tripplanner.trip.requested":
		var payload struct {
			RequestID string            `json:"requestId"`
			Request   model.TripRequest `json:"request"`
		}
		if err := json.Unmarshal(ev.Data(), &payload); err != nil {
			log.Printf("[kafka] bad trip.requested payload: %v", err)
			return
		}
		instanceID := fmt.Sprintf("go-%s", payload.RequestID)
		st := e.store.Bind(payload.RequestID, instanceID)
		if st == nil {
			log.Printf("[kafka] skipping duplicate/unknown request %s", payload.RequestID)
			return
		}
		log.Printf("[kafka] planning requestId=%s instanceId=%s", payload.RequestID, instanceID)
		plan, err := e.planner.Plan(payload.Request)
		if err != nil {
			code, message := planner.MapError(err)
			log.Printf("[kafka] planning failed instanceId=%s error=%s: %v", instanceID, code, err)
			failed := *st
			failed.Status = "failed"
			failed.Error = code
			failed.Message = message
			_ = e.emitOut("com.tripplanner.trip.failed", instanceID, &failed)
			return
		}
		awaiting := *st
		awaiting.Status = "awaiting_approval"
		awaiting.Plan = plan
		log.Printf("[kafka] emit approval.requested instanceId=%s", instanceID)
		_ = e.emitOut("com.tripplanner.trip.approval.requested", instanceID, &awaiting)
	case "com.tripplanner.trip.approval.done":
		var a model.TripApproval
		if err := json.Unmarshal(ev.Data(), &a); err != nil {
			log.Printf("[kafka] bad approval.done payload: %v", err)
			return
		}
		if !e.store.MatchesDecision(a.InstanceID, a) {
			log.Printf("[kafka] ignoring unmatched approval for %s", a.InstanceID)
			return
		}
		st := e.store.ByInstance(a.InstanceID)
		if st == nil {
			return
		}
		if a.Status == "rejected" {
			rejected := *st
			rejected.Status = "rejected"
			rejected.Confirmation = nil
			log.Printf("[kafka] emit trip.rejected instanceId=%s", a.InstanceID)
			_ = e.emitOut("com.tripplanner.trip.rejected", a.InstanceID, &rejected)
			return
		}
		confirmed := *st
		confirmed.Status = "confirmed"
		confirmed.Confirmation = &model.BookingConfirmation{
			BookingReference: fmt.Sprintf("MOS-%d", time.Now().Unix()%100000000),
			Message:          "Simulated booking confirmed. No vehicle has been reserved.",
		}
		log.Printf("[kafka] emit booking.finalized instanceId=%s ref=%s", a.InstanceID, confirmed.Confirmation.BookingReference)
		_ = e.emitOut("com.tripplanner.booking.finalized", a.InstanceID, &confirmed)
	default:
		log.Printf("[kafka] ignoring event type %s", ev.Type())
	}
}

func (e *Engine) emitOut(ceType, instanceID string, st *model.TripPlanStatus) error {
	raw, _ := json.Marshal(st)
	event := cloudevents.NewEvent()
	event.SetID(fmt.Sprintf("%d", time.Now().UnixNano()))
	event.SetType(ceType)
	event.SetSource("trip-go-engine")
	event.SetExtension("flowinstanceid", instanceID)
	_ = event.SetData(cloudevents.ApplicationJSON, json.RawMessage(raw))
	body, _ := json.Marshal(event)
	_, _, err := e.prod.SendMessage(&sarama.ProducerMessage{
		Topic: topicOut,
		Key:   sarama.StringEncoder(instanceID),
		Value: sarama.ByteEncoder(body),
	})
	if err != nil {
		log.Printf("[kafka] emit %s failed: %v", ceType, err)
	}
	return err
}
