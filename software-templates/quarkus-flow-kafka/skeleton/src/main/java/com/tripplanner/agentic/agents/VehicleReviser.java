package com.tripplanner.agentic.agents;

import com.tripplanner.guardrails.TripAppropriatenessGuardrail;
import com.tripplanner.model.TripPlan;
import com.tripplanner.model.VehicleEvaluation;
import dev.langchain4j.agentic.Agent;
import dev.langchain4j.service.UserMessage;
import dev.langchain4j.service.guardrail.OutputGuardrails;

/**
 * Revises vehicle recommendations. Uses the default ChatModel from the RH langchain4j BOM
 * (dynamic @CdiBean ChatModelSupplier is not available in quarkus-langchain4j-agentic 1.7.6).
 */
public interface VehicleReviser {

    @UserMessage("""
            You are a vehicle recommendation specialist.
            Revise the current vehicle recommendation based on the evaluation feedback.
            Keep the same format but improve the choice to address the evaluators' suggestions.

            Current recommendation: {vehicle}
            Evaluation score: {evaluation}
            Trip type: {tripType}
            Number of travelers: {travelers}
            Budget: {budget}
            Destination: {destination}
            Additional preferences: {preferences}
            Preserve the original traveler, budget, and preference constraints.
            """)
    @OutputGuardrails(value = TripAppropriatenessGuardrail.class, maxRetries = 3)
    @Agent(description = "Revises the vehicle recommendation based on evaluation feedback",
           outputKey = "vehicle")
    TripPlan.VehicleRecommendation revise(TripPlan.VehicleRecommendation vehicle,
                                          VehicleEvaluation evaluation,
                                          String tripType,
                                          String travelers,
                                          String budget,
                                          String destination,
                                          String preferences);
}
