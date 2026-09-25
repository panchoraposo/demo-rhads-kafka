package com.tripplanner.agentic;

import com.tripplanner.agentic.agents.ComfortEvaluator;
import com.tripplanner.agentic.agents.CostEstimatorAgent;
import com.tripplanner.agentic.agents.CostEvaluator;
import com.tripplanner.agentic.agents.FuelEfficiencyEvaluator;
import com.tripplanner.agentic.agents.ItineraryPlannerAgent;
import com.tripplanner.agentic.agents.SkillGuidance;
import com.tripplanner.agentic.agents.VehicleAdvisorAgent;
import com.tripplanner.agentic.agents.VehicleReviser;
import com.tripplanner.model.BookingConfirmation;
import com.tripplanner.model.ItineraryResult;
import com.tripplanner.model.TripApproval;
import com.tripplanner.model.TripPlan;
import com.tripplanner.model.TripQualityException;
import com.tripplanner.model.TripRequest;
import com.tripplanner.model.VehicleEvaluation;
import io.quarkus.logging.Log;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.inject.Inject;
import org.eclipse.microprofile.config.inject.ConfigProperty;

import java.util.UUID;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Orchestrates Red Hat langchain4j {@code @Agent} beans for trip planning.
 * Kafka owns the HITL lifecycle ({@link com.tripplanner.kafka.KafkaTripEngine}).
 */
@ApplicationScoped
public class TripPlannerService {

    private static final int MAX_REVISIONS = 3;
    private static final double EXIT_SCORE = 7.5;

    private final ExecutorService workers = Executors.newFixedThreadPool(4);

    @Inject
    SkillGuidance skillGuidance;

    @Inject
    VehicleAdvisorAgent vehicleAdvisor;

    @Inject
    ItineraryPlannerAgent itineraryPlanner;

    @Inject
    ComfortEvaluator comfortEvaluator;

    @Inject
    CostEvaluator costEvaluator;

    @Inject
    FuelEfficiencyEvaluator fuelEvaluator;

    @Inject
    VehicleReviser vehicleReviser;

    @Inject
    CostEstimatorAgent costEstimator;

    @ConfigProperty(name = "quarkus.langchain4j.openai.api-key", defaultValue = "")
    String apiKey;

    public TripPlan planFromRequest(TripRequest request) {
        if (apiKey == null || apiKey.isBlank() || "local-dev-placeholder".equals(apiKey)) {
            throw new IllegalStateException(
                    "MAAS_API_KEY is not set. Put a real key in apps/trip-quarkus/.env and restart Quarkus.");
        }
        String days = String.valueOf(request.days());
        String travelers = String.valueOf(request.travelers());
        String skills = skillGuidance.combined(request.tripType());

        CompletableFuture<TripPlan.VehicleRecommendation> vehicleFut = CompletableFuture.supplyAsync(
                () -> vehicleAdvisor.recommendVehicle(
                        request.destination(),
                        request.tripType(),
                        travelers,
                        request.budget(),
                        request.preferences(),
                        skills),
                workers);
        CompletableFuture<ItineraryResult> itineraryFut = CompletableFuture.supplyAsync(
                () -> itineraryPlanner.planItinerary(
                        request.destination(),
                        request.startDate(),
                        days,
                        request.tripType(),
                        request.preferences(),
                        skills),
                workers);

        TripPlan.VehicleRecommendation vehicle = vehicleFut.join();
        ItineraryResult itinerary = itineraryFut.join();

        vehicle = reviewVehicle(vehicle, request, days, travelers);

        TripPlan.CostEstimate costs = costEstimator.estimateCosts(
                vehicle, itinerary, days, travelers, request.budget());

        return new TripPlan(vehicle, itinerary.routeOverview(), itinerary.itinerary(), costs);
    }

    private TripPlan.VehicleRecommendation reviewVehicle(TripPlan.VehicleRecommendation vehicle,
                                                         TripRequest request,
                                                         String days,
                                                         String travelers) {
        for (int i = 0; i <= MAX_REVISIONS; i++) {
            VehicleEvaluation evaluation = evaluateVehicle(vehicle, request, days, travelers);
            Log.infof("Vehicle evaluation round %d score=%.2f", i + 1, evaluation.score());
            if (evaluation.score() >= EXIT_SCORE) {
                return vehicle;
            }
            if (i == MAX_REVISIONS) {
                throw new TripQualityException();
            }
            vehicle = vehicleReviser.revise(
                    vehicle,
                    evaluation,
                    request.tripType(),
                    travelers,
                    request.budget(),
                    request.destination(),
                    request.preferences());
        }
        throw new TripQualityException();
    }

    private VehicleEvaluation evaluateVehicle(TripPlan.VehicleRecommendation vehicle,
                                              TripRequest request,
                                              String days,
                                              String travelers) {
        CompletableFuture<VehicleEvaluation> comfort = CompletableFuture.supplyAsync(
                () -> comfortEvaluator.evaluateComfort(vehicle, request.tripType(), travelers, days),
                workers);
        CompletableFuture<VehicleEvaluation> cost = CompletableFuture.supplyAsync(
                () -> costEvaluator.evaluateCost(vehicle, request.budget(), travelers, days),
                workers);
        CompletableFuture<VehicleEvaluation> fuel = CompletableFuture.supplyAsync(
                () -> fuelEvaluator.evaluateFuelEfficiency(
                        vehicle, request.destination(), days, request.tripType()),
                workers);
        return VehicleEvaluators.aggregate(comfort.join(), cost.join(), fuel.join());
    }

    public BookingConfirmation finalizeBooking(TripApproval approval) {
        return new BookingConfirmation(
                "MOS-" + UUID.randomUUID().toString().substring(0, 8).toUpperCase(),
                "Simulated booking confirmed. No vehicle has been reserved.");
    }
}
