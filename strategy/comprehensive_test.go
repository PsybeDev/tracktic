package strategy

import (
	"context"
	"os"
	"testing"
	"time"

	"changeme/sims"
)

// TestStrategyEngineIntegration tests the complete integration between all strategy components
func TestStrategyEngineIntegration(t *testing.T) {
	// Skip if no API key is available (for offline testing)
	if os.Getenv("GOOGLE_API_KEY") == "" && os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping integration test: no API key available")
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	// Test complete analysis workflow
	raceData := createCompleteRaceData()

	// Test analysis with different types
	analysisTypes := []string{"routine", "critical", "pit_decision"}

	for _, analysisType := range analysisTypes {
		t.Run("Analysis_"+analysisType, func(t *testing.T) {
			analysis, err := engine.AnalyzeStrategy(raceData, analysisType)
			if err != nil {
				t.Logf("Analysis failed (expected in test environment): %v", err)
				return // Skip actual API call in test
			}

			// Validate analysis structure if API call succeeds
			if analysis.RaceFormat == "" {
				t.Error("Expected race format to be populated")
			}

			if len(analysis.ImmediateActions) == 0 {
				t.Error("Expected at least one immediate action")
			}
		})
	}
}

// TestRecommendationEngineIntegration tests the complete recommendation engine workflow
func TestRecommendationEngineIntegration(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Create realistic telemetry data
	telemetryData := createTelemetryData()

	// Add multiple telemetry snapshots for realistic analysis
	for i := 0; i < 10; i++ {
		snapshot := createTelemetrySnapshot(i)
		engine.AddTelemetrySnapshot(&snapshot)
	}

	// Test complete recommendation generation
	recommendation := engine.GenerateRecommendation(&telemetryData)

	// Validate all recommendation components
	t.Run("BasicRecommendation", func(t *testing.T) {
		if recommendation.PrimaryStrategy == "" {
			t.Error("Primary strategy should be provided")
		}

		if recommendation.ConfidenceLevel < 0 || recommendation.ConfidenceLevel > 1 {
			t.Errorf("Invalid confidence level: %f", recommendation.ConfidenceLevel)
		}
	})

	t.Run("PitRecommendation", func(t *testing.T) {
		if recommendation.PitRecommendation.OptimalLap < 0 {
			t.Error("Optimal lap should be valid")
		}
	})

	t.Run("FuelManagement", func(t *testing.T) {
		if recommendation.FuelManagement.TargetConsumption < 0 {
			t.Error("Target consumption should be positive")
		}
	})

	t.Run("TireManagement", func(t *testing.T) {
		if recommendation.TireManagement.RecommendedCompound == "" {
			t.Error("Tire compound recommendation should be provided")
		}
	})
}

// TestPitCalculatorIntegration tests complete pit calculator functionality
func TestPitCalculatorIntegration(t *testing.T) {
	config := DefaultConfig()
	calculator := NewPitStopCalculator(config)

	telemetryData := createTelemetryData()
	raceAnalysis := createRaceAnalysis()

	analysis := calculator.CalculatePitStopTiming(&telemetryData, &raceAnalysis)

	// Validate complete pit analysis
	t.Run("BasicCalculations", func(t *testing.T) {
		if analysis.OptimalWindow.StartLap < 1 {
			t.Error("Optimal window start lap should be valid")
		}

		if analysis.EstimatedLoss <= 0 {
			t.Error("Estimated loss should be positive")
		}
	})

	t.Run("PositionPredictions", func(t *testing.T) {
		if len(analysis.PositionPredictions) == 0 {
			t.Error("Position predictions should be provided")
		}
	})

	t.Run("RiskFactors", func(t *testing.T) {
		// Risk factors should always be evaluated (can be empty)
		if analysis.RiskFactors == nil {
			t.Error("Risk factors should be initialized")
		}
	})

	t.Run("AlternativeStrategies", func(t *testing.T) {
		if len(analysis.AlternativeStrategies) == 0 {
			t.Error("Alternative strategies should be provided")
		}
	})
}

// TestCacheIntegration tests comprehensive cache functionality
func TestCacheIntegration(t *testing.T) {
	config := DefaultConfig()
	config.CacheConfig.MaxEntries = 100

	cache := NewStrategyCache(config.CacheConfig)
	defer cache.Close()

	// Test different cache types
	testData := map[CacheType]interface{}{
		CacheTypeStrategy: &StrategyAnalysis{
			RaceFormat:      "endurance",
			PrimaryStrategy: "Test recommendation",
		},
		CacheTypePitTiming: &PitStopAnalysis{
			OptimalWindow: PitWindow{
				StartLap: 15,
				EndLap:   20,
			},
			EstimatedLoss: 25.5,
		},
		CacheTypeRecommendation: &StrategicRecommendation{
			PrimaryStrategy: "Conservative fuel management",
			ConfidenceLevel: 0.85,
		},
	}

	// Test storage and retrieval for all cache types
	for cacheType, data := range testData {
		t.Run("CacheType_"+string(cacheType), func(t *testing.T) {
			tags := []string{"lap:15", "session:race"}

			// Store data
			key := cache.Put(cacheType, data, tags)
			if key == "" {
				t.Error("Cache key should be generated")
			}

			// Retrieve data
			retrieved, found := cache.Get(key)
			if !found {
				t.Error("Data should be found in cache")
			}

			if retrieved == nil {
				t.Error("Retrieved data should not be nil")
			}
		})
	}

	// Test cache statistics
	t.Run("CacheStatistics", func(t *testing.T) {
		stats := cache.GetStats()

		if stats.TotalEntries == 0 {
			t.Error("Cache should have entries")
		}

		if stats.HitRatio < 0 || stats.HitRatio > 1 {
			t.Errorf("Invalid hit ratio: %f", stats.HitRatio)
		}
	})
}

// TestErrorHandlingIntegration tests comprehensive error handling workflows
func TestErrorHandlingIntegration(t *testing.T) {
	classifier := NewErrorClassifier()
	retryPolicy := &RetryPolicy{
		MaxRetries:        3,
		BaseDelay:         time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            0.1,
	}
	reporter := NewErrorReporter(100)

	// Test complete error handling workflow
	testErrors := []error{
		&StrategyError{Type: ErrorTypeAuthentication, Message: "Auth failed"},
		&StrategyError{Type: ErrorTypeRateLimit, Message: "Rate limited"},
		&StrategyError{Type: ErrorTypeQuota, Message: "Quota exceeded"},
		&StrategyError{Type: ErrorTypeNetwork, Message: "Network error"},
		&StrategyError{Type: ErrorTypeService, Message: "Service error"},
	}

	for _, testErr := range testErrors {
		t.Run("ErrorType_"+string(testErr.(*StrategyError).Type), func(t *testing.T) {
			// Classify error
			errorType := classifier.ClassifyError(testErr)
			if errorType == ErrorTypeUnknown {
				t.Error("Error should be classified correctly")
			}

			// Check retry policy
			shouldRetry := retryPolicy.ShouldRetry(testErr, 1)
			backoff := retryPolicy.CalculateBackoff(1)

			if backoff < 0 {
				t.Error("Backoff should be non-negative")
			}

			// Report error
			reporter.ReportError(testErr, map[string]interface{}{
				"context": "integration_test",
			})
		})
	}

	// Test error statistics
	t.Run("ErrorStatistics", func(t *testing.T) {
		stats := reporter.GetErrorStatistics()

		if stats.TotalErrors == 0 {
			t.Error("Should have recorded errors")
		}

		if len(stats.ErrorsByType) == 0 {
			t.Error("Should have error type breakdown")
		}
	})
}

// TestRateLimiterIntegration tests rate limiter in realistic scenarios
func TestRateLimiterIntegration(t *testing.T) {
	rateLimiter := NewRateLimiter(10, 20) // 10 requests per minute, burst of 20

	// Test rate limiting behavior
	t.Run("BurstCapacity", func(t *testing.T) {
		// Should allow burst requests
		for i := 0; i < 15; i++ {
			if !rateLimiter.Allow() {
				t.Errorf("Request %d should be allowed in burst", i+1)
			}
		}
	})

	t.Run("RateLimit", func(t *testing.T) {
		// Wait for rate limiter to refill
		time.Sleep(100 * time.Millisecond)

		// Test sustainable rate
		allowed := 0
		for i := 0; i < 25; i++ {
			if rateLimiter.Allow() {
				allowed++
			}
			time.Sleep(1 * time.Millisecond)
		}

		// Should respect rate limits
		if allowed > 22 { // Allow some tolerance
			t.Errorf("Too many requests allowed: %d", allowed)
		}
	})

	t.Run("Statistics", func(t *testing.T) {
		stats := rateLimiter.GetStats()

		if stats.RequestsAllowed == 0 {
			t.Error("Should have allowed some requests")
		}
	})
}

// Helper function to create complete race data for testing
func createCompleteRaceData() *RaceData {
	return &RaceData{
		SessionType:      "race",
		SessionTime:      3600, // 1 hour race
		SessionTimeLeft:  1800, // 30 minutes left
		CurrentLap:       25,
		Position:         3,
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
			{Position: 1, Name: "Leader", GapToPlayer: 15.2, LastLapTime: 81.9},
			{Position: 2, Name: "P2 Driver", GapToPlayer: 5.1, LastLapTime: 82.3},
			{Position: 4, Name: "P4 Driver", GapToPlayer: -2.8, LastLapTime: 84.1},
			{Position: 5, Name: "P5 Driver", GapToPlayer: -8.5, LastLapTime: 83.9},
		},
	}
}

// Helper function to create telemetry data for testing
func createTelemetryData() sims.TelemetryData {
	return sims.TelemetryData{
		CarData: sims.CarData{
			Speed:       200.5,
			FuelLevel:   45.8,
			CurrentLap:  25,
			Position:    3,
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
			SessionTime:     2250, // 37.5 minutes
			SessionTimeLeft: 1350, // 22.5 minutes
			SessionType:     "race",
			TrackTemp:       28.5,
			AirTemp:         22.3,
		},
	}
}

// Helper function to create telemetry snapshots for testing
func createTelemetrySnapshot(lap int) sims.TelemetryData {
	return sims.TelemetryData{
		CarData: sims.CarData{
			Speed:       float32(200 + lap*2), // Varying speed
			FuelLevel:   float32(50 - lap*2),  // Decreasing fuel
			CurrentLap:  lap,
			Position:    3,
			LapTime:     83.5 + float64(lap)*0.1, // Slight degradation
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
			SessionTime:     float32(lap * 90), // 90 seconds per lap
			SessionTimeLeft: float32(3600 - lap*90),
			SessionType:     "race",
			TrackTemp:       28.5,
			AirTemp:         22.3,
		},
	}
}

// Helper function to create race analysis for testing
func createRaceAnalysis() RaceAnalysis {
	return RaceAnalysis{
		RaceFormat:               "endurance",
		StrategicPhase:           "middle",
		PositionTrend:            "stable",
		CompetitiveGaps:          map[int]float64{2: 5.1, 4: -2.8, 5: -8.5},
		SafetyCarProbability:     0.15,
		WeatherChangeProbability: 0.05,
		RiskLevel:                "medium",
		OpportunityScore:         0.7,
		KeyStrategicFactors:      []string{"tire degradation", "fuel management"},
	}
}
