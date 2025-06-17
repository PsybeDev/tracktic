package strategy

import (
	"context"
	"sync"
	"testing"
	"time"

	"changeme/sims"
)

// TestPerformanceAndConcurrency tests performance characteristics and concurrent access
func TestPerformanceAndConcurrency(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	t.Run("ConcurrentEngineAccess", func(t *testing.T) {
		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Test concurrent access to strategy engine
		const numGoroutines = 10
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				raceData := createValidRaceData()
				raceData.CurrentLap = id + 1 // Make each request unique

				// This will likely fail due to no API key, but should not crash
				_, err := engine.AnalyzeStrategy(raceData, "routine")
				if err != nil {
					errChan <- err
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Count errors (expected due to no real API key)
		errorCount := 0
		for err := range errChan {
			errorCount++
			if err == nil {
				t.Error("Expected error due to test environment")
			}
		}

		// All requests should have failed gracefully
		if errorCount != numGoroutines {
			t.Logf("Expected %d errors, got %d (some may have been cached)", numGoroutines, errorCount)
		}
	})

	t.Run("ConcurrentCacheAccess", func(t *testing.T) {
		cache := NewStrategyCache(config.CacheConfig)
		defer cache.Close()

		// Test concurrent cache access
		const numGoroutines = 20
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Store data
				data := &StrategyAnalysis{
					PrimaryStrategy: "Concurrent test",
					RaceFormat:      "test",
				}
				key := cache.Put(CacheTypeStrategy, data, []string{"concurrent", "test"})

				// Retrieve data
				retrieved, found := cache.Get(key)
				if !found {
					t.Errorf("Goroutine %d: Failed to retrieve stored data", id)
				}

				if retrieved == nil {
					t.Errorf("Goroutine %d: Retrieved nil data", id)
				}
			}(i)
		}

		wg.Wait()

		// Check cache statistics
		stats := cache.GetStats()
		if stats.TotalEntries == 0 {
			t.Error("Cache should have entries after concurrent access")
		}
	})

	t.Run("RecommendationEnginePerformance", func(t *testing.T) {
		engine := NewRecommendationEngine(config)

		// Add substantial telemetry history
		for i := 0; i < 100; i++ {
			snapshot := createTelemetrySnapshot(i)
			engine.AddTelemetrySnapshot(&snapshot)
		}

		telemetryData := createValidTelemetryData()

		// Measure performance
		start := time.Now()
		recommendation := engine.GenerateRecommendation(&telemetryData)
		duration := time.Since(start)

		// Should complete within reasonable time (adjust based on requirements)
		if duration > 100*time.Millisecond {
			t.Errorf("Recommendation generation took too long: %v", duration)
		}

		if recommendation == nil {
			t.Error("Should generate recommendation")
		}
	})

	t.Run("MemoryUsageUnderLoad", func(t *testing.T) {
		cache := NewStrategyCache(config.CacheConfig)
		defer cache.Close()

		// Add many entries to test memory management
		for i := 0; i < 1000; i++ {
			data := &StrategyAnalysis{
				PrimaryStrategy: "Memory test",
				RaceFormat:      "test",
			}
			cache.Put(CacheTypeStrategy, data, []string{"memory", "test"})
		}

		stats := cache.GetStats()

		// Should respect memory limits
		if stats.MemoryUsageMB > float64(config.CacheConfig.MaxMemoryMB*2) {
			t.Errorf("Memory usage exceeded expectations: %.2f MB", stats.MemoryUsageMB)
		}
	})
}

// TestDataValidationAndSanitization tests input validation and data sanitization
func TestDataValidationAndSanitization(t *testing.T) {
	config := DefaultConfig()

	t.Run("RaceDataValidation", func(t *testing.T) {
		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Test various invalid race data scenarios
		testCases := []struct {
			name        string
			raceData    *RaceData
			expectError bool
		}{
			{
				name:        "NilRaceData",
				raceData:    nil,
				expectError: true,
			},
			{
				name: "NegativePosition",
				raceData: &RaceData{
					Position:        -1,
					SessionTimeLeft: 1800,
				},
				expectError: true,
			},
			{
				name: "NegativeFuelLevel",
				raceData: &RaceData{
					Position:        5,
					FuelLevel:       -10.0,
					SessionTimeLeft: 1800,
				},
				expectError: true,
			},
			{
				name: "InvalidTireWear",
				raceData: &RaceData{
					Position:        5,
					FuelLevel:       50.0,
					TireWear:        150.0, // > 100%
					SessionTimeLeft: 1800,
				},
				expectError: true,
			},
			{
				name:        "ValidData",
				raceData:    createValidRaceData(),
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := engine.AnalyzeStrategy(tc.raceData, "routine")

				if tc.expectError && err == nil {
					t.Error("Expected error for invalid data")
				}

				if !tc.expectError && err != nil {
					t.Errorf("Unexpected error for valid data: %v", err)
				}
			})
		}
	})

	t.Run("TelemetryDataSanitization", func(t *testing.T) {
		engine := NewRecommendationEngine(config)

		// Test with extreme values
		extremeData := sims.TelemetryData{
			CarData: sims.CarData{
				Speed:      float32(999999), // Extreme speed
				FuelLevel:  float32(-100),   // Negative fuel
				CurrentLap: -5,              // Negative lap
				Position:   0,               // Invalid position
				LapTime:    -1.0,            // Negative time
			},
			TireData: sims.TireData{
				FrontLeft: sims.TireWheelData{
					WearPercent: 200.0, // > 100%
					Temperature: -50.0, // Extreme cold
					Pressure:    0.0,   // No pressure
				},
				FrontRight: sims.TireWheelData{
					WearPercent: -50.0, // Negative wear
					Temperature: 500.0, // Extreme heat
					Pressure:    10.0,  // Extreme pressure
				},
				RearLeft: sims.TireWheelData{
					WearPercent: 100.0,
					Temperature: 85.0,
					Pressure:    1.8,
				},
				RearRight: sims.TireWheelData{
					WearPercent: 100.0,
					Temperature: 85.0,
					Pressure:    1.8,
				},
			},
		}

		// Should handle extreme values gracefully
		recommendation := engine.GenerateRecommendation(&extremeData)

		if recommendation == nil {
			t.Error("Should generate recommendation even with extreme data")
		}

		// Should flag data quality issues
		if recommendation.DataQuality > 0.5 {
			t.Errorf("Expected low data quality with extreme values, got %f", recommendation.DataQuality)
		}
	})

	t.Run("OpponentDataValidation", func(t *testing.T) {
		raceData := createValidRaceData()

		// Add invalid opponent data
		raceData.Opponents = append(raceData.Opponents, OpponentData{
			Position:    -1,     // Invalid position
			Name:        "",     // Empty name
			GapToPlayer: 999999, // Extreme gap
			LastLapTime: -1.0,   // Invalid time
		})

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		// Should handle invalid opponent data
		_, err = engine.AnalyzeStrategy(raceData, "routine")
		if err == nil {
			t.Error("Expected error with invalid opponent data")
		}
	})
}

// TestErrorRecoveryAndResilience tests error recovery mechanisms
func TestErrorRecoveryAndResilience(t *testing.T) {
	t.Run("CacheRecoveryAfterCorruption", func(t *testing.T) {
		config := DefaultConfig()
		cache := NewStrategyCache(config.CacheConfig)
		defer cache.Close()

		// Store valid data
		data := &StrategyAnalysis{PrimaryStrategy: "Test"}
		key := cache.Put(CacheTypeStrategy, data, []string{"test"})

		// Retrieve to confirm it's stored
		_, found := cache.Get(key)
		if !found {
			t.Error("Data should be found initially")
		}

		// Simulate corruption by clearing internal storage manually
		cache.Clear()

		// Should handle missing data gracefully
		_, found = cache.Get(key)
		if found {
			t.Error("Data should not be found after clearing")
		}

		// Should continue to work normally
		newKey := cache.Put(CacheTypeStrategy, data, []string{"recovery"})
		_, found = cache.Get(newKey)
		if !found {
			t.Error("Should be able to store data after recovery")
		}
	})

	t.Run("EngineRecoveryAfterError", func(t *testing.T) {
		config := DefaultConfig()
		config.APIKey = "invalid-key"

		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		raceData := createValidRaceData()

		// First request should fail
		_, err1 := engine.AnalyzeStrategy(raceData, "routine")
		if err1 == nil {
			t.Error("Expected error with invalid API key")
		}

		// Engine should still be responsive
		_, err2 := engine.AnalyzeStrategy(raceData, "routine")
		if err2 == nil {
			t.Error("Expected error with invalid API key on second request")
		}

		// Error types should be consistent
		if err1.Error() != err2.Error() {
			t.Error("Error handling should be consistent across requests")
		}
	})

	t.Run("RecommendationEngineResilience", func(t *testing.T) {
		config := DefaultConfig()
		engine := NewRecommendationEngine(config)

		// Generate recommendation with minimal data
		minimalData := sims.TelemetryData{}
		recommendation := engine.GenerateRecommendation(&minimalData)

		if recommendation == nil {
			t.Error("Should generate recommendation even with minimal data")
		}

		// Should indicate low confidence
		if recommendation.ConfidenceLevel > 0.3 {
			t.Errorf("Expected low confidence with minimal data, got %f", recommendation.ConfidenceLevel)
		}

		// Should still provide basic recommendations
		if recommendation.PrimaryStrategy == "" {
			t.Error("Should provide some primary strategy")
		}
	})
}

// TestBoundaryConditions tests boundary conditions and limits
func TestBoundaryConditions(t *testing.T) {
	t.Run("MaxSessionTimeValues", func(t *testing.T) {
		config := DefaultConfig()
		ctx := context.Background()
		engine, err := NewStrategyEngine(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create strategy engine: %v", err)
		}
		defer engine.Close()

		raceData := createValidRaceData()

		// Test with maximum session time
		raceData.SessionTime = 24 * 60 * 60     // 24 hours
		raceData.SessionTimeLeft = 12 * 60 * 60 // 12 hours

		_, err = engine.AnalyzeStrategy(raceData, "routine")
		// Should handle long sessions (may fail due to no API key, but shouldn't crash)
		if err == nil {
			t.Log("Long session handled successfully")
		}
	})

	t.Run("MaximumLapNumbers", func(t *testing.T) {
		engine := NewRecommendationEngine(DefaultConfig())

		telemetryData := createValidTelemetryData()
		telemetryData.CarData.CurrentLap = 999999 // Very high lap number

		recommendation := engine.GenerateRecommendation(&telemetryData)
		if recommendation == nil {
			t.Error("Should handle high lap numbers")
		}
	})

	t.Run("ExtremeTemperatureValues", func(t *testing.T) {
		engine := NewRecommendationEngine(DefaultConfig())

		telemetryData := createValidTelemetryData()

		// Test extreme cold
		telemetryData.TireData.FrontLeft.Temperature = -50.0
		telemetryData.SessionData.TrackTemp = -20.0
		telemetryData.SessionData.AirTemp = -30.0

		recommendation := engine.GenerateRecommendation(&telemetryData)
		if recommendation == nil {
			t.Error("Should handle extreme cold temperatures")
		}

		// Test extreme heat
		telemetryData.TireData.FrontLeft.Temperature = 200.0
		telemetryData.SessionData.TrackTemp = 80.0
		telemetryData.SessionData.AirTemp = 60.0

		recommendation = engine.GenerateRecommendation(&telemetryData)
		if recommendation == nil {
			t.Error("Should handle extreme hot temperatures")
		}
	})

	t.Run("PrecisionLimits", func(t *testing.T) {
		engine := NewRecommendationEngine(DefaultConfig())

		// Test with very precise values
		telemetryData := createValidTelemetryData()
		telemetryData.CarData.LapTime = 82.123456789 // High precision
		telemetryData.TireData.FrontLeft.WearPercent = 15.123456789

		recommendation := engine.GenerateRecommendation(&telemetryData)
		if recommendation == nil {
			t.Error("Should handle high precision values")
		}
	})
}

// BenchmarkStrategyEngineOperations benchmarks key operations
func BenchmarkStrategyEngineOperations(b *testing.B) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	b.Run("CacheKeyGeneration", func(b *testing.B) {
		cache := NewStrategyCache(config.CacheConfig)
		defer cache.Close()

		raceData := createValidRaceData()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.generateKey(CacheTypeStrategy, raceData)
		}
	})

	b.Run("RecommendationGeneration", func(b *testing.B) {
		engine := NewRecommendationEngine(config)

		// Pre-populate with some telemetry history
		for i := 0; i < 50; i++ {
			snapshot := createTelemetrySnapshot(i)
			engine.AddTelemetrySnapshot(&snapshot)
		}

		telemetryData := createValidTelemetryData()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine.GenerateRecommendation(&telemetryData)
		}
	})

	b.Run("CacheOperations", func(b *testing.B) {
		cache := NewStrategyCache(config.CacheConfig)
		defer cache.Close()

		data := &StrategyAnalysis{PrimaryStrategy: "Benchmark test"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := cache.Put(CacheTypeStrategy, data, []string{"benchmark"})
			cache.Get(key)
		}
	})

	b.Run("TelemetryProcessing", func(b *testing.B) {
		engine := NewRecommendationEngine(config)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			snapshot := createTelemetrySnapshot(i)
			engine.AddTelemetrySnapshot(&snapshot)
		}
	})
}

// Helper functions for advanced testing

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
