package com.tripplanner.agentic;

import com.tripplanner.model.VehicleEvaluation;

/** Aggregates parallel comfort / cost / fuel evaluator scores. */
public final class VehicleEvaluators {

    private VehicleEvaluators() {
    }

    public static VehicleEvaluation aggregate(VehicleEvaluation... votes) {
        if (votes == null || votes.length != 3) {
            throw new IllegalStateException("Expected all three vehicle evaluations");
        }
        double totalScore = 0;
        StringBuilder suggestions = new StringBuilder();
        for (VehicleEvaluation eval : votes) {
            if (eval == null || !Double.isFinite(eval.score()) || eval.score() < 1 || eval.score() > 10) {
                throw new IllegalStateException("Each vehicle evaluator must return a score between 1 and 10");
            }
            totalScore += eval.score();
            if (eval.suggestions() != null && !eval.suggestions().isBlank()) {
                if (!suggestions.isEmpty()) {
                    suggestions.append("; ");
                }
                suggestions.append(eval.suggestions());
            }
        }
        return new VehicleEvaluation(totalScore / votes.length, suggestions.toString());
    }
}
