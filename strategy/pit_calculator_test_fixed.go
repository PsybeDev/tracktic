package strategy

import (
	"testing"
	"time"

	"changeme/sims"
)

func TestNewPitStopCalculator(t *testing.T) {
	config := &Config{}
	calculator := NewPitStopCalculator(config)

	if calculator == nil {
		t.Fatal("Expected calculator to be created, got nil")
	}

	if calculator.config != config {
		t.Error("Expected config to be set correctly")
	}

	if calculator.trackDatabase == nil {
		t.Error("Expected track database to be initialized")
	}

	if calculator.positionTracker == nil {
		t.Error("Expected position tracker to be initialized")
	}

	if calculator.timingAnalyzer == nil {
		t.Error("Expected timing analyzer to be initialized")
	}
}

func TestTrackDatabaseGetTrackData(t *testing.T) {
	db := NewTrackDatabase()

	// Test known track
	spa := db.GetTrackData("Spa-Francorchamps")
	if spa == nil {
		t.Fatal("Expected Spa track data, got nil")
	}

	if spa.Name != "Spa-Francorchamps" {
		t.Errorf("Expected track name 'Spa-Francorchamps', got '%s'", spa.Name)
	}

	if spa.Length != 7.004 {
		t.Errorf("Expected track length 7.004, got %f", spa.Length)
	}

	// Test unknown track - should return generic data
	unknown := db.GetTrackData("UnknownTrack")
	if unknown == nil {
		t.Fatal("Expected generic track data for unknown track, got nil")
	}

	if unknown.Name != "UnknownTrack" {
		t.Errorf("Expected track name 'UnknownTrack', got '%s'", unknown.Name)
	}

	if unknown.Length != 5.0 {
		t.Errorf("Expected generic track length 5.0, got %f", unknown.Length)
	}
}

func TestTrackDatabaseDefaultTracks(t *testing.T) {
	db := NewTrackDatabase()

	expectedTracks := []string{"Spa-Francorchamps", "Silverstone", "Monza"}

	for _, trackName := range expectedTracks {
		trackData := db.GetTrackData(trackName)
		if trackData == nil {
			t.Errorf("Expected track data for %s, got nil", trackName)
			continue
		}

		if trackData.Name != trackName {
			t.Errorf("Expected track name %s, got %s", trackName, trackData.Name)
		}

		if trackData.Length <= 0 {
			t.Errorf("Expected positive track length for %s, got %f", trackName, trackData.Length)
		}

		if trackData.PitSpeedLimit <= 0 {
			t.Errorf("Expected positive pit speed limit for %s, got %f", trackName, trackData.PitSpeedLimit)
		}
	}
}

func TestPositionTracker(t *testing.T) {
	tracker := NewPositionTracker()

	if tracker == nil {
		t.Fatal("Expected position tracker to be created, got nil")
	}

	if tracker.playerHistory == nil {
		t.Error("Expected player history to be initialized")
	}

	if tracker.opponentHistory == nil {
		t.Error("Expected opponent history to be initialized")
	}

	if tracker.trackLength != 5.0 {
		t.Errorf("Expected default track length 5.0, got %f", tracker.trackLength)
	}
}

func TestTimingAnalyzer(t *testing.T) {
	analyzer := NewTimingAnalyzer()

	if analyzer == nil {
		t.Fatal("Expected timing analyzer to be created, got nil")
	}

	if analyzer.sectorTimes == nil {
		t.Error("Expected sector times to be initialized")
	}

	if analyzer.lapTimePatterns == nil {
		t.Error("Expected lap time patterns to be initialized")
	}
}

func TestCalculatePitStopTiming(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})

	// Create test telemetry data
	data := &sims.TelemetryData{
		Session: sims.SessionInfo{
			TrackName: "Spa-Francorchamps",
			TotalLaps: 44,
			Flag:      sims.SessionFlagGreen,
		},
		Player: sims.PlayerData{
			Position:           5,
			CurrentLap:         15,
			LapDistancePercent: 45.0,
			Speed:              180.0,
			LastLapTime:        time.Minute + time.Second*30,
			BestLapTime:        time.Minute + time.Second*28,
			CurrentLapTime:     time.Minute + time.Second*32,
			GapToAhead:         time.Second * 8,
			GapToBehind:        time.Second * 12,
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 45.0},
				FrontRight: sims.TireWheelData{WearPercent: 42.0},
				RearLeft:   sims.TireWheelData{WearPercent: 48.0},
				RearRight:  sims.TireWheelData{WearPercent: 46.0},
			},
			Fuel: sims.FuelData{
				Level:             35.0,
				EstimatedLapsLeft: 18,
				UsagePerLap:       2.5,
			},
			Pit: sims.PitData{
				IsOnPitRoad: false,
				LastPitLap:  0,
			},
		},
		Opponents: []sims.OpponentData{
			{
				CarIndex:           2,
				DriverName:         "Test Driver",
				Position:           4,
				CurrentLap:         15,
				LapDistancePercent: 50.0,
				LastLapTime:        time.Minute + time.Second*29,
				GapToPlayer:        -time.Second * 8,
				IsOnPitRoad:        false,
				LastPitLap:         3,
			},
			{
				CarIndex:           7,
				DriverName:         "Another Driver",
				Position:           6,
				CurrentLap:         15,
				LapDistancePercent: 40.0,
				LastLapTime:        time.Minute + time.Second*31,
				GapToPlayer:        time.Second * 12,
				IsOnPitRoad:        false,
				LastPitLap:         5,
			},
		},
	}

	// Create test race analysis
	raceAnalysis := &RaceAnalysis{
		RaceFormat:     "endurance",
		StrategicPhase: "middle",
	}

	analysis := calculator.CalculatePitStopTiming(data, raceAnalysis)

	if analysis == nil {
		t.Fatal("Expected pit stop analysis, got nil")
	}

	// Test basic analysis structure
	if analysis.CurrentPosition.LapDistance != 45.0 {
		t.Errorf("Expected lap distance 45.0, got %f", analysis.CurrentPosition.LapDistance)
	}

	if analysis.CurrentPosition.EstimatedSpeed != 180.0 {
		t.Errorf("Expected speed 180.0, got %f", analysis.CurrentPosition.EstimatedSpeed)
	}

	// Test optimal windows
	if len(analysis.OptimalWindows) == 0 {
		t.Error("Expected at least one optimal pit window")
	}

	// Test position predictions
	if len(analysis.EstimatedPositions) == 0 {
		t.Error("Expected future position predictions")
	}

	// Test pit loss calculation
	if analysis.PitLossCalculation.TotalPitTime <= 0 {
		t.Error("Expected positive total pit time")
	}

	// Test recommendation
	if analysis.PrimaryRecommendation.TireCompound == "" {
		t.Error("Expected tire compound recommendation")
	}

	// Test confidence calculations
	if analysis.CalculationConfidence < 0 || analysis.CalculationConfidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", analysis.CalculationConfidence)
	}

	if analysis.DataQuality < 0 || analysis.DataQuality > 1 {
		t.Errorf("Expected data quality between 0 and 1, got %f", analysis.DataQuality)
	}
}

func TestCalculateOptimalWindows(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})
	trackData := &TrackData{
		Name:         "TestTrack",
		PitLaneDelta: time.Second * 22,
	}

	// Test with high tire wear
	data := &sims.TelemetryData{
		Session: sims.SessionInfo{TotalLaps: 50},
		Player: sims.PlayerData{
			CurrentLap: 20,
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 75.0},
				FrontRight: sims.TireWheelData{WearPercent: 72.0},
				RearLeft:   sims.TireWheelData{WearPercent: 78.0},
				RearRight:  sims.TireWheelData{WearPercent: 74.0},
			},
			Fuel: sims.FuelData{EstimatedLapsLeft: 25},
		},
	}

	raceAnalysis := &RaceAnalysis{}

	windows := calculator.calculateOptimalWindows(data, trackData, raceAnalysis)

	// Should have strategic window due to high tire wear
	hasStrategicWindow := false
	for _, window := range windows {
		if window.WindowType == "strategic" {
			hasStrategicWindow = true
			if window.Confidence < 0 || window.Confidence > 1 {
				t.Errorf("Expected confidence between 0 and 1, got %f", window.Confidence)
			}
			break
		}
	}

	if !hasStrategicWindow {
		t.Error("Expected strategic pit window with high tire wear")
	}
}

func TestCalculateOptimalWindowsFuelShortage(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})
	trackData := &TrackData{
		Name:         "TestTrack",
		PitLaneDelta: time.Second * 22,
	}

	// Test with low fuel
	data := &sims.TelemetryData{
		Session: sims.SessionInfo{TotalLaps: 50},
		Player: sims.PlayerData{
			CurrentLap: 20,
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 30.0},
				FrontRight: sims.TireWheelData{WearPercent: 28.0},
				RearLeft:   sims.TireWheelData{WearPercent: 32.0},
				RearRight:  sims.TireWheelData{WearPercent: 29.0},
			},
			Fuel: sims.FuelData{EstimatedLapsLeft: 4}, // Low fuel!
		},
	}

	raceAnalysis := &RaceAnalysis{}

	windows := calculator.calculateOptimalWindows(data, trackData, raceAnalysis)

	// Should have forced window due to low fuel
	hasForcedWindow := false
	for _, window := range windows {
		if window.WindowType == "forced" {
			hasForcedWindow = true
			if window.RiskLevel != "high" {
				t.Errorf("Expected high risk level for forced window, got %s", window.RiskLevel)
			}
			break
		}
	}

	if !hasForcedWindow {
		t.Error("Expected forced pit window with low fuel")
	}
}

func TestPredictFuturePositions(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})
	trackData := &TrackData{Name: "TestTrack"}

	data := &sims.TelemetryData{
		Player: sims.PlayerData{
			Position:    5,
			CurrentLap:  15,
			LastLapTime: time.Minute + time.Second*30,
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 40.0},
				FrontRight: sims.TireWheelData{WearPercent: 38.0},
				RearLeft:   sims.TireWheelData{WearPercent: 42.0},
				RearRight:  sims.TireWheelData{WearPercent: 39.0},
			},
			Pit: sims.PitData{LastPitLap: 0},
		},
	}

	positions := calculator.predictFuturePositions(data, trackData)

	if len(positions) != 5 {
		t.Errorf("Expected 5 future positions, got %d", len(positions))
	}

	for i, pos := range positions {
		expectedLap := 15 + i + 1
		if pos.Lap != expectedLap {
			t.Errorf("Expected lap %d, got %d", expectedLap, pos.Lap)
		}

		if pos.Confidence < 0 || pos.Confidence > 1 {
			t.Errorf("Expected confidence between 0 and 1, got %f", pos.Confidence)
		}

		if len(pos.InfluencingFactors) == 0 {
			t.Error("Expected influencing factors to be populated")
		}
	}
}

func TestCalculateDetailedPitLoss(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})
	trackData := &TrackData{
		PitLaneLength:  0.5,  // 500m
		PitSpeedLimit:  60.0, // 60 km/h
		TypicalPitTime: time.Second * 25,
		Length:         5.0, // 5km track
	}

	data := &sims.TelemetryData{
		Player: sims.PlayerData{
			LastLapTime: time.Minute + time.Second*30, // 90 seconds
		},
	}

	calc := calculator.calculateDetailedPitLoss(data, trackData)

	if calc.TotalPitTime <= 0 {
		t.Error("Expected positive total pit time")
	}

	if calc.NetTimeLoss <= 0 {
		t.Error("Expected positive net time loss")
	}

	expectedComponents := []time.Duration{
		calc.PitLaneEntry,
		calc.PitLaneTravel,
		calc.StationaryTime,
		calc.PitLaneExit,
	}

	for i, component := range expectedComponents {
		if component <= 0 {
			t.Errorf("Expected positive time for component %d, got %v", i, component)
		}
	}

	// Total time should be sum of components
	expectedTotal := calc.PitLaneEntry + calc.PitLaneTravel +
		calc.StationaryTime + calc.PitLaneExit
	if calc.TotalPitTime != expectedTotal {
		t.Errorf("Expected total time %v, got %v", expectedTotal, calc.TotalPitTime)
	}
}

func TestIdentifyRiskFactorsTireDegradation(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})
	trackData := &TrackData{Name: "TestTrack"}
	raceAnalysis := &RaceAnalysis{}

	// Test high tire wear scenario
	data := &sims.TelemetryData{
		Player: sims.PlayerData{
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 85.0}, // High wear
				FrontRight: sims.TireWheelData{WearPercent: 82.0},
				RearLeft:   sims.TireWheelData{WearPercent: 87.0},
				RearRight:  sims.TireWheelData{WearPercent: 84.0},
			},
			Fuel: sims.FuelData{EstimatedLapsLeft: 15},
		},
		Opponents: []sims.OpponentData{},
	}

	risks := calculator.identifyRiskFactors(data, trackData, raceAnalysis)

	// Should identify tire degradation risk
	hasTireDegradationRisk := false
	for _, risk := range risks {
		if risk.RiskType == "tire_degradation" {
			hasTireDegradationRisk = true
			if risk.Severity != "high" {
				t.Errorf("Expected high severity for tire degradation, got %s", risk.Severity)
			}
			if risk.Probability < 0 || risk.Probability > 1 {
				t.Errorf("Expected probability between 0 and 1, got %f", risk.Probability)
			}
			break
		}
	}

	if !hasTireDegradationRisk {
		t.Error("Expected tire degradation risk with high tire wear")
	}
}

func TestIdentifyRiskFactorsFuelShortage(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})
	trackData := &TrackData{Name: "TestTrack"}
	raceAnalysis := &RaceAnalysis{}

	// Test low fuel scenario
	data := &sims.TelemetryData{
		Player: sims.PlayerData{
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 30.0},
				FrontRight: sims.TireWheelData{WearPercent: 28.0},
				RearLeft:   sims.TireWheelData{WearPercent: 32.0},
				RearRight:  sims.TireWheelData{WearPercent: 29.0},
			},
			Fuel: sims.FuelData{EstimatedLapsLeft: 3}, // Very low fuel
		},
		Opponents: []sims.OpponentData{},
	}

	risks := calculator.identifyRiskFactors(data, trackData, raceAnalysis)

	// Should identify fuel shortage risk
	hasFuelShortageRisk := false
	for _, risk := range risks {
		if risk.RiskType == "fuel_shortage" {
			hasFuelShortageRisk = true
			if risk.Severity != "critical" {
				t.Errorf("Expected critical severity for fuel shortage, got %s", risk.Severity)
			}
			if risk.Probability != 1.0 {
				t.Errorf("Expected probability 1.0 for fuel shortage, got %f", risk.Probability)
			}
			break
		}
	}

	if !hasFuelShortageRisk {
		t.Error("Expected fuel shortage risk with very low fuel")
	}
}

func TestGenerateAlternativeOptions(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})

	analysis := &PitStopAnalysis{}
	data := &sims.TelemetryData{
		Player: sims.PlayerData{CurrentLap: 20},
	}

	alternatives := calculator.generateAlternativeOptions(analysis, data)

	if len(alternatives) < 2 {
		t.Error("Expected at least 2 alternative options")
	}

	for _, alt := range alternatives {
		if alt.Lap <= 0 {
			t.Error("Expected positive lap number for alternative")
		}

		if alt.TireCompound == "" {
			t.Error("Expected tire compound to be specified")
		}

		if alt.FuelLoad <= 0 {
			t.Error("Expected positive fuel load")
		}

		if len(alt.Pros) == 0 {
			t.Error("Expected pros to be populated")
		}

		if len(alt.Cons) == 0 {
			t.Error("Expected cons to be populated")
		}

		if alt.RiskLevel == "" {
			t.Error("Expected risk level to be specified")
		}
	}
}

func TestCalculateConfidence(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})

	// Add some history for better confidence
	for i := 0; i < 15; i++ {
		snapshot := PositionSnapshot{
			TrackPosition: float64(i) * 0.1,
			Speed:         150.0,
		}
		calculator.positionTracker.playerHistory = append(
			calculator.positionTracker.playerHistory,
			snapshot,
		)
	}

	data := &sims.TelemetryData{
		Session: sims.SessionInfo{Flag: sims.SessionFlagGreen},
		Player: sims.PlayerData{
			GapToAhead:  time.Second * 15,
			GapToBehind: time.Second * 20,
		},
	}

	confidence := calculator.calculateConfidence(data)

	if confidence < 0 || confidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", confidence)
	}

	// With good history and clear track position, should have decent confidence
	if confidence < 0.5 {
		t.Errorf("Expected confidence >= 0.5 with good conditions, got %f", confidence)
	}
}

func TestAssessDataQuality(t *testing.T) {
	calculator := NewPitStopCalculator(&Config{})

	// Add various types of data for quality assessment
	for i := 0; i < 10; i++ {
		snapshot := PositionSnapshot{
			TrackPosition: float64(i) * 0.1,
		}
		calculator.positionTracker.playerHistory = append(
			calculator.positionTracker.playerHistory,
			snapshot,
		)
	}

	for i := 0; i < 5; i++ {
		pattern := LapTimePattern{
			BaseLapTime: time.Minute + time.Second*30,
		}
		calculator.timingAnalyzer.lapTimePatterns = append(
			calculator.timingAnalyzer.lapTimePatterns,
			pattern,
		)
	}

	calculator.positionTracker.opponentHistory[1] = []PositionSnapshot{{}}
	calculator.positionTracker.opponentHistory[2] = []PositionSnapshot{{}}
	calculator.positionTracker.opponentHistory[3] = []PositionSnapshot{{}}
	calculator.positionTracker.opponentHistory[4] = []PositionSnapshot{{}}

	quality := calculator.assessDataQuality()

	if quality < 0 || quality > 1 {
		t.Errorf("Expected data quality between 0 and 1, got %f", quality)
	}

	// With good data availability, should have high quality
	if quality < 0.8 {
		t.Errorf("Expected data quality >= 0.8 with good data, got %f", quality)
	}
}

// Benchmark tests
func BenchmarkCalculatePitStopTiming(b *testing.B) {
	calculator := NewPitStopCalculator(&Config{})

	data := &sims.TelemetryData{
		Session: sims.SessionInfo{
			TrackName: "Spa-Francorchamps",
			TotalLaps: 44,
		},
		Player: sims.PlayerData{
			Position:           5,
			CurrentLap:         15,
			LapDistancePercent: 45.0,
			LastLapTime:        time.Minute + time.Second*30,
			Tires: sims.TireData{
				FrontLeft:  sims.TireWheelData{WearPercent: 45.0},
				FrontRight: sims.TireWheelData{WearPercent: 42.0},
				RearLeft:   sims.TireWheelData{WearPercent: 48.0},
				RearRight:  sims.TireWheelData{WearPercent: 46.0},
			},
			Fuel: sims.FuelData{
				Level:             35.0,
				EstimatedLapsLeft: 18,
			},
		},
		Opponents: make([]sims.OpponentData, 10),
	}

	raceAnalysis := &RaceAnalysis{
		RaceFormat:     "endurance",
		StrategicPhase: "middle",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculator.CalculatePitStopTiming(data, raceAnalysis)
	}
}
