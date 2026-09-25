package model

type TripRequest struct {
	Destination string `json:"destination"`
	StartDate   string `json:"startDate"`
	Days        int    `json:"days"`
	TripType    string `json:"tripType"`
	Travelers   int    `json:"travelers"`
	Budget      string `json:"budget"`
	Preferences string `json:"preferences"`
}

type VehicleRecommendation struct {
	Type      string `json:"type"`
	Model     string `json:"model"`
	Reasoning string `json:"reasoning"`
}

type DayItinerary struct {
	Day           int    `json:"day"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	OvernightStop string `json:"overnightStop"`
}

type CostEstimate struct {
	VehiclePerDay string `json:"vehiclePerDay"`
	Fuel          string `json:"fuel"`
	Tolls         string `json:"tolls"`
	Accommodation string `json:"accommodation"`
	Food          string `json:"food"`
	Activities    string `json:"activities"`
	Total         string `json:"total"`
}

// TripPlan matches the Quarkus LangChain4j JSON shape so the same UI can render either app.
type TripPlan struct {
	Vehicle       VehicleRecommendation `json:"vehicle"`
	RouteOverview string                `json:"routeOverview"`
	Itinerary     []DayItinerary        `json:"itinerary"`
	Costs         CostEstimate          `json:"costs"`
}

type BookingConfirmation struct {
	BookingReference string `json:"bookingReference"`
	Message          string `json:"message"`
}

type TripApproval struct {
	InstanceID string `json:"instanceId"`
	Status     string `json:"status"` // approved | rejected
}

type TripPlanStatus struct {
	RequestID    string               `json:"requestId"`
	InstanceID   string               `json:"instanceId"`
	Request      TripRequest          `json:"request"`
	Status       string               `json:"status"`
	Plan         *TripPlan            `json:"plan"`
	Confirmation *BookingConfirmation `json:"confirmation"`
	Error        string               `json:"error"`
	Message      string               `json:"message"`
}

type TripError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
