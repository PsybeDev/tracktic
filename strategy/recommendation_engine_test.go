package strategy

import (
	"testing"
	"time"

	"changeme/sims"
)

func TestNewRecommendationEngine(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	if engine == nil {
		t.Fatal("Expected engine to be created, got nil")
	}

	if engine.config != config {
		t.Error("Expected config to be set correctly")
	}

	if len(engine.telemetryHistory) != 0 {
		t.Error("Expected empty telemetry history initially")
	}
}

func TestCalculateAverageWear(t *testing.T) {
	tires := sims.TireData{
		FrontLeft:  sims.TireWheelData{WearPercent: 10.0},
		FrontRight: sims.TireWheelData{WearPercent: 15.0},
		RearLeft:   sims.TireWheelData{WearPercent: 20.0},
		RearRight:  sims.TireWheelData{WearPercent: 25.0},
	}

	expectedAvg := (10.0 + 15.0 + 20.0 + 25.0) / 4.0
	actualAvg := CalculateAverageWear(tires)

	if actualAvg != expectedAvg {
		t.Errorf("Expected average wear %.2f, got %.2f", expectedAvg, actualAvg)
	}
}

func TestCalculateAverageTireTemp(t *testing.T) {
	tires := sims.TireData{
		FrontLeft:  sims.TireWheelData{Temperature: 80.0},
		FrontRight: sims.TireWheelData{Temperature: 85.0},
		RearLeft:   sims.TireWheelData{Temperature: 90.0},
		RearRight:  sims.TireWheelData{Temperature: 95.0},
	}

	expectedAvg := (80.0 + 85.0 + 90.0 + 95.0) / 4.0
	actualAvg := CalculateAverageTireTemp(tires)

	if actualAvg != expectedAvg {
		t.Errorf("Expected average temperature %.2f, got %.2f", expectedAvg, actualAvg)
	}
}

func TestAddTelemetrySnapshot(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Create test telemetry data
	testData := &sims.TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: sims.SimulatorTypeACC,
		IsConnected:   true,
		Session: sims.SessionInfo{
			Type:      sims.SessionTypeRace,
			TotalLaps: 30,
			TrackName: "Spa-Francorchamps",
		},
		Player: sims.PlayerData{
			Position:   5,
			CurrentLap: 10,
			Fuel: sims.FuelData{
				Level:       25.5,
				UsagePerLap: 2.3,
			},
			Tires: sims.TireData{
				Compound:   "medium",
				FrontLeft:  sims.TireWheelData{WearPercent: 15.0, Temperature: 85.0},
				FrontRight: sims.TireWheelData{WearPercent: 18.0, Temperature: 87.0},
				RearLeft:   sims.TireWheelData{WearPercent: 12.0, Temperature: 82.0},
				RearRight:  sims.TireWheelData{WearPercent: 14.0, Temperature: 84.0},
			},
		},
	}

	// Add the snapshot
	engine.AddTelemetrySnapshot(testData)

	// Verify snapshot was added
	if len(engine.telemetryHistory) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(engine.telemetryHistory))
	}

	if engine.telemetryHistory[0].Data != testData {
		t.Error("Expected snapshot data to match input data")
	}
}

func TestLapAnalysisWithMultipleDataPoints(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Add multiple telemetry snapshots with different lap times
	lapTimes := []time.Duration{
		time.Minute*1 + time.Second*23,                        // 1:23.000
		time.Minute*1 + time.Second*22 + time.Millisecond*500, // 1:22.500
		time.Minute*1 + time.Second*23 + time.Millisecond*200, // 1:23.200
		time.Minute*1 + time.Second*22 + time.Millisecond*800, // 1:22.800
		time.Minute*1 + time.Second*23 + time.Millisecond*100, // 1:23.100
	}

	for i, lapTime := range lapTimes {
		testData := &sims.TelemetryData{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			SimulatorType: sims.SimulatorTypeACC,
			IsConnected:   true,
			Player: sims.PlayerData{
				CurrentLap:  i + 1,
				LastLapTime: lapTime,
				BestLapTime: time.Minute*1 + time.Second*22,
			},
		}
		engine.AddTelemetrySnapshot(testData)
	}

	// Check that lap analysis was performed
	if len(engine.lapAnalysis.RecentLapTimes) == 0 {
		t.Error("Expected recent lap times to be populated")
	}

	if engine.lapAnalysis.ConsistencyScore < 0 || engine.lapAnalysis.ConsistencyScore > 1 {
		t.Errorf("Expected consistency score between 0 and 1, got %.3f", engine.lapAnalysis.ConsistencyScore)
	}

	if engine.lapAnalysis.TrendDirection == "" {
		t.Error("Expected trend direction to be set")
	}
}

func TestFuelAnalysisCalculation(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Add multiple snapshots with decreasing fuel levels
	fuelLevels := []float64{30.0, 27.7, 25.4, 23.1, 20.8} // ~2.3L per lap consumption

	for i, fuelLevel := range fuelLevels {
		testData := &sims.TelemetryData{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			SimulatorType: sims.SimulatorTypeACC,
			IsConnected:   true,
			Player: sims.PlayerData{
				CurrentLap: i + 1,
				Fuel: sims.FuelData{
					Level: fuelLevel,
				},
			},
		}
		engine.AddTelemetrySnapshot(testData)
	}

	// Check fuel analysis calculations
	if engine.fuelAnalysis.AverageConsumption <= 0 {
		t.Error("Expected positive average fuel consumption")
	}

	// Should be approximately 2.3L per lap
	expectedConsumption := 2.3
	tolerance := 0.5
	if engine.fuelAnalysis.AverageConsumption < expectedConsumption-tolerance ||
		engine.fuelAnalysis.AverageConsumption > expectedConsumption+tolerance {
		t.Errorf("Expected consumption around %.1f, got %.3f", expectedConsumption, engine.fuelAnalysis.AverageConsumption)
	}
}

func TestTireAnalysisWithWearProgression(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Add snapshots with increasing tire wear
	wearProgression := []float64{10.0, 15.0, 20.0, 25.0, 30.0}

	for i, wear := range wearProgression {
		testData := &sims.TelemetryData{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			SimulatorType: sims.SimulatorTypeACC,
			IsConnected:   true,
			Player: sims.PlayerData{
				CurrentLap: i + 1,
				Tires: sims.TireData{
					FrontLeft:  sims.TireWheelData{WearPercent: wear},
					FrontRight: sims.TireWheelData{WearPercent: wear + 2},
					RearLeft:   sims.TireWheelData{WearPercent: wear - 1},
					RearRight:  sims.TireWheelData{WearPercent: wear + 1},
				},
				Pit: sims.PitData{
					LastPitLap: 0, // No previous pit stop
				},
			},
		}
		engine.AddTelemetrySnapshot(testData)
	}

	// Check tire analysis
	if engine.tireAnalysis.DegradationRate <= 0 {
		t.Error("Expected positive tire degradation rate")
	}

	if engine.tireAnalysis.OptimalStintLength <= 0 {
		t.Error("Expected positive optimal stint length")
	}
}

func TestGenerateRecommendation(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Create comprehensive test data
	testData := &sims.TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: sims.SimulatorTypeACC,
		IsConnected:   true,
		Session: sims.SessionInfo{
			Type:             sims.SessionTypeRace,
			TotalLaps:        30,
			TrackName:        "Spa-Francorchamps",
			AirTemperature:   22.0,
			TrackTemperature: 28.0,
			Flag:             sims.SessionFlagGreen,
		},
		Player: sims.PlayerData{
			Position:       5,
			CurrentLap:     15,
			LastLapTime:    time.Minute*1 + time.Second*23,
			BestLapTime:    time.Minute*1 + time.Second*22,
			CurrentLapTime: time.Minute*1 + time.Second*23 + time.Millisecond*500,
			Fuel: sims.FuelData{
				Level:             20.5,
				UsagePerLap:       2.2,
				EstimatedLapsLeft: 9,
			},
			Tires: sims.TireData{
				Compound:   "medium",
				FrontLeft:  sims.TireWheelData{WearPercent: 45.0, Temperature: 85.0},
				FrontRight: sims.TireWheelData{WearPercent: 48.0, Temperature: 87.0},
				RearLeft:   sims.TireWheelData{WearPercent: 42.0, Temperature: 82.0},
				RearRight:  sims.TireWheelData{WearPercent: 44.0, Temperature: 84.0},
			},
			Pit: sims.PitData{
				LastPitLap:       5,
				PitWindowOpen:    true,
				EstimatedPitTime: time.Second * 25,
			},
		},
		Opponents: []sims.OpponentData{
			{
				Position:    4,
				DriverName:  "Test Driver 1",
				GapToPlayer: time.Second * 3,
				LastLapTime: time.Minute*1 + time.Second*22 + time.Millisecond*800,
				LastPitLap:  3,
			},
			{
				Position:    6,
				DriverName:  "Test Driver 2",
				GapToPlayer: -time.Second * 2,
				LastLapTime: time.Minute*1 + time.Second*23 + time.Millisecond*200,
				LastPitLap:  6,
			},
		},
	}

	// Generate recommendation
	recommendation := engine.GenerateRecommendation(testData)

	// Verify recommendation structure
	if recommendation == nil {
		t.Fatal("Expected recommendation to be generated, got nil")
	}

	if recommendation.PrimaryStrategy == "" {
		t.Error("Expected primary strategy to be set")
	}

	if recommendation.ConfidenceLevel < 0 || recommendation.ConfidenceLevel > 1 {
		t.Errorf("Expected confidence level between 0 and 1, got %.3f", recommendation.ConfidenceLevel)
	}

	if recommendation.RiskAssessment == "" {
		t.Error("Expected risk assessment to be set")
	}

	if len(recommendation.ImmediateActions) == 0 {
		t.Error("Expected immediate actions to be provided")
	}

	if len(recommendation.LapTargets) == 0 {
		t.Error("Expected lap targets to be provided")
	}

	// Verify pit recommendation is present
	if recommendation.PitRecommendation.OptimalLap <= 0 {
		t.Error("Expected positive optimal pit lap")
	}

	// Verify fuel management plan
	if recommendation.FuelManagement.CurrentConsumption <= 0 {
		t.Error("Expected positive current consumption in fuel management")
	}

	// Verify tire management plan
	if recommendation.TireManagement.OptimalStintLength <= 0 {
		t.Error("Expected positive optimal stint length in tire management")
	}

	// Verify finish prediction
	if recommendation.FinishPrediction.EstimatedPosition <= 0 {
		t.Error("Expected positive estimated finish position")
	}
}

func TestRaceFormatDetection(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	tests := []struct {
		name          string
		totalLaps     int
		timeRemaining time.Duration
		expected      string
	}{
		{"Sprint race - few laps", 10, 0, "sprint"},
		{"Standard race - medium laps", 25, 0, "standard"},
		{"Endurance race - many laps", 60, 0, "endurance"},
		{"Sprint time-based", 0, time.Minute * 45, "sprint"},
		{"Standard time-based", 0, time.Hour + time.Minute*30, "standard"},
		{"Endurance time-based", 0, time.Hour * 3, "endurance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := &sims.TelemetryData{
				Session: sims.SessionInfo{
					TotalLaps:     tt.totalLaps,
					TimeRemaining: tt.timeRemaining,
					SessionTime:   tt.timeRemaining * 2, // Assume we're halfway through
				},
			}

			engine.AddTelemetrySnapshot(testData)

			if engine.raceAnalysis.RaceFormat != tt.expected {
				t.Errorf("Expected race format %s, got %s", tt.expected, engine.raceAnalysis.RaceFormat)
			}
		})
	}
}

func TestDataQualityCalculation(t *testing.T) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Initially should have low data quality
	quality := engine.calculateDataQuality()
	if quality > 0.3 {
		t.Errorf("Expected low initial data quality, got %.3f", quality)
	}

	// Add several telemetry snapshots
	for i := 0; i < 10; i++ {
		testData := &sims.TelemetryData{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			SimulatorType: sims.SimulatorTypeACC,
			IsConnected:   true,
			Player: sims.PlayerData{
				CurrentLap:  i + 1,
				LastLapTime: time.Minute*1 + time.Second*23,
				Fuel: sims.FuelData{
					Level:       30.0 - float64(i)*2.3,
					UsagePerLap: 2.3,
				},
				Tires: sims.TireData{
					FrontLeft: sims.TireWheelData{WearPercent: float64(i * 5)},
				},
			},
		}
		engine.AddTelemetrySnapshot(testData)
	}

	// Should have better data quality now
	quality = engine.calculateDataQuality()
	if quality < 0.7 {
		t.Errorf("Expected high data quality with sufficient data, got %.3f", quality)
	}
}

// Benchmark tests for performance
func BenchmarkAddTelemetrySnapshot(b *testing.B) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	testData := &sims.TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: sims.SimulatorTypeACC,
		IsConnected:   true,
		Player: sims.PlayerData{
			CurrentLap: 10,
			Fuel:       sims.FuelData{Level: 25.0},
			Tires: sims.TireData{
				FrontLeft: sims.TireWheelData{WearPercent: 30.0},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.AddTelemetrySnapshot(testData)
	}
}

func BenchmarkGenerateRecommendation(b *testing.B) {
	config := DefaultConfig()
	engine := NewRecommendationEngine(config)

	// Pre-populate with some data
	for i := 0; i < 10; i++ {
		testData := &sims.TelemetryData{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Minute),
			SimulatorType: sims.SimulatorTypeACC,
			IsConnected:   true,
			Player: sims.PlayerData{
				CurrentLap:  i + 1,
				LastLapTime: time.Minute*1 + time.Second*23,
				Fuel:        sims.FuelData{Level: 30.0 - float64(i)*2.3},
			},
		}
		engine.AddTelemetrySnapshot(testData)
	}

	testData := &sims.TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: sims.SimulatorTypeACC,
		IsConnected:   true,
		Session: sims.SessionInfo{
			Type:      sims.SessionTypeRace,
			TotalLaps: 30,
		},
		Player: sims.PlayerData{
			Position:    5,
			CurrentLap:  15,
			LastLapTime: time.Minute*1 + time.Second*23,
			Fuel:        sims.FuelData{Level: 20.0},
			Tires:       sims.TireData{Compound: "medium"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.GenerateRecommendation(testData)
	}
}
