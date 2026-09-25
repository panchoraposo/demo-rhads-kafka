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

// Engine consumes trip.requested / approval.done on flow-in and emits outcomes on flow-out
// (same CloudEvents contract as Quarkus Flow workshop step-04).
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
	group, err := sarama.NewConsumerGroup(brokers, "trip-go-flow-engine", cfg)
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
				log.Printf("engine consume: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
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
		if err := json.Unmarshal(msg.Value, &ev); err == nil {
			h.e.handle(ev)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func (e *Engine) handle(ev cloudevents.Event) {
	switch ev.Type() {
	case "com.tripplanner.trip.requested":
		var payload struct {
			RequestID string            `json:"requestId"`
			Request   model.TripRequest `json:"request"`
		}
		if err := json.Unmarshal(ev.Data(), &payload); err != nil {
			return
		}
		instanceID := fmt.Sprintf("go-%s", payload.RequestID)
		st := e.store.Bind(payload.RequestID, instanceID)
		if st == nil {
			st = &model.TripPlanStatus{RequestID: payload.RequestID, InstanceID: instanceID, Request: payload.Request}
		}
		plan, err := e.planner.Plan(payload.Request)
		if err != nil {
			st.Status = "failed"
			st.Error = "planning_failed"
			st.Message = err.Error()
			_ = e.emitOut("com.tripplanner.trip.failed", instanceID, st)
			return
		}
		st.Status = "awaiting_approval"
		st.Plan = plan
		e.store.Update(st)
		_ = e.emitOut("com.tripplanner.trip.approval.requested", instanceID, st)
	case "com.tripplanner.trip.approval.done":
		var a model.TripApproval
		if err := json.Unmarshal(ev.Data(), &a); err != nil {
			return
		}
		st := e.store.ByInstance(a.InstanceID)
		if st == nil || !e.store.MatchesDecision(a.InstanceID, a) {
			return
		}
		if a.Status == "rejected" {
			st.Status = "rejected"
			e.store.Update(st)
			_ = e.emitOut("com.tripplanner.trip.rejected", a.InstanceID, st)
			return
		}
		st.Status = "confirmed"
		st.Confirmation = &model.BookingConfirmation{
			Reference: fmt.Sprintf("MOS-%d", time.Now().Unix()%100000000),
			Message:   "Simulated booking confirmed. No vehicle has been reserved.",
		}
		e.store.Update(st)
		_ = e.emitOut("com.tripplanner.booking.finalized", a.InstanceID, st)
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
	_, _, err := e.prod.SendMessage(&sarama.ProducerMessage{Topic: topicOut, Key: sarama.StringEncoder(instanceID), Value: sarama.ByteEncoder(body)})
	return err
}
