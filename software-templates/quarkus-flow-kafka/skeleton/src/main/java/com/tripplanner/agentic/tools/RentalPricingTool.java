package com.tripplanner.agentic.tools;

import com.tripplanner.guardrails.RentalEstimateInputGuardrail;
import dev.langchain4j.agent.tool.Tool;
import io.quarkiverse.langchain4j.guardrails.ToolInputGuardrails;
import jakarta.enterprise.context.ApplicationScoped;
import org.jboss.logging.Logger;

import java.util.Locale;
import java.util.Map;

@ApplicationScoped
public class RentalPricingTool {

    // Fictional workshop rates in whole euros per day, not live rental prices.
    // Other currencies use a fixed demo scale (USD 1.1, CLP 1000, ARS 1200 per euro).
    public static final Map<String, Integer> DAILY_RATES = Map.of(
            "compact", 45, "estate", 65, "suv", 80, "mpv", 90);

    private static final ThreadLocal<String> ACTIVE = new ThreadLocal<>();

    private static final Logger LOG = Logger.getLogger(RentalPricingTool.class);

    public static void use(String currency) {
        ACTIVE.set(normalize(currency));
    }

    public static void clear() {
        ACTIVE.remove();
    }

    public static String normalize(String currency) {
        if (currency == null || currency.isBlank()) {
            return "EUR";
        }
        return switch (currency.trim().toUpperCase(Locale.ROOT)) {
            case "USD", "CLP", "ARS" -> currency.trim().toUpperCase(Locale.ROOT);
            default -> "EUR";
        };
    }

    public static String describe(String currency) {
        return switch (normalize(currency)) {
            case "USD" -> "US dollars (USD)";
            case "CLP" -> "Chilean pesos (CLP)";
            case "ARS" -> "Argentine pesos (ARS)";
            default -> "euros (EUR)";
        };
    }

    public static int convertFromEur(int eurAmount, String currency) {
        return switch (normalize(currency)) {
            case "USD" -> (eurAmount * 11 + 5) / 10;
            case "CLP" -> eurAmount * 1_000;
            case "ARS" -> eurAmount * 1_200;
            default -> eurAmount;
        };
    }

    @Tool("Calculate a fictional workshop rental estimate for the vehicle category and rental days. Amounts use the traveler's selected currency.")
    @ToolInputGuardrails(RentalEstimateInputGuardrail.class)
    public RentalEstimate estimateRental(String category, int days) {
        String currency = normalize(ACTIVE.get());
        int dailyRate = convertFromEur(DAILY_RATES.get(category), currency);
        int rentalTotal = dailyRate * days;
        LOG.infof("Rental calculation executed: category=%s, days=%d, currency=%s, total=%d",
                category, days, currency, rentalTotal);
        return new RentalEstimate(category, days, currency, dailyRate, rentalTotal);
    }

    public record RentalEstimate(String category, int days, String currency, int dailyRate, int rentalTotal) {}
}
