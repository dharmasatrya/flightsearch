package aggregator

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dharmasatrya/flightsearch/internal/models"
	"github.com/dharmasatrya/flightsearch/internal/providers"
	"github.com/dharmasatrya/flightsearch/internal/ratelimit"
)

type Config struct {
	Timeout     time.Duration
	MaxRetries  int
	RetryDelays []time.Duration
	RateLimiter *ratelimit.ProviderLimiter
}

type Aggregator struct {
	providers []providers.Provider
	config    Config
}

type Result struct {
	Flights            []models.Flight
	ProvidersQueried   int
	ProvidersSucceeded int
	ProvidersFailed    int
	FailedProviders    []string
}

func NewAggregator(providerList []providers.Provider, config Config) *Aggregator {
	return &Aggregator{
		providers: providerList,
		config:    config,
	}
}

func (a *Aggregator) Search(ctx context.Context, req models.SearchRequest) (*Result, error) {
	searchCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	result := &Result{
		Flights:          make([]models.Flight, 0),
		ProvidersQueried: len(a.providers),
	}

	type providerResult struct {
		provider string
		flights  []models.Flight
		err      error
	}

	resultCh := make(chan providerResult, len(a.providers))
	var wg sync.WaitGroup

	for _, p := range a.providers {
		wg.Add(1)
		go func(provider providers.Provider) {
			defer wg.Done()

			if a.config.RateLimiter != nil {
				if err := a.config.RateLimiter.Wait(searchCtx, provider.Name()); err != nil {
					resultCh <- providerResult{
						provider: provider.Name(),
						err:      err,
					}
					return
				}
			}

			flights, err := a.searchWithRetry(searchCtx, provider, req)
			resultCh <- providerResult{
				provider: provider.Name(),
				flights:  flights,
				err:      err,
			}
		}(p)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var mu sync.Mutex
	for pr := range resultCh {
		if pr.err != nil {
			log.Printf("Provider %s failed: %v", pr.provider, pr.err)
			mu.Lock()
			result.ProvidersFailed++
			result.FailedProviders = append(result.FailedProviders, pr.provider)
			mu.Unlock()
		} else {
			mu.Lock()
			result.ProvidersSucceeded++
			result.Flights = append(result.Flights, pr.flights...)
			mu.Unlock()
		}
	}

	return result, nil
}

func (a *Aggregator) searchWithRetry(ctx context.Context, provider providers.Provider, req models.SearchRequest) ([]models.Flight, error) {
	var lastErr error

	for attempt := 0; attempt <= a.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if attempt > 0 {
			delayIdx := attempt - 1
			if delayIdx >= len(a.config.RetryDelays) {
				delayIdx = len(a.config.RetryDelays) - 1
			}
			delay := a.config.RetryDelays[delayIdx]

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		flights, err := provider.Search(ctx, req)
		if err == nil {
			return flights, nil
		}

		lastErr = err
		log.Printf("Provider %s attempt %d failed: %v", provider.Name(), attempt+1, err)
	}

	return nil, lastErr
}

func (a *Aggregator) SearchRoundTrip(ctx context.Context, req models.SearchRequest) (*Result, *Result, error) {
	if req.ReturnDate == nil || *req.ReturnDate == "" {
		outbound, err := a.Search(ctx, req)
		return outbound, nil, err
	}

	searchCtx, cancel := context.WithTimeout(ctx, a.config.Timeout*2)
	defer cancel()

	type searchResult struct {
		result   *Result
		err      error
		isReturn bool
	}

	resultCh := make(chan searchResult, 2)

	go func() {
		result, err := a.Search(searchCtx, req)
		resultCh <- searchResult{result: result, err: err, isReturn: false}
	}()

	go func() {
		returnReq := models.SearchRequest{
			Origin:        req.Destination,
			Destination:   req.Origin,
			DepartureDate: *req.ReturnDate,
			Passengers:    req.Passengers,
			CabinClass:    req.CabinClass,
			Filters:       req.Filters,
			SortBy:        req.SortBy,
			SortOrder:     req.SortOrder,
		}
		result, err := a.Search(searchCtx, returnReq)
		resultCh <- searchResult{result: result, err: err, isReturn: true}
	}()

	var outbound, returnResult *Result
	var outboundErr, returnErr error

	for i := 0; i < 2; i++ {
		sr := <-resultCh
		if sr.isReturn {
			returnResult = sr.result
			returnErr = sr.err
		} else {
			outbound = sr.result
			outboundErr = sr.err
		}
	}

	if outboundErr != nil {
		return nil, nil, outboundErr
	}

	if returnErr != nil {
		log.Printf("Return flight search failed: %v", returnErr)
		return outbound, nil, nil
	}

	return outbound, returnResult, nil
}
