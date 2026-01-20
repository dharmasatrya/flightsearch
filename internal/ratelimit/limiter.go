package ratelimit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type ProviderLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	defaults RateLimitConfig
}

type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

func DefaultConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         20,
	}
}

func NewProviderLimiter(config RateLimitConfig) *ProviderLimiter {
	return &ProviderLimiter{
		limiters: make(map[string]*rate.Limiter),
		defaults: config,
	}
}

func NewProviderLimiterWithDefaults() *ProviderLimiter {
	return NewProviderLimiter(DefaultConfig())
}

func (p *ProviderLimiter) GetLimiter(provider string) *rate.Limiter {
	p.mu.RLock()
	limiter, exists := p.limiters[provider]
	p.mu.RUnlock()

	if exists {
		return limiter
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if limiter, exists = p.limiters[provider]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(p.defaults.RequestsPerSecond), p.defaults.BurstSize)
	p.limiters[provider] = limiter
	return limiter
}

func (p *ProviderLimiter) SetProviderLimit(provider string, rps float64, burst int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.limiters[provider] = rate.NewLimiter(rate.Limit(rps), burst)
}

func (p *ProviderLimiter) Wait(ctx context.Context, provider string) error {
	return p.GetLimiter(provider).Wait(ctx)
}
