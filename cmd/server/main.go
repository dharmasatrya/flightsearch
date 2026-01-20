package main

import (
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/dharmasatrya/flightsearch/internal/aggregator"
	"github.com/dharmasatrya/flightsearch/internal/cache"
	"github.com/dharmasatrya/flightsearch/internal/handler"
	"github.com/dharmasatrya/flightsearch/internal/providers"
	"github.com/dharmasatrya/flightsearch/internal/ratelimit"
)

type Config struct {
	Port         string
	CacheEnabled bool
	RedisHost    string
	RedisPort    string
	RedisTTL     time.Duration
}

func main() {
	cfg := loadConfig()
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())

	providerList, err := initializeProviders()
	if err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}
	log.Printf("Initialized %d flight providers", len(providerList))

	rateLimiter := ratelimit.NewProviderLimiterWithDefaults()
	rateLimiter.SetProviderLimit("garuda", 20, 30)
	rateLimiter.SetProviderLimit("lionair", 15, 25)
	rateLimiter.SetProviderLimit("batikair", 15, 25)
	rateLimiter.SetProviderLimit("airasia", 10, 20)

	aggConfig := aggregator.Config{
		Timeout:    2 * time.Second,
		MaxRetries: 3,
		RetryDelays: []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			400 * time.Millisecond,
		},
		RateLimiter: rateLimiter,
	}
	agg := aggregator.NewAggregator(providerList, aggConfig)

	var flightCache cache.Cache
	if cfg.CacheEnabled {
		redisCache, err := cache.NewRedisCache(cache.RedisConfig{
			Host: cfg.RedisHost,
			Port: cfg.RedisPort,
			TTL:  cfg.RedisTTL,
		})
		if err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		flightCache = redisCache
		log.Printf("Redis cache enabled (host: %s:%s, TTL: %v)", cfg.RedisHost, cfg.RedisPort, cfg.RedisTTL)
	} else {
		flightCache = cache.NewNoOpCache()
		log.Println("Cache disabled")
	}

	searchHandler := handler.NewSearchHandler(agg, flightCache)

	api := e.Group("/api/v1")
	api.POST("/flights/search", searchHandler.Search)
	e.GET("/health", handler.HealthHandler)

	log.Printf("Starting flight aggregator server on port %s", cfg.Port)

	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func loadConfig() Config {
	cfg := Config{
		Port:         getEnv("PORT", "8080"),
		CacheEnabled: getEnvBool("CACHE_ENABLED", true),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		RedisTTL:     getEnvDuration("REDIS_TTL", 5*time.Minute),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func initializeProviders() ([]providers.Provider, error) {
	var providerList []providers.Provider

	garuda, err := providers.NewGarudaProvider()
	if err != nil {
		return nil, err
	}
	providerList = append(providerList, garuda)

	lionair, err := providers.NewLionAirProvider()
	if err != nil {
		return nil, err
	}
	providerList = append(providerList, lionair)

	batikair, err := providers.NewBatikAirProvider()
	if err != nil {
		return nil, err
	}
	providerList = append(providerList, batikair)

	airasia, err := providers.NewAirAsiaProvider()
	if err != nil {
		return nil, err
	}
	providerList = append(providerList, airasia)

	return providerList, nil
}
