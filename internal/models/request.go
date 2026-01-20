package models

type SearchFilters struct {
	PriceMin         *float64 `json:"price_min,omitempty"`
	PriceMax         *float64 `json:"price_max,omitempty"`
	MaxStops         *int     `json:"max_stops,omitempty"`
	Airlines         []string `json:"airlines,omitempty"`
	DepartureTimeMin *string  `json:"departure_time_min,omitempty"`
	DepartureTimeMax *string  `json:"departure_time_max,omitempty"`
	ArrivalTimeMin   *string  `json:"arrival_time_min,omitempty"`
	ArrivalTimeMax   *string  `json:"arrival_time_max,omitempty"`
	MaxDuration      *int     `json:"max_duration,omitempty"`
}

type SearchRequest struct {
	Origin        string         `json:"origin"`
	Destination   string         `json:"destination"`
	DepartureDate string         `json:"departure_date"`
	ReturnDate    *string        `json:"return_date,omitempty"`
	Passengers    int            `json:"passengers"`
	CabinClass    string         `json:"cabin_class"`
	Filters       *SearchFilters `json:"filters,omitempty"`
	SortBy        string         `json:"sort_by,omitempty"`
	SortOrder     string         `json:"sort_order,omitempty"`
}

func (r *SearchRequest) Validate() error {
	if r.Origin == "" {
		return ErrMissingOrigin
	}
	if r.Destination == "" {
		return ErrMissingDestination
	}
	if r.DepartureDate == "" {
		return ErrMissingDepartureDate
	}
	if r.Passengers <= 0 {
		r.Passengers = 1
	}
	if r.CabinClass == "" {
		r.CabinClass = "economy"
	}
	if r.SortBy == "" {
		r.SortBy = "best_value"
	}
	if r.SortOrder == "" {
		r.SortOrder = "asc"
	}
	return nil
}

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

const (
	ErrMissingOrigin        ValidationError = "origin is required"
	ErrMissingDestination   ValidationError = "destination is required"
	ErrMissingDepartureDate ValidationError = "departure_date is required"
)
