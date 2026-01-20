package ranking

import (
	"math"

	"github.com/dharmasatrya/flightsearch/internal/models"
)

const (
	PriceWeight    = 0.5
	DurationWeight = 0.3
	StopsWeight    = 0.2
)

func CalculateScores(flights []models.Flight) []models.Flight {
	if len(flights) == 0 {
		return flights
	}

	maxPrice := findMaxPrice(flights)
	maxDuration := findMaxDuration(flights)

	result := make([]models.Flight, len(flights))
	for i, f := range flights {
		result[i] = f
		result[i].BestValueScore = CalculateBestValue(f, maxPrice, maxDuration)
	}

	return result
}

// Lower score = better value
func CalculateBestValue(flight models.Flight, maxPrice, maxDuration float64) float64 {
	priceScore := 0.0
	if maxPrice > 0 {
		priceScore = (flight.Price.Amount / maxPrice) * 100
	}

	durationScore := 0.0
	if maxDuration > 0 {
		durationScore = (float64(flight.Duration.TotalMinutes) / maxDuration) * 100
	}

	stopsScore := float64(flight.Stops) * 15
	score := (priceScore * PriceWeight) + (durationScore * DurationWeight) + (stopsScore * StopsWeight)

	return math.Round(score*100) / 100
}

func findMaxPrice(flights []models.Flight) float64 {
	maxPrice := 0.0
	for _, f := range flights {
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
	}
	return maxPrice
}

func findMaxDuration(flights []models.Flight) float64 {
	maxDuration := 0.0
	for _, f := range flights {
		dur := float64(f.Duration.TotalMinutes)
		if dur > maxDuration {
			maxDuration = dur
		}
	}
	return maxDuration
}
