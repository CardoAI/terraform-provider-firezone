package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/time/rate"
)

// Default client tuning. All values are overridable via environment variables
// so the provider can be matched to the Firezone account's server-side rate
// limit (per-account api_refill_rate / api_capacity).
const (
	// defaultRateLimitRPS is the sustained request rate (tokens/sec). Mirrors
	// Firezone's default API_REFILL_RATE.
	defaultRateLimitRPS = 10.0
	// defaultRateLimitBurst is the token-bucket size: how many requests may be
	// issued back-to-back before pacing kicks in.
	defaultRateLimitBurst = 30
	defaultMaxRetries     = 6
	defaultHTTPTimeout    = 30 * time.Second
	// retryBaseDelay is the base for exponential backoff (base * 2^attempt).
	retryBaseDelay    = 1 * time.Second
	defaultMaxBackoff = 30 * time.Second
)

// Global rate limiter shared across all client instances in the process so the
// per-account server-side limit is respected regardless of how many resources
// Terraform reconciles in parallel.
var (
	globalRateLimiter *rate.Limiter
	rateLimiterOnce   sync.Once
)

func getGlobalRateLimiter() *rate.Limiter {
	rateLimiterOnce.Do(func() {
		rps := getEnvFloat("FIREZONE_RATE_LIMIT_RPS", defaultRateLimitRPS)
		burst := getEnvInt("FIREZONE_RATE_LIMIT_BURST", defaultRateLimitBurst)
		globalRateLimiter = rate.NewLimiter(rate.Limit(rps), burst)
	})
	return globalRateLimiter
}

type Client struct {
	BaseURL     string
	ApiKey      string
	HTTPClient  *http.Client
	MaxRetries  int
	MaxBackoff  time.Duration
	rateLimiter *rate.Limiter
}

func NewClient(baseURL, apiKey string) *Client {
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Client{
		BaseURL: baseURL,
		ApiKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout:   defaultHTTPTimeout,
			Transport: transport,
		},
		MaxRetries:  getEnvInt("FIREZONE_MAX_RETRIES", defaultMaxRetries),
		MaxBackoff:  time.Duration(getEnvInt("FIREZONE_RETRY_MAX_BACKOFF_SECS", int(defaultMaxBackoff.Seconds()))) * time.Second,
		rateLimiter: getGlobalRateLimiter(),
	}
}

// doRequest preserves the original signature used throughout the provider and
// delegates to the context-aware implementation.
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	return c.doRequestCtx(context.Background(), method, path, body)
}

func (c *Client) doRequestCtx(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
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
		// Token-bucket rate limiting: allows bursts up to the bucket size and
		// then paces requests, instead of strictly serializing every call.
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		var reqBody io.Reader
		if jsonData != nil {
			reqBody = bytes.NewBuffer(jsonData)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.MaxRetries {
				time.Sleep(c.calculateBackoffWithJitter(attempt))
				continue
			}
			return nil, err
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			if attempt < c.MaxRetries {
				time.Sleep(c.calculateBackoffWithJitter(attempt))
				continue
			}
			return nil, err
		}

		// Rate limited (429): honor Retry-After when present, otherwise use
		// jittered exponential backoff.
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limit exceeded (429)")
			if attempt < c.MaxRetries {
				backoff := c.retryAfterOrBackoff(resp.Header.Get("Retry-After"), attempt)
				tflog.Warn(ctx, "Firezone API rate limited; retrying", map[string]interface{}{
					"backoff":     backoff.String(),
					"attempt":     attempt + 1,
					"max_retries": c.MaxRetries,
				})
				time.Sleep(backoff)
				continue
			}
			return nil, fmt.Errorf("rate limit exceeded after %d retries", c.MaxRetries)
		}

		// Transient gateway/server errors: retry with backoff.
		if resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout {
			lastErr = fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
			if attempt < c.MaxRetries {
				backoff := c.calculateBackoffWithJitter(attempt)
				tflog.Warn(ctx, "Firezone API transient server error; retrying", map[string]interface{}{
					"status":      resp.StatusCode,
					"backoff":     backoff.String(),
					"attempt":     attempt + 1,
					"max_retries": c.MaxRetries,
				})
				time.Sleep(backoff)
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
		}

		return respBody, nil
	}

	return nil, lastErr
}

// retryAfterOrBackoff parses a Retry-After header (delay in seconds) and falls
// back to jittered exponential backoff when it is absent or unparseable.
func (c *Client) retryAfterOrBackoff(retryAfter string, attempt int) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return c.calculateBackoffWithJitter(attempt)
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := retryBaseDelay * time.Duration(math.Pow(2, float64(attempt)))
	if backoff > c.MaxBackoff {
		backoff = c.MaxBackoff
	}
	return backoff
}

func (c *Client) calculateBackoffWithJitter(attempt int) time.Duration {
	base := c.calculateBackoff(attempt)
	// Full-spectrum jitter of ±25% to avoid a thundering herd of retries.
	jitter := time.Duration(rand.Int63n(int64(base)/2+1)) - (base / 4)
	backoff := base + jitter
	if backoff < retryBaseDelay {
		backoff = retryBaseDelay
	}
	return backoff
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			return i
		}
	}
	return fallback
}
