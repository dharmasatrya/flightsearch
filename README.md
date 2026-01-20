# Flight Search Aggregation System

A Go-based flight search and aggregation system that fetches from 4 mock Indonesian airline APIs, normalizes data, and returns unified search results with filtering and sorting capabilities.

## Features

- **Multi-Provider Aggregation**: Parallel fetching from Garuda Indonesia, Lion Air, Batik Air, and AirAsia
- **Data Normalization**: Unified flight model from different API formats
- **Filtering**: Price range, stops, airlines, departure/arrival time windows, max duration
- **Sorting**: Price, duration, departure time, arrival time, best value score
- **Best Value Scoring**: Weighted algorithm combining price, duration, and stops
- **Caching**: Redis cache with configurable TTL (can be disabled for easier run)
- **Rate Limiting**: Per-provider rate limiting using token bucket algorithm
- **Retry Logic**: Exponential backoff for failed requests
- **Round-Trip Support**: Parallel search for outbound and return flights
- **Indonesia Timezone Handling**: WIB/WITA/WIT timezone support

## Project Structure

```
flightsearch/
├── cmd/server/main.go
├── internal/
│   ├── models/
│   ├── providers/
│   │   └── data/
│   ├── aggregator/
│   ├── filter/
│   ├── ranking/
│   ├── cache/
│   ├── ratelimit/
│   ├── timezone/
│   └── handler/
├── pkg/currency/
├── docs/
│   ├── swagger.yaml
│   ├── postman_collection.json
│   └── APPLICATION_FLOW.md
├── go.mod
├── Dockerfile
├── DESIGN.md
└── README.md
```

## Prerequisites

- Go 1.21+
- Redis (optional, can be disabled)

## Installation

```bash
# Download dependencies
go mod download

# Build the application
go build -o flightsearch ./cmd/server
```

## Running the Server

### With Redis (Default)

Redis is required by default for caching. Start Redis first:

```bash
# Start Redis using Docker
docker run -d --name redis -p 6379:6379 redis:alpine

# Run the application
go run ./cmd/server
```

### Without Cache (Development/Testing)

To run without Redis, disable caching:

```bash
CACHE_ENABLED=false go run ./cmd/server
```

### Using Docker Compose

Create a `docker-compose.yml`:

```yaml
version: '3.8'
services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  flightsearch:
    build: .
    ports:
      - "8080:8080"
    environment:
      - CACHE_ENABLED=true
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    depends_on:
      - redis

volumes:
  redis_data:
```

Then run:
```bash
docker-compose up -d
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `CACHE_ENABLED` | `true` | Enable Redis cache (boolean) |
| `REDIS_HOST` | `localhost` | Redis server host |
| `REDIS_PORT` | `6379` | Redis server port |
| `REDIS_TTL` | `5m` | Cache TTL (e.g., `5m`, `300s`, `1h`) |

### Example Configurations

**Development (no cache):**
```bash
CACHE_ENABLED=false go run ./cmd/server
```

**Production with remote Redis:**
```bash
CACHE_ENABLED=true \
REDIS_HOST=redis.example.com \
REDIS_PORT=6379 \
REDIS_TTL=10m \
PORT=8080 \
go run ./cmd/server
```

## API Endpoints

### POST /api/v1/flights/search

Search for flights with optional filtering and sorting.

**Request Body:**

```json
{
  "origin": "CGK",
  "destination": "DPS",
  "departure_date": "2025-12-15",
  "return_date": null,
  "passengers": 1,
  "cabin_class": "economy",
  "filters": {
    "price_min": 0,
    "price_max": 2000000,
    "max_stops": 1,
    "airlines": ["GA", "QZ"],
    "departure_time_min": "06:00",
    "departure_time_max": "18:00"
  },
  "sort_by": "best_value",
  "sort_order": "asc"
}
```

**Response:**

```json
{
  "search_criteria": {
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "passengers": 1,
    "cabin_class": "economy",
    "sort_by": "best_value",
    "sort_order": "asc"
  },
  "metadata": {
    "total_results": 12,
    "providers_queried": 4,
    "providers_succeeded": 4,
    "providers_failed": 0,
    "search_time_ms": 285,
    "cache_hit": false
  },
  "flights": [
    {
      "id": "QZ-001",
      "provider": "airasia",
      "airline": {
        "code": "QZ",
        "name": "AirAsia Indonesia"
      },
      "flight_number": "QZ 7520",
      "departure": {
        "airport": "CGK",
        "city": "Jakarta",
        "time": "2025-12-15T06:30:00+07:00",
        "timezone": "WIB"
      },
      "arrival": {
        "airport": "DPS",
        "city": "Bali",
        "time": "2025-12-15T09:15:00+08:00",
        "timezone": "WITA"
      },
      "duration": {
        "hours": 1,
        "minutes": 45,
        "total_minutes": 105
      },
      "stops": 0,
      "price": {
        "amount": 650000,
        "currency": "IDR",
        "formatted": "IDR 650.000"
      },
      "available_seats": 85,
      "cabin_class": "economy",
      "aircraft": "Airbus A320",
      "baggage": {
        "cabin_kg": 7,
        "checked_kg": 0
      },
      "best_value_score": 25.5
    }
  ]
}
```

### GET /health

Health check endpoint.

```json
{
  "status": "ok"
}
```

## Filter Options

| Filter | Type | Description |
|--------|------|-------------|
| `price_min` | float | Minimum price in IDR |
| `price_max` | float | Maximum price in IDR |
| `max_stops` | int | Maximum number of stops (0 = direct only) |
| `airlines` | array | Whitelist of airline codes (e.g., ["GA", "JT"]) |
| `departure_time_min` | string | Earliest departure time (HH:MM) |
| `departure_time_max` | string | Latest departure time (HH:MM) |
| `arrival_time_min` | string | Earliest arrival time (HH:MM) |
| `arrival_time_max` | string | Latest arrival time (HH:MM) |
| `max_duration` | int | Maximum flight duration in minutes |

## Sort Options

Default sorting is by **best value score** (weighted combination of price, duration, and stops).

| Sort By | Description |
|---------|-------------|
| `price` | Sort by price |
| `duration` | Sort by flight duration |
| `departure` | Sort by departure time |
| `arrival` | Sort by arrival time |
| `stops` | Sort by number of stops |

## Best Value Scoring

The best value score is calculated using:

```
Score = (PriceScore × 0.5) + (DurationScore × 0.3) + (StopsScore × 0.2)
```

Where:
- `PriceScore`: Normalized price (0-100, lower is better)
- `DurationScore`: Normalized duration (0-100, shorter is better)
- `StopsScore`: Number of stops × 15

Lower scores indicate better value.

## Provider Simulations

| Provider | Latency | Failure Rate |
|----------|---------|--------------|
| Garuda Indonesia | 50-100ms | 0% |
| Lion Air | 100-200ms | 0% |
| Batik Air | 200-400ms | 0% |
| AirAsia | 50-150ms | 10% |

## Indonesia Timezone Support

- **WIB (UTC+7)**
- **WITA (UTC+8)**
- **WIT (UTC+9)**

## Example Requests

### Basic Search

```bash
curl -X POST http://localhost:8080/api/v1/flights/search \
  -H "Content-Type: application/json" \
  -d '{"origin":"CGK","destination":"DPS","departure_date":"2025-12-15","passengers":1,"cabin_class":"economy"}'
```

### Search with Filters

```bash
curl -X POST http://localhost:8080/api/v1/flights/search \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "passengers": 1,
    "cabin_class": "economy",
    "filters": {
      "price_max": 1500000,
      "max_stops": 0,
      "departure_time_min": "06:00",
      "departure_time_max": "12:00"
    },
    "sort_by": "price",
    "sort_order": "asc"
  }'
```

### Round-Trip Search

```bash
curl -X POST http://localhost:8080/api/v1/flights/search \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "return_date": "2025-12-20",
    "passengers": 2,
    "cabin_class": "economy",
    "sort_by": "best_value"
  }'
```

## Documentation

API specs and testing tools are in the `docs/` folder:

- **OpenAPI/Swagger**: `docs/swagger.yaml` - import into [Swagger Editor](https://editor.swagger.io) or your preferred tool
- **Postman Collection**: `docs/postman_collection.json` - import into Postman for quick API testing
- **Application Flow**: `docs/APPLICATION_FLOW.md` - request flow diagrams and architecture overview
