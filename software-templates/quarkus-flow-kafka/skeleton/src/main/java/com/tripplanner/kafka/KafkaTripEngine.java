package com.tripplanner.kafka;

import java.net.URI;
import java.util.UUID;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionStage;

import org.eclipse.microprofile.reactive.messaging.Channel;
import org.eclipse.microprofile.reactive.messaging.Emitter;
import org.eclipse.microprofile.reactive.messaging.Incoming;
import org.eclipse.microprofile.reactive.messaging.Message;
import org.eclipse.microprofile.reactive.messaging.Metadata;
import org.jboss.logging.Logger;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.tripplanner.agentic.TripPlannerService;
import com.tripplanner.model.BookingConfirmation;
import com.tripplanner.model.PlanningRequest;
import com.tripplanner.model.TripApproval;
import com.tripplanner.model.TripPlan;
import com.tripplanner.model.TripPlanStatus;
import com.tripplanner.store.TripPlanStore;
import io.smallrye.common.annotation.Blocking;
import io.smallrye.reactive.messaging.ce.CloudEventMetadata;
import io.smallrye.reactive.messaging.ce.OutgoingCloudEventMetadata;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;

/**
 * Consumes CloudEvents on Kafka {@code flow-in}, runs LangChain4j planning, emits outcomes on {@code flow-out}.
 * Same event contract as the Go trip planner (workshop HITL topics).
 */
@ApplicationScoped
public class KafkaTripEngine {
    private static final Logger LOG = Logger.getLogger(KafkaTripEngine.class);

    @Inject
    TripPlanStore store;

    @Inject
    TripPlannerService planner;

    @Inject
    ObjectMapper objectMapper;

    @Channel("trip-out")
    Emitter<TripPlanStatus> tripOut;

    @Incoming("trip-in")
    @Blocking
    public CompletionStage<Void> onTripIn(Message<String> message) {
        CloudEventMetadata<?> event = message.getMetadata(CloudEventMetadata.class).orElse(null);
        if (event == null) {
            return message.ack();
        }
        try {
            switch (event.getType()) {
                case "com.tripplanner.trip.requested" -> handleRequested(message.getPayload());
                case "com.tripplanner.trip.approval.done" -> handleApproval(message.getPayload());
                default -> LOG.debugf("Ignoring event type %s", event.getType());
            }
        } catch (Exception e) {
            LOG.error("Failed to process trip Kafka event", e);
        }
        return message.ack();
    }

    private void handleRequested(String payload) throws Exception {
        PlanningRequest input = objectMapper.readValue(payload, PlanningRequest.class);
        String instanceId = "q-" + input.requestId();
        TripPlanStatus status;
        try {
            status = store.bind(input, instanceId);
        } catch (IllegalStateException e) {
            LOG.warnf("Skipping duplicate planning request %s", input.requestId());
            return;
        }
        try {
            TripPlan plan = planner.planFromRequest(input.request());
            TripPlanStatus awaiting = new TripPlanStatus(
                    status.requestId(), instanceId, status.request(), "awaiting_approval",
                    plan, null, null, null);
            emit("com.tripplanner.trip.approval.requested", instanceId, awaiting);
        } catch (Exception failure) {
            LOG.errorf(failure, "Trip planning failed for %s", instanceId);
            TripPlanStatus failed = status.failed(failure);
            emit("com.tripplanner.trip.failed", instanceId, failed);
        }
    }

    private void handleApproval(String payload) throws Exception {
        TripApproval approval = objectMapper.readValue(payload, TripApproval.class);
        if (!store.matchesDecision(approval.instanceId(), approval)) {
            LOG.warnf("Ignoring unmatched approval for %s", approval.instanceId());
            return;
        }
        TripPlanStatus previous = store.byInstanceId(approval.instanceId());
        if (previous == null) {
            return;
        }
        if ("rejected".equals(approval.status())) {
            TripPlanStatus rejected = new TripPlanStatus(
                    previous.requestId(), previous.instanceId(), previous.request(), "rejected",
                    previous.plan(), null, null, null);
            emit("com.tripplanner.trip.rejected", approval.instanceId(), rejected);
            return;
        }
        BookingConfirmation confirmation = planner.finalizeBooking(approval);
        TripPlanStatus confirmed = new TripPlanStatus(
                previous.requestId(), previous.instanceId(), previous.request(), "confirmed",
                previous.plan(), confirmation, null, null);
        emit("com.tripplanner.booking.finalized", approval.instanceId(), confirmed);
    }

    private void emit(String type, String instanceId, TripPlanStatus status) {
        OutgoingCloudEventMetadata<TripPlanStatus> metadata = OutgoingCloudEventMetadata.<TripPlanStatus>builder()
                .withId(UUID.randomUUID().toString())
                .withSource(URI.create("engine:/trip-quarkus"))
                .withType(type)
                .withDataContentType("application/json")
                .withExtension("flowinstanceid", instanceId)
                .build();
        tripOut.send(Message.of(status, Metadata.of(metadata)).withNack(failure -> {
            LOG.errorf(failure, "Failed to publish %s for %s", type, instanceId);
            store.submissionFailed(status.requestId(), failure);
            return CompletableFuture.completedFuture(null);
        }));
    }
}
