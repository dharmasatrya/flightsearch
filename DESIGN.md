# Design Notes

Quick notes on why certain things are built the way they are.

## The Core Problem

We're aggregating flight data from 4 different airline APIs, each with their own data format, response times, and reliability. The goal is to merge them into a single unified response without making users wait forever or miss out on results because one provider is slow.

## Key Decisions

### Why Parallel Fetching?

Each provider takes 50-400ms to respond. If we queried them sequentially, that's potentially 1.6 seconds just waiting. By using goroutines, all 4 providers are queried simultaneously - total wait time becomes the slowest provider, not the sum.

The tricky part is collecting results safely. We use channels to gather flights as they come in, with a 2-second timeout so one slow provider doesn't hold up everything.

### Handling Provider Failures Gracefully

AirAsia has a simulated 10% failure rate. Instead of failing the whole request, we return partial results. If 3 out of 4 providers succeed, the user still gets useful data. The response metadata shows which providers failed so clients can decide how to handle it.

We also retry failed requests with exponential backoff (100ms → 200ms → 400ms). This catches network issues without spamming the provider.

### The Data Normalization Mess

This was the hardest part. Each provider returns data differently:

- **Time formats**: Garuda uses `+0700`, AirAsia uses `+07:00`, Lion Air passes a timezone name separately
- **Duration**: Some return minutes as int, AirAsia returns hours as float, Batik Air returns "2h 15m" strings
- **Stops**: Could be an int, a boolean `is_direct` + count, or just an array of layovers to count
- **Baggage**: Structured objects vs "7kg cabin, 20kg checked" strings

The solution: each provider adapter handles its own parsing and outputs the same `Flight` struct. Messy parsing logic stays isolated - adding a new provider doesn't touch existing ones.

This is quite rigid and will break when provider changes their format without notifying us. But per my experience with third parties there is simply nothing we can do but pray haha.

### Why Filter Before Scoring?

The best value score requires finding max price/duration across all flights to normalize. If we score first then filter, we waste cycles scoring flights that get filtered out anyway. More importantly, if someone filters to "direct flights only", the scoring should compare against other direct flights, not against all flights including those with stops.

### Timezone Handling

Indonesia spans 3 timezones (WIB, WITA, WIT) and a flight from Jakarta (WIB) to Bali (WITA) crosses timezone boundaries. We map each airport code to its timezone and convert times accordingly.

## Interesting Parts

**Cache key generation**: We SHA256 hash the search parameters to create cache keys. This handles the case where two requests with the same params should hit the same cache entry, regardless of how the JSON was formatted.

**Rate limiting per provider**: Premium airlines (Garuda) get higher limits than budget ones (AirAsia). This reflects real-world API quotas where we might have different contracts with different providers. This is my assumption that they have API quotas.

**Best value scoring**: The weights (50% price, 30% duration, 20% stops) are my opinion on what matters most but adjustable. The formula normalizes values to 0-100 so a IDR 500k price difference is comparable to a 30-minute duration difference.
