package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/dharmasatrya/flightsearch/internal/aggregator"
	"github.com/dharmasatrya/flightsearch/internal/cache"
	"github.com/dharmasatrya/flightsearch/internal/filter"
	"github.com/dharmasatrya/flightsearch/internal/models"
)

type SearchHandler struct {
	aggregator *aggregator.Aggregator
	cache      cache.Cache
}

func NewSearchHandler(agg *aggregator.Aggregator, c cache.Cache) *SearchHandler {
	return &SearchHandler{
		aggregator: agg,
		cache:      c,
	}
}

func (h *SearchHandler) Search(c echo.Context) error {
	startTime := time.Now()
	ctx := c.Request().Context()

	var req models.SearchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Failed to parse request body: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
	}

	cacheHit := false
	if cachedFlights, found := h.cache.Get(ctx, req); found {
		cacheHit = true
		filtered := filter.Apply(cachedFlights, req.Filters, req.SortBy, req.SortOrder)

		return c.JSON(http.StatusOK, models.SearchResponse{
			SearchCriteria: buildSearchCriteria(req),
			Metadata: models.SearchMetadata{
				TotalResults:       len(filtered),
				ProvidersQueried:   4,
				ProvidersSucceeded: 4,
				ProvidersFailed:    0,
				SearchTimeMs:       time.Since(startTime).Milliseconds(),
				CacheHit:           cacheHit,
			},
			Flights: filtered,
		})
	}

	if req.ReturnDate != nil && *req.ReturnDate != "" {
		return h.handleRoundTrip(c, req, startTime)
	}

	result, err := h.aggregator.Search(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "search_error",
			Message: "Failed to search flights: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
	}

	_ = h.cache.Set(ctx, req, result.Flights)
	filtered := filter.Apply(result.Flights, req.Filters, req.SortBy, req.SortOrder)

	return c.JSON(http.StatusOK, models.SearchResponse{
		SearchCriteria: buildSearchCriteria(req),
		Metadata: models.SearchMetadata{
			TotalResults:       len(filtered),
			ProvidersQueried:   result.ProvidersQueried,
			ProvidersSucceeded: result.ProvidersSucceeded,
			ProvidersFailed:    result.ProvidersFailed,
			FailedProviders:    result.FailedProviders,
			SearchTimeMs:       time.Since(startTime).Milliseconds(),
			CacheHit:           cacheHit,
		},
		Flights: filtered,
	})
}

func (h *SearchHandler) handleRoundTrip(c echo.Context, req models.SearchRequest, startTime time.Time) error {
	ctx := c.Request().Context()

	outbound, returnResult, err := h.aggregator.SearchRoundTrip(ctx, req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "search_error",
			Message: "Failed to search flights: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
	}

	outboundFiltered := filter.Apply(outbound.Flights, req.Filters, req.SortBy, req.SortOrder)

	var returnFiltered []models.Flight
	var returnMeta *aggregator.Result
	if returnResult != nil {
		returnFiltered = filter.Apply(returnResult.Flights, req.Filters, req.SortBy, req.SortOrder)
		returnMeta = returnResult
	}

	totalQueried := outbound.ProvidersQueried
	totalSucceeded := outbound.ProvidersSucceeded
	totalFailed := outbound.ProvidersFailed
	failedProviders := outbound.FailedProviders

	if returnMeta != nil {
		totalQueried += returnMeta.ProvidersQueried
		totalSucceeded += returnMeta.ProvidersSucceeded
		totalFailed += returnMeta.ProvidersFailed
		failedProviders = append(failedProviders, returnMeta.FailedProviders...)
	}

	failedProviders = uniqueStrings(failedProviders)

	return c.JSON(http.StatusOK, models.RoundTripResponse{
		SearchCriteria: buildSearchCriteria(req),
		Metadata: models.SearchMetadata{
			TotalResults:       len(outboundFiltered) + len(returnFiltered),
			ProvidersQueried:   totalQueried,
			ProvidersSucceeded: totalSucceeded,
			ProvidersFailed:    totalFailed,
			FailedProviders:    failedProviders,
			SearchTimeMs:       time.Since(startTime).Milliseconds(),
			CacheHit:           false,
		},
		OutboundFlights: outboundFiltered,
		ReturnFlights:   returnFiltered,
	})
}

func buildSearchCriteria(req models.SearchRequest) models.SearchCriteria {
	return models.SearchCriteria{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		ReturnDate:    req.ReturnDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
		Filters:       req.Filters,
		SortBy:        req.SortBy,
		SortOrder:     req.SortOrder,
	}
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func HealthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}
