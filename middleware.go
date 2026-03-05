package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Options struct {
	// ServiceURL is the base URL of the rate limiter service
	// e.g. "https://rate-limiter.josephmohanty.me"
	ServiceURL string

	// Bucket is the named bucket config to use (e.g. "default", "strict", "relaxed")
	// Defaults to "default" if empty
	Bucket string

	// KeyFunc extracts the rate limit key from the request
	// Defaults to IP-based key if nil
	KeyFunc func(r *http.Request) string

	// OnLimited is called when a request is rate limited
	// Defaults to a 429 JSON response if nil
	OnLimited func(w http.ResponseWriter, r *http.Request, retryAfter int)

	// Timeout for requests to the rate limiter service
	// Defaults to 2 seconds
	Timeout time.Duration
}

type checkRequest struct {
	Key    string `json:"key"`
	Bucket string `json:"bucket"`
}

type checkResponse struct {
	Allowed    bool   `json:"allowed"`
	Remaining  int    `json:"remaining"`
	RetryAfter int    `json:"retry_after"`
	Message    string `json:"message"`
}

// RateLimit returns a middleware that rate limits requests using the
// rate limiter service at the given URL
func RateLimit(opts Options) func(http.Handler) http.Handler {
	if opts.Bucket == "" {
		opts.Bucket = "default"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Second
	}
	if opts.KeyFunc == nil {
		opts.KeyFunc = defaultKeyFunc
	}
	if opts.OnLimited == nil {
		opts.OnLimited = defaultOnLimited
	}

	client := &http.Client{Timeout: opts.Timeout}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := opts.KeyFunc(r)

			result, err := check(client, opts.ServiceURL, key, opts.Bucket)
			if err != nil {
				// Fail open — rate limiter service is unreachable
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))

			if !result.Allowed {
				opts.OnLimited(w, r, result.RetryAfter)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func check(client *http.Client, serviceURL, key, bucket string) (*checkResponse, error) {
	body, err := json.Marshal(checkRequest{Key: key, Bucket: bucket})
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(
		serviceURL+"/check",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result checkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func defaultKeyFunc(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return fmt.Sprintf("ip:%s", ip)
	}
	return fmt.Sprintf("ip:%s", r.RemoteAddr)
}

func defaultOnLimited(w http.ResponseWriter, r *http.Request, retryAfter int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(map[string]any{
		"error":       "rate limit exceeded",
		"retry_after": retryAfter,
	})
}
