package providers

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/dharmasatrya/flightsearch/internal/models"
	"github.com/dharmasatrya/flightsearch/internal/providers/data"
	"github.com/dharmasatrya/flightsearch/internal/timezone"
	"github.com/dharmasatrya/flightsearch/pkg/currency"
)

type garudaResponse struct {
	Flights []garudaFlight `json:"flights"`
}

type garudaFlight struct {
	FlightID     string          `json:"flight_id"`
	Airline      garudaAirline   `json:"airline"`
	FlightNumber string          `json:"flight_number"`
	Departure    garudaLocation  `json:"departure"`
	Arrival      garudaLocation  `json:"arrival"`
	Duration     int             `json:"duration_minutes"`
	Stops        int             `json:"stops"`
	Layovers     []garudaLayover `json:"layovers,omitempty"`
	Price        garudaPrice     `json:"price"`
	Seats        int             `json:"available_seats"`
	CabinClass   string          `json:"cabin_class"`
	Aircraft     string          `json:"aircraft"`
	Amenities    []string        `json:"amenities"`
	Baggage      garudaBaggage   `json:"baggage"`
}

type garudaAirline struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type garudaLocation struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Terminal string `json:"terminal"`
	Time     string `json:"time"`
}

type garudaLayover struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Duration int    `json:"duration"`
}

type garudaPrice struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type garudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type GarudaProvider struct {
	flights []garudaFlight
}

func NewGarudaProvider() (*GarudaProvider, error) {
	var resp garudaResponse
	if err := json.Unmarshal(data.GarudaData, &resp); err != nil {
		return nil, err
	}
	return &GarudaProvider{flights: resp.Flights}, nil
}

func (p *GarudaProvider) Name() string {
	return "garuda"
}

func (p *GarudaProvider) Search(ctx context.Context, req models.SearchRequest) ([]models.Flight, error) {
	delay := time.Duration(50+rand.Intn(50)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var results []models.Flight
	for _, f := range p.flights {
		if !strings.EqualFold(f.Departure.Airport, req.Origin) ||
			!strings.EqualFold(f.Arrival.Airport, req.Destination) {
			continue
		}

		if !strings.EqualFold(f.CabinClass, req.CabinClass) {
			continue
		}

		depTime, err := timezone.ParseTimeWithOffset(f.Departure.Time, "")
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

func (p *GarudaProvider) normalize(f garudaFlight) (models.Flight, error) {
	depTime, err := timezone.ParseTimeWithOffset(f.Departure.Time, "")
	if err != nil {
		return models.Flight{}, err
	}

	arrTime, err := timezone.ParseTimeWithOffset(f.Arrival.Time, "")
	if err != nil {
		return models.Flight{}, err
	}

	depTime = timezone.ConvertToTimezone(depTime, f.Departure.Airport)
	arrTime = timezone.ConvertToTimezone(arrTime, f.Arrival.Airport)

	layovers := make([]models.Layover, len(f.Layovers))
	for i, l := range f.Layovers {
		layovers[i] = models.Layover{
			Airport:  l.Airport,
			City:     l.City,
			Duration: l.Duration,
		}
	}

	hours := f.Duration / 60
	mins := f.Duration % 60

	var depTerminal, arrTerminal *string
	if f.Departure.Terminal != "" {
		t := f.Departure.Terminal
		depTerminal = &t
	}
	if f.Arrival.Terminal != "" {
		t := f.Arrival.Terminal
		arrTerminal = &t
	}

	var aircraft *string
	if f.Aircraft != "" {
		a := f.Aircraft
		aircraft = &a
	}

	return models.Flight{
		ID:       f.FlightID,
		Provider: p.Name(),
		Airline: models.Airline{
			Code: f.Airline.Code,
			Name: f.Airline.Name,
		},
		FlightNumber: f.FlightNumber,
		Departure: models.Location{
			Airport:  f.Departure.Airport,
			City:     f.Departure.City,
			Terminal: depTerminal,
			Time:     depTime,
			Timezone: timezone.GetTimezoneByAirport(f.Departure.Airport),
		},
		Arrival: models.Location{
			Airport:  f.Arrival.Airport,
			City:     f.Arrival.City,
			Terminal: arrTerminal,
			Time:     arrTime,
			Timezone: timezone.GetTimezoneByAirport(f.Arrival.Airport),
		},
		Duration: models.Duration{
			Hours:        hours,
			Minutes:      mins,
			TotalMinutes: f.Duration,
		},
		Stops:    f.Stops,
		Layovers: layovers,
		Price: models.Price{
			Amount:    f.Price.Amount,
			Currency:  f.Price.Currency,
			Formatted: currency.FormatIDR(f.Price.Amount),
		},
		AvailableSeats: f.Seats,
		CabinClass:     f.CabinClass,
		Aircraft:       aircraft,
		Amenities:      f.Amenities,
		Baggage: models.Baggage{
			CabinKg:   float64(f.Baggage.CarryOn),
			CheckedKg: float64(f.Baggage.Checked),
		},
	}, nil
}
