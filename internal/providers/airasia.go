package providers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
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

var ErrAirAsiaTemporaryFailure = errors.New("temporary service unavailable")

type airasiaResponse struct {
	FlightOffers []airasiaFlight `json:"flight_offers"`
}

type airasiaFlight struct {
	OfferID          string          `json:"offer_id"`
	MarketingCarrier airasiaCarrier  `json:"marketing_carrier"`
	FlightNum        string          `json:"flight_num"`
	From             airasiaLocation `json:"from"`
	To               airasiaLocation `json:"to"`
	DepartAt         string          `json:"depart_at"`
	ArriveAt         string          `json:"arrive_at"`
	DurationHours    float64         `json:"duration_hours"`
	DirectFlight     bool            `json:"direct_flight"`
	Stops            []airasiaStop   `json:"stops"`
	PriceIDR         float64         `json:"price_idr"`
	SeatsLeft        int             `json:"seats_left"`
	TravelClass      string          `json:"travel_class"`
	Equipment        string          `json:"equipment"`
	Perks            []string        `json:"perks"`
	BaggageInfo      string          `json:"baggage_info"`
}

type airasiaCarrier struct {
	AirlineCode string `json:"airline_code"`
	AirlineName string `json:"airline_name"`
}

type airasiaLocation struct {
	IATA     string `json:"iata"`
	CityName string `json:"city_name"`
}

type airasiaStop struct {
	StopAirport     string `json:"stop_airport"`
	StopCity        string `json:"stop_city"`
	StopDurationMin int    `json:"stop_duration_mins"`
}

type AirAsiaProvider struct {
	flights []airasiaFlight
}

func NewAirAsiaProvider() (*AirAsiaProvider, error) {
	var resp airasiaResponse
	if err := json.Unmarshal(data.AirAsiaData, &resp); err != nil {
		return nil, err
	}
	return &AirAsiaProvider{flights: resp.FlightOffers}, nil
}

func (p *AirAsiaProvider) Name() string {
	return "airasia"
}

func (p *AirAsiaProvider) Search(ctx context.Context, req models.SearchRequest) ([]models.Flight, error) {
	delay := time.Duration(50+rand.Intn(100)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if rand.Float64() < 0.1 {
		return nil, ErrAirAsiaTemporaryFailure
	}

	var results []models.Flight
	for _, f := range p.flights {
		if !strings.EqualFold(f.From.IATA, req.Origin) ||
			!strings.EqualFold(f.To.IATA, req.Destination) {
			continue
		}

		if !strings.EqualFold(f.TravelClass, req.CabinClass) {
			continue
		}

		depTime, err := timezone.ParseTimeWithOffset(f.DepartAt, "")
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

func (p *AirAsiaProvider) normalize(f airasiaFlight) (models.Flight, error) {
	depTime, err := timezone.ParseTimeWithOffset(f.DepartAt, "")
	if err != nil {
		return models.Flight{}, err
	}

	arrTime, err := timezone.ParseTimeWithOffset(f.ArriveAt, "")
	if err != nil {
		return models.Flight{}, err
	}

	depTime = timezone.ConvertToTimezone(depTime, f.From.IATA)
	arrTime = timezone.ConvertToTimezone(arrTime, f.To.IATA)

	totalMinutes := int(math.Round(f.DurationHours * 60))
	hours := totalMinutes / 60
	mins := totalMinutes % 60

	stops := len(f.Stops)
	if f.DirectFlight {
		stops = 0
	}

	layovers := make([]models.Layover, len(f.Stops))
	for i, s := range f.Stops {
		layovers[i] = models.Layover{
			Airport:  s.StopAirport,
			City:     s.StopCity,
			Duration: s.StopDurationMin,
		}
	}

	cabinKg := parseAirAsiaBaggage(f.BaggageInfo)

	var aircraft *string
	if f.Equipment != "" {
		a := f.Equipment
		aircraft = &a
	}

	return models.Flight{
		ID:       f.OfferID,
		Provider: p.Name(),
		Airline: models.Airline{
			Code: f.MarketingCarrier.AirlineCode,
			Name: f.MarketingCarrier.AirlineName,
		},
		FlightNumber: f.FlightNum,
		Departure: models.Location{
			Airport:  f.From.IATA,
			City:     f.From.CityName,
			Terminal: nil,
			Time:     depTime,
			Timezone: timezone.GetTimezoneByAirport(f.From.IATA),
		},
		Arrival: models.Location{
			Airport:  f.To.IATA,
			City:     f.To.CityName,
			Terminal: nil,
			Time:     arrTime,
			Timezone: timezone.GetTimezoneByAirport(f.To.IATA),
		},
		Duration: models.Duration{
			Hours:        hours,
			Minutes:      mins,
			TotalMinutes: totalMinutes,
		},
		Stops:    stops,
		Layovers: layovers,
		Price: models.Price{
			Amount:    f.PriceIDR,
			Currency:  "IDR",
			Formatted: currency.FormatIDR(f.PriceIDR),
		},
		AvailableSeats: f.SeatsLeft,
		CabinClass:     f.TravelClass,
		Aircraft:       aircraft,
		Amenities:      f.Perks,
		Baggage: models.Baggage{
			CabinKg:   cabinKg,
			CheckedKg: 0,
		},
	}, nil
}

func parseAirAsiaBaggage(s string) float64 {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*kg`)
	matches := re.FindStringSubmatch(strings.ToLower(s))
	if len(matches) >= 2 {
		if v, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return v
		}
	}
	return 7
}
