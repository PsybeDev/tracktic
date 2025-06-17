package strategy

import (
	"strings"
	"testing"
)

// TestRaceFormatDetector tests the race format detection logic
func TestRaceFormatDetector(t *testing.T) {
	config := DefaultConfig()
	detector := NewRaceFormatDetector(config)

	tests := []struct {
		name         string
		raceData     *RaceData
		configFormat string
		expected     string
	}{
		{
			name: "Sprint race by lap count",
			raceData: &RaceData{
				TotalLaps:   10,
				CurrentLap:  5,
				SessionTime: 1800,
			},
			configFormat: "auto",
			expected:     "sprint",
		},
		{
			name: "Endurance race by lap count",
			raceData: &RaceData{
				TotalLaps:   100,
				CurrentLap:  20,
				SessionTime: 7200,
			},
			configFormat: "auto",
			expected:     "endurance",
		},
		{
			name: "Standard race by lap count",
			raceData: &RaceData{
				TotalLaps:   25,
				CurrentLap:  10,
				SessionTime: 3600,
			},
			configFormat: "auto",
			expected:     "standard",
		},
		{
			name: "Sprint race by time (20 minutes)",
			raceData: &RaceData{
				TotalLaps:       0,
				SessionTime:     20 * 60,
				SessionTimeLeft: 10 * 60,
			},
			configFormat: "auto",
			expected:     "sprint",
		},
		{
			name: "Endurance race by time (3 hours)",
			raceData: &RaceData{
				TotalLaps:       0,
				SessionTime:     3 * 3600,
				SessionTimeLeft: 1.5 * 3600,
			},
			configFormat: "auto",
			expected:     "endurance",
		},
		{
			name: "Manual override to sprint",
			raceData: &RaceData{
				TotalLaps:  50, // Would normally be standard
				CurrentLap: 25,
			},
			configFormat: "sprint",
			expected:     "sprint",
		},
		{
			name: "Manual override to endurance",
			raceData: &RaceData{
				TotalLaps:  20, // Would normally be standard
				CurrentLap: 10,
			},
			configFormat: "endurance",
			expected:     "endurance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.AnalysisPreferences.RaceFormat = tt.configFormat
			detector.config = config

			result := detector.DetectRaceFormat(tt.raceData)
			if result != tt.expected {
				t.Errorf("Expected race format %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestPromptTemplates tests that all prompt templates are properly defined
func TestPromptTemplates(t *testing.T) {
	formats := []string{"sprint", "endurance", "standard"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			template := GetPromptTemplate(format)

			if template == nil {
				t.Fatal("Template should not be nil")
			}

			if template.RaceFormat != format {
				t.Errorf("Expected race format %s, got %s", format, template.RaceFormat)
			}

			// Check required fields are not empty
			if template.SystemContext == "" {
				t.Error("SystemContext should not be empty")
			}

			if template.StrategyFocus == "" {
				t.Error("StrategyFocus should not be empty")
			}

			if len(template.KeyFactors) == 0 {
				t.Error("KeyFactors should not be empty")
			}

			if len(template.DecisionCriteria) == 0 {
				t.Error("DecisionCriteria should not be empty")
			}

			if template.OutputGuidance == "" {
				t.Error("OutputGuidance should not be empty")
			}
		})
	}
}

// TestSprintRaceTemplate tests specific sprint race template content
func TestSprintRaceTemplate(t *testing.T) {
	template := getSprintRaceTemplate()

	// Check sprint-specific keywords in system context
	sprintKeywords := []string{"sprint", "aggressive", "short", "intense", "minimal pit"}
	for _, keyword := range sprintKeywords {
		if !strings.Contains(strings.ToLower(template.SystemContext), strings.ToLower(keyword)) {
			t.Errorf("Sprint template should contain keyword '%s' in SystemContext", keyword)
		}
	}

	// Check strategy focus contains sprint-specific elements
	sprintStrategyElements := []string{"track position", "minimal pit stops", "aggressive positioning"}
	for _, element := range sprintStrategyElements {
		if !strings.Contains(strings.ToLower(template.StrategyFocus), strings.ToLower(element)) {
			t.Errorf("Sprint template should contain strategy element '%s'", element)
		}
	}

	// Verify key factors include sprint-specific items
	if len(template.KeyFactors) < 5 {
		t.Error("Sprint template should have at least 5 key factors")
	}

	// Check for sprint-specific decision criteria
	if len(template.DecisionCriteria) < 4 {
		t.Error("Sprint template should have at least 4 decision criteria")
	}
}

// TestEnduranceRaceTemplate tests specific endurance race template content
func TestEnduranceRaceTemplate(t *testing.T) {
	template := getEnduranceRaceTemplate()

	// Check endurance-specific keywords
	enduranceKeywords := []string{"endurance", "consistency", "marathon", "long-term", "tire management"}
	for _, keyword := range enduranceKeywords {
		if !strings.Contains(strings.ToLower(template.SystemContext), strings.ToLower(keyword)) {
			t.Errorf("Endurance template should contain keyword '%s' in SystemContext", keyword)
		}
	}

	// Check strategy focus contains endurance-specific elements
	enduranceStrategyElements := []string{"stint lengths", "fuel management", "consistency", "reliability"}
	for _, element := range enduranceStrategyElements {
		if !strings.Contains(strings.ToLower(template.StrategyFocus), strings.ToLower(element)) {
			t.Errorf("Endurance template should contain strategy element '%s'", element)
		}
	}

	// Verify comprehensive key factors for endurance
	if len(template.KeyFactors) < 6 {
		t.Error("Endurance template should have at least 6 key factors")
	}

	// Check for endurance-specific decision criteria
	if len(template.DecisionCriteria) < 5 {
		t.Error("Endurance template should have at least 5 decision criteria")
	}
}

// TestPromptBuilder tests the prompt builder functionality
func TestPromptBuilder(t *testing.T) {
	config := DefaultConfig()
	config.AnalysisPreferences.RaceFormat = "auto"

	builder := NewPromptBuilder(config)

	if builder == nil {
		t.Fatal("PromptBuilder should not be nil")
	}

	if builder.detector == nil {
		t.Fatal("PromptBuilder detector should not be nil")
	}
}

// TestBuildSpecializedPrompt tests prompt construction for different scenarios
func TestBuildSpecializedPrompt(t *testing.T) {
	config := DefaultConfig()
	config.AnalysisPreferences.RaceFormat = "auto"
	config.AnalysisPreferences.IncludeOpponentData = true

	builder := NewPromptBuilder(config)

	tests := []struct {
		name         string
		raceData     *RaceData
		analysisType string
		expectFormat string
	}{
		{
			name: "Sprint race prompt",
			raceData: &RaceData{
				SessionType:     "race",
				TrackName:       "Monza",
				TotalLaps:       12,
				CurrentLap:      6,
				Position:        4,
				FuelLevel:       75.0,
				TireWear:        45.0,
				TireCompound:    "soft",
				Weather:         "dry",
				WeatherForecast: "dry",
				RemainingLaps:   6,
				Opponents: []OpponentData{
					{Position: 3, Name: "Driver A", GapToPlayer: 1.5, LastLapTime: 80.5},
				},
			},
			analysisType: "routine",
			expectFormat: "sprint",
		},
		{
			name: "Endurance race prompt",
			raceData: &RaceData{
				SessionType:     "race",
				TrackName:       "Le Mans",
				TotalLaps:       200,
				CurrentLap:      50,
				Position:        2,
				FuelLevel:       60.0,
				TireWear:        65.0,
				TireCompound:    "medium",
				Weather:         "dry",
				WeatherForecast: "light_rain",
				RemainingLaps:   150,
			},
			analysisType: "pit_decision",
			expectFormat: "endurance",
		},
		{
			name: "Standard race prompt",
			raceData: &RaceData{
				SessionType:   "race",
				TrackName:     "Silverstone",
				TotalLaps:     25,
				CurrentLap:    15,
				Position:      3,
				FuelLevel:     45.0,
				TireWear:      70.0,
				TireCompound:  "medium",
				Weather:       "dry",
				RemainingLaps: 10,
			},
			analysisType: "critical",
			expectFormat: "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := builder.BuildSpecializedPrompt(tt.raceData, tt.analysisType, config)

			if err != nil {
				t.Fatalf("BuildSpecializedPrompt failed: %v", err)
			}

			if len(prompt) < 500 {
				t.Error("Generated prompt is too short")
			}

			// Check that prompt contains race format
			if !strings.Contains(strings.ToUpper(prompt), strings.ToUpper(tt.expectFormat)) {
				t.Errorf("Prompt should contain race format '%s'", tt.expectFormat)
			}

			// Check that prompt contains track name
			if !strings.Contains(prompt, tt.raceData.TrackName) {
				t.Error("Prompt should contain track name")
			}

			// Check that prompt contains analysis type context
			switch tt.analysisType {
			case "critical":
				if !strings.Contains(prompt, "CRITICAL") {
					t.Error("Critical analysis prompt should contain 'CRITICAL'")
				}
			case "pit_decision":
				if !strings.Contains(prompt, "PIT") {
					t.Error("Pit decision prompt should contain 'PIT'")
				}
			case "routine":
				if !strings.Contains(prompt, "ROUTINE") {
					t.Error("Routine analysis prompt should contain 'ROUTINE'")
				}
			}

			// Check JSON format specification
			if !strings.Contains(prompt, "JSON format") {
				t.Error("Prompt should specify JSON format")
			}

			// Check for race format-specific content
			switch tt.expectFormat {
			case "sprint":
				if !strings.Contains(strings.ToLower(prompt), "aggressive") {
					t.Error("Sprint prompt should contain aggressive strategy guidance")
				}
				if !strings.Contains(strings.ToLower(prompt), "track position") {
					t.Error("Sprint prompt should emphasize track position")
				}
			case "endurance":
				if !strings.Contains(strings.ToLower(prompt), "consistency") {
					t.Error("Endurance prompt should contain consistency guidance")
				}
				if !strings.Contains(strings.ToLower(prompt), "stint") {
					t.Error("Endurance prompt should mention stint strategy")
				}
			case "standard":
				if !strings.Contains(strings.ToLower(prompt), "balanced") {
					t.Error("Standard prompt should contain balanced approach")
				}
			}
		})
	}
}

// TestGetRaceFormatAnalysis tests the race format analysis function
func TestGetRaceFormatAnalysis(t *testing.T) {
	config := DefaultConfig()
	builder := NewPromptBuilder(config)

	formats := []string{"sprint", "endurance", "standard"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			analysis := builder.GetRaceFormatAnalysis(format)

			if len(analysis) < 100 {
				t.Error("Race format analysis should be substantial")
			}

			if !strings.Contains(strings.ToUpper(analysis), strings.ToUpper(format)) {
				t.Errorf("Analysis should contain race format '%s'", format)
			}

			// Check for required sections
			if !strings.Contains(analysis, "ANALYSIS") {
				t.Error("Analysis should contain 'ANALYSIS' header")
			}

			if !strings.Contains(analysis, "Strategic Factors") {
				t.Error("Analysis should contain 'Strategic Factors' section")
			}

			if !strings.Contains(analysis, "Decision Framework") {
				t.Error("Analysis should contain 'Decision Framework' section")
			}
		})
	}
}

// TestPromptConsistency tests that prompts are consistently formatted
func TestPromptConsistency(t *testing.T) {
	config := DefaultConfig()
	config.AnalysisPreferences.IncludeOpponentData = true

	builder := NewPromptBuilder(config)

	raceData := &RaceData{
		SessionType:     "race",
		TrackName:       "Spa-Francorchamps",
		TotalLaps:       25,
		CurrentLap:      12,
		Position:        3,
		FuelLevel:       55.0,
		TireWear:        60.0,
		TireCompound:    "medium",
		Weather:         "dry",
		WeatherForecast: "dry",
		RemainingLaps:   13,
	}

	analysisTypes := []string{"routine", "critical", "pit_decision"}

	for _, analysisType := range analysisTypes {
		t.Run(analysisType, func(t *testing.T) {
			prompt, err := builder.BuildSpecializedPrompt(raceData, analysisType, config)

			if err != nil {
				t.Fatalf("Failed to build prompt: %v", err)
			}

			// Check for consistent sections
			requiredSections := []string{
				"CURRENT RACE SITUATION",
				"CAR STATUS",
				"TRACK CONDITIONS",
				"STRATEGY PREFERENCES",
				"DECISION CRITERIA",
				"ANALYSIS REQUEST",
				"OUTPUT REQUIREMENTS",
				"JSON format",
			}

			for _, section := range requiredSections {
				if !strings.Contains(prompt, section) {
					t.Errorf("Prompt missing required section: %s", section)
				}
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkDetectRaceFormat(b *testing.B) {
	config := DefaultConfig()
	detector := NewRaceFormatDetector(config)

	raceData := &RaceData{
		TotalLaps:   25,
		CurrentLap:  12,
		SessionTime: 3600,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectRaceFormat(raceData)
	}
}

func BenchmarkBuildSpecializedPrompt(b *testing.B) {
	config := DefaultConfig()
	builder := NewPromptBuilder(config)

	raceData := &RaceData{
		SessionType:   "race",
		TrackName:     "Silverstone",
		TotalLaps:     25,
		CurrentLap:    15,
		Position:      3,
		FuelLevel:     55.0,
		TireWear:      60.0,
		TireCompound:  "medium",
		Weather:       "dry",
		RemainingLaps: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.BuildSpecializedPrompt(raceData, "routine", config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetPromptTemplate(b *testing.B) {
	formats := []string{"sprint", "endurance", "standard"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		format := formats[i%len(formats)]
		GetPromptTemplate(format)
	}
}
