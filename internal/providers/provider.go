package providers

import (
	"context"

	"github.com/dharmasatrya/flightsearch/internal/models"
)

type Provider interface {
	Name() string
	Search(ctx context.Context, req models.SearchRequest) ([]models.Flight, error)
}

type ProviderError struct {
	Provider string
	Err      error
}

func (e *ProviderError) Error() string {
	return e.Provider + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

func NewProviderError(provider string, err error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Err:      err,
	}
}
