package providers

import (
	"context"
	"encoding/json"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dharmasatrya/flightsearch/internal/models"
	"github.com/dharmasatrya/flightsearch/internal/providers/data"
	"github.com/dharmasatrya/flightsearch/internal/timezone"
	"github.com/dharmasatrya/flightsearch/pkg/currency"
)

type lionResponse struct {
	Results []lionFlight `json:"results"`
}

type lionFlight struct {
	ID          string         `json:"id"`
	Carrier     lionCarrier    `json:"carrier"`
	FlightCode  string         `json:"flight_code"`
	Origin      lionAirport    `json:"origin"`
	Destination lionAirport    `json:"destination"`
	Schedule    lionSchedule   `json:"schedule"`
	FlightTime  int            `json:"flight_time"`
	IsDirect    bool           `json:"is_direct"`
	StopCount   int            `json:"stop_count"`
	Stopovers   []lionStopover `json:"stopovers,omitempty"`
	Pricing     lionPricing    `json:"pricing"`
	Seats       int            `json:"seats_remaining"`
	Class       string         `json:"class"`
	PlaneType   string         `json:"plane_type"`
	Services    []string       `json:"services"`
	Baggage     lionBaggage    `json:"baggage"`
}

type lionCarrier struct {
	IATA     string `json:"iata"`
	FullName string `json:"full_name"`
}

type lionAirport struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Gate string `json:"gate"`
}

type lionSchedule struct {
	Departure string `json:"departure"`
	Arrival   string `json:"arrival"`
	Timezone  string `json:"timezone"`
}

type lionStopover struct {
	AirportCode string `json:"airport_code"`
	CityName    string `json:"city_name"`
	WaitTime    int    `json:"wait_time"`
}

type lionPricing struct {
	Total    float64 `json:"total"`
	Currency string  `json:"currency_code"`
}

type lionBaggage struct {
	Cabin string `json:"cabin"`
	Hold  string `json:"hold"`
}

type LionAirProvider struct {
	flights []lionFlight
}

func NewLionAirProvider() (*LionAirProvider, error) {
	var resp lionResponse
	if err := json.Unmarshal(data.LionAirData, &resp); err != nil {
		return nil, err
	}
	return &LionAirProvider{flights: resp.Results}, nil
}

func (p *LionAirProvider) Name() string {
	return "lionair"
}

func (p *LionAirProvider) Search(ctx context.Context, req models.SearchRequest) ([]models.Flight, error) {
	delay := time.Duration(100+rand.Intn(100)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var results []models.Flight
	for _, f := range p.flights {
		if !strings.EqualFold(f.Origin.Code, req.Origin) ||
			!strings.EqualFold(f.Destination.Code, req.Destination) {
			continue
		}

		if !strings.EqualFold(f.Class, req.CabinClass) {
			continue
		}

		depTime, err := timezone.ParseTimeWithOffset(f.Schedule.Departure, f.Schedule.Timezone)
		if err != nil {
			continue
		}

		reqDate, err := time.Parse("2006-01-02", req.DepartureDate)
		if err != nil {
			continue
		}
		if depTime.Year() != reqDate.Year() || depTime.Month() != reqDate.Month() || depTime.Day() != reqDate.Day() {
			continue
		}

		flight, err := p.normalize(f)
		if err != nil {
			continue
		}
		results = append(results, flight)
	}

	return results, nil
}

func (p *LionAirProvider) normalize(f lionFlight) (models.Flight, error) {
	depTime, err := timezone.ParseTimeWithOffset(f.Schedule.Departure, f.Schedule.Timezone)
	if err != nil {
		return models.Flight{}, err
	}

	arrTime, err := timezone.ParseTimeWithOffset(f.Schedule.Arrival, f.Schedule.Timezone)
	if err != nil {
		return models.Flight{}, err
	}

	arrTime = timezone.ConvertToTimezone(arrTime, f.Destination.Code)

	stops := f.StopCount
	if f.IsDirect {
		stops = 0
	}

	layovers := make([]models.Layover, len(f.Stopovers))
	for i, s := range f.Stopovers {
		layovers[i] = models.Layover{
			Airport:  s.AirportCode,
			City:     s.CityName,
			Duration: s.WaitTime,
		}
	}

	hours := f.FlightTime / 60
	mins := f.FlightTime % 60

	cabinKg := parseBaggageWeight(f.Baggage.Cabin)
	checkedKg := parseBaggageWeight(f.Baggage.Hold)

	var depTerminal, arrTerminal *string
	if f.Origin.Gate != "" {
		t := f.Origin.Gate
		depTerminal = &t
	}
	if f.Destination.Gate != "" {
		t := f.Destination.Gate
		arrTerminal = &t
	}

	var aircraft *string
	if f.PlaneType != "" {
		a := f.PlaneType
		aircraft = &a
	}

	return models.Flight{
		ID:       f.ID,
		Provider: p.Name(),
		Airline: models.Airline{
			Code: f.Carrier.IATA,
			Name: f.Carrier.FullName,
		},
		FlightNumber: f.FlightCode,
		Departure: models.Location{
			Airport:  f.Origin.Code,
			City:     f.Origin.Name,
			Terminal: depTerminal,
			Time:     depTime,
			Timezone: timezone.GetTimezoneByAirport(f.Origin.Code),
		},
		Arrival: models.Location{
			Airport:  f.Destination.Code,
			City:     f.Destination.Name,
			Terminal: arrTerminal,
			Time:     arrTime,
			Timezone: timezone.GetTimezoneByAirport(f.Destination.Code),
		},
		Duration: models.Duration{
			Hours:        hours,
			Minutes:      mins,
			TotalMinutes: f.FlightTime,
		},
		Stops:    stops,
		Layovers: layovers,
		Price: models.Price{
			Amount:    f.Pricing.Total,
			Currency:  f.Pricing.Currency,
			Formatted: currency.FormatIDR(f.Pricing.Total),
		},
		AvailableSeats: f.Seats,
		CabinClass:     f.Class,
		Aircraft:       aircraft,
		Amenities:      f.Services,
		Baggage: models.Baggage{
			CabinKg:   cabinKg,
			CheckedKg: checkedKg,
		},
	}, nil
}

func parseBaggageWeight(s string) float64 {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*kg`)
	matches := re.FindStringSubmatch(strings.ToLower(s))
	if len(matches) >= 2 {
		if v, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return v
		}
	}
	return 0
}
