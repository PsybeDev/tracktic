package strategy

import (
	"os"
	"testing"
	"time"
)

// TestNewStrategyManager tests creating a new strategy manager
func TestNewStrategyManager(t *testing.T) {
	// Skip if no API key is available
	if os.Getenv("GOOGLE_API_KEY") == "" && os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping test: no API key available")
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	manager, err := NewStrategyManager(config)
	if err != nil {
		t.Fatalf("Failed to create strategy manager: %v", err)
	}
	defer manager.Close()

	if !manager.isRunning {
		t.Error("Strategy manager should be running")
	}

	if manager.engine == nil {
		t.Error("Strategy engine should not be nil")
	}

	if manager.config == nil {
		t.Error("Config should not be nil")
	}
}

// TestStrategyManagerHealthCheck tests the health check functionality
func TestStrategyManagerHealthCheck(t *testing.T) {
	// Test with invalid config (no API key)
	config := DefaultConfig()
	config.APIKey = "" // Invalid

	manager := &StrategyManager{
		config:    config,
		isRunning: false,
	}

	err := manager.IsHealthy()
	if err == nil {
		t.Error("Health check should fail for non-running manager")
	}
}

// TestCreateSampleRaceData tests the sample data creation
func TestCreateSampleRaceData(t *testing.T) {
	raceData := CreateSampleRaceData()

	if raceData == nil {
		t.Fatal("Sample race data should not be nil")
	}

	err := ValidateRaceData(raceData)
	if err != nil {
		t.Errorf("Sample race data should be valid: %v", err)
	}

	// Check specific fields
	if raceData.SessionType != "race" {
		t.Errorf("Expected session type 'race', got '%s'", raceData.SessionType)
	}

	if raceData.TrackName != "Silverstone" {
		t.Errorf("Expected track name 'Silverstone', got '%s'", raceData.TrackName)
	}

	if len(raceData.Opponents) != 2 {
		t.Errorf("Expected 2 opponents, got %d", len(raceData.Opponents))
	}
}

// TestValidateRaceData tests race data validation
func TestValidateRaceData(t *testing.T) {
	tests := []struct {
		name      string
		raceData  *RaceData
		expectErr bool
	}{
		{
			name:      "Nil data",
			raceData:  nil,
			expectErr: true,
		},
		{
			name: "Missing session type",
			raceData: &RaceData{
				TrackName:  "Test Track",
				CurrentLap: 1,
				Position:   1,
				FuelLevel:  50.0,
				TireWear:   30.0,
			},
			expectErr: true,
		},
		{
			name: "Missing track name",
			raceData: &RaceData{
				SessionType: "race",
				CurrentLap:  1,
				Position:    1,
				FuelLevel:   50.0,
				TireWear:    30.0,
			},
			expectErr: true,
		},
		{
			name: "Negative current lap",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  -1,
				Position:    1,
				FuelLevel:   50.0,
				TireWear:    30.0,
			},
			expectErr: true,
		},
		{
			name: "Invalid position",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  1,
				Position:    0,
				FuelLevel:   50.0,
				TireWear:    30.0,
			},
			expectErr: true,
		},
		{
			name: "Invalid fuel level (negative)",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  1,
				Position:    1,
				FuelLevel:   -10.0,
				TireWear:    30.0,
			},
			expectErr: true,
		},
		{
			name: "Invalid fuel level (over 100)",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  1,
				Position:    1,
				FuelLevel:   150.0,
				TireWear:    30.0,
			},
			expectErr: true,
		},
		{
			name: "Invalid tire wear (negative)",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  1,
				Position:    1,
				FuelLevel:   50.0,
				TireWear:    -10.0,
			},
			expectErr: true,
		},
		{
			name: "Invalid tire wear (over 100)",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  1,
				Position:    1,
				FuelLevel:   50.0,
				TireWear:    150.0,
			},
			expectErr: true,
		},
		{
			name: "Valid data",
			raceData: &RaceData{
				SessionType: "race",
				TrackName:   "Test Track",
				CurrentLap:  5,
				Position:    3,
				FuelLevel:   75.5,
				TireWear:    45.0,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRaceData(tt.raceData)

			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestFormatAnalysisForDisplay tests the analysis formatting function
func TestFormatAnalysisForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		analysis *StrategyAnalysis
		contains []string
	}{
		{
			name:     "Nil analysis",
			analysis: nil,
			contains: []string{"No analysis available"},
		},
		{
			name: "Complete analysis",
			analysis: &StrategyAnalysis{
				CurrentSituation:        "Good position, holding P3",
				PrimaryStrategy:         "Continue current stint, pit on lap 15",
				Confidence:              0.85,
				PitWindowOpen:           true,
				RecommendedLap:          15,
				TireRecommendation:      "medium",
				FuelStrategy:            "Conserve fuel for final 5 laps",
				ImmediateActions:        []string{"Maintain gap to car behind", "Monitor tire temps"},
				LapTargets:              map[string]float64{"current_stint": 83.5, "after_pit": 82.8},
				RiskFactors:             []string{"Tire degradation in final sector"},
				Opportunities:           []string{"Undercut car ahead if they pit late"},
				EstimatedFinishPosition: 3,
				EstimatedFinishTime:     "1:25:30",
			},
			contains: []string{
				"STRATEGY ANALYSIS",
				"85%",
				"Good position",
				"Continue current stint",
				"PIT WINDOW OPEN",
				"Recommended pit lap: 15",
				"medium",
				"Conserve fuel",
				"IMMEDIATE ACTIONS",
				"Maintain gap",
				"LAP TARGETS",
				"83.500",
				"RISKS",
				"Tire degradation",
				"OPPORTUNITIES",
				"Undercut car",
				"P3",
				"1:25:30",
			},
		},
		{
			name: "Minimal analysis",
			analysis: &StrategyAnalysis{
				CurrentSituation: "Analysis in progress",
				PrimaryStrategy:  "Hold position",
				Confidence:       0.6,
			},
			contains: []string{
				"60%",
				"Analysis in progress",
				"Hold position",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAnalysisForDisplay(tt.analysis)

			for _, expected := range tt.contains {
				if !containsIgnoreCase(result, expected) {
					t.Errorf("Formatted output should contain '%s'\nGot: %s", expected, result)
				}
			}
		})
	}
}

// TestGetPriority tests the priority assignment function
func TestGetPriority(t *testing.T) {
	tests := []struct {
		analysisType     string
		expectedPriority int
	}{
		{"critical", 100},
		{"pit_decision", 80},
		{"safety_car", 70},
		{"weather_change", 60},
		{"routine", 30},
		{"unknown", 50},
		{"", 50},
	}

	for _, tt := range tests {
		t.Run(tt.analysisType, func(t *testing.T) {
			priority := getPriority(tt.analysisType)
			if priority != tt.expectedPriority {
				t.Errorf("Expected priority %d for type '%s', got %d",
					tt.expectedPriority, tt.analysisType, priority)
			}
		})
	}
}

// TestAnalysisRequestStruct tests the analysis request structure
func TestAnalysisRequestStruct(t *testing.T) {
	raceData := CreateSampleRaceData()
	responseChan := make(chan AnalysisResult, 1)

	request := AnalysisRequest{
		RaceData:     raceData,
		AnalysisType: "test",
		Priority:     50,
		RequestTime:  time.Now(),
		ResponseChan: responseChan,
	}

	if request.RaceData != raceData {
		t.Error("Race data not properly assigned")
	}

	if request.AnalysisType != "test" {
		t.Error("Analysis type not properly assigned")
	}

	if request.Priority != 50 {
		t.Error("Priority not properly assigned")
	}

	if request.ResponseChan != responseChan {
		t.Error("Response channel not properly assigned")
	}

	if request.RequestTime.IsZero() {
		t.Error("Request time should not be zero")
	}
}

// TestAnalysisResultStruct tests the analysis result structure
func TestAnalysisResultStruct(t *testing.T) {
	analysis := &StrategyAnalysis{
		CurrentSituation: "Test analysis",
		PrimaryStrategy:  "Test strategy",
		Confidence:       0.8,
	}

	result := AnalysisResult{
		Analysis:  analysis,
		Error:     nil,
		Duration:  100 * time.Millisecond,
		RequestID: "test-123",
	}

	if result.Analysis != analysis {
		t.Error("Analysis not properly assigned")
	}

	if result.Error != nil {
		t.Error("Error should be nil")
	}

	if result.Duration != 100*time.Millisecond {
		t.Error("Duration not properly assigned")
	}

	if result.RequestID != "test-123" {
		t.Error("Request ID not properly assigned")
	}
}

// Benchmark tests
func BenchmarkCreateSampleRaceData(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data := CreateSampleRaceData()
		if data == nil {
			b.Fatal("Sample data should not be nil")
		}
	}
}

func BenchmarkValidateRaceData(b *testing.B) {
	raceData := CreateSampleRaceData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateRaceData(raceData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFormatAnalysisForDisplay(b *testing.B) {
	analysis := &StrategyAnalysis{
		CurrentSituation:        "Good position, holding P3",
		PrimaryStrategy:         "Continue current stint, pit on lap 15",
		Confidence:              0.85,
		PitWindowOpen:           true,
		RecommendedLap:          15,
		TireRecommendation:      "medium",
		FuelStrategy:            "Conserve fuel for final 5 laps",
		ImmediateActions:        []string{"Maintain gap to car behind", "Monitor tire temps"},
		LapTargets:              map[string]float64{"current_stint": 83.5},
		RiskFactors:             []string{"Tire degradation in final sector"},
		Opportunities:           []string{"Undercut car ahead if they pit late"},
		EstimatedFinishPosition: 3,
		EstimatedFinishTime:     "1:25:30",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := FormatAnalysisForDisplay(analysis)
		if len(result) == 0 {
			b.Fatal("Formatted result should not be empty")
		}
	}
}
