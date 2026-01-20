package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/dharmasatrya/flightsearch/internal/models"
)

type Cache interface {
	Get(ctx context.Context, req models.SearchRequest) ([]models.Flight, bool)
	Set(ctx context.Context, req models.SearchRequest, flights []models.Flight) error
	Close() error
}

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	TTL      time.Duration
}

func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
		TTL:      5 * time.Minute,
	}
}

func NewRedisCache(cfg RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
		ttl:    cfg.TTL,
	}, nil
}

func (c *RedisCache) Get(ctx context.Context, req models.SearchRequest) ([]models.Flight, bool) {
	key := generateKey(req)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}

	var flights []models.Flight
	if err := json.Unmarshal(data, &flights); err != nil {
		return nil, false
	}

	return flights, true
}

func (c *RedisCache) Set(ctx context.Context, req models.SearchRequest, flights []models.Flight) error {
	key := generateKey(req)

	data, err := json.Marshal(flights)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

type NoOpCache struct{}

func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

func (c *NoOpCache) Get(ctx context.Context, req models.SearchRequest) ([]models.Flight, bool) {
	return nil, false
}

func (c *NoOpCache) Set(ctx context.Context, req models.SearchRequest, flights []models.Flight) error {
	return nil
}

func (c *NoOpCache) Close() error {
	return nil
}

func generateKey(req models.SearchRequest) string {
	keyData := struct {
		Origin        string
		Destination   string
		DepartureDate string
		ReturnDate    string
		Passengers    int
		CabinClass    string
	}{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
	}

	if req.ReturnDate != nil {
		keyData.ReturnDate = *req.ReturnDate
	}

	data, _ := json.Marshal(keyData)
	hash := sha256.Sum256(data)
	return "flight:" + hex.EncodeToString(hash[:])
}
