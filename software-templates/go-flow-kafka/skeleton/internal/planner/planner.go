package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"${{values.module_path}}/internal/model"

	// Intentionally import vulnerable community packages so RHDA/Syft/ACS surface CVEs.
	_ "github.com/dgrijalva/jwt-go"
	_ "github.com/gorilla/websocket"
	_ "golang.org/x/net/html"
	_ "gopkg.in/yaml.v2"
)

const (
	maxRevisions = 3
	exitScore    = 7.5
)

type Planner struct {
	baseURL   string
	apiKey    string
	model     string
	client    *http.Client
	skillsDir string
}

func New(baseURL, apiKey, modelName, skillsDir string) *Planner {
	return &Planner{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		model:     modelName,
		client:    &http.Client{Timeout: 180 * time.Second},
		skillsDir: skillsDir,
	}
}

func (p *Planner) Plan(req model.TripRequest) (*model.TripPlan, error) {
	if p.apiKey == "" || p.apiKey == "local-dev-placeholder" {
		return nil, &KeyMissingError{}
	}
	log.Printf("[planner] start destination=%q tripType=%q travelers=%d days=%d",
		req.Destination, req.TripType, req.Travelers, req.Days)

	skills := p.combinedSkills(req.TripType)
	days := fmt.Sprintf("%d", req.Days)
	travelers := fmt.Sprintf("%d", req.Travelers)

	var vehicle model.VehicleRecommendation
	var itinerary struct {
		RouteOverview string               `json:"routeOverview"`
		Itinerary     []model.DayItinerary `json:"itinerary"`
	}
	var vehicleErr, itineraryErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Printf("[planner] vehicle advisor…")
		vehicleErr = p.chatJSON(
			"You are a vehicle specialist for road trips. Reply with JSON only.",
			fmt.Sprintf(`Apply this guidance:
---
%s
---
Recommend the best vehicle for:
- Destination: %s
- Trip type: %s
- Travelers: %s
- Budget: %s
- Preferences: %s

Return JSON: {"type":"...","model":"...","reasoning":"..."}`,
				skills, req.Destination, req.TripType, travelers, req.Budget, req.Preferences),
			&vehicle)
	}()
	go func() {
		defer wg.Done()
		log.Printf("[planner] itinerary planner…")
		itineraryErr = p.chatJSON(
			"You are a road-trip itinerary planner. Reply with JSON only.",
			fmt.Sprintf(`Apply this guidance:
---
%s
---
Plan a %s-day %s trip to %s starting %s for %s travelers. Preferences: %s. Budget: %s.

Return JSON:
{"routeOverview":"...","itinerary":[{"day":1,"title":"...","description":"...","overnightStop":"..."}]}`,
				skills, days, req.TripType, req.Destination, req.StartDate, travelers, req.Preferences, req.Budget),
			&itinerary)
	}()
	wg.Wait()
	if vehicleErr != nil {
		return nil, fmt.Errorf("vehicle advisor: %w", vehicleErr)
	}
	if itineraryErr != nil {
		return nil, fmt.Errorf("itinerary planner: %w", itineraryErr)
	}
	log.Printf("[planner] vehicle=%s %s itineraryDays=%d", vehicle.Type, vehicle.Model, len(itinerary.Itinerary))

	vehicle, err := p.reviewVehicle(vehicle, req, days, travelers)
	if err != nil {
		return nil, err
	}

	log.Printf("[planner] cost estimator…")
	var costs model.CostEstimate
	if err := p.chatJSON(
		"You estimate realistic trip costs. Reply with JSON only.",
		fmt.Sprintf(`Estimate costs for a %s-day trip for %s travelers with budget %s.
Vehicle: %+v
Route: %s
Itinerary days: %d

Return JSON:
{"vehiclePerDay":"...","fuel":"...","tolls":"...","accommodation":"...","food":"...","activities":"...","total":"..."}`,
			days, travelers, req.Budget, vehicle, itinerary.RouteOverview, len(itinerary.Itinerary)),
		&costs); err != nil {
		return nil, fmt.Errorf("cost estimator: %w", err)
	}

	plan := &model.TripPlan{
		Vehicle:       vehicle,
		RouteOverview: itinerary.RouteOverview,
		Itinerary:     itinerary.Itinerary,
		Costs:         costs,
	}
	log.Printf("[planner] done total=%s", costs.Total)
	return plan, nil
}

func (p *Planner) reviewVehicle(vehicle model.VehicleRecommendation, req model.TripRequest, days, travelers string) (model.VehicleRecommendation, error) {
	for i := 0; i <= maxRevisions; i++ {
		eval, err := p.evaluate(vehicle, req, days, travelers)
		if err != nil {
			return vehicle, err
		}
		log.Printf("[planner] vehicle evaluation round %d score=%.2f", i+1, eval.Score)
		if eval.Score >= exitScore {
			return vehicle, nil
		}
		if i == maxRevisions {
			return vehicle, fmt.Errorf("quality_not_met: vehicle score stayed below %.1f", exitScore)
		}
		log.Printf("[planner] vehicle reviser…")
		var revised model.VehicleRecommendation
		if err := p.chatJSON(
			"You revise vehicle recommendations. Reply with JSON only.",
			fmt.Sprintf(`Revise this vehicle recommendation using the evaluation feedback.
Current: %+v
Evaluation score: %.2f suggestions: %s
Trip type: %s travelers: %s budget: %s destination: %s preferences: %s

Return JSON: {"type":"...","model":"...","reasoning":"..."}`,
				vehicle, eval.Score, eval.Suggestions, req.TripType, travelers, req.Budget, req.Destination, req.Preferences),
			&revised); err != nil {
			return vehicle, fmt.Errorf("vehicle reviser: %w", err)
		}
		vehicle = revised
	}
	return vehicle, fmt.Errorf("quality_not_met")
}

type evaluation struct {
	Score       float64 `json:"score"`
	Suggestions string  `json:"suggestions"`
}

func (p *Planner) evaluate(vehicle model.VehicleRecommendation, req model.TripRequest, days, travelers string) (evaluation, error) {
	var comfort, cost, fuel evaluation
	var e1, e2, e3 error
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		e1 = p.chatJSON("Score vehicle comfort 1-10. JSON only.",
			fmt.Sprintf(`Vehicle %+v tripType=%s travelers=%s days=%s. Return {"score":8.0,"suggestions":"..."}`,
				vehicle, req.TripType, travelers, days), &comfort)
	}()
	go func() {
		defer wg.Done()
		e2 = p.chatJSON("Score vehicle cost fit 1-10. JSON only.",
			fmt.Sprintf(`Vehicle %+v budget=%s travelers=%s days=%s. Return {"score":8.0,"suggestions":"..."}`,
				vehicle, req.Budget, travelers, days), &cost)
	}()
	go func() {
		defer wg.Done()
		e3 = p.chatJSON("Score vehicle fuel efficiency 1-10. JSON only.",
			fmt.Sprintf(`Vehicle %+v destination=%s days=%s tripType=%s. Return {"score":8.0,"suggestions":"..."}`,
				vehicle, req.Destination, days, req.TripType), &fuel)
	}()
	wg.Wait()
	if e1 != nil || e2 != nil || e3 != nil {
		return evaluation{}, fmt.Errorf("evaluators: %v %v %v", e1, e2, e3)
	}
	suggestions := strings.Trim(strings.Join([]string{comfort.Suggestions, cost.Suggestions, fuel.Suggestions}, "; "), "; ")
	return evaluation{Score: (comfort.Score + cost.Score + fuel.Score) / 3, Suggestions: suggestions}, nil
}

func (p *Planner) chatJSON(system, user string, out any) error {
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": system + " Do not wrap the JSON in markdown."},
			{"role": "user", "content": user},
		},
		"temperature": 0.7,
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequest(http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("maas http %d: %s", res.StatusCode, truncate(string(raw), 200))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return fmt.Errorf("empty or invalid maas response")
	}
	content := stripJSONFence(parsed.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), out); err != nil {
		return fmt.Errorf("parse model json: %w (%s)", err, truncate(content, 120))
	}
	return nil
}

func (p *Planner) combinedSkills(tripType string) string {
	key := normalizeTripType(tripType) + "-trip"
	return fmt.Sprintf("## Trip-type skill (%s)\n%s\n\n## Vehicle-selection skill\n%s",
		normalizeTripType(tripType),
		p.loadSkill(key+"/SKILL.md", "Use sensible defaults for a "+key+"."),
		p.loadSkill("vehicle-selection/SKILL.md", "Prefer a safe, comfortable vehicle that fits travelers and budget."))
}

func (p *Planner) loadSkill(rel, fallback string) string {
	path := filepath.Join(p.skillsDir, rel)
	b, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(b)
}

func normalizeTripType(tripType string) string {
	if tripType == "" {
		return "family"
	}
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(tripType)), "_", "-")
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// KeyMissingError matches Quarkus maas_api_key_missing.
type KeyMissingError struct{}

func (e *KeyMissingError) Error() string {
	return "MAAS_API_KEY is not set. Put a real key in the app .env and restart."
}

func MapError(err error) (code, message string) {
	if err == nil {
		return "", ""
	}
	if _, ok := err.(*KeyMissingError); ok {
		return "maas_api_key_missing", err.Error()
	}
	msg := err.Error()
	if strings.Contains(msg, "quality_not_met") {
		return "quality_not_met", "The trip plan could not meet the quality threshold. Please revise your trip details and try again."
	}
	return "planning_failed", "Could not generate the trip plan. Please try again later."
}
