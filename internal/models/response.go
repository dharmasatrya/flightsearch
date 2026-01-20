package models

type SearchMetadata struct {
	TotalResults       int      `json:"total_results"`
	ProvidersQueried   int      `json:"providers_queried"`
	ProvidersSucceeded int      `json:"providers_succeeded"`
	ProvidersFailed    int      `json:"providers_failed"`
	FailedProviders    []string `json:"failed_providers,omitempty"`
	SearchTimeMs       int64    `json:"search_time_ms"`
	CacheHit           bool     `json:"cache_hit"`
}

type SearchCriteria struct {
	Origin        string         `json:"origin"`
	Destination   string         `json:"destination"`
	DepartureDate string         `json:"departure_date"`
	ReturnDate    *string        `json:"return_date,omitempty"`
	Passengers    int            `json:"passengers"`
	CabinClass    string         `json:"cabin_class"`
	Filters       *SearchFilters `json:"filters,omitempty"`
	SortBy        string         `json:"sort_by"`
	SortOrder     string         `json:"sort_order"`
}

type SearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       SearchMetadata `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}

type RoundTripResponse struct {
	SearchCriteria  SearchCriteria `json:"search_criteria"`
	Metadata        SearchMetadata `json:"metadata"`
	OutboundFlights []Flight       `json:"outbound_flights"`
	ReturnFlights   []Flight       `json:"return_flights"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
