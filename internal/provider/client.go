package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"math"
	"math/rand"
	"strconv"
	"sync"
)

// Global rate limiter shared across all client instances
var (
	globalRateLimiter *rateLimiter
	rateLimiterOnce   sync.Once
)

type rateLimiter struct {
	mu            sync.Mutex
	lastRequestAt time.Time
	minInterval   time.Duration
}

func getGlobalRateLimiter() *rateLimiter {
	rateLimiterOnce.Do(func() {
		globalRateLimiter = &rateLimiter{
			minInterval: 200 * time.Millisecond, // Minimum 200ms between ANY requests
		}
	})
	return globalRateLimiter
}

func (rl *rateLimiter) wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if !rl.lastRequestAt.IsZero() {
		elapsed := time.Since(rl.lastRequestAt)
		if elapsed < rl.minInterval {
			time.Sleep(rl.minInterval - elapsed)
		}
	}
	rl.lastRequestAt = time.Now()
}

type Client struct {
	BaseURL       string
	ApiKey        string
	HTTPClient    *http.Client
	MaxRetries    int
	RetryDelay    time.Duration
	rateLimiter   *rateLimiter
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:     baseURL,
		ApiKey:      apiKey,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		MaxRetries:  20,
		RetryDelay:  2 * time.Second,
		rateLimiter: getGlobalRateLimiter(),
	}
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	var jsonData []byte
	
	if body != nil {
		var err error
		jsonData, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// Global rate limiting: ensure minimum delay between ALL requests across all goroutines
		c.rateLimiter.wait()
		
		// Reset body for retry
		if jsonData != nil {
			reqBody = bytes.NewBuffer(jsonData)
		}

		req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.MaxRetries {
				time.Sleep(c.calculateBackoff(attempt))
				continue
			}
			return nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			lastErr = err
			if attempt < c.MaxRetries {
				time.Sleep(c.calculateBackoff(attempt))
				continue
			}
			return nil, err
		}

		// Handle rate limiting (429)
		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limit exceeded (429)")
			if attempt < c.MaxRetries {
				// Check for Retry-After header
				retryAfter := resp.Header.Get("Retry-After")
				var backoff time.Duration

				if retryAfter != "" {
					// Try to parse Retry-After as seconds
					if seconds, err := strconv.Atoi(retryAfter); err == nil {
						backoff = time.Duration(seconds) * time.Second
					} else {
						// Fallback to calculated backoff
						backoff = c.calculateBackoffWithJitter(attempt)
					}
				} else {
					backoff = c.calculateBackoffWithJitter(attempt)
				}

				fmt.Printf("Rate limited, retrying in %v (attempt %d/%d)...\n", backoff, attempt+1, c.MaxRetries)
				time.Sleep(backoff)
				continue
			}
			return nil, fmt.Errorf("rate limit exceeded after %d retries", c.MaxRetries)
		}

		// Handle transient gateway/server errors (502, 503, 504) — retry with backoff
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			lastErr = fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
			if attempt < c.MaxRetries {
				backoff := c.calculateBackoffWithJitter(attempt)
				fmt.Printf("Transient server error %d, retrying in %v (attempt %d/%d)...\n", resp.StatusCode, backoff, attempt+1, c.MaxRetries)
				time.Sleep(backoff)
				continue
			}
			return nil, lastErr
		}

		// Handle other errors
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
		}

		return respBody, nil
	}

	return nil, lastErr
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 2s, 4s, 8s, 16s, 32s, 64s, 90s (capped)
	backoff := c.RetryDelay * time.Duration(math.Pow(2, float64(attempt)))
	// Cap at 90 seconds
	if backoff > 90*time.Second {
		backoff = 90 * time.Second
	}
	return backoff
}

func (c *Client) calculateBackoffWithJitter(attempt int) time.Duration {
	// Exponential backoff with jitter to prevent thundering herd
	baseBackoff := c.calculateBackoff(attempt)
	
	// Add random jitter: ±25% of the base backoff
	jitter := time.Duration(rand.Int63n(int64(baseBackoff) / 2))
	backoff := baseBackoff + jitter - (baseBackoff / 4)
	
	// Ensure minimum of 2 seconds
	if backoff < 2*time.Second {
		backoff = 2 * time.Second
	}
	
	return backoff
}
