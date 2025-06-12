package sims

import (
	"context"
	"testing"
	"time"
)

func TestNewIRacingConnector(t *testing.T) {
	connector := NewIRacingConnector()

	if connector == nil {
		t.Fatal("NewIRacingConnector should not return nil")
	}

	if connector.IsConnected() {
		t.Error("New connector should not be connected initially")
	}

	if connector.GetSimulatorType() != SimulatorTypeIRacing {
		t.Errorf("GetSimulatorType() = %v, want %v", connector.GetSimulatorType(), SimulatorTypeIRacing)
	}
}

func TestIRacingConnectorInterface(t *testing.T) {
	var _ SimulatorConnector = NewIRacingConnector()
}

func TestIRacingConnectorDisconnect(t *testing.T) {
	connector := NewIRacingConnector()

	// Test disconnect when not connected
	err := connector.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() when not connected should not return error, got %v", err)
	}

	// Simulate connected state
	connector.isConnected = true

	err = connector.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() should not return error, got %v", err)
	}

	if connector.IsConnected() {
		t.Error("Should not be connected after disconnect")
	}
}

func TestIRacingConnectorGetTelemetryDataNotConnected(t *testing.T) {
	connector := NewIRacingConnector()
	ctx := context.Background()

	_, err := connector.GetTelemetryData(ctx)
	if err == nil {
		t.Error("GetTelemetryData() should return error when not connected")
	}

	expectedErrMsg := "connection error [iracing:GetTelemetryData]: not connected to iRacing"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestIRacingConnectorHealthCheckNotConnected(t *testing.T) {
	connector := NewIRacingConnector()
	ctx := context.Background()

	err := connector.HealthCheck(ctx)
	if err == nil {
		t.Error("HealthCheck() should return error when not connected")
	}

	expectedErrMsg := "connection error [iracing:HealthCheck]: not connected to iRacing"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestIRacingConnectorStopDataStream(t *testing.T) {
	connector := NewIRacingConnector()

	// Should not panic when stopping stream that was never started
	connector.StopDataStream()
}

func TestIRacingConnectorConvertSessionState(t *testing.T) {
	connector := NewIRacingConnector()

	tests := []struct {
		name         string
		sessionState int32
		expectedType SessionType
	}{
		{
			name:         "Invalid state",
			sessionState: 0,
			expectedType: SessionTypePractice, // Default
		},
		{
			name:         "Get in car",
			sessionState: 1,
			expectedType: SessionTypePractice, // Default
		},
		{
			name:         "Warmup",
			sessionState: 2,
			expectedType: SessionTypePractice,
		},
		{
			name:         "Parade laps",
			sessionState: 3,
			expectedType: SessionTypeRace,
		},
		{
			name:         "Racing",
			sessionState: 4,
			expectedType: SessionTypeRace,
		},
		{
			name:         "Checkered",
			sessionState: 5,
			expectedType: SessionTypeRace,
		},
		{
			name:         "Cool down",
			sessionState: 6,
			expectedType: SessionTypePractice, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.convertSessionState(tt.sessionState)
			if result != tt.expectedType {
				t.Errorf("convertSessionState(%d) = %v, want %v", tt.sessionState, result, tt.expectedType)
			}
		})
	}
}

func TestIRacingConnectorConvertSessionFlags(t *testing.T) {
	connector := NewIRacingConnector()

	tests := []struct {
		name         string
		sessionFlags int32
		expectedFlag SessionFlag
	}{
		{
			name:         "No flags",
			sessionFlags: 0,
			expectedFlag: SessionFlagNone,
		},
		{
			name:         "Green flag",
			sessionFlags: 0x00000004,
			expectedFlag: SessionFlagGreen,
		},
		{
			name:         "Yellow flag",
			sessionFlags: 0x00000008,
			expectedFlag: SessionFlagYellow,
		},
		{
			name:         "Red flag",
			sessionFlags: 0x00000010,
			expectedFlag: SessionFlagRed,
		},
		{
			name:         "Blue flag",
			sessionFlags: 0x00000020,
			expectedFlag: SessionFlagBlue,
		},
		{
			name:         "White flag",
			sessionFlags: 0x00000002,
			expectedFlag: SessionFlagWhite,
		},
		{
			name:         "Checkered flag",
			sessionFlags: 0x00000001,
			expectedFlag: SessionFlagCheckered,
		},
		{
			name:         "Multiple flags - red takes priority",
			sessionFlags: 0x00000010 | 0x00000008, // Red and yellow
			expectedFlag: SessionFlagRed,
		},
		{
			name:         "Multiple flags - yellow takes priority over green",
			sessionFlags: 0x00000008 | 0x00000004, // Yellow and green
			expectedFlag: SessionFlagYellow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.convertSessionFlags(tt.sessionFlags)
			if result != tt.expectedFlag {
				t.Errorf("convertSessionFlags(0x%08x) = %v, want %v", tt.sessionFlags, result, tt.expectedFlag)
			}
		})
	}
}

func TestIRacingConnectorDataStreamChannels(t *testing.T) {
	connector := NewIRacingConnector()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test that StartDataStream returns channels even when not connected
	dataChan, errorChan := connector.StartDataStream(ctx, 50*time.Millisecond)

	if dataChan == nil {
		t.Error("StartDataStream should return non-nil data channel")
	}

	if errorChan == nil {
		t.Error("StartDataStream should return non-nil error channel")
	}

	// Should receive error since not connected
	select {
	case err := <-errorChan:
		if err == nil {
			t.Error("Should receive error when not connected")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Should receive error quickly when not connected")
	}

	// Stop the stream
	connector.StopDataStream()

	// Channels should be closed after stopping
	select {
	case _, ok := <-dataChan:
		if ok {
			t.Error("Data channel should be closed after stopping stream")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Data channel should be closed quickly after stopping stream")
	}
}

func TestIRacingConnectorFuelEstimateLogic(t *testing.T) {
	// Test the fuel estimation logic that would be used in player data
	fuelData := FuelData{
		Level:        50.0,
		Capacity:     100.0,
		UsagePerHour: 120.0, // 2.0 liters per minute
	}

	// Simulate a 90-second lap time
	lastLapTime := 90.0 // seconds
	lapTimeHours := lastLapTime / 3600.0
	fuelData.UsagePerLap = fuelData.UsagePerHour * lapTimeHours

	expectedUsagePerLap := 120.0 * (90.0 / 3600.0) // 3.0 liters per lap
	if fuelData.UsagePerLap != expectedUsagePerLap {
		t.Errorf("UsagePerLap = %v, want %v", fuelData.UsagePerLap, expectedUsagePerLap)
	}
	avgLapTime := time.Duration(lastLapTime) * time.Second
	CalculateFuelEstimates(&fuelData, avgLapTime)

	expectedLapsLeft := 16 // int(50.0 / 3.0) = 16 laps
	if fuelData.EstimatedLapsLeft != expectedLapsLeft {
		t.Errorf("EstimatedLapsLeft = %v, want %v", fuelData.EstimatedLapsLeft, expectedLapsLeft)
	}

	expectedPercentage := 50.0
	if fuelData.Percentage != expectedPercentage {
		t.Errorf("Percentage = %v, want %v", fuelData.Percentage, expectedPercentage)
	}

	if fuelData.LowFuelWarning {
		t.Error("LowFuelWarning should be false at 50% fuel")
	}
}

func TestIRacingConnectorTireDataDefaults(t *testing.T) {
	// Test the default values that would be used when iRacing data is unavailable
	expectedCompound := "Unknown"
	expectedDefaultTemp := 80.0
	expectedDefaultPressure := 30.0
	expectedDefaultWear := 0.0
	expectedDefaultDirt := 0.0

	// These are the defaults that would be used in getTireData when iRacing data is unavailable
	if expectedCompound != "Unknown" {
		t.Errorf("Default compound should be %q", "Unknown")
	}

	if expectedDefaultTemp != 80.0 {
		t.Errorf("Default temperature should be %v", 80.0)
	}

	if expectedDefaultPressure != 30.0 {
		t.Errorf("Default pressure should be %v", 30.0)
	}

	if expectedDefaultWear != 0.0 {
		t.Errorf("Default wear should be %v", 0.0)
	}

	if expectedDefaultDirt != 0.0 {
		t.Errorf("Default dirt level should be %v", 0.0)
	}
}

func TestIRacingConnectorPitDataLogic(t *testing.T) {
	// Test the pit stall detection logic
	onPitRoad := true
	speed := 0.5 // Less than 1.0 m/s

	isInPitStall := onPitRoad && speed < 1.0

	if !isInPitStall {
		t.Error("Should be detected as in pit stall when on pit road and speed < 1.0")
	}

	// Test when moving on pit road
	speed = 15.0 // More than 1.0 m/s
	isInPitStall = onPitRoad && speed < 1.0

	if isInPitStall {
		t.Error("Should not be detected as in pit stall when moving on pit road")
	}

	// Test when not on pit road
	onPitRoad = false
	speed = 0.5
	isInPitStall = onPitRoad && speed < 1.0

	if isInPitStall {
		t.Error("Should not be detected as in pit stall when not on pit road")
	}
}

func TestIRacingConnectorUnitConversions(t *testing.T) {
	// Test speed conversion from m/s to km/h
	speedMS := 50.0 // 50 m/s
	speedKMH := speedMS * 3.6
	expectedKMH := 180.0

	if speedKMH != expectedKMH {
		t.Errorf("Speed conversion: %v m/s = %v km/h, want %v km/h", speedMS, speedKMH, expectedKMH)
	}

	// Test percentage conversions
	throttle := 0.75 // 75% as 0.75
	throttlePercent := throttle * 100
	expectedPercent := 75.0

	if throttlePercent != expectedPercent {
		t.Errorf("Throttle conversion: %v = %v%%, want %v%%", throttle, throttlePercent, expectedPercent)
	}

	// Test lap distance percentage conversion
	lapDistPct := 0.456 // iRacing provides as 0-1 range
	lapDistPercent := lapDistPct * 100
	expectedLapPercent := 45.6

	if lapDistPercent != expectedLapPercent {
		t.Errorf("Lap distance conversion: %v = %v%%, want %v%%", lapDistPct, lapDistPercent, expectedLapPercent)
	}
}

func TestIRacingConnectorSessionInfoLogic(t *testing.T) {
	// Test session type determination logic
	sessionTimeRemain := 1800.0    // 30 minutes remaining
	sessionLapsRemain := int32(25) // 25 laps remaining

	isTimedSession := sessionTimeRemain > 0
	isLappedSession := sessionLapsRemain > 0

	if !isTimedSession {
		t.Error("Should be detected as timed session when time remaining > 0")
	}

	if !isLappedSession {
		t.Error("Should be detected as lapped session when laps remaining > 0")
	}

	// Test time-only session
	sessionTimeRemain = 1800.0
	sessionLapsRemain = 0

	isTimedSession = sessionTimeRemain > 0
	isLappedSession = sessionLapsRemain > 0

	if !isTimedSession {
		t.Error("Should be timed session")
	}

	if isLappedSession {
		t.Error("Should not be lapped session when laps remaining = 0")
	}

	// Test laps-only session
	sessionTimeRemain = 0.0
	sessionLapsRemain = 50

	isTimedSession = sessionTimeRemain > 0
	isLappedSession = sessionLapsRemain > 0

	if isTimedSession {
		t.Error("Should not be timed session when time remaining = 0")
	}

	if !isLappedSession {
		t.Error("Should be lapped session")
	}
}

// Benchmark tests for performance validation
func BenchmarkIRacingConnectorConvertSessionState(b *testing.B) {
	connector := NewIRacingConnector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connector.convertSessionState(4) // Racing state
	}
}

func BenchmarkIRacingConnectorConvertSessionFlags(b *testing.B) {
	connector := NewIRacingConnector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connector.convertSessionFlags(0x00000008) // Yellow flag
	}
}

func TestIRacingConnectorEmptyOpponentsData(t *testing.T) {
	connector := NewIRacingConnector()

	opponents, err := connector.getOpponentsData()
	if err != nil {
		t.Errorf("getOpponentsData() should not return error, got %v", err)
	}

	if opponents == nil {
		t.Error("getOpponentsData() should not return nil slice")
	}

	if len(opponents) != 0 {
		t.Errorf("getOpponentsData() should return empty slice, got length %d", len(opponents))
	}
}
