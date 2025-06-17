package strategy

import (
	"testing"
	"time"

	"changeme/sims"
)

// TestEdgeCasesAndIntegration provides comprehensive edge case testing for the strategy engine
func TestEdgeCasesAndIntegration(t *testing.T) {
	// Create a basic config for testing using the DefaultConfig
	config := DefaultConfig()

	t.Run("ExtremeTelemetryValues", func(t *testing.T) {
		engine := NewRecommendationEngine(config)

		// Create telemetry with extreme values
		telemetry := createExtremeTelemetryData()

		// Should handle extreme values gracefully without panicking
		recommendation := engine.GenerateRecommendation(&telemetry)

		if recommendation == nil {
			t.Fatal("Should generate recommendation even with extreme values")
		}

		// Basic validation
		if recommendation.PrimaryStrategy == "" {
			t.Error("Should have a primary strategy")
		}

		if recommendation.ConfidenceLevel < 0 || recommendation.ConfidenceLevel > 1 {
			t.Errorf("Confidence level should be between 0 and 1, got %f", recommendation.ConfidenceLevel)
		}
	})

	t.Run("CacheIntegrationTest", func(t *testing.T) {
		cacheConfig := DefaultCacheConfig()
		cache := NewStrategyCache(cacheConfig)

		// Test basic cache operations
		testData := "test data"
		key := cache.Put(CacheTypeStrategy, testData, []string{"test"})

		retrieved, exists := cache.Get(key)
		if !exists {
			t.Error("Cache should return stored data")
		}

		if retrieved != testData {
			t.Error("Retrieved data should match stored data")
		}

		// Test cache statistics
		stats := cache.GetStats()
		if stats.TotalEntries != 1 {
			t.Errorf("Expected 1 cache entry, got %d", stats.TotalEntries)
		}

		// Test cache removal
		removed := cache.Remove(key)
		if !removed {
			t.Error("Should successfully remove cache entry")
		}

		_, exists = cache.Get(key)
		if exists {
			t.Error("Cache entry should not exist after removal")
		}
	})

	t.Run("ErrorHandlingIntegration", func(t *testing.T) {
		classifier := NewErrorClassifier()
		reporter := NewErrorReporter(10)

		// Create various error types
		errors := []*StrategyError{
			{Type: ErrorTypeNetwork, Code: "NET001", Message: "Network error", Retryable: true},
			{Type: ErrorTypeRateLimit, Code: "RATE001", Message: "Rate limit exceeded", Retryable: true},
			{Type: ErrorTypeAuthentication, Code: "AUTH001", Message: "Invalid API key", Retryable: false},
			{Type: ErrorTypeTimeout, Code: "TIME001", Message: "Request timeout", Retryable: true},
		}

		// Report all errors
		for _, err := range errors {
			reporter.ReportError(err)
		}

		// Check error statistics
		stats := reporter.GetErrorStats()
		if stats[ErrorTypeNetwork] != 1 {
			t.Errorf("Expected 1 network error, got %d", stats[ErrorTypeNetwork])
		}

		// Check recent errors
		recentErrors := reporter.GetRecentErrors(2)
		if len(recentErrors) != 2 {
			t.Errorf("Expected 2 recent errors, got %d", len(recentErrors))
		}

		// Test error classification with context
		contextData := map[string]interface{}{
			"operation":         "strategy_analysis",
			"telemetry_quality": 0.85,
		}

		testErr := &StrategyError{
			Type:    ErrorTypeValidation,
			Code:    "VAL001",
			Message: "Invalid telemetry data",
		}

		classified := classifier.ClassifyError(testErr, contextData)
		if classified.Context["operation"] != "strategy_analysis" {
			t.Error("Context should be preserved in classified error")
		}
	})

	t.Run("RecommendationEngineComprehensive", func(t *testing.T) {
		engine := NewRecommendationEngine(config)

		// Test with various race scenarios
		scenarios := []struct {
			name      string
			telemetry sims.TelemetryData
		}{
			{"LowFuel", createLowFuelScenario()},
			{"HighTireWear", createHighTireWearScenario()},
			{"PitWindowOpen", createPitWindowScenario()},
			{"EnduranceRace", createEnduranceRaceScenario()},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				recommendation := engine.GenerateRecommendation(&scenario.telemetry)

				if recommendation == nil {
					t.Fatalf("Should generate recommendation for %s scenario", scenario.name)
				}

				// Validate basic recommendation structure
				if recommendation.PrimaryStrategy == "" {
					t.Error("Should have primary strategy")
				}

				// Pit recommendation should exist for scenarios that need it
				if (scenario.name == "LowFuel" || scenario.name == "HighTireWear") && !recommendation.PitRecommendation.ShouldPit {
					t.Errorf("Should recommend pit for %s scenario", scenario.name)
				}
			})
		}
	})
}

// Helper functions to create test data

func createExtremeTelemetryData() sims.TelemetryData {
	return sims.TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: sims.SimulatorTypeIRacing,
		IsConnected:   true,
		Session: sims.SessionInfo{
			Type:             sims.SessionTypeRace,
			Format:           sims.RaceFormatSprint,
			Flag:             sims.SessionFlagGreen,
			TimeRemaining:    -1 * time.Hour, // Negative time (extreme)
			LapsRemaining:    -5,             // Negative laps (extreme)
			TotalLaps:        50,
			SessionTime:      2 * time.Hour,
			IsTimedSession:   false,
			IsLappedSession:  true,
			TrackName:        "Test Track",
			TrackLength:      5.0,
			AirTemperature:   -50.0, // Extreme cold
			TrackTemperature: 80.0,  // Extreme heat
		},
		Player: sims.PlayerData{
			Position:           1,
			CurrentLap:         30,
			LapDistancePercent: 150.0, // Over 100% (extreme)
			LastLapTime:        30 * time.Second,
			BestLapTime:        25 * time.Second,
			CurrentLapTime:     35 * time.Second,
			Fuel: sims.FuelData{
				Level:             -10.0, // Negative fuel (extreme)
				Capacity:          80.0,
				Percentage:        -12.5, // Negative percentage
				UsagePerLap:       2.1,
				EstimatedLapsLeft: -5, // Negative laps
			},
			Tires: sims.TireData{
				Compound: "Medium",
				FrontLeft: sims.TireWheelData{
					Temperature: 200.0, // Extreme heat
					Pressure:    50.0,  // Extreme pressure
					WearPercent: 150.0, // Over 100% wear
				},
				FrontRight: sims.TireWheelData{
					Temperature: -20.0, // Extreme cold
					Pressure:    5.0,   // Very low pressure
					WearPercent: 200.0, // Way over 100%
				},
				RearLeft: sims.TireWheelData{
					Temperature: 85.0,
					Pressure:    27.0,
					WearPercent: 35.0,
				},
				RearRight: sims.TireWheelData{
					Temperature: 87.0,
					Pressure:    26.5,
					WearPercent: 38.0,
				},
			},
			Speed:    500.0,   // Unrealistic speed
			RPM:      15000.0, // Very high RPM
			Gear:     -1,      // Negative gear
			Throttle: 150.0,   // Over 100%
			Brake:    -50.0,   // Negative brake
		},
	}
}

func createLowFuelScenario() sims.TelemetryData {
	data := createBasicTelemetryData()
	data.Player.Fuel.Level = 5.0       // Very low fuel
	data.Player.Fuel.Percentage = 6.25 // ~6% fuel remaining
	data.Player.Fuel.EstimatedLapsLeft = 2
	data.Player.Fuel.LowFuelWarning = true
	return data
}

func createHighTireWearScenario() sims.TelemetryData {
	data := createBasicTelemetryData()
	data.Player.Tires.FrontLeft.WearPercent = 85.0
	data.Player.Tires.FrontRight.WearPercent = 87.0
	data.Player.Tires.RearLeft.WearPercent = 82.0
	data.Player.Tires.RearRight.WearPercent = 84.0
	data.Player.Tires.WearLevel = sims.TireWearWorn
	return data
}

func createPitWindowScenario() sims.TelemetryData {
	data := createBasicTelemetryData()
	data.Player.Pit.PitWindowOpen = true
	data.Player.Pit.PitWindowLapsLeft = 5
	data.Player.CurrentLap = 25
	return data
}

func createEnduranceRaceScenario() sims.TelemetryData {
	data := createBasicTelemetryData()
	data.Session.Format = sims.RaceFormatEndurance
	data.Session.IsTimedSession = true
	data.Session.TimeRemaining = 2 * time.Hour // 2 hours remaining
	data.Session.TotalLaps = 200               // Long race
	data.Player.CurrentLap = 100
	return data
}

func createBasicTelemetryData() sims.TelemetryData {
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
			Fuel: sims.FuelData{
				Level:             45.5,
				Capacity:          80.0,
				Percentage:        56.875,
				UsagePerLap:       2.1,
				EstimatedLapsLeft: 21,
			},
			Tires: sims.TireData{
				Compound: "Medium",
				FrontLeft: sims.TireWheelData{
					Temperature: 85.0,
					Pressure:    27.5,
					WearPercent: 35.0,
				},
				FrontRight: sims.TireWheelData{
					Temperature: 87.0,
					Pressure:    27.3,
					WearPercent: 37.0,
				},
				RearLeft: sims.TireWheelData{
					Temperature: 82.0,
					Pressure:    26.8,
					WearPercent: 33.0,
				},
				RearRight: sims.TireWheelData{
					Temperature: 84.0,
					Pressure:    26.9,
					WearPercent: 34.0,
				},
				WearLevel: sims.TireWearGood,
				TempLevel: sims.TireTempOptimal,
			},
			Pit: sims.PitData{
				IsOnPitRoad:       false,
				IsInPitStall:      false,
				PitWindowOpen:     false,
				PitWindowLapsLeft: 10,
				LastPitLap:        15,
				EstimatedPitTime:  23 * time.Second,
			},
			Speed:    175.5,
			RPM:      7500.0,
			Gear:     5,
			Throttle: 85.0,
			Brake:    0.0,
		},
	}
}
