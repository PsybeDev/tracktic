package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter for API requests
type RateLimiter struct {
	maxRequests int           // Maximum requests per minute
	burstLimit  int           // Maximum burst requests
	window      time.Duration // Time window for rate limiting (1 minute)

	// Token bucket implementation
	tokens     int
	lastRefill time.Time
	mutex      sync.Mutex

	// Request tracking for detailed monitoring
	requestHistory []time.Time
	historyMutex   sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with the specified configuration
func NewRateLimiter(maxRequestsPerMinute, burstLimit int) *RateLimiter {
	rl := &RateLimiter{
		maxRequests:    maxRequestsPerMinute,
		burstLimit:     burstLimit,
		window:         time.Minute,
		tokens:         burstLimit, // Start with full burst capacity
		lastRefill:     time.Now(),
		requestHistory: make([]time.Time, 0),
	}

	return rl
}

// Allow checks if a request can be made immediately
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.refillTokens()

	if rl.tokens > 0 {
		rl.tokens--
		rl.recordRequest()
		return true
	}

	return false
}

// Wait blocks until a request can be made, respecting the rate limit
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		// Calculate wait time until next token is available
		waitTime := rl.nextTokenAvailableIn()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to check again
		}
	}
}

// WaitN waits for n tokens to become available
func (rl *RateLimiter) WaitN(ctx context.Context, n int) error {
	if n > rl.burstLimit {
		return fmt.Errorf("requested tokens (%d) exceeds burst limit (%d)", n, rl.burstLimit)
	}

	for {
		rl.mutex.Lock()
		rl.refillTokens()

		if rl.tokens >= n {
			rl.tokens -= n
			for i := 0; i < n; i++ {
				rl.recordRequest()
			}
			rl.mutex.Unlock()
			return nil
		}
		rl.mutex.Unlock()

		// Calculate wait time until n tokens are available
		waitTime := rl.nextNTokensAvailableIn(n)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop to check again
		}
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.mutex.Lock()
	rl.refillTokens()
	availableTokens := rl.tokens
	rl.mutex.Unlock()

	rl.historyMutex.RLock()
	defer rl.historyMutex.RUnlock()

	now := time.Now()
	recentRequests := 0

	// Count requests in the last minute
	for _, reqTime := range rl.requestHistory {
		if now.Sub(reqTime) <= rl.window {
			recentRequests++
		}
	}

	return RateLimiterStats{
		MaxRequestsPerMinute: rl.maxRequests,
		BurstLimit:           rl.burstLimit,
		AvailableTokens:      availableTokens,
		RequestsInLastMinute: recentRequests,
		NextTokenIn:          rl.nextTokenAvailableIn(),
	}
}

// RateLimiterStats provides information about the current state of the rate limiter
type RateLimiterStats struct {
	MaxRequestsPerMinute int
	BurstLimit           int
	AvailableTokens      int
	RequestsInLastMinute int
	NextTokenIn          time.Duration
}

// Internal methods

func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Calculate how many tokens to add based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.maxRequests) / 60.0)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.burstLimit {
			rl.tokens = rl.burstLimit
		}
		rl.lastRefill = now
	}
}

func (rl *RateLimiter) recordRequest() {
	rl.historyMutex.Lock()
	defer rl.historyMutex.Unlock()

	now := time.Now()
	rl.requestHistory = append(rl.requestHistory, now)

	// Clean old history entries (keep last 2 minutes for safety)
	cutoff := now.Add(-2 * rl.window)
	i := 0
	for i < len(rl.requestHistory) && rl.requestHistory[i].Before(cutoff) {
		i++
	}
	rl.requestHistory = rl.requestHistory[i:]
}

func (rl *RateLimiter) nextTokenAvailableIn() time.Duration {
	if rl.tokens > 0 {
		return 0
	}

	// Time per token = 60 seconds / maxRequests
	timePerToken := time.Duration(60.0/float64(rl.maxRequests)) * time.Second
	return timePerToken
}

func (rl *RateLimiter) nextNTokensAvailableIn(n int) time.Duration {
	if rl.tokens >= n {
		return 0
	}

	tokensNeeded := n - rl.tokens
	timePerToken := time.Duration(60.0/float64(rl.maxRequests)) * time.Second
	return time.Duration(tokensNeeded) * timePerToken
}

// Reset clears the rate limiter state (useful for testing)
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.tokens = rl.burstLimit
	rl.lastRefill = time.Now()

	rl.historyMutex.Lock()
	rl.requestHistory = rl.requestHistory[:0]
	rl.historyMutex.Unlock()
}
