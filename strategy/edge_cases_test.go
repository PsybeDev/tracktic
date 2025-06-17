package strategy

import (
	"context"
	"testing"

	"changeme/sims"
)

// TestStrategyEngineEdgeCases tests edge cases for strategy engine functionality
func TestStrategyEngineEdgeCases(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key" // Use dummy key for testing

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	t.Run("EmptyRaceData", func(t *testing.T) {
		// Test with minimal race data
		raceData := &RaceData{}

		analysis, err := engine.AnalyzeStrategy(raceData, "routine")
		if err == nil {
			t.Error("Expected error with empty race data")
		}

		if analysis != nil {
			t.Error("Expected nil analysis with invalid data")
		}
	})

	t.Run("InvalidAnalysisType", func(t *testing.T) {
		raceData := createValidRaceData()

		analysis, err := engine.AnalyzeStrategy(raceData, "invalid_type")
		if err == nil {
			t.Error("Expected error with invalid analysis type")
		}

		if analysis != nil {
			t.Error("Expected nil analysis with invalid type")
		}
	})

	t.Run("ExtremeSessionTimeValues", func(t *testing.T) {
		raceData := createValidRaceData()

		// Test with negative session time
		raceData.SessionTimeLeft = -100
		_, err := engine.AnalyzeStrategy(raceData, "routine")
		if err == nil {
			t.Error("Expected error with negative session time")
		}

		// Test with zero session time
		raceData.SessionTimeLeft = 0
		_, err = engine.AnalyzeStrategy(raceData, "routine")
		if err == nil {
			t.Error("Expected error with zero session time")
		}
	})

	t.Run("CacheKeyGeneration", func(t *testing.T) {
		raceData1 := createValidRaceData()
		raceData2 := createValidRaceData()

		// Same data should generate same cache key
		key1 := engine.cache.generateKey(CacheTypeStrategy, raceData1)
		key2 := engine.cache.generateKey(CacheTypeStrategy, raceData2)

		if key1 != key2 {
			t.Error("Same race data should generate same cache key")
		}

		// Different data should generate different cache keys
		raceData2.CurrentLap = 99
		key3 := engine.cache.generateKey(CacheTypeStrategy, raceData2)

		if key1 == key3 {
			t.Error("Different race data should generate different cache keys")
		}
	})
}

// TestRecommendationEngineEdgeCases tests edge cases for recommendation engine
func TestRecommendationEngineEdgeCases(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	t.Run("EmptyTelemetryHistory", func(t *testing.T) {
		// Test with no telemetry history
		telemetryData := createValidTelemetryData()
		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should still generate a recommendation, but with lower confidence
		if recommendation == nil {
			t.Error("Expected recommendation even with no history")
		}

		if recommendation.ConfidenceLevel > 0.5 {
			t.Errorf("Expected low confidence with no history, got %f", recommendation.ConfidenceLevel)
		}
	})
	t.Run("ExtremeWearValues", func(t *testing.T) {
		telemetryData := createValidTelemetryData()

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
		telemetryData := createValidTelemetryData()
		telemetryData.Player.Fuel.Level = 0.0

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should recommend immediate pit stop
		if !recommendation.PitRecommendation.ShouldPit {
			t.Error("Should recommend pit with zero fuel")
		}

		// Should flag as high risk
		if recommendation.RiskAssessment != "high" {
			t.Errorf("Expected high risk with zero fuel, got %s", recommendation.RiskAssessment)
		}
	})

	t.Run("LapAnalysisConsistency", func(t *testing.T) {
		// Add telemetry with consistent lap times
		for i := 0; i < 10; i++ {
			snapshot := createTelemetrySnapshot(i)
			snapshot.CarData.LapTime = 85.0 // Consistent lap time
			engine.AddTelemetrySnapshot(&snapshot)
		}

		telemetryData := createValidTelemetryData()
		telemetryData.CarData.LapTime = 85.0

		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should have high confidence due to consistency
		if recommendation.ConfidenceLevel < 0.8 {
			t.Errorf("Expected high confidence with consistent times, got %f", recommendation.ConfidenceLevel)
		}
	})

	t.Run("TelemetryHistoryLimit", func(t *testing.T) {
		engine := NewRecommendationEngine(config)

		// Add more than the history limit
		for i := 0; i < 1500; i++ { // Assuming 1000 is the limit
			snapshot := createTelemetrySnapshot(i)
			engine.AddTelemetrySnapshot(&snapshot)
		}

		// Should not exceed the limit
		if len(engine.telemetryHistory) > 1000 {
			t.Errorf("Telemetry history exceeded limit: %d", len(engine.telemetryHistory))
		}
	})
}

// TestCacheEdgeCases tests edge cases for caching functionality
func TestCacheEdgeCases(t *testing.T) {
	config := DefaultConfig()
	config.CacheConfig.MaxEntries = 5 // Small cache for testing

	cache := NewStrategyCache(config.CacheConfig)
	defer cache.Close()

	t.Run("CacheEviction", func(t *testing.T) {
		// Fill cache beyond capacity
		for i := 0; i < 10; i++ {
			data := &StrategyAnalysis{PrimaryStrategy: "Test"}
			cache.Put(CacheTypeStrategy, data, []string{})
		}

		stats := cache.GetStats()
		if stats.TotalEntries > 5 {
			t.Errorf("Cache exceeded max entries: %d", stats.TotalEntries)
		}
	})

	t.Run("NilData", func(t *testing.T) {
		// Test with nil data
		key := cache.Put(CacheTypeStrategy, nil, []string{})
		if key == "" {
			t.Error("Should generate key even for nil data")
		}

		retrieved, found := cache.Get(key)
		if !found {
			t.Error("Should find cached nil data")
		}

		if retrieved != nil {
			t.Error("Retrieved data should be nil")
		}
	})

	t.Run("TaggedInvalidation", func(t *testing.T) {
		// Store data with tags
		data1 := &StrategyAnalysis{PrimaryStrategy: "Test1"}
		data2 := &StrategyAnalysis{PrimaryStrategy: "Test2"}

		key1 := cache.Put(CacheTypeStrategy, data1, []string{"lap:10", "session:race"})
		key2 := cache.Put(CacheTypeStrategy, data2, []string{"lap:11", "session:race"})

		// Invalidate by tag
		count := cache.RemoveByTag("session:race")
		if count != 2 {
			t.Errorf("Expected to remove 2 entries, removed %d", count)
		}

		// Should not find the data anymore
		_, found1 := cache.Get(key1)
		_, found2 := cache.Get(key2)

		if found1 || found2 {
			t.Error("Data should be invalidated by tag")
		}
	})

	t.Run("EmptyTags", func(t *testing.T) {
		data := &StrategyAnalysis{PrimaryStrategy: "Test"}
		key := cache.Put(CacheTypeStrategy, data, []string{})

		if key == "" {
			t.Error("Should generate key with empty tags")
		}

		retrieved, found := cache.Get(key)
		if !found {
			t.Error("Should find data with empty tags")
		}

		if retrieved == nil {
			t.Error("Retrieved data should not be nil")
		}
	})
}

// TestErrorHandlingEdgeCases tests edge cases for error handling
func TestErrorHandlingEdgeCases(t *testing.T) {
	classifier := NewErrorClassifier()
	reporter := NewErrorReporter(5) // Small capacity for testing

	t.Run("UnknownErrorType", func(t *testing.T) {
		// Test with unknown error
		unknownErr := &CustomError{message: "Unknown error"}
		errorType := classifier.ClassifyError(unknownErr, map[string]interface{}{})

		if errorType != ErrorTypeUnknown {
			t.Errorf("Expected unknown error type, got %s", errorType)
		}
	})

	t.Run("ErrorReporterCapacity", func(t *testing.T) {
		// Fill reporter beyond capacity
		for i := 0; i < 10; i++ {
			err := &StrategyError{
				Type:    ErrorTypeNetwork,
				Message: "Test error",
			}
			reporter.ReportError(err, map[string]interface{}{
				"index": i,
			})
		}

		stats := reporter.GetErrorStatistics()
		// Should not exceed capacity
		if len(reporter.errors) > 5 {
			t.Errorf("Error reporter exceeded capacity: %d", len(reporter.errors))
		}

		// But should track total count
		if stats.TotalErrors != 10 {
			t.Errorf("Expected total errors 10, got %d", stats.TotalErrors)
		}
	})

	t.Run("NilErrorContext", func(t *testing.T) {
		err := &StrategyError{
			Type:    ErrorTypeNetwork,
			Message: "Test error",
		}

		// Should handle nil context gracefully
		reporter.ReportError(err, nil)

		stats := reporter.GetErrorStatistics()
		if stats.TotalErrors == 0 {
			t.Error("Should have recorded error with nil context")
		}
	})
}

// TestRateLimiterEdgeCases tests edge cases for rate limiting
func TestRateLimiterEdgeCases(t *testing.T) {
	t.Run("ZeroRequestsPerMinute", func(t *testing.T) {
		rateLimiter := NewRateLimiter(0, 1) // Zero rate, 1 burst

		// Should allow burst token
		if !rateLimiter.Allow() {
			t.Error("Should allow burst token")
		}

		// Should not allow more
		if rateLimiter.Allow() {
			t.Error("Should not allow more than burst with zero rate")
		}
	})

	t.Run("NegativeValues", func(t *testing.T) {
		// Should handle negative values gracefully
		rateLimiter := NewRateLimiter(-10, -5)

		// Should still function (implementation specific)
		stats := rateLimiter.GetStats()
		if stats == nil {
			t.Error("Should return valid stats even with negative values")
		}
	})

	t.Run("HighFrequencyRequests", func(t *testing.T) {
		rateLimiter := NewRateLimiter(100, 10) // High rate, low burst

		// Make many requests quickly
		allowed := 0
		for i := 0; i < 20; i++ {
			if rateLimiter.Allow() {
				allowed++
			}
		}

		// Should respect burst limit initially
		if allowed > 15 { // Allow some tolerance
			t.Errorf("Too many requests allowed: %d", allowed)
		}
	})
}

// TestEnhancedFuelAndTireStrategies tests enhanced fuel and tire strategy methods
func TestEnhancedFuelAndTireStrategies(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	telemetryData := createValidTelemetryData()

	t.Run("FuelStrategyScenarios", func(t *testing.T) {
		// Test different fuel scenarios
		scenarios := []struct {
			fuelLevel    float32
			expectedRisk string
		}{
			{50.0, "low"},     // High fuel
			{25.0, "medium"},  // Medium fuel
			{5.0, "high"},     // Low fuel
			{0.0, "critical"}, // No fuel
		}

		for _, scenario := range scenarios {
			telemetryData.CarData.FuelLevel = scenario.fuelLevel
			recommendation := engine.GenerateRecommendation(&telemetryData)

			// Test that fuel management considers fuel level
			if recommendation.FuelManagement.TargetConsumption <= 0 {
				t.Error("Fuel management should provide target consumption")
			}
		}
	})

	t.Run("TireStrategyProgression", func(t *testing.T) {
		// Test tire strategy with wear progression
		wearLevels := []float32{10.0, 30.0, 60.0, 90.0}

		for _, wearLevel := range wearLevels {
			telemetryData.TireData.FrontLeft.WearPercent = wearLevel
			telemetryData.TireData.FrontRight.WearPercent = wearLevel
			telemetryData.TireData.RearLeft.WearPercent = wearLevel
			telemetryData.TireData.RearRight.WearPercent = wearLevel

			recommendation := engine.GenerateRecommendation(&telemetryData)

			// High wear should recommend pit
			if wearLevel > 80.0 && !recommendation.PitRecommendation.ShouldPit {
				t.Errorf("Should recommend pit with %f%% tire wear", wearLevel)
			}
		}
	})

	t.Run("WeatherConsiderations", func(t *testing.T) {
		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should have weather considerations
		if recommendation.WeatherConsiderations.CurrentConditions == "" {
			t.Error("Should provide current weather conditions")
		}
	})

	t.Run("ThreatAndOpportunityDetection", func(t *testing.T) {
		recommendation := engine.GenerateRecommendation(&telemetryData)

		// Should identify threats and opportunities
		if len(recommendation.ThreatsAndOpportunities.Threats) == 0 &&
			len(recommendation.ThreatsAndOpportunities.Opportunities) == 0 {
			t.Error("Should identify at least some threats or opportunities")
		}
	})
}

// Helper functions for testing

type CustomError struct {
	message string
}

func (e *CustomError) Error() string {
	return e.message
}

func createValidRaceData() *RaceData {
	return &RaceData{
		SessionType:      "race",
		SessionTime:      3600,
		SessionTimeLeft:  1800,
		CurrentLap:       25,
		Position:         5,
		FuelLevel:        45.5,
		FuelConsumption:  2.8,
		TireWear:         65.0,
		TireCompound:     "medium",
		CurrentLapTime:   83.456,
		BestLapTime:      82.123,
		AverageLapTime:   83.789,
		TrackName:        "Silverstone",
		TrackTemp:        28.5,
		AirTemp:          22.3,
		Weather:          "dry",
		WeatherForecast:  "dry",
		TotalLaps:        45,
		RemainingLaps:    20,
		SafetyCarActive:  false,
		YellowFlagSector: 0,
		Opponents: []OpponentData{
			{Position: 4, Name: "Driver A", GapToPlayer: 2.5, LastLapTime: 82.8},
			{Position: 6, Name: "Driver B", GapToPlayer: -1.2, LastLapTime: 84.1},
		},
	}
}

func createValidTelemetryData() sims.TelemetryData {
	return sims.TelemetryData{
		CarData: sims.CarData{
			Speed:       200.5,
			FuelLevel:   45.8,
			CurrentLap:  25,
			Position:    5,
			LapTime:     83.456,
			BestLapTime: 82.1,
			LastLapTime: 83.2,
		},
		TireData: sims.TireData{
			FrontLeft: sims.TireWheelData{
				WearPercent: 15.5,
				Temperature: 85.0,
				Pressure:    1.8,
			},
			FrontRight: sims.TireWheelData{
				WearPercent: 16.2,
				Temperature: 86.0,
				Pressure:    1.8,
			},
			RearLeft: sims.TireWheelData{
				WearPercent: 14.8,
				Temperature: 88.0,
				Pressure:    1.9,
			},
			RearRight: sims.TireWheelData{
				WearPercent: 15.1,
				Temperature: 89.0,
				Pressure:    1.9,
			},
		},
		SessionData: sims.SessionData{
			SessionTime:     2250,
			SessionTimeLeft: 1350,
			SessionType:     "race",
			TrackTemp:       28.5,
			AirTemp:         22.3,
		},
	}
}

func createTelemetrySnapshot(lap int) sims.TelemetryData {
	return sims.TelemetryData{
		CarData: sims.CarData{
			Speed:       float32(200 + lap*2),
			FuelLevel:   float32(50 - lap*2),
			CurrentLap:  lap,
			Position:    5,
			LapTime:     83.5 + float64(lap)*0.1,
			BestLapTime: 82.1,
			LastLapTime: 83.2 + float64(lap)*0.08,
		},
		TireData: sims.TireData{
			FrontLeft: sims.TireWheelData{
				WearPercent: float32(lap * 1.5),
				Temperature: 85.0 + float32(lap)*0.5,
				Pressure:    1.8 + float32(lap)*0.01,
			},
			FrontRight: sims.TireWheelData{
				WearPercent: float32(lap * 1.6),
				Temperature: 86.0 + float32(lap)*0.5,
				Pressure:    1.8 + float32(lap)*0.01,
			},
			RearLeft: sims.TireWheelData{
				WearPercent: float32(lap * 1.4),
				Temperature: 88.0 + float32(lap)*0.6,
				Pressure:    1.9 + float32(lap)*0.01,
			},
			RearRight: sims.TireWheelData{
				WearPercent: float32(lap * 1.5),
				Temperature: 89.0 + float32(lap)*0.6,
				Pressure:    1.9 + float32(lap)*0.01,
			},
		},
		SessionData: sims.SessionData{
			SessionTime:     float32(lap * 90),
			SessionTimeLeft: float32(3600 - lap*90),
			SessionType:     "race",
			TrackTemp:       28.5,
			AirTemp:         22.3,
		},
	}
}
