// Rate Limiting and Error Handling Demo Test
// This demonstrates the enhanced rate limiting and error handling capabilities
// Run from strategy/: go test -v -run TestRateLimitingDemo
package strategy

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRateLimitingDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo test in short mode")
	}

	fmt.Println("\n=== TrackTic AI Race Strategist - Rate Limiting & Error Handling Demo ===")
	fmt.Println()

	// Create a config with rate limiting for demonstration
	config := DefaultConfig()
	config.MaxRequestsPerMinute = 6 // Conservative rate limit
	config.BurstLimit = 2           // Allow 2 immediate requests
	config.RetryAttempts = 2        // Moderate retry attempts
	config.RequestTimeout = 15 * time.Second

	// Use fake API key for demo to show error handling
	fmt.Println("⚠️  Using fake API key for demonstration of error handling...")
	config.APIKey = "demo-fake-api-key-for-error-handling"

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	fmt.Printf("✅ Strategy engine created with rate limiting:\n")
	fmt.Printf("   • Max requests per minute: %d\n", config.MaxRequestsPerMinute)
	fmt.Printf("   • Burst limit: %d\n", config.BurstLimit)
	fmt.Printf("   • Retry attempts: %d\n", config.RetryAttempts)
	fmt.Printf("   • Request timeout: %v\n", config.RequestTimeout)
	fmt.Println()

	// Demonstrate rate limiter stats
	fmt.Println("📊 Initial Rate Limiter Status:")
	printRateLimiterStatsDemo(engine)

	// Create sample race data for testing
	raceData := createSampleRaceDataDemo()

	// Demonstration 1: Rate limiting in action
	fmt.Println("🚥 Demonstration 1: Rate Limiting Behavior")
	fmt.Println("Making multiple rapid requests to demonstrate rate limiting...")

	for i := 1; i <= 4; i++ {
		fmt.Printf("\n🔄 Request #%d at %s\n", i, time.Now().Format("15:04:05.000"))

		start := time.Now()
		_, err := engine.AnalyzeStrategy(raceData, "routine")
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("❌ Request failed after %v: %v\n", elapsed, err)
		} else {
			fmt.Printf("✅ Request succeeded after %v\n", elapsed)
		}

		// Show rate limiter stats after each request
		stats := engine.GetRateLimiterStats()
		fmt.Printf("   Rate limiter: %d tokens available, next token in %v\n",
			stats.AvailableTokens, stats.NextTokenIn)
	}

	fmt.Println("\n📊 Rate Limiter Status After Requests:")
	printRateLimiterStatsDemo(engine)

	// Demonstration 2: Error handling and classification
	fmt.Println("\n🔍 Demonstration 2: Error Handling & Classification")

	// Show accumulated errors
	errorStats := engine.GetErrorStats()
	if len(errorStats) > 0 {
		fmt.Println("Error statistics:")
		for errorType, count := range errorStats {
			fmt.Printf("   • %s: %d occurrences\n", errorType.String(), count)
		}
	} else {
		fmt.Println("No errors recorded yet.")
	}

	// Show recent errors with details
	recentErrors := engine.GetRecentErrors(5)
	if len(recentErrors) > 0 {
		fmt.Printf("\n📝 Recent Errors (%d shown):\n", len(recentErrors))
		for i, err := range recentErrors {
			fmt.Printf("   %d. [%s] %s: %s\n",
				i+1, err.Timestamp.Format("15:04:05"), err.Type.String(), err.Message)
			if err.Retryable {
				fmt.Printf("      Retryable: Yes (retry after %v)\n", err.GetRetryAfter())
			} else {
				fmt.Printf("      Retryable: No\n")
			}
		}
	}

	// Demonstration 3: Health check
	fmt.Println("\n🏥 Demonstration 3: Health Check")
	err = engine.HealthCheck()
	if err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
		fmt.Println("   This is expected with a fake API key.")
	} else {
		fmt.Println("✅ Health check passed - all systems operational")
	}

	// Demonstration 4: Config updates
	fmt.Println("\n⚙️  Demonstration 4: Configuration Updates")
	fmt.Println("Updating rate limiting configuration...")

	newConfig := DefaultConfig()
	newConfig.MaxRequestsPerMinute = 12 // Double the rate limit
	newConfig.BurstLimit = 4            // Double the burst
	newConfig.APIKey = config.APIKey    // Keep the same API key

	err = engine.UpdateConfig(newConfig)
	if err != nil {
		fmt.Printf("❌ Failed to update config: %v\n", err)
	} else {
		fmt.Println("✅ Configuration updated successfully")
		fmt.Println("📊 Updated Rate Limiter Status:")
		printRateLimiterStatsDemo(engine)
	}

	fmt.Println("\n🎯 Demonstration Complete!")
	fmt.Println("Key features demonstrated:")
	fmt.Println("  ✅ Token bucket rate limiting with burst capacity")
	fmt.Println("  ✅ Intelligent error classification and retry policies")
	fmt.Println("  ✅ Comprehensive error reporting and statistics")
	fmt.Println("  ✅ Real-time monitoring and health checks")
	fmt.Println("  ✅ Dynamic configuration updates")
	fmt.Println("  ✅ Context-aware timeout handling")
}

func printRateLimiterStatsDemo(engine *StrategyEngine) {
	stats := engine.GetRateLimiterStats()
	fmt.Printf("   • Available tokens: %d / %d\n", stats.AvailableTokens, stats.BurstLimit)
	fmt.Printf("   • Requests in last minute: %d / %d\n", stats.RequestsInLastMinute, stats.MaxRequestsPerMinute)
	fmt.Printf("   • Next token available in: %v\n", stats.NextTokenIn)
}

func createSampleRaceDataDemo() *RaceData {
	return &RaceData{
		SessionType:      "race",
		SessionTime:      1200, // 20 minutes into race
		SessionTimeLeft:  2400, // 40 minutes remaining
		CurrentLap:       15,
		Position:         4,
		FuelLevel:        78.5,
		FuelConsumption:  2.1,
		TireWear:         35.0,
		TireCompound:     "medium",
		CurrentLapTime:   92.3,
		BestLapTime:      91.1,
		AverageLapTime:   92.0,
		RecentLapTimes:   []float64{92.1, 91.8, 92.4, 91.9, 92.3},
		TrackName:        "Silverstone GP",
		TrackTemp:        28.5,
		AirTemp:          22.0,
		Weather:          "dry",
		WeatherForecast:  "stable",
		TotalLaps:        52,
		RemainingLaps:    37,
		SafetyCarActive:  false,
		YellowFlagSector: 0,
		Opponents: []OpponentData{
			{Position: 3, Name: "Driver A", GapToPlayer: -2.1, LastLapTime: 91.8, TireAge: 12, RecentPitLap: 8},
			{Position: 5, Name: "Driver B", GapToPlayer: 1.8, LastLapTime: 92.5, TireAge: 15, RecentPitLap: 0},
		},
	}
}
