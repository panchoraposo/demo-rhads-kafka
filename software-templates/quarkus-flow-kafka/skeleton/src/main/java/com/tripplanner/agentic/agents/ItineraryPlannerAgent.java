package com.tripplanner.agentic.agents;

import com.tripplanner.guardrails.TripSafetyGuardrail;
import com.tripplanner.model.ItineraryResult;
import dev.langchain4j.agentic.Agent;
import dev.langchain4j.service.UserMessage;
import dev.langchain4j.service.guardrail.OutputGuardrails;

/**
 * Itinerary agent. Skill markdown under {@code classpath:skills/{tripType}-trip/}
 * is loaded by {@link SkillGuidance} (Red Hat langchain4j BOM has no skills extension).
 */
public interface ItineraryPlannerAgent {

    @UserMessage("""
            You are an expert trip itinerary planner.
            Apply this trip-type guidance:
            ---
            {skillGuidance}
            ---
            Create a detailed day-by-day itinerary and a route overview for the trip.
            Include a title, description, and overnight stop for each day.
            Consider the travel dates when suggesting activities and seasonal attractions.

            - Destination: {destination}
            - Start date: {startDate}
            - Duration: {days} days
            - Trip type: {tripType}
            - Additional preferences: {preferences}
            """)
    @Agent(description = "Creates a detailed day-by-day itinerary and route overview",
           outputKey = "itineraryResult")
    @OutputGuardrails(value = TripSafetyGuardrail.class, maxRetries = 3)
    ItineraryResult planItinerary(String destination,
                                  String startDate,
                                  String days,
                                  String tripType,
                                  String preferences,
                                  String skillGuidance);
}
