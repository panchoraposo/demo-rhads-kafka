package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"${{values.module_path}}/internal/model"

	// Intentionally import vulnerable community packages so RHDA/Syft/ACS surface CVEs.
	_ "github.com/dgrijalva/jwt-go"
	_ "github.com/gorilla/websocket"
	_ "gopkg.in/yaml.v2"
	_ "golang.org/x/net/html"
)

type Planner struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func New(baseURL, apiKey, modelName string) *Planner {
	return &Planner{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   modelName,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Planner) Plan(req model.TripRequest) (*model.TripPlan, error) {
	if p.apiKey == "" {
		return stubPlan(req), nil
	}
	prompt := fmt.Sprintf(
		"Plan a %s trip to %s for %d travelers over %d days starting %s. Budget %.0f. Preferences: %s. Reply with a short summary, 3 itinerary bullets, a vehicle suggestion, and a cost estimate.",
		req.TripType, req.Destination, req.Travelers, req.Days, req.StartDate, req.Budget, req.Preferences,
	)
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a concise trip planner. Prefer bullet-friendly text."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	}
	b, _ := json.Marshal(body)
	httpReq, err := http.NewRequest(http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return stubPlan(req), nil
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(httpReq)
	if err != nil {
		return stubPlan(req), nil
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return stubPlan(req), nil
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return stubPlan(req), nil
	}
	content := parsed.Choices[0].Message.Content
	return &model.TripPlan{
		Summary:   content,
		Itinerary: []string{"See summary for day-by-day plan"},
		Vehicle:   "Suggested in summary",
		Estimate:  fmt.Sprintf("Within budget %.0f (model estimate)", req.Budget),
	}, nil
}

func stubPlan(req model.TripRequest) *model.TripPlan {
	return &model.TripPlan{
		Summary:   fmt.Sprintf("Demo plan for %s (%s, %d days, %d travelers).", req.Destination, req.TripType, req.Days, req.Travelers),
		Itinerary: []string{"Day 1: arrive and settle", "Day 2: highlights", "Day 3: return prep"},
		Vehicle:   "Mid-size SUV",
		Estimate:  fmt.Sprintf("~%.0f total (stub — set MAAS_API_KEY for live model)", req.Budget*0.8),
	}
}
