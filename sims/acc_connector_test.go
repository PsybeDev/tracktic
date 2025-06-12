package sims

import (
	"context"
	"math"
	"testing"
	"time"
)

// floatEquals checks if two float64 values are approximately equal within a tolerance
func floatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestNewACCConnector(t *testing.T) {
	connector := NewACCConnector()

	if connector == nil {
		t.Fatal("NewACCConnector() returned nil")
	}

	if connector.IsConnected() {
		t.Error("New connector should not be connected")
	}

	if connector.GetSimulatorType() != SimulatorTypeACC {
		t.Errorf("GetSimulatorType() = %v, want %v", connector.GetSimulatorType(), SimulatorTypeACC)
	}
}

func TestACCConnector_GetSimulatorType(t *testing.T) {
	connector := NewACCConnector()

	simType := connector.GetSimulatorType()
	if simType != SimulatorTypeACC {
		t.Errorf("GetSimulatorType() = %v, want %v", simType, SimulatorTypeACC)
	}
}

func TestACCConnector_IsConnected_WhenNotConnected(t *testing.T) {
	connector := NewACCConnector()

	if connector.IsConnected() {
		t.Error("IsConnected() should return false for new connector")
	}
}

func TestACCConnector_GetTelemetryData_WhenNotConnected(t *testing.T) {
	connector := NewACCConnector()
	ctx := context.Background()

	data, err := connector.GetTelemetryData(ctx)

	if err == nil {
		t.Error("GetTelemetryData() should return error when not connected")
	}

	if data != nil {
		t.Error("GetTelemetryData() should return nil data when not connected")
	}
}

func TestACCConnector_HealthCheck_WhenNotConnected(t *testing.T) {
	connector := NewACCConnector()
	ctx := context.Background()

	err := connector.HealthCheck(ctx)

	if err == nil {
		t.Error("HealthCheck() should return error when not connected")
	}
}

func TestACCConnector_Disconnect_WhenNotConnected(t *testing.T) {
	connector := NewACCConnector()

	err := connector.Disconnect()

	if err != nil {
		t.Errorf("Disconnect() should not return error when not connected: %v", err)
	}
}

func TestACCConnector_StopDataStream(t *testing.T) {
	connector := NewACCConnector()

	// Should not panic when stopping stream that wasn't started
	connector.StopDataStream()
}

func TestACCConnector_ConvertSessionType(t *testing.T) {
	connector := NewACCConnector()

	tests := []struct {
		name     string
		accType  int32
		expected SessionType
	}{
		{"Practice", 0, SessionTypePractice},
		{"Qualifying", 1, SessionTypeQualifying},
		{"Race", 2, SessionTypeRace},
		{"Hotlap", 3, SessionTypeHotlap},
		{"Unknown", 99, SessionTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.convertSessionType(tt.accType)
			if result != tt.expected {
				t.Errorf("convertSessionType(%d) = %v, want %v", tt.accType, result, tt.expected)
			}
		})
	}
}

func TestACCConnector_ConvertSessionFlag(t *testing.T) {
	connector := NewACCConnector()

	tests := []struct {
		name     string
		graphics ACCGraphics
		expected SessionFlag
	}{
		{
			name: "Red flag",
			graphics: ACCGraphics{
				GlobalRed: 1,
			},
			expected: SessionFlagRed,
		},
		{
			name: "Yellow flag",
			graphics: ACCGraphics{
				GlobalYellow: 1,
			},
			expected: SessionFlagYellow,
		},
		{
			name: "Green flag",
			graphics: ACCGraphics{
				GlobalGreen: 1,
			},
			expected: SessionFlagGreen,
		},
		{
			name: "Checkered flag",
			graphics: ACCGraphics{
				GlobalChequered: 1,
			},
			expected: SessionFlagCheckered,
		},
		{
			name: "White flag",
			graphics: ACCGraphics{
				GlobalWhite: 1,
			},
			expected: SessionFlagWhite,
		},
		{
			name: "Local blue flag",
			graphics: ACCGraphics{
				Flag: 1,
			},
			expected: SessionFlagBlue,
		},
		{
			name: "No flag",
			graphics: ACCGraphics{
				Flag: 0,
			},
			expected: SessionFlagNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.convertSessionFlag(&tt.graphics)
			if result != tt.expected {
				t.Errorf("convertSessionFlag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestACCConnector_ConvertUTF16ToString(t *testing.T) {
	connector := NewACCConnector()

	tests := []struct {
		name     string
		input    []uint16
		expected string
	}{
		{
			name:     "Simple string",
			input:    []uint16{'T', 'e', 's', 't', 0},
			expected: "Test",
		},
		{
			name:     "Empty string",
			input:    []uint16{0},
			expected: "",
		},
		{
			name:     "String without null terminator",
			input:    []uint16{'A', 'B', 'C'},
			expected: "ABC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.convertUTF16ToString(tt.input)
			if result != tt.expected {
				t.Errorf("convertUTF16ToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestACCConnector_ConvertToTelemetryData(t *testing.T) {
	connector := NewACCConnector()

	// Create sample ACC data
	physics := &ACCPhysics{
		Fuel:           45.5,
		RPM:            7500,
		Gear:           4,
		SpeedKMH:       180.5,
		Gas:            0.85,
		Brake:          0.0,
		Clutch:         0.0,
		SteerAngle:     -15.5,
		TyreWear:       [4]float32{0.15, 0.16, 0.18, 0.17},
		WheelsPressure: [4]float32{28.5, 28.3, 27.8, 27.9},
		TyreTempI:      [4]float32{85.0, 87.0, 82.0, 83.0},
		TyreDirtyLevel: [4]float32{0.1, 0.1, 0.2, 0.2},
		AirTemp:        25.0,
		RoadTemp:       35.0,
	}

	graphics := &ACCGraphics{
		ACSessionType:         2, // Race
		Position:              5,
		CompletedLaps:         9, // Will be +1 for current lap
		NormalizedCarPosition: 0.455,
		ILastTime:             90000, // 90 seconds in milliseconds
		IBestTime:             88000, // 88 seconds in milliseconds
		ICurrentTime:          25000, // 25 seconds in milliseconds
		SessionTimeLeft:       1800,  // 30 minutes in seconds
		FuelXLap:              2.2,
		FuelEstimatedLaps:     20,
		IsInPitLane:           0,
		IsInPit:               0,
		GapAhead:              3000, // 3 seconds in milliseconds
		GapBehind:             2000, // 2 seconds in milliseconds
		GlobalGreen:           1,
		Clock:                 900, // 15 minutes in seconds
	}

	static := &ACCStatic{
		MaxFuel:           60.0,
		IsTimedRace:       0,      // Lapped race
		TrackSPlineLength: 4200.0, // 4.2km in meters
		Track:             [33]uint16{'T', 'e', 's', 't', ' ', 'T', 'r', 'a', 'c', 'k', 0},
		DryTyresName:      [33]uint16{'M', 'e', 'd', 'i', 'u', 'm', 0},
	}

	telemetry := connector.convertToTelemetryData(physics, graphics, static)

	// Verify basic metadata
	if telemetry.SimulatorType != SimulatorTypeACC {
		t.Errorf("SimulatorType = %v, want %v", telemetry.SimulatorType, SimulatorTypeACC)
	}

	if !telemetry.IsConnected {
		t.Error("IsConnected should be true")
	}

	// Verify session data
	if telemetry.Session.Type != SessionTypeRace {
		t.Errorf("Session.Type = %v, want %v", telemetry.Session.Type, SessionTypeRace)
	}

	if telemetry.Session.Flag != SessionFlagGreen {
		t.Errorf("Session.Flag = %v, want %v", telemetry.Session.Flag, SessionFlagGreen)
	}

	if telemetry.Session.TrackName != "Test Track" {
		t.Errorf("Session.TrackName = %v, want %v", telemetry.Session.TrackName, "Test Track")
	}

	if telemetry.Session.TrackLength != 4.2 {
		t.Errorf("Session.TrackLength = %v, want %v", telemetry.Session.TrackLength, 4.2)
	}

	if telemetry.Session.Format != RaceFormatSprint {
		t.Errorf("Session.Format = %v, want %v", telemetry.Session.Format, RaceFormatSprint)
	}

	// Verify player data
	if telemetry.Player.Position != 5 {
		t.Errorf("Player.Position = %v, want %v", telemetry.Player.Position, 5)
	}

	if telemetry.Player.CurrentLap != 10 {
		t.Errorf("Player.CurrentLap = %v, want %v", telemetry.Player.CurrentLap, 10)
	}

	if !floatEquals(telemetry.Player.LapDistancePercent, 45.5, 0.01) {
		t.Errorf("Player.LapDistancePercent = %v, want %v", telemetry.Player.LapDistancePercent, 45.5)
	}

	expectedLastLapTime := 90 * time.Second
	if telemetry.Player.LastLapTime != expectedLastLapTime {
		t.Errorf("Player.LastLapTime = %v, want %v", telemetry.Player.LastLapTime, expectedLastLapTime)
	}

	expectedBestLapTime := 88 * time.Second
	if telemetry.Player.BestLapTime != expectedBestLapTime {
		t.Errorf("Player.BestLapTime = %v, want %v", telemetry.Player.BestLapTime, expectedBestLapTime)
	}

	// Verify fuel data
	if telemetry.Player.Fuel.Level != 45.5 {
		t.Errorf("Player.Fuel.Level = %v, want %v", telemetry.Player.Fuel.Level, 45.5)
	}

	if telemetry.Player.Fuel.Capacity != 60.0 {
		t.Errorf("Player.Fuel.Capacity = %v, want %v", telemetry.Player.Fuel.Capacity, 60.0)
	}
	if !floatEquals(telemetry.Player.Fuel.UsagePerLap, 2.2, 0.001) {
		t.Errorf("Player.Fuel.UsagePerLap = %v, want %v", telemetry.Player.Fuel.UsagePerLap, 2.2)
	}

	// Verify tire data
	if telemetry.Player.Tires.Compound != "Medium" {
		t.Errorf("Player.Tires.Compound = %v, want %v", telemetry.Player.Tires.Compound, "Medium")
	}

	if !floatEquals(telemetry.Player.Tires.FrontLeft.WearPercent, 15.0, 0.001) {
		t.Errorf("Player.Tires.FrontLeft.WearPercent = %v, want %v", telemetry.Player.Tires.FrontLeft.WearPercent, 15.0)
	}

	if telemetry.Player.Tires.FrontLeft.Temperature != 85.0 {
		t.Errorf("Player.Tires.FrontLeft.Temperature = %v, want %v", telemetry.Player.Tires.FrontLeft.Temperature, 85.0)
	}

	if telemetry.Player.Tires.FrontLeft.Pressure != 28.5 {
		t.Errorf("Player.Tires.FrontLeft.Pressure = %v, want %v", telemetry.Player.Tires.FrontLeft.Pressure, 28.5)
	}

	// Verify car state
	if telemetry.Player.Speed != 180.5 {
		t.Errorf("Player.Speed = %v, want %v", telemetry.Player.Speed, 180.5)
	}

	if telemetry.Player.RPM != 7500 {
		t.Errorf("Player.RPM = %v, want %v", telemetry.Player.RPM, 7500)
	}

	if telemetry.Player.Gear != 4 {
		t.Errorf("Player.Gear = %v, want %v", telemetry.Player.Gear, 4)
	}

	if telemetry.Player.Throttle != 85.0 {
		t.Errorf("Player.Throttle = %v, want %v", telemetry.Player.Throttle, 85.0)
	}

	// Verify pit data
	if telemetry.Player.Pit.IsOnPitRoad {
		t.Error("Player.Pit.IsOnPitRoad should be false")
	}

	if telemetry.Player.Pit.IsInPitStall {
		t.Error("Player.Pit.IsInPitStall should be false")
	}
}
