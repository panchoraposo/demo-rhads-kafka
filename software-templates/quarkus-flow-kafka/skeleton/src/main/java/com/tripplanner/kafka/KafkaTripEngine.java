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

import com.fasterxml.jackson.databind.JsonNode;
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
import io.smallrye.reactive.messaging.kafka.api.IncomingKafkaRecordMetadata;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;

/**
 * Consumes CloudEvents on Kafka {@code flow-in}, runs LangChain4j planning, emits outcomes on {@code flow-out}.
 * Same event contract as the Go trip planner (workshop HITL topics).
 * Also supports in-process kicks when Kafka CloudEvent metadata is missing (common in local/Dev Spaces).
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
        String payload = message.getPayload();
        CloudEventMetadata<?> event = message.getMetadata(CloudEventMetadata.class).orElse(null);
        String type = event != null ? event.getType() : null;
        if (type == null || type.isBlank()) {
            type = inferType(payload);
            LOG.warnf("[kafka] trip-in without CloudEvent metadata; inferred type=%s payload=%s",
                    type, truncate(payload));
            message.getMetadata(IncomingKafkaRecordMetadata.class).ifPresent(meta ->
                    LOG.infof("[kafka] trip-in topic=%s partition=%d offset=%d",
                            meta.getTopic(), meta.getPartition(), meta.getOffset()));
        } else {
            LOG.infof("[kafka] trip-in type=%s id=%s", type, event.getId());
        }
        try {
            switch (type) {
                case "com.tripplanner.trip.requested" -> handleRequested(unwrapData(payload));
                case "com.tripplanner.trip.approval.done" -> handleApproval(unwrapData(payload));
                default -> LOG.infof("[kafka] ignoring event type %s", type);
            }
        } catch (Exception e) {
            LOG.error("[kafka] failed to process trip-in event", e);
        }
        return message.ack();
    }

    /** In-process planning kick (used when Kafka loopback does not deliver CloudEvents). */
    public void processRequested(PlanningRequest input) {
        try {
            LOG.infof("[engine] in-process planning requestId=%s destination=%s",
                    input.requestId(), input.request() != null ? input.request().destination() : "?");
            handleRequested(objectMapper.writeValueAsString(input));
        } catch (Exception e) {
            LOG.errorf(e, "[engine] in-process planning failed requestId=%s", input.requestId());
            store.submissionFailed(input.requestId(), e);
        }
    }

    /** In-process approval kick. */
    public void processApproval(TripApproval approval) {
        try {
            LOG.infof("[engine] in-process approval instanceId=%s status=%s",
                    approval.instanceId(), approval.status());
            handleApproval(objectMapper.writeValueAsString(approval));
        } catch (Exception e) {
            LOG.errorf(e, "[engine] in-process approval failed instanceId=%s", approval.instanceId());
        }
    }

    private void handleRequested(String payload) throws Exception {
        PlanningRequest input = objectMapper.readValue(payload, PlanningRequest.class);
        String instanceId = "q-" + input.requestId();
        TripPlanStatus status;
        try {
            status = store.bind(input, instanceId);
        } catch (IllegalStateException e) {
            LOG.warnf("[engine] skipping duplicate planning request %s", input.requestId());
            return;
        }
        LOG.infof("[engine] planning start requestId=%s instanceId=%s", input.requestId(), instanceId);
        try {
            TripPlan plan = planner.planFromRequest(input.request());
            TripPlanStatus awaiting = new TripPlanStatus(
                    status.requestId(), instanceId, status.request(), "awaiting_approval",
                    plan, null, null, null);
            LOG.infof("[engine] planning done instanceId=%s → awaiting_approval", instanceId);
            emit("com.tripplanner.trip.approval.requested", instanceId, awaiting);
        } catch (Exception failure) {
            LOG.errorf(failure, "[engine] planning failed instanceId=%s", instanceId);
            TripPlanStatus failed = status.failed(failure);
            emit("com.tripplanner.trip.failed", instanceId, failed);
        }
    }

    private void handleApproval(String payload) throws Exception {
        TripApproval approval = objectMapper.readValue(payload, TripApproval.class);
        if (!store.matchesDecision(approval.instanceId(), approval)) {
            LOG.warnf("[engine] ignoring unmatched approval for %s", approval.instanceId());
            return;
        }
        TripPlanStatus previous = store.byInstanceId(approval.instanceId());
        if (previous == null) {
            LOG.warnf("[engine] approval for unknown instance %s", approval.instanceId());
            return;
        }
        if ("rejected".equals(approval.status())) {
            TripPlanStatus rejected = new TripPlanStatus(
                    previous.requestId(), previous.instanceId(), previous.request(), "rejected",
                    previous.plan(), null, null, null);
            LOG.infof("[engine] emit rejected instanceId=%s", approval.instanceId());
            emit("com.tripplanner.trip.rejected", approval.instanceId(), rejected);
            return;
        }
        BookingConfirmation confirmation = planner.finalizeBooking(approval);
        TripPlanStatus confirmed = new TripPlanStatus(
                previous.requestId(), previous.instanceId(), previous.request(), "confirmed",
                previous.plan(), confirmation, null, null);
        LOG.infof("[engine] emit confirmed instanceId=%s ref=%s",
                approval.instanceId(), confirmation.bookingReference());
        emit("com.tripplanner.booking.finalized", approval.instanceId(), confirmed);
    }

    private void emit(String type, String instanceId, TripPlanStatus status) {
        // Always update the in-memory store so HTTP waiters unblock even if Kafka emit fails.
        store.acceptLocal(instanceId, status);
        OutgoingCloudEventMetadata<TripPlanStatus> metadata = OutgoingCloudEventMetadata.<TripPlanStatus>builder()
                .withId(UUID.randomUUID().toString())
                .withSource(URI.create("engine:/trip-quarkus"))
                .withType(type)
                .withDataContentType("application/json")
                .withExtension("flowinstanceid", instanceId)
                .build();
        LOG.infof("[kafka] publish %s instanceId=%s status=%s", type, instanceId, status.status());
        tripOut.send(Message.of(status, Metadata.of(metadata)).withNack(failure -> {
            LOG.errorf(failure, "[kafka] failed to publish %s for %s", type, instanceId);
            return CompletableFuture.completedFuture(null);
        }));
    }

    private String inferType(String payload) {
        try {
            JsonNode node = objectMapper.readTree(payload);
            if (node.has("type") && node.get("type").isTextual()) {
                return node.get("type").asText();
            }
            if (node.has("requestId") && node.has("request")) {
                return "com.tripplanner.trip.requested";
            }
            if (node.has("instanceId") && node.has("status")
                    && ("approved".equals(node.path("status").asText())
                    || "rejected".equals(node.path("status").asText()))) {
                return "com.tripplanner.trip.approval.done";
            }
        } catch (Exception ignored) {
            // fall through
        }
        return "unknown";
    }

    /** Unwrap structured CloudEvent JSON {@code data} when the connector did not strip it. */
    private String unwrapData(String payload) throws Exception {
        JsonNode node = objectMapper.readTree(payload);
        if (node.has("data") && (node.has("specversion") || node.has("type"))) {
            JsonNode data = node.get("data");
            return data.isTextual() ? data.asText() : objectMapper.writeValueAsString(data);
        }
        return payload;
    }

    private static String truncate(String value) {
        if (value == null) {
            return "";
        }
        return value.length() <= 240 ? value : value.substring(0, 240) + "…";
    }
}
