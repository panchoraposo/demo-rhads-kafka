package com.tripplanner.agentic.agents;

import jakarta.enterprise.context.ApplicationScoped;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.util.Locale;

/**
 * Loads demo skill markdown from {@code classpath:skills/} without the community
 * {@code quarkus-langchain4j-skills} extension (not in the Red Hat langchain4j BOM).
 */
@ApplicationScoped
public class SkillGuidance {

    public String forTripType(String tripType) {
        String key = normalizeTripType(tripType) + "-trip";
        return load("skills/" + key + "/SKILL.md",
                "Use sensible defaults for a " + key + ".");
    }

    public String forVehicleSelection() {
        return load("skills/vehicle-selection/SKILL.md",
                "Prefer a safe, comfortable vehicle that fits travelers and budget.");
    }

    /** Combined guidance passed into the agent graph as {@code skillGuidance}. */
    public String combined(String tripType) {
        return """
                ## Trip-type skill (%s)
                %s

                ## Vehicle-selection skill
                %s
                """.formatted(normalizeTripType(tripType), forTripType(tripType), forVehicleSelection());
    }

    private static String normalizeTripType(String tripType) {
        if (tripType == null || tripType.isBlank()) {
            return "family";
        }
        return tripType.trim().toLowerCase(Locale.ROOT).replace('_', '-');
    }

    private static String load(String classpath, String fallback) {
        try (InputStream in = Thread.currentThread().getContextClassLoader().getResourceAsStream(classpath)) {
            if (in == null) {
                return fallback;
            }
            return new String(in.readAllBytes(), StandardCharsets.UTF_8);
        } catch (IOException e) {
            return fallback;
        }
    }
}
