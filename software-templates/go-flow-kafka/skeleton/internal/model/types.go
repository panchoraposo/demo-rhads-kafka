package model

type TripRequest struct {
	Destination string  `json:"destination"`
	StartDate   string  `json:"startDate"`
	Days        int     `json:"days"`
	TripType    string  `json:"tripType"`
	Travelers   int     `json:"travelers"`
	Budget      float64 `json:"budget"`
	Preferences string  `json:"preferences"`
}

type TripPlan struct {
	Summary   string   `json:"summary"`
	Itinerary []string `json:"itinerary"`
	Vehicle   string   `json:"vehicle"`
	Estimate  string   `json:"estimate"`
}

type BookingConfirmation struct {
	Reference string `json:"reference"`
	Message   string `json:"message"`
}

type TripApproval struct {
	InstanceID string `json:"instanceId"`
	Status     string `json:"status"` // approved | rejected
}

type TripPlanStatus struct {
	RequestID    string                `json:"requestId"`
	InstanceID   string                `json:"instanceId"`
	Request      TripRequest           `json:"request"`
	Status       string                `json:"status"`
	Plan         *TripPlan             `json:"plan,omitempty"`
	Confirmation *BookingConfirmation  `json:"confirmation,omitempty"`
	Error        string                `json:"error,omitempty"`
	Message      string                `json:"message,omitempty"`
}
