package com.tripplanner.agentic.agents;

import com.tripplanner.guardrails.TripAppropriatenessGuardrail;
import com.tripplanner.model.TripPlan;
import dev.langchain4j.agentic.Agent;
import dev.langchain4j.service.UserMessage;
import dev.langchain4j.service.guardrail.OutputGuardrails;

/**
 * Vehicle advisor. Skill markdown under {@code classpath:skills/vehicle-selection/}
 * is loaded by {@link SkillGuidance} (no community skills extension — Red Hat BOM only).
 */
public interface VehicleAdvisorAgent {

    @UserMessage("""
            You are a vehicle specialist for road trips.
            Apply this vehicle-selection guidance:
            ---
            {skillGuidance}
            ---
            Based on the guidance and the trip details below, recommend exactly one rental vehicle.
            Follow the vehicle-selection skill and return the make and model from the matching row.
            A different destination, trip type, traveler count, budget tier, or preference must change the model.
            The model field must be a specific make and model, never only a category such as "SUV" or "MPV".
            The type field must be one of compact, estate, suv, mpv, or convertible.

            - Destination: {destination}
            - Trip type: {tripType}
            - Number of travelers: {travelers}
            - Budget: {budget}
            - Additional preferences: {preferences}

            The budget names the traveler's currency. Do not quote prices in a different currency.
            In the reasoning, name the destination cue, trip type, traveler count, and budget tier that selected this model.
            """)
    @Agent(description = "Recommends the best vehicle for the trip based on destination, travelers, and budget",
           outputKey = "vehicle")
    @OutputGuardrails(value = TripAppropriatenessGuardrail.class, maxRetries = 3)
    TripPlan.VehicleRecommendation recommendVehicle(String destination,
                                                    String tripType,
                                                    String travelers,
                                                    String budget,
                                                    String preferences,
                                                    String skillGuidance);
}
