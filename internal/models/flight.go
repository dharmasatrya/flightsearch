package models

import "time"

type Airline struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Location struct {
	Airport  string    `json:"airport"`
	City     string    `json:"city"`
	Terminal *string   `json:"terminal,omitempty"`
	Time     time.Time `json:"time"`
	Timezone string    `json:"timezone"`
}

type Duration struct {
	Hours        int `json:"hours"`
	Minutes      int `json:"minutes"`
	TotalMinutes int `json:"total_minutes"`
}

type Layover struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Duration int    `json:"duration_minutes"`
}

type Price struct {
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Formatted string  `json:"formatted"`
}

type Baggage struct {
	CabinKg   float64 `json:"cabin_kg"`
	CheckedKg float64 `json:"checked_kg"`
}

type Flight struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	Airline        Airline   `json:"airline"`
	FlightNumber   string    `json:"flight_number"`
	Departure      Location  `json:"departure"`
	Arrival        Location  `json:"arrival"`
	Duration       Duration  `json:"duration"`
	Stops          int       `json:"stops"`
	Layovers       []Layover `json:"layovers,omitempty"`
	Price          Price     `json:"price"`
	AvailableSeats int       `json:"available_seats"`
	CabinClass     string    `json:"cabin_class"`
	Aircraft       *string   `json:"aircraft,omitempty"`
	Amenities      []string  `json:"amenities,omitempty"`
	Baggage        Baggage   `json:"baggage"`
	BestValueScore float64   `json:"best_value_score,omitempty"`
}
