# rate-limiter-middleware

A Go HTTP middleware that integrates with [rate-limiter](https://github.com/joymohanty8999/rate-limiter) — a production-grade Token Bucket rate limiting service backed by Redis.

## Installation
```bash
go get github.com/joymohanty8999/rate-limiter-middleware
```

## Quick Start
```go
import middleware "github.com/joymohanty8999/rate-limiter-middleware"

limiter := middleware.RateLimit(middleware.Options{
    ServiceURL: "https://rate-limiter.josephmohanty.me",
    Bucket:     "default",
})

mux.Handle("/api/", limiter(yourHandler))
```

## Options

| Field | Type | Default | Description |
|---|---|---|---|
| `ServiceURL` | `string` | required | Base URL of the rate limiter service |
| `Bucket` | `string` | `"default"` | Named bucket config to use |
| `KeyFunc` | `func(*http.Request) string` | IP-based | Extracts the rate limit key from the request |
| `OnLimited` | `func(w, r, retryAfter)` | 429 JSON | Called when a request is rate limited |
| `Timeout` | `time.Duration` | `2s` | Timeout for requests to the rate limiter service |

## Examples

**Rate limit by IP (default):**
```go
limiter := middleware.RateLimit(middleware.Options{
    ServiceURL: "https://rate-limiter.josephmohanty.me",
    Bucket:     "default",
})
```

**Rate limit by user ID:**
```go
limiter := middleware.RateLimit(middleware.Options{
    ServiceURL: "https://rate-limiter.josephmohanty.me",
    Bucket:     "strict",
    KeyFunc: func(r *http.Request) string {
        return "user:" + r.Header.Get("X-User-ID")
    },
})
```

**Rate limit by API key:**
```go
limiter := middleware.RateLimit(middleware.Options{
    ServiceURL: "https://rate-limiter.josephmohanty.me",
    Bucket:     "default",
    KeyFunc: func(r *http.Request) string {
        return "apikey:" + r.Header.Get("X-API-Key")
    },
})
```

**Custom rejection response:**
```go
limiter := middleware.RateLimit(middleware.Options{
    ServiceURL: "https://rate-limiter.josephmohanty.me",
    Bucket:     "default",
    OnLimited: func(w http.ResponseWriter, r *http.Request, retryAfter int) {
        http.Error(w, "slow down!", http.StatusTooManyRequests)
    },
})
```

## Response Headers

The middleware sets the following headers on every request:

| Header | Description |
|---|---|
| `X-RateLimit-Remaining` | Tokens remaining in the bucket |
| `Retry-After` | Seconds until next token (on 429 only) |

## Fail Open

If the rate limiter service is unreachable, the middleware **fails open** — requests are allowed through. This ensures your service stays available even if the rate limiter goes down.

## Buckets

Buckets are configured on the rate limiter service in `config.yaml`:

| Bucket | Capacity | Refill Rate |
|---|---|---|
| `default` | 10 tokens | 1/sec |
| `strict` | 5 tokens | 0.5/sec |
| `relaxed` | 100 tokens | 10/sec |

## Related

- [rate-limiter](https://github.com/joymohanty8999/rate-limiter) — the rate limiting service this middleware talks to
