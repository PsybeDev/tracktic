package sims

import (
	"testing"
	"time"
)

func TestCalculateRaceFormat(t *testing.T) {
	tests := []struct {
		name     string
		session  SessionInfo
		expected RaceFormat
	}{
		{
			name: "Long timed session should be endurance",
			session: SessionInfo{
				IsTimedSession:  true,
				TimeRemaining:   90 * time.Minute,
				IsLappedSession: false,
			},
			expected: RaceFormatEndurance,
		},
		{
			name: "Short timed session should be sprint",
			session: SessionInfo{
				IsTimedSession:  true,
				TimeRemaining:   30 * time.Minute,
				IsLappedSession: false,
			},
			expected: RaceFormatSprint,
		},
		{
			name: "Many laps should be endurance",
			session: SessionInfo{
				IsTimedSession:  false,
				IsLappedSession: true,
				TotalLaps:       100,
			},
			expected: RaceFormatEndurance,
		},
		{
			name: "Few laps should be sprint",
			session: SessionInfo{
				IsTimedSession:  false,
				IsLappedSession: true,
				TotalLaps:       25,
			},
			expected: RaceFormatSprint,
		},
		{
			name: "Unknown session type should be unknown format",
			session: SessionInfo{
				IsTimedSession:  false,
				IsLappedSession: false,
			},
			expected: RaceFormatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateRaceFormat(&tt.session)
			if result != tt.expected {
				t.Errorf("CalculateRaceFormat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateTireWearLevel(t *testing.T) {
	tests := []struct {
		name     string
		tires    TireData
		expected TireWearLevel
	}{
		{
			name: "Fresh tires",
			tires: TireData{
				FrontLeft:  TireWheelData{WearPercent: 5},
				FrontRight: TireWheelData{WearPercent: 5},
				RearLeft:   TireWheelData{WearPercent: 5},
				RearRight:  TireWheelData{WearPercent: 5},
			},
			expected: TireWearFresh,
		},
		{
			name: "Good tires",
			tires: TireData{
				FrontLeft:  TireWheelData{WearPercent: 20},
				FrontRight: TireWheelData{WearPercent: 20},
				RearLeft:   TireWheelData{WearPercent: 20},
				RearRight:  TireWheelData{WearPercent: 20},
			},
			expected: TireWearGood,
		},
		{
			name: "Medium wear tires",
			tires: TireData{
				FrontLeft:  TireWheelData{WearPercent: 45},
				FrontRight: TireWheelData{WearPercent: 45},
				RearLeft:   TireWheelData{WearPercent: 45},
				RearRight:  TireWheelData{WearPercent: 45},
			},
			expected: TireWearMedium,
		},
		{
			name: "Worn tires",
			tires: TireData{
				FrontLeft:  TireWheelData{WearPercent: 75},
				FrontRight: TireWheelData{WearPercent: 75},
				RearLeft:   TireWheelData{WearPercent: 75},
				RearRight:  TireWheelData{WearPercent: 75},
			},
			expected: TireWearWorn,
		},
		{
			name: "Critical wear tires",
			tires: TireData{
				FrontLeft:  TireWheelData{WearPercent: 90},
				FrontRight: TireWheelData{WearPercent: 90},
				RearLeft:   TireWheelData{WearPercent: 90},
				RearRight:  TireWheelData{WearPercent: 90},
			},
			expected: TireWearCritical,
		},
		{
			name: "Uneven wear should average",
			tires: TireData{
				FrontLeft:  TireWheelData{WearPercent: 10},
				FrontRight: TireWheelData{WearPercent: 20},
				RearLeft:   TireWheelData{WearPercent: 30},
				RearRight:  TireWheelData{WearPercent: 40},
			},
			expected: TireWearGood, // Average: 25%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTireWearLevel(&tt.tires)
			if result != tt.expected {
				t.Errorf("CalculateTireWearLevel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateTireTempLevel(t *testing.T) {
	tests := []struct {
		name     string
		tires    TireData
		expected TireTempLevel
	}{
		{
			name: "Cold tires",
			tires: TireData{
				FrontLeft:  TireWheelData{Temperature: 60},
				FrontRight: TireWheelData{Temperature: 60},
				RearLeft:   TireWheelData{Temperature: 60},
				RearRight:  TireWheelData{Temperature: 60},
			},
			expected: TireTempCold,
		},
		{
			name: "Optimal temperature tires",
			tires: TireData{
				FrontLeft:  TireWheelData{Temperature: 85},
				FrontRight: TireWheelData{Temperature: 85},
				RearLeft:   TireWheelData{Temperature: 85},
				RearRight:  TireWheelData{Temperature: 85},
			},
			expected: TireTempOptimal,
		},
		{
			name: "Hot tires",
			tires: TireData{
				FrontLeft:  TireWheelData{Temperature: 110},
				FrontRight: TireWheelData{Temperature: 110},
				RearLeft:   TireWheelData{Temperature: 110},
				RearRight:  TireWheelData{Temperature: 110},
			},
			expected: TireTempHot,
		},
		{
			name: "Overheating tires",
			tires: TireData{
				FrontLeft:  TireWheelData{Temperature: 130},
				FrontRight: TireWheelData{Temperature: 130},
				RearLeft:   TireWheelData{Temperature: 130},
				RearRight:  TireWheelData{Temperature: 130},
			},
			expected: TireTempOverheat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTireTempLevel(&tt.tires)
			if result != tt.expected {
				t.Errorf("CalculateTireTempLevel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateFuelEstimates(t *testing.T) {
	tests := []struct {
		name            string
		fuel            FuelData
		averageLapTime  time.Duration
		expectedLaps    int
		expectedPercent float64
		expectedWarning bool
	}{
		{
			name: "Normal fuel level",
			fuel: FuelData{
				Level:       50.0,
				Capacity:    100.0,
				UsagePerLap: 2.5,
			},
			averageLapTime:  90 * time.Second,
			expectedLaps:    20,
			expectedPercent: 50.0,
			expectedWarning: false,
		},
		{
			name: "Low fuel warning",
			fuel: FuelData{
				Level:       10.0,
				Capacity:    100.0,
				UsagePerLap: 2.5,
			},
			averageLapTime:  90 * time.Second,
			expectedLaps:    4,
			expectedPercent: 10.0,
			expectedWarning: true,
		},
		{
			name: "Full tank",
			fuel: FuelData{
				Level:       100.0,
				Capacity:    100.0,
				UsagePerLap: 2.0,
			},
			averageLapTime:  85 * time.Second,
			expectedLaps:    50,
			expectedPercent: 100.0,
			expectedWarning: false,
		},
		{
			name: "Zero usage per lap",
			fuel: FuelData{
				Level:       50.0,
				Capacity:    100.0,
				UsagePerLap: 0.0,
			},
			averageLapTime:  90 * time.Second,
			expectedLaps:    0,
			expectedPercent: 50.0,
			expectedWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fuel := tt.fuel // Copy to avoid modifying test data
			CalculateFuelEstimates(&fuel, tt.averageLapTime)

			if fuel.EstimatedLapsLeft != tt.expectedLaps {
				t.Errorf("EstimatedLapsLeft = %v, want %v", fuel.EstimatedLapsLeft, tt.expectedLaps)
			}

			if fuel.Percentage != tt.expectedPercent {
				t.Errorf("Percentage = %v, want %v", fuel.Percentage, tt.expectedPercent)
			}

			if fuel.LowFuelWarning != tt.expectedWarning {
				t.Errorf("LowFuelWarning = %v, want %v", fuel.LowFuelWarning, tt.expectedWarning)
			}

			// Check estimated time calculation when usage > 0
			if tt.fuel.UsagePerLap > 0 && tt.averageLapTime > 0 {
				expectedTime := time.Duration(tt.expectedLaps) * tt.averageLapTime
				if fuel.EstimatedTimeLeft != expectedTime {
					t.Errorf("EstimatedTimeLeft = %v, want %v", fuel.EstimatedTimeLeft, expectedTime)
				}
			}
		})
	}
}

func TestTelemetryDataValidation(t *testing.T) {
	// Test that TelemetryData structure can be created and populated
	now := time.Now()

	data := &TelemetryData{
		Timestamp:     now,
		SimulatorType: SimulatorTypeIRacing,
		IsConnected:   true,
		Session: SessionInfo{
			Type:             SessionTypeRace,
			Format:           RaceFormatSprint,
			Flag:             SessionFlagGreen,
			TimeRemaining:    30 * time.Minute,
			LapsRemaining:    25,
			TotalLaps:        30,
			SessionTime:      15 * time.Minute,
			IsTimedSession:   false,
			IsLappedSession:  true,
			TrackName:        "Test Track",
			TrackLength:      4.2,
			AirTemperature:   25.0,
			TrackTemperature: 35.0,
		},
		Player: PlayerData{
			Position:           5,
			CurrentLap:         10,
			LapDistancePercent: 45.5,
			LastLapTime:        90 * time.Second,
			BestLapTime:        88 * time.Second,
			CurrentLapTime:     25 * time.Second,
			GapToLeader:        15 * time.Second,
			GapToAhead:         3 * time.Second,
			GapToBehind:        2 * time.Second,
			Fuel: FuelData{
				Level:             45.0,
				Capacity:          60.0,
				Percentage:        75.0,
				UsagePerLap:       2.2,
				UsagePerHour:      88.0,
				EstimatedLapsLeft: 20,
				EstimatedTimeLeft: 30 * time.Minute,
				LowFuelWarning:    false,
			},
			Tires: TireData{
				Compound:  "Medium",
				WearLevel: TireWearGood,
				TempLevel: TireTempOptimal,
				FrontLeft: TireWheelData{
					Temperature: 85.0,
					Pressure:    28.5,
					WearPercent: 15.0,
					DirtLevel:   0.1,
				},
				FrontRight: TireWheelData{
					Temperature: 87.0,
					Pressure:    28.3,
					WearPercent: 16.0,
					DirtLevel:   0.1,
				},
				RearLeft: TireWheelData{
					Temperature: 82.0,
					Pressure:    27.8,
					WearPercent: 18.0,
					DirtLevel:   0.2,
				},
				RearRight: TireWheelData{
					Temperature: 83.0,
					Pressure:    27.9,
					WearPercent: 17.0,
					DirtLevel:   0.2,
				},
			},
			Pit: PitData{
				IsOnPitRoad:       false,
				IsInPitStall:      false,
				PitWindowOpen:     true,
				PitWindowLapsLeft: 5,
				LastPitLap:        0,
				LastPitTime:       0,
				EstimatedPitTime:  25 * time.Second,
				PitSpeedLimit:     80.0,
			},
			Speed:    180.5,
			RPM:      7500,
			Gear:     4,
			Throttle: 85.0,
			Brake:    0.0,
			Clutch:   0.0,
			Steering: -15.5,
		},
		Opponents: []OpponentData{
			{
				CarIndex:           1,
				DriverName:         "Test Driver 1",
				CarNumber:          "42",
				Position:           1,
				CurrentLap:         10,
				LapDistancePercent: 55.0,
				LastLapTime:        89 * time.Second,
				BestLapTime:        87 * time.Second,
				GapToPlayer:        -15 * time.Second,
				IsOnPitRoad:        false,
				IsInPitStall:       false,
				LastPitLap:         0,
				EstimatedPitTime:   24 * time.Second,
			},
		},
	}

	// Basic validation tests
	if data.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if data.SimulatorType != SimulatorTypeIRacing {
		t.Errorf("SimulatorType = %v, want %v", data.SimulatorType, SimulatorTypeIRacing)
	}

	if !data.IsConnected {
		t.Error("IsConnected should be true")
	}

	if data.Session.Type != SessionTypeRace {
		t.Errorf("Session.Type = %v, want %v", data.Session.Type, SessionTypeRace)
	}

	if data.Player.Position != 5 {
		t.Errorf("Player.Position = %v, want %v", data.Player.Position, 5)
	}

	if len(data.Opponents) != 1 {
		t.Errorf("len(Opponents) = %v, want %v", len(data.Opponents), 1)
	}

	if data.Opponents[0].DriverName != "Test Driver 1" {
		t.Errorf("Opponents[0].DriverName = %v, want %v", data.Opponents[0].DriverName, "Test Driver 1")
	}
}
