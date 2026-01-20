package filter

import (
	"sort"
	"strings"
	"time"

	"github.com/dharmasatrya/flightsearch/internal/models"
	"github.com/dharmasatrya/flightsearch/internal/ranking"
)

func Apply(flights []models.Flight, filters *models.SearchFilters, sortBy, sortOrder string) []models.Flight {
	filtered := applyFilters(flights, filters)

	if sortBy == "best_value" {
		filtered = ranking.CalculateScores(filtered)
	}

	sorted := applySort(filtered, sortBy, sortOrder)

	return sorted
}

func applyFilters(flights []models.Flight, filters *models.SearchFilters) []models.Flight {
	if filters == nil {
		return flights
	}

	result := make([]models.Flight, 0, len(flights))

	for _, f := range flights {
		if matchesFilters(f, filters) {
			result = append(result, f)
		}
	}

	return result
}

func matchesFilters(f models.Flight, filters *models.SearchFilters) bool {
	if filters.PriceMin != nil && f.Price.Amount < *filters.PriceMin {
		return false
	}
	if filters.PriceMax != nil && f.Price.Amount > *filters.PriceMax {
		return false
	}

	if filters.MaxStops != nil && f.Stops > *filters.MaxStops {
		return false
	}

	if len(filters.Airlines) > 0 {
		found := false
		for _, airline := range filters.Airlines {
			if strings.EqualFold(f.Airline.Code, airline) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filters.DepartureTimeMin != nil {
		minTime, err := parseTimeOfDay(*filters.DepartureTimeMin)
		if err == nil {
			depTime := f.Departure.Time.Hour()*60 + f.Departure.Time.Minute()
			if depTime < minTime {
				return false
			}
		}
	}
	if filters.DepartureTimeMax != nil {
		maxTime, err := parseTimeOfDay(*filters.DepartureTimeMax)
		if err == nil {
			depTime := f.Departure.Time.Hour()*60 + f.Departure.Time.Minute()
			if depTime > maxTime {
				return false
			}
		}
	}

	if filters.ArrivalTimeMin != nil {
		minTime, err := parseTimeOfDay(*filters.ArrivalTimeMin)
		if err == nil {
			arrTime := f.Arrival.Time.Hour()*60 + f.Arrival.Time.Minute()
			if arrTime < minTime {
				return false
			}
		}
	}
	if filters.ArrivalTimeMax != nil {
		maxTime, err := parseTimeOfDay(*filters.ArrivalTimeMax)
		if err == nil {
			arrTime := f.Arrival.Time.Hour()*60 + f.Arrival.Time.Minute()
			if arrTime > maxTime {
				return false
			}
		}
	}

	if filters.MaxDuration != nil && f.Duration.TotalMinutes > *filters.MaxDuration {
		return false
	}

	return true
}

func parseTimeOfDay(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

func applySort(flights []models.Flight, sortBy, sortOrder string) []models.Flight {
	if len(flights) == 0 {
		return flights
	}

	ascending := strings.ToLower(sortOrder) != "desc"

	switch strings.ToLower(sortBy) {
	case "price":
		sort.Slice(flights, func(i, j int) bool {
			if ascending {
				return flights[i].Price.Amount < flights[j].Price.Amount
			}
			return flights[i].Price.Amount > flights[j].Price.Amount
		})

	case "duration":
		sort.Slice(flights, func(i, j int) bool {
			if ascending {
				return flights[i].Duration.TotalMinutes < flights[j].Duration.TotalMinutes
			}
			return flights[i].Duration.TotalMinutes > flights[j].Duration.TotalMinutes
		})

	case "departure":
		sort.Slice(flights, func(i, j int) bool {
			if ascending {
				return flights[i].Departure.Time.Before(flights[j].Departure.Time)
			}
			return flights[i].Departure.Time.After(flights[j].Departure.Time)
		})

	case "arrival":
		sort.Slice(flights, func(i, j int) bool {
			if ascending {
				return flights[i].Arrival.Time.Before(flights[j].Arrival.Time)
			}
			return flights[i].Arrival.Time.After(flights[j].Arrival.Time)
		})

	case "best_value":
		sort.Slice(flights, func(i, j int) bool {
			if ascending {
				return flights[i].BestValueScore < flights[j].BestValueScore
			}
			return flights[i].BestValueScore > flights[j].BestValueScore
		})

	case "stops":
		sort.Slice(flights, func(i, j int) bool {
			if ascending {
				return flights[i].Stops < flights[j].Stops
			}
			return flights[i].Stops > flights[j].Stops
		})

	default:
		// Default to price ascending
		sort.Slice(flights, func(i, j int) bool {
			return flights[i].Price.Amount < flights[j].Price.Amount
		})
	}

	return flights
}
