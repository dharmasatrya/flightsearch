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

type batikResponse struct {
	Data struct {
		AvailableFlights []batikFlight `json:"availableFlights"`
	} `json:"data"`
}

type batikFlight struct {
	FlightID         string            `json:"flightId"`
	OperatingCarrier batikCarrier      `json:"operatingCarrier"`
	FlightNo         string            `json:"flightNo"`
	DepartureInfo    batikLocationInfo `json:"departureInfo"`
	ArrivalInfo      batikArrivalInfo  `json:"arrivalInfo"`
	TravelTime       string            `json:"travelTime"`
	NumberOfStops    int               `json:"numberOfStops"`
	ConnectionPoints []batikConnection `json:"connectionPoints,omitempty"`
	Fare             batikFare         `json:"fare"`
	SeatsAvailable   int               `json:"seatsAvailable"`
	CabinType        string            `json:"cabinType"`
	AircraftType     string            `json:"aircraftType"`
	IncludedServices []string          `json:"includedServices"`
	BaggageAllowance string            `json:"baggageAllowance"`
}

type batikCarrier struct {
	CarrierCode string `json:"carrierCode"`
	CarrierName string `json:"carrierName"`
}

type batikLocationInfo struct {
	AirportCode   string `json:"airportCode"`
	CityName      string `json:"cityName"`
	TerminalNo    string `json:"terminalNo"`
	DepartureTime string `json:"departureTime"`
}

type batikArrivalInfo struct {
	AirportCode string `json:"airportCode"`
	CityName    string `json:"cityName"`
	TerminalNo  string `json:"terminalNo"`
	ArrivalTime string `json:"arrivalTime"`
}

type batikConnection struct {
	Airport        string `json:"airport"`
	City           string `json:"city"`
	LayoverMinutes int    `json:"layoverMinutes"`
}

type batikFare struct {
	TotalPrice   float64 `json:"totalPrice"`
	CurrencyCode string  `json:"currencyCode"`
}

type BatikAirProvider struct {
	flights []batikFlight
}

func NewBatikAirProvider() (*BatikAirProvider, error) {
	var resp batikResponse
	if err := json.Unmarshal(data.BatikAirData, &resp); err != nil {
		return nil, err
	}
	return &BatikAirProvider{flights: resp.Data.AvailableFlights}, nil
}

func (p *BatikAirProvider) Name() string {
	return "batikair"
}

func (p *BatikAirProvider) Search(ctx context.Context, req models.SearchRequest) ([]models.Flight, error) {
	delay := time.Duration(200+rand.Intn(200)) * time.Millisecond
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var results []models.Flight
	for _, f := range p.flights {
		if !strings.EqualFold(f.DepartureInfo.AirportCode, req.Origin) ||
			!strings.EqualFold(f.ArrivalInfo.AirportCode, req.Destination) {
			continue
		}

		if !strings.EqualFold(f.CabinType, req.CabinClass) {
			continue
		}

		depTime, err := timezone.ParseTimeWithOffset(f.DepartureInfo.DepartureTime, "")
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

func (p *BatikAirProvider) normalize(f batikFlight) (models.Flight, error) {
	depTime, err := timezone.ParseTimeWithOffset(f.DepartureInfo.DepartureTime, "")
	if err != nil {
		return models.Flight{}, err
	}

	arrTime, err := timezone.ParseTimeWithOffset(f.ArrivalInfo.ArrivalTime, "")
	if err != nil {
		return models.Flight{}, err
	}

	depTime = timezone.ConvertToTimezone(depTime, f.DepartureInfo.AirportCode)
	arrTime = timezone.ConvertToTimezone(arrTime, f.ArrivalInfo.AirportCode)

	totalMinutes := parseTravelTime(f.TravelTime)
	hours := totalMinutes / 60
	mins := totalMinutes % 60

	layovers := make([]models.Layover, len(f.ConnectionPoints))
	for i, c := range f.ConnectionPoints {
		layovers[i] = models.Layover{
			Airport:  c.Airport,
			City:     c.City,
			Duration: c.LayoverMinutes,
		}
	}

	cabinKg, checkedKg := parseBatikBaggage(f.BaggageAllowance)

	var depTerminal, arrTerminal *string
	if f.DepartureInfo.TerminalNo != "" {
		t := f.DepartureInfo.TerminalNo
		depTerminal = &t
	}
	if f.ArrivalInfo.TerminalNo != "" {
		t := f.ArrivalInfo.TerminalNo
		arrTerminal = &t
	}

	var aircraft *string
	if f.AircraftType != "" {
		a := f.AircraftType
		aircraft = &a
	}

	return models.Flight{
		ID:       f.FlightID,
		Provider: p.Name(),
		Airline: models.Airline{
			Code: f.OperatingCarrier.CarrierCode,
			Name: f.OperatingCarrier.CarrierName,
		},
		FlightNumber: f.FlightNo,
		Departure: models.Location{
			Airport:  f.DepartureInfo.AirportCode,
			City:     f.DepartureInfo.CityName,
			Terminal: depTerminal,
			Time:     depTime,
			Timezone: timezone.GetTimezoneByAirport(f.DepartureInfo.AirportCode),
		},
		Arrival: models.Location{
			Airport:  f.ArrivalInfo.AirportCode,
			City:     f.ArrivalInfo.CityName,
			Terminal: arrTerminal,
			Time:     arrTime,
			Timezone: timezone.GetTimezoneByAirport(f.ArrivalInfo.AirportCode),
		},
		Duration: models.Duration{
			Hours:        hours,
			Minutes:      mins,
			TotalMinutes: totalMinutes,
		},
		Stops:    f.NumberOfStops,
		Layovers: layovers,
		Price: models.Price{
			Amount:    f.Fare.TotalPrice,
			Currency:  f.Fare.CurrencyCode,
			Formatted: currency.FormatIDR(f.Fare.TotalPrice),
		},
		AvailableSeats: f.SeatsAvailable,
		CabinClass:     f.CabinType,
		Aircraft:       aircraft,
		Amenities:      f.IncludedServices,
		Baggage: models.Baggage{
			CabinKg:   cabinKg,
			CheckedKg: checkedKg,
		},
	}, nil
}

func parseTravelTime(s string) int {
	re := regexp.MustCompile(`(?:(\d+)h)?\s*(?:(\d+)m)?`)
	matches := re.FindStringSubmatch(s)

	var hours, mins int
	if len(matches) >= 2 && matches[1] != "" {
		hours, _ = strconv.Atoi(matches[1])
	}
	if len(matches) >= 3 && matches[2] != "" {
		mins, _ = strconv.Atoi(matches[2])
	}

	return hours*60 + mins
}

func parseBatikBaggage(s string) (cabin, checked float64) {
	s = strings.ToLower(s)

	cabinRe := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*kg\s*cabin`)
	checkedRe := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*kg\s*checked`)

	if matches := cabinRe.FindStringSubmatch(s); len(matches) >= 2 {
		cabin, _ = strconv.ParseFloat(matches[1], 64)
	}
	if matches := checkedRe.FindStringSubmatch(s); len(matches) >= 2 {
		checked, _ = strconv.ParseFloat(matches[1], 64)
	}

	return cabin, checked
}
