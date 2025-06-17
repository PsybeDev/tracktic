package strategy

import (
	"testing"
	"time"

	"changeme/sims"
)

func TestStrategyEngineEdgeCases(t *testing.T) {
	engine := NewStrategyEngine()

	t.Run("ExtremeWearValues", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Test with 100% tire wear
		telemetryData.Player.Tires.FrontLeft.WearPercent = 100.0
		telemetryData.Player.Tires.FrontRight.WearPercent = 100.0
		telemetryData.Player.Tires.RearLeft.WearPercent = 100.0
		telemetryData.Player.Tires.RearRight.WearPercent = 100.0

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should recommend immediate pit stop
		if !recommendation.PitRecommendation.ShouldPit {
			t.Error("Should recommend pit with 100% tire wear")
		}
	})

	t.Run("ZeroFuelLevel", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()
		telemetryData.Player.Fuel.Level = 0.0

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should recommend immediate pit stop
		if !recommendation.PitRecommendation.ShouldPit {
			t.Error("Should recommend pit with zero fuel")
		}
	})

	t.Run("ExtremeTireTemperatures", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Test tire temperature extremes
		telemetryData.Player.Tires.FrontLeft.Temperature = -10.0  // Extremely cold
		telemetryData.Player.Tires.FrontRight.Temperature = 200.0 // Extremely hot
		telemetryData.Player.Tires.RearLeft.Temperature = 80.0    // Normal
		telemetryData.Player.Tires.RearRight.Temperature = 85.0   // Normal

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should generate some recommendation about tire management
		if recommendation == nil {
			t.Error("Should generate recommendation with extreme tire temperatures")
		}
	})

	t.Run("InconsistentTelemetryData", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Create inconsistent data - current lap higher than total laps
		telemetryData.Player.CurrentLap = 100
		telemetryData.Session.TotalLaps = 50

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should still generate a recommendation despite inconsistency
		if recommendation == nil {
			t.Error("Should handle inconsistent telemetry data gracefully")
		}
	})

	t.Run("NegativeValues", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Set some negative values that shouldn't be negative
		telemetryData.Player.Speed = -50.0
		telemetryData.Player.RPM = -1000.0
		telemetryData.Player.Fuel.Level = -10.0

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should handle negative values gracefully
		if recommendation == nil {
			t.Error("Should handle negative values gracefully")
		}
	})
}

func TestCacheEdgeCases(t *testing.T) {
	config := DefaultCacheConfig()
	cache := NewStrategyCache(config)

	t.Run("ExcessiveDataSize", func(t *testing.T) {
		// Create very large telemetry data
		largeData := createValidTelemetryDataFixed()

		// Add many opponents to make data large
		for i := 0; i < 1000; i++ {
			opponent := sims.OpponentData{
				CarIndex:           i,
				DriverName:         "Driver" + string(rune(i)),
				CarNumber:          string(rune(i)),
				Position:           i + 1,
				CurrentLap:         10,
				LapDistancePercent: 50.0,
				LastLapTime:        90 * time.Second,
				BestLapTime:        85 * time.Second,
			}
			largeData.Opponents = append(largeData.Opponents, opponent)
		}

		err := cache.Store("large_data", &largeData)
		if err != nil {
			t.Logf("Cache rejected large data as expected: %v", err)
		}
	})

	t.Run("RapidCacheOperations", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Perform rapid cache operations
		for i := 0; i < 100; i++ {
			key := "rapid_test_" + string(rune(i))
			_ = cache.Store(key, &telemetryData)
			_, _ = cache.Get(key)
			cache.Delete(key)
		}

		// Cache should still be functional
		_ = cache.Store("final_test", &telemetryData)
		retrieved, exists := cache.Get("final_test")
		if !exists || retrieved == nil {
			t.Error("Cache should still be functional after rapid operations")
		}
	})
}

func TestErrorHandlingEdgeCases(t *testing.T) {
	classifier := NewErrorClassifier()
	reporter := NewErrorReporter(10)

	t.Run("ErrorChaining", func(t *testing.T) {
		// Create a chain of errors
		baseErr := NewStrategyError(ErrorTypeNetwork, "NET001", "Connection failed", nil)
		wrappedErr := NewStrategyError(ErrorTypeTimeout, "TIME001", "Request timed out", baseErr)
		finalErr := NewStrategyError(ErrorTypeUnknown, "UNK001", "General failure", wrappedErr)

		if finalErr.Unwrap() != wrappedErr {
			t.Error("Error unwrapping should work correctly")
		}
	})

	t.Run("ErrorReporting", func(t *testing.T) {
		// Report multiple errors
		for i := 0; i < 10; i++ {
			err := NewStrategyError(ErrorTypeNetwork, "NET001", "Test error", nil)
			reporter.ReportError(err)
		}

		stats := reporter.GetErrorStats()
		if stats[ErrorTypeNetwork] != 10 {
			t.Errorf("Expected 10 network errors, got %d", stats[ErrorTypeNetwork])
		}

		recentErrors := reporter.GetRecentErrors(5)
		if len(recentErrors) != 5 {
			t.Errorf("Expected 5 recent errors, got %d", len(recentErrors))
		}
	})

	t.Run("ErrorClassification", func(t *testing.T) {
		// Test error classification with context
		contextData := map[string]interface{}{
			"request_id": "test123",
			"operation":  "strategy_generation",
		}

		baseErr := NewStrategyError(ErrorTypeNetwork, "NET001", "Test error", nil)
		classified := classifier.ClassifyError(baseErr, contextData)

		if classified.Context["request_id"] != "test123" {
			t.Error("Context data should be preserved in classified error")
		}
	})
}

func TestRateLimiterEdgeCases(t *testing.T) {
	config := DefaultRateLimiterConfig()
	limiter := NewTokenBucketRateLimiter(config)

	t.Run("BurstRequests", func(t *testing.T) {
		// Try to make many requests at once
		successCount := 0
		for i := 0; i < 100; i++ {
			if limiter.Allow() {
				successCount++
			}
		}

		// Should allow some but not all requests
		if successCount == 0 {
			t.Error("Rate limiter should allow some requests")
		}
		if successCount == 100 {
			t.Error("Rate limiter should not allow all burst requests")
		}
	})

	t.Run("RateLimiterStats", func(t *testing.T) {
		stats := limiter.GetStats()

		// Stats should not be nil and should have sensible values
		if stats.TokensRemaining < 0 {
			t.Error("Tokens remaining should not be negative")
		}
		if stats.RequestsAllowed < 0 {
			t.Error("Requests allowed should not be negative")
		}
		if stats.RequestsRejected < 0 {
			t.Error("Requests rejected should not be negative")
		}
	})
}

func TestRecommendationEngineComplexScenarios(t *testing.T) {
	engine := NewRecommendationEngine()

	t.Run("MultipleStrategyOptions", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Create a scenario where multiple strategies are viable
		telemetryData.Player.Fuel.Level = 30.0 // Medium fuel
		telemetryData.Player.CurrentLap = 25   // Mid-race
		telemetryData.Session.TotalLaps = 50   // Half distance
		telemetryData.Player.Position = 5      // Competitive position

		// Set tire wear to moderate level
		telemetryData.Player.Tires.FrontLeft.WearPercent = 60.0
		telemetryData.Player.Tires.FrontRight.WearPercent = 62.0
		telemetryData.Player.Tires.RearLeft.WearPercent = 58.0
		telemetryData.Player.Tires.RearRight.WearPercent = 61.0

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should generate a comprehensive recommendation
		if recommendation == nil {
			t.Error("Should generate recommendation for complex scenario")
		}

		// Should have fuel and tire recommendations
		if recommendation.FuelRecommendation == nil {
			t.Error("Should have fuel recommendation")
		}
		if recommendation.TireRecommendation == nil {
			t.Error("Should have tire recommendation")
		}
	})

	t.Run("EnduranceRaceScenario", func(t *testing.T) {
		telemetryData := createValidTelemetryDataFixed()

		// Set up for endurance race
		telemetryData.Session.Format = sims.RaceFormatEndurance
		telemetryData.Session.IsTimedSession = true
		telemetryData.Session.TimeRemaining = 120 * time.Minute // 2 hours remaining
		telemetryData.Player.CurrentLap = 50

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should account for long race duration
		if recommendation == nil {
			t.Error("Should generate recommendation for endurance race")
		}
	})
}

func createValidTelemetryDataFixed() sims.TelemetryData {
	return sims.TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: sims.SimulatorTypeIRacing,
		IsConnected:   true,
		Session: sims.SessionInfo{
			Type:             sims.SessionTypeRace,
			Format:           sims.RaceFormatSprint,
			Flag:             sims.SessionFlagGreen,
			TimeRemaining:    30 * time.Minute,
			LapsRemaining:    20,
			TotalLaps:        50,
			SessionTime:      45 * time.Minute,
			IsTimedSession:   false,
			IsLappedSession:  true,
			TrackName:        "Silverstone GP",
			TrackLength:      5.891,
			AirTemperature:   22.0,
			TrackTemperature: 28.0,
		},
		Player: sims.PlayerData{
			Position:           5,
			CurrentLap:         30,
			LapDistancePercent: 25.0,
			LastLapTime:        88 * time.Second,
			BestLapTime:        85 * time.Second,
			CurrentLapTime:     45 * time.Second,
			GapToLeader:        15 * time.Second,
			GapToAhead:         3 * time.Second,
			GapToBehind:        2 * time.Second,
			Fuel: sims.FuelData{
				Level:             45.5,
				Capacity:          80.0,
				Percentage:        56.875,
				UsagePerLap:       2.1,
				UsagePerHour:      50.0,
				EstimatedLapsLeft: 21,
				EstimatedTimeLeft: 30 * time.Minute,
				LowFuelWarning:    false,
			},
			Tires: sims.TireData{
				Compound: "Medium",
				FrontLeft: sims.TireWheelData{
					Temperature: 85.0,
					Pressure:    27.5,
					WearPercent: 35.0,
					DirtLevel:   5.0,
				},
				FrontRight: sims.TireWheelData{
					Temperature: 87.0,
					Pressure:    27.3,
					WearPercent: 37.0,
					DirtLevel:   4.0,
				},
				RearLeft: sims.TireWheelData{
					Temperature: 82.0,
					Pressure:    26.8,
					WearPercent: 33.0,
					DirtLevel:   6.0,
				},
				RearRight: sims.TireWheelData{
					Temperature: 84.0,
					Pressure:    26.9,
					WearPercent: 34.0,
					DirtLevel:   5.5,
				},
				WearLevel: sims.TireWearGood,
				TempLevel: sims.TireTempOptimal,
			},
			Pit: sims.PitData{
				IsOnPitRoad:       false,
				IsInPitStall:      false,
				PitWindowOpen:     true,
				PitWindowLapsLeft: 10,
				LastPitLap:        15,
				LastPitTime:       25 * time.Second,
				EstimatedPitTime:  23 * time.Second,
				PitSpeedLimit:     80.0,
			},
			Speed:    175.5,
			RPM:      7500.0,
			Gear:     5,
			Throttle: 85.0,
			Brake:    0.0,
			Clutch:   0.0,
			Steering: -15.5,
		},
		Opponents: []sims.OpponentData{
			{
				CarIndex:           1,
				DriverName:         "John Doe",
				CarNumber:          "42",
				Position:           1,
				CurrentLap:         30,
				LapDistancePercent: 35.0,
				LastLapTime:        86 * time.Second,
				BestLapTime:        84 * time.Second,
				GapToPlayer:        -15 * time.Second,
				IsOnPitRoad:        false,
				IsInPitStall:       false,
				LastPitLap:         10,
				EstimatedPitTime:   23 * time.Second,
			},
		},
	}
}
