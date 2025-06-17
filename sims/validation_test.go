package sims

import (
	"math"
	"testing"
	"time"
)

func TestDefaultValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()

	if config == nil {
		t.Fatal("DefaultValidationConfig returned nil")
	}

	// Test some key values
	if config.MaxSpeed != 500.0 {
		t.Errorf("Expected MaxSpeed 500.0, got %f", config.MaxSpeed)
	}

	if config.MaxRPM != 15000.0 {
		t.Errorf("Expected MaxRPM 15000.0, got %f", config.MaxRPM)
	}

	if config.MaxLapTime != 15*time.Minute {
		t.Errorf("Expected MaxLapTime 15m, got %v", config.MaxLapTime)
	}
}

func TestNewDataValidator(t *testing.T) {
	// Test with nil config (should use defaults)
	validator := NewDataValidator(nil)
	if validator == nil {
		t.Fatal("NewDataValidator returned nil")
	}
	if validator.config == nil {
		t.Fatal("DataValidator config is nil")
	}

	// Test with custom config
	customConfig := &ValidationConfig{MaxSpeed: 300.0}
	validator2 := NewDataValidator(customConfig)
	if validator2.config.MaxSpeed != 300.0 {
		t.Errorf("Expected custom MaxSpeed 300.0, got %f", validator2.config.MaxSpeed)
	}
}

func TestValidateSpeed(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name     string
		speed    float64
		hasError bool
	}{
		{"Valid speed", 120.5, false},
		{"Zero speed", 0.0, false},
		{"Max speed", 500.0, false},
		{"Negative speed", -10.0, true},
		{"Excessive speed", 600.0, true},
		{"NaN speed", math.NaN(), true},
		{"Infinite speed", math.Inf(1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSpeed(tt.speed)
			if tt.hasError && err == nil {
				t.Errorf("Expected error for speed %f, got nil", tt.speed)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for speed %f, got %v", tt.speed, err)
			}
		})
	}
}

func TestValidateRPM(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name     string
		rpm      float64
		hasError bool
	}{
		{"Valid RPM", 6500.0, false},
		{"Zero RPM", 0.0, false},
		{"Max RPM", 15000.0, false},
		{"Negative RPM", -100.0, true},
		{"Excessive RPM", 20000.0, true},
		{"NaN RPM", math.NaN(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRPM(tt.rpm)
			if tt.hasError && err == nil {
				t.Errorf("Expected error for RPM %f, got nil", tt.rpm)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for RPM %f, got %v", tt.rpm, err)
			}
		})
	}
}

func TestValidateTemperature(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name       string
		temp       float64
		isTireTemp bool
		hasError   bool
	}{
		{"Valid tire temp", 85.0, true, false},
		{"Max tire temp", 150.0, true, false},
		{"Min tire temp", 0.0, true, false},
		{"Excessive tire temp", 200.0, true, true},
		{"Valid air temp", 25.0, false, false},
		{"Cold air temp", -15.0, false, false},
		{"Hot air temp", 50.0, false, false},
		{"Excessive cold air", -30.0, false, true},
		{"Excessive hot air", 70.0, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.isTireTemp {
				err = validator.ValidateTireTemperature(tt.temp)
			} else {
				err = validator.ValidateAirTemperature(tt.temp)
			}

			if tt.hasError && err == nil {
				t.Errorf("Expected error for temp %f, got nil", tt.temp)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for temp %f, got %v", tt.temp, err)
			}
		})
	}
}

func TestValidatePressure(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name     string
		pressure float64
		hasError bool
	}{
		{"Valid pressure", 28.5, false},
		{"Min pressure", 10.0, false},
		{"Max pressure", 50.0, false},
		{"Low pressure", 5.0, true},
		{"High pressure", 60.0, true},
		{"Zero pressure", 0.0, true},
		{"Negative pressure", -5.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateTirePressure(tt.pressure)
			if tt.hasError && err == nil {
				t.Errorf("Expected error for pressure %f, got nil", tt.pressure)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for pressure %f, got %v", tt.pressure, err)
			}
		})
	}
}

func TestValidateLapTime(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name     string
		lapTime  time.Duration
		hasError bool
	}{
		{"Valid lap time", 90 * time.Second, false},
		{"Min lap time", 30 * time.Second, false},
		{"Long lap time", 5 * time.Minute, false},
		{"Max lap time", 15 * time.Minute, false},
		{"Too short", 20 * time.Second, true},
		{"Too long", 20 * time.Minute, true},
		{"Zero lap time", 0, true},
		{"Negative lap time", -10 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateLapTime(tt.lapTime)
			if tt.hasError && err == nil {
				t.Errorf("Expected error for lap time %v, got nil", tt.lapTime)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for lap time %v, got %v", tt.lapTime, err)
			}
		})
	}
}

func TestValidatePosition(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name     string
		position int
		hasError bool
	}{
		{"Valid position", 5, false},
		{"First position", 1, false},
		{"Last position", 100, false},
		{"Zero position", 0, true},
		{"Negative position", -1, true},
		{"Excessive position", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePosition(tt.position)
			if tt.hasError && err == nil {
				t.Errorf("Expected error for position %d, got nil", tt.position)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for position %d, got %v", tt.position, err)
			}
		})
	}
}

func TestValidatePercentage(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name       string
		percentage float64
		hasError   bool
	}{
		{"Valid percentage", 75.5, false},
		{"Zero percentage", 0.0, false},
		{"Full percentage", 100.0, false},
		{"Negative percentage", -5.0, true},
		{"Excessive percentage", 105.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePercentage(tt.percentage)
			if tt.hasError && err == nil {
				t.Errorf("Expected error for percentage %f, got nil", tt.percentage)
			}
			if !tt.hasError && err != nil {
				t.Errorf("Expected no error for percentage %f, got %v", tt.percentage, err)
			}
		})
	}
}

func TestValidateSimulatorType(t *testing.T) {
	validator := NewDataValidator(nil)

	validTypes := []SimulatorType{
		SimulatorTypeIRacing,
		SimulatorTypeACC,
		SimulatorTypeLMU,
	}

	for _, simType := range validTypes {
		t.Run(string(simType), func(t *testing.T) {
			if !validator.isValidSimulatorType(simType) {
				t.Errorf("Expected %s to be valid", simType)
			}
		})
	}

	invalidTypes := []SimulatorType{
		"invalid",
		"",
		"random_sim",
	}

	for _, simType := range invalidTypes {
		t.Run(string(simType), func(t *testing.T) {
			if validator.isValidSimulatorType(simType) {
				t.Errorf("Expected %s to be invalid", simType)
			}
		})
	}
}

func TestValidateSessionType(t *testing.T) {
	validator := NewDataValidator(nil)

	validTypes := []SessionType{
		SessionTypePractice,
		SessionTypeQualifying,
		SessionTypeRace,
		SessionTypeHotlap,
		SessionTypeUnknown,
	}

	for _, sessionType := range validTypes {
		t.Run(string(sessionType), func(t *testing.T) {
			if !validator.isValidSessionType(sessionType) {
				t.Errorf("Expected %s to be valid", sessionType)
			}
		})
	}

	invalidTypes := []SessionType{
		"invalid",
		"",
		"warmup",
	}

	for _, sessionType := range invalidTypes {
		t.Run(string(sessionType), func(t *testing.T) {
			if validator.isValidSessionType(sessionType) {
				t.Errorf("Expected %s to be invalid", sessionType)
			}
		})
	}
}

func TestSanitizeSpeed(t *testing.T) {
	validator := NewDataValidator(nil)

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Valid speed", 120.5, 120.5},
		{"Negative speed", -10.0, 0.0},
		{"Excessive speed", 600.0, 500.0},
		{"NaN speed", math.NaN(), 0.0},
		{"Infinite speed", math.Inf(1), 500.0},
		{"Negative infinite", math.Inf(-1), 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.SanitizeSpeed(tt.input)
			if math.IsNaN(tt.expected) {
				if !math.IsNaN(result) {
					t.Errorf("Expected NaN, got %f", result)
				}
			} else if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestValidateTelemetryData(t *testing.T) {
	validator := NewDataValidator(nil)

	// Test nil data
	errors := validator.ValidateTelemetryData(nil)
	if len(errors) == 0 {
		t.Error("Expected error for nil telemetry data")
	}

	// Test valid data
	validData := &TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: SimulatorTypeIRacing,
		IsConnected:   true,
		Session: SessionInfo{
			Type:             SessionTypeRace,
			Format:           RaceFormatSprint,
			Flag:             SessionFlagGreen,
			TimeRemaining:    30 * time.Minute,
			LapsRemaining:    10,
			TotalLaps:        20,
			SessionTime:      15 * time.Minute,
			TrackName:        "Silverstone",
			TrackLength:      5.891,
			AirTemperature:   25.0,
			TrackTemperature: 35.0,
		},
		Player: PlayerData{
			Position:           5,
			CurrentLap:         8,
			LapDistancePercent: 45.5,
			LastLapTime:        90 * time.Second,
			BestLapTime:        88 * time.Second,
			CurrentLapTime:     45 * time.Second, Fuel: FuelData{
				Level:             45.5,
				Capacity:          65.0,
				Percentage:        70.0,
				EstimatedLapsLeft: 12,
				UsagePerLap:       2.1,
				UsagePerHour:      125.0,
			},
			Tires: TireData{
				FrontLeft: TireWheelData{
					Temperature: 85.0,
					Pressure:    28.5,
					WearPercent: 15.0,
				},
				FrontRight: TireWheelData{
					Temperature: 87.0,
					Pressure:    28.3,
					WearPercent: 16.0,
				},
				RearLeft: TireWheelData{
					Temperature: 82.0,
					Pressure:    27.8,
					WearPercent: 12.0,
				},
				RearRight: TireWheelData{
					Temperature: 84.0,
					Pressure:    27.9,
					WearPercent: 13.0,
				},
			},
		}, Opponents: []OpponentData{
			{
				CarIndex:           42,
				CarNumber:          "42",
				Position:           4,
				CurrentLap:         8,
				LapDistancePercent: 55.0,
				LastLapTime:        89 * time.Second,
				BestLapTime:        87 * time.Second,
				GapToPlayer:        -5 * time.Second,
				DriverName:         "Test Driver",
			},
		},
	}

	errors = validator.ValidateTelemetryData(validData)
	if len(errors) > 0 {
		t.Errorf("Expected no errors for valid data, got %d errors: %v", len(errors), errors)
	}

	// Test invalid data
	invalidData := &TelemetryData{
		Timestamp:     time.Time{}, // Zero timestamp
		SimulatorType: "invalid",   // Invalid simulator type
		Session: SessionInfo{
			Type:        "invalid", // Invalid session type
			TrackLength: -1.0,      // Invalid track length
		}, Player: PlayerData{
			Position:    0,  // Invalid position
			CurrentLap:  -1, // Invalid lap
			LastLapTime: 0,  // Invalid lap time
			Fuel: FuelData{
				Level:    -10.0, // Invalid fuel
				Capacity: 0.0,   // Invalid capacity
			},
		},
	}

	errors = validator.ValidateTelemetryData(invalidData)
	if len(errors) == 0 {
		t.Error("Expected errors for invalid data, got none")
	}

	// Check that we have multiple validation errors
	if len(errors) < 5 {
		t.Errorf("Expected at least 5 validation errors, got %d", len(errors))
	}
}

func TestSanitizeTelemetryData(t *testing.T) {
	validator := NewDataValidator(nil)

	// Create data with various invalid values
	data := &TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: SimulatorTypeIRacing,
		Session: SessionInfo{
			TrackLength:      -1.0,  // Invalid, should be clamped
			AirTemperature:   100.0, // Too hot, should be clamped
			TrackTemperature: -50.0, // Too cold, should be clamped
		}, Player: PlayerData{
			Position:           -5,    // Invalid, should be clamped to 1
			LapDistancePercent: 150.0, // Too high, should be clamped to 100
			Fuel: FuelData{
				Level:      -10.0, // Negative, should be clamped to 0
				Percentage: 120.0, // Too high, should be clamped to 100
			},
			Tires: TireData{
				FrontLeft: TireWheelData{
					Temperature: 200.0, // Too hot, should be clamped
					Pressure:    5.0,   // Too low, should be clamped
					WearPercent: -10.0, // Negative, should be clamped to 0
				},
			},
		},
	}

	sanitizedData := validator.SanitizeTelemetryData(data)

	// Verify sanitization
	if sanitizedData.Session.TrackLength < 0.5 {
		t.Errorf("Track length not properly sanitized: %f", sanitizedData.Session.TrackLength)
	}

	if sanitizedData.Session.AirTemperature > 60.0 {
		t.Errorf("Air temperature not properly sanitized: %f", sanitizedData.Session.AirTemperature)
	}

	if sanitizedData.Player.Position < 1 {
		t.Errorf("Position not properly sanitized: %d", sanitizedData.Player.Position)
	}

	if sanitizedData.Player.LapDistancePercent > 100.0 {
		t.Errorf("Lap distance percent not properly sanitized: %f", sanitizedData.Player.LapDistancePercent)
	}
	if sanitizedData.Player.Fuel.Level < 0.0 {
		t.Errorf("Fuel level not properly sanitized: %f", sanitizedData.Player.Fuel.Level)
	}

	if sanitizedData.Player.Tires.FrontLeft.Temperature > 150.0 {
		t.Errorf("Tire temperature not properly sanitized: %f", sanitizedData.Player.Tires.FrontLeft.Temperature)
	}

	if sanitizedData.Player.Tires.FrontLeft.Pressure < 10.0 {
		t.Errorf("Tire pressure not properly sanitized: %f", sanitizedData.Player.Tires.FrontLeft.Pressure)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "Speed",
		Value:   -10.0,
		Message: "speed cannot be negative",
	}

	expectedMsg := "validation error for field 'Speed': speed cannot be negative (value: -10)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}
