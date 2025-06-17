package strategy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	t.Run("Allow requests within burst limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 3) // 10 per minute, burst of 3

		// Should allow 3 requests immediately
		for i := 0; i < 3; i++ {
			if !rl.Allow() {
				t.Errorf("Request %d should be allowed within burst limit", i+1)
			}
		}

		// 4th request should be denied
		if rl.Allow() {
			t.Error("4th request should be denied when burst limit exceeded")
		}
	})

	t.Run("Refill tokens over time", func(t *testing.T) {
		rl := NewRateLimiter(60, 2) // 60 per minute (1 per second), burst of 2

		// Use up burst capacity
		rl.Allow()
		rl.Allow()

		// Should be denied immediately
		if rl.Allow() {
			t.Error("Request should be denied after burst exhausted")
		}

		// Manually advance time by setting lastRefill
		rl.lastRefill = time.Now().Add(-2 * time.Second)

		// Should allow 2 more requests after refill
		if !rl.Allow() {
			t.Error("Request should be allowed after token refill")
		}
		if !rl.Allow() {
			t.Error("Second request should be allowed after token refill")
		}

		// Should be denied again
		if rl.Allow() {
			t.Error("Request should be denied after using refilled tokens")
		}
	})

	t.Run("Wait for token availability", func(t *testing.T) {
		rl := NewRateLimiter(120, 1) // 120 per minute (2 per second), burst of 1

		// Use up the single token
		if !rl.Allow() {
			t.Error("First request should be allowed")
		}

		// Wait should complete within 1 second
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		start := time.Now()
		err := rl.Wait(ctx)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Wait should not return error: %v", err)
		}

		if elapsed < 400*time.Millisecond || elapsed > 1*time.Second {
			t.Errorf("Wait time should be around 500ms, got %v", elapsed)
		}
	})

	t.Run("Wait with context cancellation", func(t *testing.T) {
		rl := NewRateLimiter(1, 1) // Very slow rate

		// Use up the token
		rl.Allow()

		// Create context that will be cancelled quickly
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := rl.Wait(ctx)
		if err == nil {
			t.Error("Wait should return error when context is cancelled")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("Get stats", func(t *testing.T) {
		rl := NewRateLimiter(60, 3)

		stats := rl.GetStats()
		if stats.MaxRequestsPerMinute != 60 {
			t.Errorf("Expected max requests 60, got %d", stats.MaxRequestsPerMinute)
		}
		if stats.BurstLimit != 3 {
			t.Errorf("Expected burst limit 3, got %d", stats.BurstLimit)
		}
		if stats.AvailableTokens != 3 {
			t.Errorf("Expected 3 available tokens, got %d", stats.AvailableTokens)
		}

		// Use one token
		rl.Allow()

		stats = rl.GetStats()
		if stats.AvailableTokens != 2 {
			t.Errorf("Expected 2 available tokens after using one, got %d", stats.AvailableTokens)
		}
		if stats.RequestsInLastMinute != 1 {
			t.Errorf("Expected 1 request in last minute, got %d", stats.RequestsInLastMinute)
		}
	})
}

func TestErrorClassifier(t *testing.T) {
	classifier := NewErrorClassifier()

	t.Run("Classify network errors", func(t *testing.T) {
		netErr := &net.DNSError{Err: "no such host", Name: "example.com"}
		stratError := classifier.ClassifyError(netErr, nil)

		if stratError.Type != ErrorTypeNetwork {
			t.Errorf("Expected network error type, got %s", stratError.Type.String())
		}
		if !stratError.Retryable {
			t.Error("Network errors should be retryable")
		}
	})

	t.Run("Classify timeout errors", func(t *testing.T) {
		timeoutErr := context.DeadlineExceeded
		stratError := classifier.ClassifyError(timeoutErr, nil)

		if stratError.Type != ErrorTypeTimeout {
			t.Errorf("Expected timeout error type, got %s", stratError.Type.String())
		}
		if !stratError.Retryable {
			t.Error("Timeout errors should be retryable")
		}
	})

	t.Run("Classify rate limit errors", func(t *testing.T) {
		rateLimitErr := errors.New("rate limit exceeded")
		stratError := classifier.ClassifyError(rateLimitErr, nil)

		if stratError.Type != ErrorTypeRateLimit {
			t.Errorf("Expected rate limit error type, got %s", stratError.Type.String())
		}
		if !stratError.Retryable {
			t.Error("Rate limit errors should be retryable")
		}
		if stratError.RetryAfter != 60*time.Second {
			t.Errorf("Expected 60s retry delay, got %v", stratError.RetryAfter)
		}
	})

	t.Run("Classify authentication errors", func(t *testing.T) {
		authErr := errors.New("unauthorized: invalid api key")
		stratError := classifier.ClassifyError(authErr, nil)

		if stratError.Type != ErrorTypeAuthentication {
			t.Errorf("Expected authentication error type, got %s", stratError.Type.String())
		}
		if stratError.Retryable {
			t.Error("Authentication errors should not be retryable")
		}
	})

	t.Run("Classify server errors", func(t *testing.T) {
		serverErr := errors.New("internal server error")
		stratError := classifier.ClassifyError(serverErr, nil)

		if stratError.Type != ErrorTypeInternalServer {
			t.Errorf("Expected server error type, got %s", stratError.Type.String())
		}
		if !stratError.Retryable {
			t.Error("Server errors should be retryable")
		}
	})

	t.Run("Include context data", func(t *testing.T) {
		contextData := map[string]interface{}{
			"operation": "test",
			"attempt":   1,
		}

		err := errors.New("test error")
		stratError := classifier.ClassifyError(err, contextData)

		if stratError.Context["operation"] != "test" {
			t.Error("Context data should be preserved")
		}
		if stratError.Context["attempt"] != 1 {
			t.Error("Context data should be preserved")
		}
	})
}

func TestRetryPolicy(t *testing.T) {
	policy := NewDefaultRetryPolicy()

	t.Run("Should retry network errors", func(t *testing.T) {
		stratError := &StrategyError{
			Type:      ErrorTypeNetwork,
			Retryable: true,
		}

		if !policy.ShouldRetry(stratError, 0) {
			t.Error("Should retry network errors on first attempt")
		}
		if !policy.ShouldRetry(stratError, 1) {
			t.Error("Should retry network errors on second attempt")
		}
		if !policy.ShouldRetry(stratError, 2) {
			t.Error("Should retry network errors on third attempt")
		}
		if policy.ShouldRetry(stratError, 3) {
			t.Error("Should not retry network errors after max attempts")
		}
	})

	t.Run("Should not retry authentication errors", func(t *testing.T) {
		stratError := &StrategyError{
			Type:      ErrorTypeAuthentication,
			Retryable: false,
		}

		if policy.ShouldRetry(stratError, 0) {
			t.Error("Should not retry authentication errors")
		}
	})

	t.Run("Calculate backoff delays", func(t *testing.T) {
		stratError := &StrategyError{
			Type:       ErrorTypeNetwork,
			RetryAfter: 1 * time.Second,
		}

		// Test exponential backoff with jitter
		policy.BackoffStrategy = BackoffExponential

		delay1 := policy.CalculateBackoff(stratError, 0)
		delay2 := policy.CalculateBackoff(stratError, 1)
		delay3 := policy.CalculateBackoff(stratError, 2)

		if delay1 != 1*time.Second {
			t.Errorf("First delay should be 1s, got %v", delay1)
		}
		if delay2 != 2*time.Second {
			t.Errorf("Second delay should be 2s, got %v", delay2)
		}
		if delay3 != 4*time.Second {
			t.Errorf("Third delay should be 4s, got %v", delay3)
		}
	})
}

func TestErrorReporter(t *testing.T) {
	reporter := NewErrorReporter(5) // Keep last 5 errors

	t.Run("Report and count errors", func(t *testing.T) {
		// Report different types of errors
		reporter.ReportError(&StrategyError{Type: ErrorTypeNetwork})
		reporter.ReportError(&StrategyError{Type: ErrorTypeNetwork})
		reporter.ReportError(&StrategyError{Type: ErrorTypeRateLimit})

		stats := reporter.GetErrorStats()
		if stats[ErrorTypeNetwork] != 2 {
			t.Errorf("Expected 2 network errors, got %d", stats[ErrorTypeNetwork])
		}
		if stats[ErrorTypeRateLimit] != 1 {
			t.Errorf("Expected 1 rate limit error, got %d", stats[ErrorTypeRateLimit])
		}
	})

	t.Run("Maintain error history", func(t *testing.T) {
		reporter := NewErrorReporter(3) // Keep last 3 errors

		// Report 5 errors
		for i := 0; i < 5; i++ {
			reporter.ReportError(&StrategyError{
				Type:    ErrorTypeNetwork,
				Message: fmt.Sprintf("Error %d", i),
			})
		}

		recentErrors := reporter.GetRecentErrors(10)
		if len(recentErrors) != 3 {
			t.Errorf("Expected 3 errors in history, got %d", len(recentErrors))
		}

		// Should have the last 3 errors (2, 3, 4)
		if recentErrors[0].Message != "Error 2" {
			t.Errorf("Expected 'Error 2', got '%s'", recentErrors[0].Message)
		}
		if recentErrors[2].Message != "Error 4" {
			t.Errorf("Expected 'Error 4', got '%s'", recentErrors[2].Message)
		}
	})

	t.Run("Get limited recent errors", func(t *testing.T) {
		reporter := NewErrorReporter(10)

		// Report 5 errors
		for i := 0; i < 5; i++ {
			reporter.ReportError(&StrategyError{Type: ErrorTypeNetwork})
		}

		recentErrors := reporter.GetRecentErrors(3)
		if len(recentErrors) != 3 {
			t.Errorf("Expected 3 errors when limiting to 3, got %d", len(recentErrors))
		}
	})
}
