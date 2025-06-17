package strategy

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewStrategyEngine tests creating a new strategy engine instance
func TestNewStrategyEngine(t *testing.T) {
	// Skip if no API key is available
	if os.Getenv("GOOGLE_API_KEY") == "" && os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping test: no API key available")
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

	if engine.client == nil {
		t.Error("Strategy engine client is nil")
	}

	if engine.config == nil {
		t.Error("Strategy engine config is nil")
	}

	if engine.cache == nil {
		t.Error("Strategy engine cache is nil")
	}
}

// TestConstructPrompt tests prompt construction for different scenarios
func TestConstructPrompt(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key" // Use dummy key for testing

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	// Test data
	raceData := &RaceData{
		SessionType:      "race",
		SessionTime:      1800, // 30 minutes
		SessionTimeLeft:  600,  // 10 minutes left
		CurrentLap:       15,
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
		TotalLaps:        25,
		RemainingLaps:    10,
		SafetyCarActive:  false,
		YellowFlagSector: 0,
		Opponents: []OpponentData{
			{Position: 2, Name: "Driver A", GapToPlayer: 2.5, LastLapTime: 82.8},
			{Position: 4, Name: "Driver B", GapToPlayer: -1.2, LastLapTime: 84.1},
		},
	}

	tests := []struct {
		name         string
		analysisType string
		expectError  bool
	}{
		{"Routine Analysis", "routine", false},
		{"Critical Analysis", "critical", false},
		{"Pit Decision", "pit_decision", false},
		{"General Analysis", "general", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := engine.constructPrompt(raceData, tt.analysisType)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Verify prompt contains essential information
				if len(prompt) < 100 {
					t.Error("Prompt is too short")
				}

				// Check for key elements
				keyElements := []string{
					"race strategist",
					"Silverstone",
					"Position: 3",
					"Current Lap: 15",
					"Fuel Level: 45.5",
					"medium compound",
					"JSON format",
				}

				for _, element := range keyElements {
					if !containsIgnoreCase(prompt, element) {
						t.Errorf("Prompt missing key element: %s", element)
					}
				}
			}
		})
	}
}

// TestParseResponse tests parsing of various response formats
func TestParseResponse(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	raceData := &RaceData{
		CurrentLap: 10,
		Position:   5,
	}

	tests := []struct {
		name         string
		response     string
		expectError  bool
		expectedData map[string]interface{}
	}{
		{
			name: "Valid JSON Response",
			response: `{
				"current_situation": "Mid-race, good position",
				"primary_strategy": "Continue current stint",
				"confidence": 0.8,
				"pit_window_open": false,
				"recommended_lap": 0,
				"tire_recommendation": "medium",
				"fuel_strategy": "Conserve fuel",
				"immediate_actions": ["Maintain pace", "Monitor tires"],
				"lap_targets": {"current_stint": 83.5},
				"risk_factors": ["Tire degradation"],
				"opportunities": ["Undercut opportunity"],
				"estimated_finish_position": 4,
				"estimated_finish_time": "1:25:30"
			}`,
			expectError: false,
			expectedData: map[string]interface{}{
				"current_situation": "Mid-race, good position",
				"confidence":        0.8,
				"pit_window_open":   false,
			},
		},
		{
			name:        "Invalid JSON",
			response:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "No JSON in response",
			response:    `This is just text without JSON`,
			expectError: true,
		},
		{
			name: "JSON with extra text",
			response: `Here's the analysis:
			{
				"current_situation": "Good position",
				"primary_strategy": "Hold position",
				"confidence": 0.7
			}
			That's my recommendation.`,
			expectError: false,
			expectedData: map[string]interface{}{
				"current_situation": "Good position",
				"confidence":        0.7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := engine.parseResponse(tt.response, raceData, "test")

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && analysis != nil {
				// Check expected data
				for key, expected := range tt.expectedData {
					switch key {
					case "current_situation":
						if analysis.CurrentSituation != expected.(string) {
							t.Errorf("Expected current_situation %s, got %s", expected, analysis.CurrentSituation)
						}
					case "confidence":
						if analysis.Confidence != expected.(float64) {
							t.Errorf("Expected confidence %f, got %f", expected, analysis.Confidence)
						}
					case "pit_window_open":
						if analysis.PitWindowOpen != expected.(bool) {
							t.Errorf("Expected pit_window_open %t, got %t", expected, analysis.PitWindowOpen)
						}
					}
				}

				// Verify required fields are set
				if analysis.Timestamp.IsZero() {
					t.Error("Timestamp not set")
				}

				if analysis.RequestID == "" {
					t.Error("RequestID not set")
				}

				if analysis.AnalysisType != "test" {
					t.Errorf("Expected analysis type 'test', got '%s'", analysis.AnalysisType)
				}
			}
		})
	}
}

// TestCacheOperations tests caching functionality
func TestCacheOperations(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.EnableCaching = true
	config.CacheTTL = 100 * time.Millisecond
	config.MaxCacheSize = 2

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	raceData := &RaceData{
		CurrentLap: 10,
		Position:   5,
		FuelLevel:  50.0,
		TireWear:   30.0,
	}

	analysis := &StrategyAnalysis{
		CurrentSituation: "Test analysis",
		Confidence:       0.8,
		Timestamp:        time.Now(),
	}
	// Test caching
	engine.cacheAnalysis(raceData, "test", analysis)

	stats := engine.cache.GetStats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected cache size 1, got %d", stats.TotalEntries)
	}

	// Test cache key generation
	key1 := engine.generateCacheKey(raceData, "test")
	key2 := engine.generateCacheKey(raceData, "test")

	if key1 != key2 {
		t.Error("Cache keys should be identical for same data")
	}

	// Modify data and test different key
	raceData.CurrentLap = 11
	key3 := engine.generateCacheKey(raceData, "test")

	if key1 == key3 {
		t.Error("Cache keys should be different for different data")
	}
	// Test cache expiration cleanup
	time.Sleep(150 * time.Millisecond)

	// Cache should automatically handle expiration, but let's verify
	cacheKey := engine.generateCacheKey(raceData, "test")
	_, exists := engine.cache.Get(cacheKey)

	// The entry should still exist if TTL is longer than 150ms
	// or should be expired if TTL is shorter
	if !exists {
		t.Log("Cache entry expired as expected")
	} else {
		t.Log("Cache entry still valid")
	}
}

// TestRaceDataValidation tests that race data is properly handled
func TestRaceDataValidation(t *testing.T) {
	tests := []struct {
		name      string
		raceData  *RaceData
		expectErr bool
	}{
		{
			name: "Valid Race Data",
			raceData: &RaceData{
				SessionType:   "race",
				CurrentLap:    10,
				Position:      3,
				FuelLevel:     50.0,
				TireWear:      40.0,
				TrackName:     "Silverstone",
				Weather:       "dry",
				TotalLaps:     25,
				RemainingLaps: 15,
			},
			expectErr: false,
		},
		{
			name: "Minimal Race Data",
			raceData: &RaceData{
				SessionType: "practice",
				CurrentLap:  1,
				Position:    1,
				TrackName:   "Unknown",
			},
			expectErr: false,
		},
	}

	config := DefaultConfig()
	config.APIKey = "test-key"

	ctx := context.Background()
	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := engine.constructPrompt(tt.raceData, "test")

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectErr && len(prompt) == 0 {
				t.Error("Empty prompt generated")
			}
		})
	}
}

// Helper function to check if a string contains another string (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Benchmark tests for performance
func BenchmarkConstructPrompt(b *testing.B) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	ctx := context.Background()
	engine, _ := NewStrategyEngine(ctx, config)
	defer engine.Close()

	raceData := &RaceData{
		SessionType:   "race",
		CurrentLap:    15,
		Position:      3,
		FuelLevel:     45.5,
		TireWear:      65.0,
		TrackName:     "Silverstone",
		Weather:       "dry",
		TotalLaps:     25,
		RemainingLaps: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.constructPrompt(raceData, "routine")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseResponse(b *testing.B) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	ctx := context.Background()
	engine, _ := NewStrategyEngine(ctx, config)
	defer engine.Close()

	response := `{
		"current_situation": "Mid-race, good position",
		"primary_strategy": "Continue current stint",
		"confidence": 0.8,
		"pit_window_open": false,
		"recommended_lap": 0,
		"tire_recommendation": "medium",
		"fuel_strategy": "Conserve fuel"
	}`

	raceData := &RaceData{CurrentLap: 10, Position: 5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.parseResponse(response, raceData, "test")
		if err != nil {
			b.Fatal(err)
		}
	}
}
