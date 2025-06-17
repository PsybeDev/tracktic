package strategy

import (
	"context"
	"testing"
	"time"
)

func TestStrategyEngineRateLimitingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Rate limiting behavior", func(t *testing.T) {
		// Create config with very low rate limits for testing
		config := DefaultConfig()
		config.MaxRequestsPerMinute = 2
		config.BurstLimit = 1
		config.RetryAttempts = 1
		config.APIKey = "test-key-for-rate-limit-testing"

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Check initial rate limiter stats
		stats := engine.GetRateLimiterStats()
		if stats.AvailableTokens != 1 {
			t.Errorf("Expected 1 available token initially, got %d", stats.AvailableTokens)
		}
		if stats.MaxRequestsPerMinute != 2 {
			t.Errorf("Expected max 2 requests per minute, got %d", stats.MaxRequestsPerMinute)
		}
	})

	t.Run("Error handling and reporting", func(t *testing.T) {
		config := DefaultConfig()
		config.APIKey = "" // Invalid API key to trigger auth error
		config.RetryAttempts = 1

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err == nil {
			defer engine.Close()
			t.Error("Expected error creating engine with invalid API key")
		}

		// Test with valid config but fake API key that will cause auth errors
		config.APIKey = "fake-api-key-that-will-fail"
		engine, err = NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Create minimal race data
		raceData := &RaceData{
			SessionType:    "race",
			TrackName:      "Test Track",
			CurrentLap:     5,
			Position:       3,
			FuelLevel:      75.0,
			TireWear:       25.0,
			TireCompound:   "medium",
			CurrentLapTime: 90.5,
			BestLapTime:    89.2,
			AverageLapTime: 90.1,
		}

		// This should fail due to invalid API key
		_, err = engine.AnalyzeStrategy(raceData, "routine")
		if err == nil {
			t.Error("Expected error with fake API key")
		}

		// Check that error was reported
		errorStats := engine.GetErrorStats()
		if len(errorStats) == 0 {
			t.Error("Expected error statistics to be populated")
		}

		recentErrors := engine.GetRecentErrors(5)
		if len(recentErrors) == 0 {
			t.Error("Expected recent errors to be recorded")
		}
	})

	t.Run("Config update", func(t *testing.T) {
		config := DefaultConfig()
		config.MaxRequestsPerMinute = 10
		config.BurstLimit = 3

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Check initial stats
		initialStats := engine.GetRateLimiterStats()
		if initialStats.MaxRequestsPerMinute != 10 {
			t.Errorf("Expected 10 requests per minute initially, got %d", initialStats.MaxRequestsPerMinute)
		}

		// Update config
		newConfig := DefaultConfig()
		newConfig.MaxRequestsPerMinute = 20
		newConfig.BurstLimit = 5
		newConfig.APIKey = config.APIKey

		err = engine.UpdateConfig(newConfig)
		if err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}

		// Check updated stats
		updatedStats := engine.GetRateLimiterStats()
		if updatedStats.MaxRequestsPerMinute != 20 {
			t.Errorf("Expected 20 requests per minute after update, got %d", updatedStats.MaxRequestsPerMinute)
		}
		if updatedStats.BurstLimit != 5 {
			t.Errorf("Expected burst limit 5 after update, got %d", updatedStats.BurstLimit)
		}
	})

	t.Run("Health check", func(t *testing.T) {
		config := DefaultConfig()
		config.APIKey = "fake-api-key" // Will fail health check

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Health check should fail with fake API key
		err = engine.HealthCheck()
		if err == nil {
			t.Error("Expected health check to fail with fake API key")
		}
	})
}

func TestStrategyEngineErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Recovery from temporary failures", func(t *testing.T) {
		config := DefaultConfig()
		config.RetryAttempts = 3
		config.RetryDelay = 100 * time.Millisecond
		config.APIKey = "test-key-for-recovery-testing"

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Test that the engine properly handles and reports different error types
		initialErrorCount := len(engine.GetErrorStats())

		// Simulate some analysis attempts that will fail
		raceData := &RaceData{
			SessionType:  "race",
			TrackName:    "Test Track",
			CurrentLap:   1,
			Position:     1,
			FuelLevel:    100.0,
			TireWear:     0.0,
			TireCompound: "soft",
		}

		// This will likely fail due to fake API key, but should exercise error handling
		_, err = engine.AnalyzeStrategy(raceData, "routine")

		// Check that errors were properly classified and reported
		finalErrorCount := len(engine.GetErrorStats())
		if finalErrorCount <= initialErrorCount {
			t.Error("Expected error count to increase after failed analysis")
		}

		recentErrors := engine.GetRecentErrors(10)
		if len(recentErrors) == 0 {
			t.Error("Expected recent errors to be recorded")
		}

		// Verify error has proper classification
		lastError := recentErrors[len(recentErrors)-1]
		if lastError.Type == ErrorTypeUnknown {
			t.Error("Error should be properly classified, not unknown")
		}
		if lastError.Timestamp.IsZero() {
			t.Error("Error should have timestamp")
		}
	})
}

func TestRateLimiterIntegration(t *testing.T) {
	t.Run("Rate limiter blocks excessive requests", func(t *testing.T) {
		rateLimiter := NewRateLimiter(6, 2) // 6 per minute, burst of 2

		// Should allow 2 immediate requests
		if !rateLimiter.Allow() {
			t.Error("First request should be allowed")
		}
		if !rateLimiter.Allow() {
			t.Error("Second request should be allowed")
		}

		// Third request should be blocked
		if rateLimiter.Allow() {
			t.Error("Third request should be blocked")
		}

		// Stats should reflect current state
		stats := rateLimiter.GetStats()
		if stats.AvailableTokens != 0 {
			t.Errorf("Expected 0 available tokens, got %d", stats.AvailableTokens)
		}
		if stats.RequestsInLastMinute != 2 {
			t.Errorf("Expected 2 requests in last minute, got %d", stats.RequestsInLastMinute)
		}
	})

	t.Run("Wait respects context cancellation", func(t *testing.T) {
		rateLimiter := NewRateLimiter(1, 1) // Very restrictive

		// Use up the token
		rateLimiter.Allow()

		// Create context that cancels quickly
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := rateLimiter.Wait(ctx)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("Wait should return error when context is cancelled")
		}

		if elapsed > 200*time.Millisecond {
			t.Errorf("Wait should respect context timeout, took %v", elapsed)
		}
	})
}
