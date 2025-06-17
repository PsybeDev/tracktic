package sims

import (
	"fmt"
	"math"
	"time"
)

// ValidationError represents a data validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// DataValidator provides validation functions for telemetry data
type DataValidator struct {
	config *ValidationConfig
}

// ValidationConfig contains validation parameters
type ValidationConfig struct {
	// Speed limits (km/h)
	MaxSpeed float64
	MinSpeed float64

	// RPM limits
	MaxRPM float64
	MinRPM float64

	// Fuel limits (liters)
	MaxFuel float64
	MinFuel float64

	// Temperature limits (Celsius)
	MaxTireTemp float64
	MinTireTemp float64
	MaxAirTemp  float64
	MinAirTemp  float64

	// Pressure limits (PSI)
	MaxTirePressure float64
	MinTirePressure float64

	// Lap time limits
	MaxLapTime time.Duration
	MinLapTime time.Duration

	// Position limits
	MaxPosition int
	MinPosition int

	// Percentage limits (0-100)
	MaxPercentage float64
	MinPercentage float64

	// Track limits
	MaxTrackLength float64 // km
	MinTrackLength float64 // km
}

// DefaultValidationConfig returns sensible default validation parameters
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxSpeed:        500.0, // 500 km/h max speed
		MinSpeed:        0.0,
		MaxRPM:          15000.0, // 15k RPM max
		MinRPM:          0.0,
		MaxFuel:         200.0, // 200L max fuel capacity
		MinFuel:         0.0,
		MaxTireTemp:     150.0, // 150°C max tire temp
		MinTireTemp:     0.0,
		MaxAirTemp:      60.0,             // 60°C max air temp
		MinAirTemp:      -20.0,            // -20°C min air temp
		MaxTirePressure: 50.0,             // 50 PSI max
		MinTirePressure: 10.0,             // 10 PSI min
		MaxLapTime:      15 * time.Minute, // 15 minutes max lap
		MinLapTime:      30 * time.Second, // 30 seconds min lap
		MaxPosition:     100,              // 100 cars max
		MinPosition:     1,
		MaxPercentage:   100.0,
		MinPercentage:   0.0,
		MaxTrackLength:  25.0, // 25km max track length
		MinTrackLength:  0.5,  // 0.5km min track length
	}
}

// NewDataValidator creates a new data validator with the given config
func NewDataValidator(config *ValidationConfig) *DataValidator {
	if config == nil {
		config = DefaultValidationConfig()
	}
	return &DataValidator{config: config}
}

// ValidateTelemetryData validates complete telemetry data structure
func (v *DataValidator) ValidateTelemetryData(data *TelemetryData) []error {
	var errors []error

	if data == nil {
		return []error{&ValidationError{"TelemetryData", nil, "data is nil"}}
	}

	// Validate timestamp
	if data.Timestamp.IsZero() {
		errors = append(errors, &ValidationError{"Timestamp", data.Timestamp, "timestamp is zero"})
	}

	// Validate simulator type
	if !v.isValidSimulatorType(data.SimulatorType) {
		errors = append(errors, &ValidationError{"SimulatorType", data.SimulatorType, "invalid simulator type"})
	}

	// Validate session data
	sessionErrors := v.ValidateSessionInfo(&data.Session)
	errors = append(errors, sessionErrors...)

	// Validate player data
	playerErrors := v.ValidatePlayerData(&data.Player)
	errors = append(errors, playerErrors...)

	// Validate opponents data
	for i, opponent := range data.Opponents {
		opponentErrors := v.ValidateOpponentData(&opponent)
		for _, err := range opponentErrors {
			if valErr, ok := err.(*ValidationError); ok {
				valErr.Field = fmt.Sprintf("Opponents[%d].%s", i, valErr.Field)
			}
			errors = append(errors, err)
		}
	}

	return errors
}

// ValidateSessionInfo validates session information
func (v *DataValidator) ValidateSessionInfo(session *SessionInfo) []error {
	var errors []error

	if session == nil {
		return []error{&ValidationError{"SessionInfo", nil, "session info is nil"}}
	}

	// Validate session type
	if !v.isValidSessionType(session.Type) {
		errors = append(errors, &ValidationError{"Type", session.Type, "invalid session type"})
	}

	// Validate session flag
	if !v.isValidSessionFlag(session.Flag) {
		errors = append(errors, &ValidationError{"Flag", session.Flag, "invalid session flag"})
	}

	// Validate race format
	if !v.isValidRaceFormat(session.Format) {
		errors = append(errors, &ValidationError{"Format", session.Format, "invalid race format"})
	}

	// Validate time remaining (should be non-negative for timed sessions)
	if session.IsTimedSession && session.TimeRemaining < 0 {
		errors = append(errors, &ValidationError{"TimeRemaining", session.TimeRemaining, "negative time remaining in timed session"})
	}

	// Validate laps remaining (should be non-negative for lapped sessions)
	if session.IsLappedSession && session.LapsRemaining < 0 {
		errors = append(errors, &ValidationError{"LapsRemaining", session.LapsRemaining, "negative laps remaining in lapped session"})
	}

	// Validate track length
	if session.TrackLength < v.config.MinTrackLength || session.TrackLength > v.config.MaxTrackLength {
		errors = append(errors, &ValidationError{"TrackLength", session.TrackLength,
			fmt.Sprintf("track length outside valid range [%.1f, %.1f] km", v.config.MinTrackLength, v.config.MaxTrackLength)})
	}

	// Validate temperatures
	if session.AirTemperature < v.config.MinAirTemp || session.AirTemperature > v.config.MaxAirTemp {
		errors = append(errors, &ValidationError{"AirTemperature", session.AirTemperature,
			fmt.Sprintf("air temperature outside valid range [%.1f, %.1f]°C", v.config.MinAirTemp, v.config.MaxAirTemp)})
	}

	if session.TrackTemperature < v.config.MinAirTemp || session.TrackTemperature > v.config.MaxAirTemp {
		errors = append(errors, &ValidationError{"TrackTemperature", session.TrackTemperature,
			fmt.Sprintf("track temperature outside valid range [%.1f, %.1f]°C", v.config.MinAirTemp, v.config.MaxAirTemp)})
	}

	return errors
}

// ValidatePlayerData validates player telemetry data
func (v *DataValidator) ValidatePlayerData(player *PlayerData) []error {
	var errors []error

	if player == nil {
		return []error{&ValidationError{"PlayerData", nil, "player data is nil"}}
	}

	// Validate position
	if player.Position < v.config.MinPosition || player.Position > v.config.MaxPosition {
		errors = append(errors, &ValidationError{"Position", player.Position,
			fmt.Sprintf("position outside valid range [%d, %d]", v.config.MinPosition, v.config.MaxPosition)})
	}

	// Validate current lap (should be positive)
	if player.CurrentLap < 0 {
		errors = append(errors, &ValidationError{"CurrentLap", player.CurrentLap, "current lap cannot be negative"})
	}

	// Validate lap distance percentage
	if player.LapDistancePercent < v.config.MinPercentage || player.LapDistancePercent > v.config.MaxPercentage {
		errors = append(errors, &ValidationError{"LapDistancePercent", player.LapDistancePercent,
			fmt.Sprintf("lap distance percentage outside valid range [%.1f, %.1f]", v.config.MinPercentage, v.config.MaxPercentage)})
	}

	// Validate lap times (should be positive and within reasonable bounds)
	if player.LastLapTime > 0 && (player.LastLapTime < v.config.MinLapTime || player.LastLapTime > v.config.MaxLapTime) {
		errors = append(errors, &ValidationError{"LastLapTime", player.LastLapTime,
			fmt.Sprintf("last lap time outside valid range [%v, %v]", v.config.MinLapTime, v.config.MaxLapTime)})
	}

	if player.BestLapTime > 0 && (player.BestLapTime < v.config.MinLapTime || player.BestLapTime > v.config.MaxLapTime) {
		errors = append(errors, &ValidationError{"BestLapTime", player.BestLapTime,
			fmt.Sprintf("best lap time outside valid range [%v, %v]", v.config.MinLapTime, v.config.MaxLapTime)})
	}

	// Validate speed
	if player.Speed < v.config.MinSpeed || player.Speed > v.config.MaxSpeed {
		errors = append(errors, &ValidationError{"Speed", player.Speed,
			fmt.Sprintf("speed outside valid range [%.1f, %.1f] km/h", v.config.MinSpeed, v.config.MaxSpeed)})
	}

	// Validate RPM
	if player.RPM < v.config.MinRPM || player.RPM > v.config.MaxRPM {
		errors = append(errors, &ValidationError{"RPM", player.RPM,
			fmt.Sprintf("RPM outside valid range [%.0f, %.0f]", v.config.MinRPM, v.config.MaxRPM)})
	}

	// Validate gear (should be -1 to 8 for reverse and gears 1-8)
	if player.Gear < -1 || player.Gear > 8 {
		errors = append(errors, &ValidationError{"Gear", player.Gear, "gear outside valid range [-1, 8]"})
	}

	// Validate input percentages
	percentageFields := map[string]float64{
		"Throttle": player.Throttle,
		"Brake":    player.Brake,
		"Clutch":   player.Clutch,
	}

	for field, value := range percentageFields {
		if value < v.config.MinPercentage || value > v.config.MaxPercentage {
			errors = append(errors, &ValidationError{field, value,
				fmt.Sprintf("%s percentage outside valid range [%.1f, %.1f]", field, v.config.MinPercentage, v.config.MaxPercentage)})
		}
	}

	// Validate steering angle (typically -90 to +90 degrees)
	if player.Steering < -180.0 || player.Steering > 180.0 {
		errors = append(errors, &ValidationError{"Steering", player.Steering, "steering angle outside valid range [-180, 180] degrees"})
	}

	// Validate fuel data
	fuelErrors := v.ValidateFuelData(&player.Fuel)
	for _, err := range fuelErrors {
		if valErr, ok := err.(*ValidationError); ok {
			valErr.Field = "Fuel." + valErr.Field
		}
		errors = append(errors, err)
	}

	// Validate tire data
	tireErrors := v.ValidateTireData(&player.Tires)
	for _, err := range tireErrors {
		if valErr, ok := err.(*ValidationError); ok {
			valErr.Field = "Tires." + valErr.Field
		}
		errors = append(errors, err)
	}

	return errors
}

// ValidateFuelData validates fuel information
func (v *DataValidator) ValidateFuelData(fuel *FuelData) []error {
	var errors []error

	if fuel == nil {
		return []error{&ValidationError{"FuelData", nil, "fuel data is nil"}}
	}

	// Validate fuel level
	if fuel.Level < v.config.MinFuel || fuel.Level > v.config.MaxFuel {
		errors = append(errors, &ValidationError{"Level", fuel.Level,
			fmt.Sprintf("fuel level outside valid range [%.1f, %.1f] L", v.config.MinFuel, v.config.MaxFuel)})
	}

	// Validate fuel capacity
	if fuel.Capacity < v.config.MinFuel || fuel.Capacity > v.config.MaxFuel {
		errors = append(errors, &ValidationError{"Capacity", fuel.Capacity,
			fmt.Sprintf("fuel capacity outside valid range [%.1f, %.1f] L", v.config.MinFuel, v.config.MaxFuel)})
	}

	// Validate fuel level doesn't exceed capacity
	if fuel.Level > fuel.Capacity {
		errors = append(errors, &ValidationError{"Level", fuel.Level, "fuel level exceeds capacity"})
	}

	// Validate fuel percentage
	if fuel.Percentage < v.config.MinPercentage || fuel.Percentage > v.config.MaxPercentage {
		errors = append(errors, &ValidationError{"Percentage", fuel.Percentage,
			fmt.Sprintf("fuel percentage outside valid range [%.1f, %.1f]", v.config.MinPercentage, v.config.MaxPercentage)})
	}

	// Validate usage per lap (should be positive if set)
	if fuel.UsagePerLap < 0 {
		errors = append(errors, &ValidationError{"UsagePerLap", fuel.UsagePerLap, "fuel usage per lap cannot be negative"})
	}

	// Validate estimated laps left (should be non-negative)
	if fuel.EstimatedLapsLeft < 0 {
		errors = append(errors, &ValidationError{"EstimatedLapsLeft", fuel.EstimatedLapsLeft, "estimated laps left cannot be negative"})
	}

	return errors
}

// ValidateTireData validates tire information
func (v *DataValidator) ValidateTireData(tires *TireData) []error {
	var errors []error

	if tires == nil {
		return []error{&ValidationError{"TireData", nil, "tire data is nil"}}
	}

	// Validate tire wear level
	if !v.isValidTireWearLevel(tires.WearLevel) {
		errors = append(errors, &ValidationError{"WearLevel", tires.WearLevel, "invalid tire wear level"})
	}

	// Validate tire temp level
	if !v.isValidTireTempLevel(tires.TempLevel) {
		errors = append(errors, &ValidationError{"TempLevel", tires.TempLevel, "invalid tire temperature level"})
	}

	// Validate individual tire data
	tireWheels := map[string]TireWheelData{
		"FrontLeft":  tires.FrontLeft,
		"FrontRight": tires.FrontRight,
		"RearLeft":   tires.RearLeft,
		"RearRight":  tires.RearRight,
	}

	for position, tire := range tireWheels {
		wheelErrors := v.ValidateTireWheelData(&tire)
		for _, err := range wheelErrors {
			if valErr, ok := err.(*ValidationError); ok {
				valErr.Field = position + "." + valErr.Field
			}
			errors = append(errors, err)
		}
	}

	return errors
}

// ValidateTireWheelData validates individual tire wheel data
func (v *DataValidator) ValidateTireWheelData(tire *TireWheelData) []error {
	var errors []error

	if tire == nil {
		return []error{&ValidationError{"TireWheelData", nil, "tire wheel data is nil"}}
	}

	// Validate temperature
	if tire.Temperature < v.config.MinTireTemp || tire.Temperature > v.config.MaxTireTemp {
		errors = append(errors, &ValidationError{"Temperature", tire.Temperature,
			fmt.Sprintf("tire temperature outside valid range [%.1f, %.1f]°C", v.config.MinTireTemp, v.config.MaxTireTemp)})
	}

	// Validate pressure
	if tire.Pressure < v.config.MinTirePressure || tire.Pressure > v.config.MaxTirePressure {
		errors = append(errors, &ValidationError{"Pressure", tire.Pressure,
			fmt.Sprintf("tire pressure outside valid range [%.1f, %.1f] PSI", v.config.MinTirePressure, v.config.MaxTirePressure)})
	}

	// Validate wear percentage
	if tire.WearPercent < v.config.MinPercentage || tire.WearPercent > v.config.MaxPercentage {
		errors = append(errors, &ValidationError{"WearPercent", tire.WearPercent,
			fmt.Sprintf("tire wear percentage outside valid range [%.1f, %.1f]", v.config.MinPercentage, v.config.MaxPercentage)})
	}

	// Validate dirt level percentage
	if tire.DirtLevel < v.config.MinPercentage || tire.DirtLevel > v.config.MaxPercentage {
		errors = append(errors, &ValidationError{"DirtLevel", tire.DirtLevel,
			fmt.Sprintf("tire dirt level outside valid range [%.1f, %.1f]", v.config.MinPercentage, v.config.MaxPercentage)})
	}

	return errors
}

// ValidateOpponentData validates opponent driver data
func (v *DataValidator) ValidateOpponentData(opponent *OpponentData) []error {
	var errors []error

	if opponent == nil {
		return []error{&ValidationError{"OpponentData", nil, "opponent data is nil"}}
	}

	// Validate position
	if opponent.Position < v.config.MinPosition || opponent.Position > v.config.MaxPosition {
		errors = append(errors, &ValidationError{"Position", opponent.Position,
			fmt.Sprintf("position outside valid range [%d, %d]", v.config.MinPosition, v.config.MaxPosition)})
	}

	// Validate current lap (should be non-negative)
	if opponent.CurrentLap < 0 {
		errors = append(errors, &ValidationError{"CurrentLap", opponent.CurrentLap, "current lap cannot be negative"})
	}

	// Validate lap distance percentage
	if opponent.LapDistancePercent < v.config.MinPercentage || opponent.LapDistancePercent > v.config.MaxPercentage {
		errors = append(errors, &ValidationError{"LapDistancePercent", opponent.LapDistancePercent,
			fmt.Sprintf("lap distance percentage outside valid range [%.1f, %.1f]", v.config.MinPercentage, v.config.MaxPercentage)})
	}

	// Validate lap times (should be positive and within reasonable bounds)
	if opponent.LastLapTime > 0 && (opponent.LastLapTime < v.config.MinLapTime || opponent.LastLapTime > v.config.MaxLapTime) {
		errors = append(errors, &ValidationError{"LastLapTime", opponent.LastLapTime,
			fmt.Sprintf("last lap time outside valid range [%v, %v]", v.config.MinLapTime, v.config.MaxLapTime)})
	}

	if opponent.BestLapTime > 0 && (opponent.BestLapTime < v.config.MinLapTime || opponent.BestLapTime > v.config.MaxLapTime) {
		errors = append(errors, &ValidationError{"BestLapTime", opponent.BestLapTime,
			fmt.Sprintf("best lap time outside valid range [%v, %v]", v.config.MinLapTime, v.config.MaxLapTime)})
	}

	return errors
}

// SanitizeTelemetryData cleans and corrects telemetry data where possible
func (v *DataValidator) SanitizeTelemetryData(data *TelemetryData) *TelemetryData {
	if data == nil {
		return nil
	}

	// Make a copy to avoid modifying the original
	sanitized := *data

	// Set timestamp if zero
	if sanitized.Timestamp.IsZero() {
		sanitized.Timestamp = time.Now()
	}

	// Sanitize player data
	sanitized.Player = v.sanitizePlayerData(&sanitized.Player)

	// Sanitize session data
	sanitized.Session = v.sanitizeSessionInfo(&sanitized.Session)

	// Sanitize opponents data
	for i := range sanitized.Opponents {
		sanitized.Opponents[i] = v.sanitizeOpponentData(&sanitized.Opponents[i])
	}

	return &sanitized
}

// Helper methods for data sanitization
func (v *DataValidator) sanitizePlayerData(player *PlayerData) PlayerData {
	sanitized := *player

	// Clamp speed to valid range
	sanitized.Speed = v.clamp(sanitized.Speed, v.config.MinSpeed, v.config.MaxSpeed)

	// Clamp RPM to valid range
	sanitized.RPM = v.clamp(sanitized.RPM, v.config.MinRPM, v.config.MaxRPM)

	// Clamp percentages
	sanitized.Throttle = v.clamp(sanitized.Throttle, v.config.MinPercentage, v.config.MaxPercentage)
	sanitized.Brake = v.clamp(sanitized.Brake, v.config.MinPercentage, v.config.MaxPercentage)
	sanitized.Clutch = v.clamp(sanitized.Clutch, v.config.MinPercentage, v.config.MaxPercentage)
	sanitized.LapDistancePercent = v.clamp(sanitized.LapDistancePercent, v.config.MinPercentage, v.config.MaxPercentage)

	// Clamp steering angle
	sanitized.Steering = v.clamp(sanitized.Steering, -180.0, 180.0)

	// Ensure position is at least 1
	if sanitized.Position < 1 {
		sanitized.Position = 1
	}

	// Ensure current lap is non-negative
	if sanitized.CurrentLap < 0 {
		sanitized.CurrentLap = 0
	}

	// Sanitize fuel data
	sanitized.Fuel = v.sanitizeFuelData(&sanitized.Fuel)

	// Sanitize tire data
	sanitized.Tires = v.sanitizeTireData(&sanitized.Tires)

	return sanitized
}

func (v *DataValidator) sanitizeSessionInfo(session *SessionInfo) SessionInfo {
	sanitized := *session

	// Clamp temperatures
	sanitized.AirTemperature = v.clamp(sanitized.AirTemperature, v.config.MinAirTemp, v.config.MaxAirTemp)
	sanitized.TrackTemperature = v.clamp(sanitized.TrackTemperature, v.config.MinAirTemp, v.config.MaxAirTemp)

	// Clamp track length
	sanitized.TrackLength = v.clamp(sanitized.TrackLength, v.config.MinTrackLength, v.config.MaxTrackLength)

	// Ensure non-negative values
	if sanitized.TimeRemaining < 0 {
		sanitized.TimeRemaining = 0
	}
	if sanitized.LapsRemaining < 0 {
		sanitized.LapsRemaining = 0
	}

	return sanitized
}

func (v *DataValidator) sanitizeFuelData(fuel *FuelData) FuelData {
	sanitized := *fuel

	// Clamp fuel values
	sanitized.Level = v.clamp(sanitized.Level, v.config.MinFuel, v.config.MaxFuel)
	sanitized.Capacity = v.clamp(sanitized.Capacity, v.config.MinFuel, v.config.MaxFuel)
	sanitized.Percentage = v.clamp(sanitized.Percentage, v.config.MinPercentage, v.config.MaxPercentage)

	// Ensure level doesn't exceed capacity
	if sanitized.Level > sanitized.Capacity {
		sanitized.Level = sanitized.Capacity
	}

	// Ensure non-negative values
	if sanitized.UsagePerLap < 0 {
		sanitized.UsagePerLap = 0
	}
	if sanitized.EstimatedLapsLeft < 0 {
		sanitized.EstimatedLapsLeft = 0
	}

	return sanitized
}

func (v *DataValidator) sanitizeTireData(tires *TireData) TireData {
	sanitized := *tires

	// Sanitize individual tire data
	sanitized.FrontLeft = v.sanitizeTireWheelData(&sanitized.FrontLeft)
	sanitized.FrontRight = v.sanitizeTireWheelData(&sanitized.FrontRight)
	sanitized.RearLeft = v.sanitizeTireWheelData(&sanitized.RearLeft)
	sanitized.RearRight = v.sanitizeTireWheelData(&sanitized.RearRight)

	return sanitized
}

func (v *DataValidator) sanitizeTireWheelData(tire *TireWheelData) TireWheelData {
	sanitized := *tire

	// Clamp tire values
	sanitized.Temperature = v.clamp(sanitized.Temperature, v.config.MinTireTemp, v.config.MaxTireTemp)
	sanitized.Pressure = v.clamp(sanitized.Pressure, v.config.MinTirePressure, v.config.MaxTirePressure)
	sanitized.WearPercent = v.clamp(sanitized.WearPercent, v.config.MinPercentage, v.config.MaxPercentage)
	sanitized.DirtLevel = v.clamp(sanitized.DirtLevel, v.config.MinPercentage, v.config.MaxPercentage)

	return sanitized
}

func (v *DataValidator) sanitizeOpponentData(opponent *OpponentData) OpponentData {
	sanitized := *opponent

	// Ensure position is at least 1
	if sanitized.Position < 1 {
		sanitized.Position = 1
	}

	// Ensure current lap is non-negative
	if sanitized.CurrentLap < 0 {
		sanitized.CurrentLap = 0
	}

	// Clamp lap distance percentage
	sanitized.LapDistancePercent = v.clamp(sanitized.LapDistancePercent, v.config.MinPercentage, v.config.MaxPercentage)

	return sanitized
}

// Individual validation methods for testing and specific use cases

// ValidateSpeed validates a speed value
func (v *DataValidator) ValidateSpeed(speed float64) error {
	if math.IsNaN(speed) || math.IsInf(speed, 0) {
		return &ValidationError{"Speed", speed, "speed is NaN or infinite"}
	}
	if speed < v.config.MinSpeed || speed > v.config.MaxSpeed {
		return &ValidationError{"Speed", speed,
			fmt.Sprintf("speed outside valid range [%.1f, %.1f] km/h", v.config.MinSpeed, v.config.MaxSpeed)}
	}
	return nil
}

// ValidateRPM validates an RPM value
func (v *DataValidator) ValidateRPM(rpm float64) error {
	if math.IsNaN(rpm) || math.IsInf(rpm, 0) {
		return &ValidationError{"RPM", rpm, "RPM is NaN or infinite"}
	}
	if rpm < v.config.MinRPM || rpm > v.config.MaxRPM {
		return &ValidationError{"RPM", rpm,
			fmt.Sprintf("RPM outside valid range [%.0f, %.0f]", v.config.MinRPM, v.config.MaxRPM)}
	}
	return nil
}

// ValidateTireTemperature validates a tire temperature value
func (v *DataValidator) ValidateTireTemperature(temp float64) error {
	if math.IsNaN(temp) || math.IsInf(temp, 0) {
		return &ValidationError{"TireTemperature", temp, "tire temperature is NaN or infinite"}
	}
	if temp < v.config.MinTireTemp || temp > v.config.MaxTireTemp {
		return &ValidationError{"TireTemperature", temp,
			fmt.Sprintf("tire temperature outside valid range [%.1f, %.1f]°C", v.config.MinTireTemp, v.config.MaxTireTemp)}
	}
	return nil
}

// ValidateAirTemperature validates an air temperature value
func (v *DataValidator) ValidateAirTemperature(temp float64) error {
	if math.IsNaN(temp) || math.IsInf(temp, 0) {
		return &ValidationError{"AirTemperature", temp, "air temperature is NaN or infinite"}
	}
	if temp < v.config.MinAirTemp || temp > v.config.MaxAirTemp {
		return &ValidationError{"AirTemperature", temp,
			fmt.Sprintf("air temperature outside valid range [%.1f, %.1f]°C", v.config.MinAirTemp, v.config.MaxAirTemp)}
	}
	return nil
}

// ValidateTirePressure validates a tire pressure value
func (v *DataValidator) ValidateTirePressure(pressure float64) error {
	if math.IsNaN(pressure) || math.IsInf(pressure, 0) {
		return &ValidationError{"TirePressure", pressure, "tire pressure is NaN or infinite"}
	}
	if pressure < v.config.MinTirePressure || pressure > v.config.MaxTirePressure {
		return &ValidationError{"TirePressure", pressure,
			fmt.Sprintf("tire pressure outside valid range [%.1f, %.1f] PSI", v.config.MinTirePressure, v.config.MaxTirePressure)}
	}
	return nil
}

// ValidateLapTime validates a lap time value
func (v *DataValidator) ValidateLapTime(lapTime time.Duration) error {
	if lapTime <= 0 {
		return &ValidationError{"LapTime", lapTime, "lap time must be positive"}
	}
	if lapTime < v.config.MinLapTime || lapTime > v.config.MaxLapTime {
		return &ValidationError{"LapTime", lapTime,
			fmt.Sprintf("lap time outside valid range [%v, %v]", v.config.MinLapTime, v.config.MaxLapTime)}
	}
	return nil
}

// ValidatePosition validates a position value
func (v *DataValidator) ValidatePosition(position int) error {
	if position < v.config.MinPosition || position > v.config.MaxPosition {
		return &ValidationError{"Position", position,
			fmt.Sprintf("position outside valid range [%d, %d]", v.config.MinPosition, v.config.MaxPosition)}
	}
	return nil
}

// ValidatePercentage validates a percentage value
func (v *DataValidator) ValidatePercentage(percentage float64) error {
	if math.IsNaN(percentage) || math.IsInf(percentage, 0) {
		return &ValidationError{"Percentage", percentage, "percentage is NaN or infinite"}
	}
	if percentage < v.config.MinPercentage || percentage > v.config.MaxPercentage {
		return &ValidationError{"Percentage", percentage,
			fmt.Sprintf("percentage outside valid range [%.1f, %.1f]", v.config.MinPercentage, v.config.MaxPercentage)}
	}
	return nil
}

// Data sanitization methods

// SanitizeSpeed cleans and corrects a speed value
func (v *DataValidator) SanitizeSpeed(speed float64) float64 {
	return v.clamp(speed, v.config.MinSpeed, v.config.MaxSpeed)
}

// SanitizeRPM cleans and corrects an RPM value
func (v *DataValidator) SanitizeRPM(rpm float64) float64 {
	return v.clamp(rpm, v.config.MinRPM, v.config.MaxRPM)
}

// SanitizeTemperature cleans and corrects a temperature value
func (v *DataValidator) SanitizeTemperature(temp float64, isTireTemp bool) float64 {
	if isTireTemp {
		return v.clamp(temp, v.config.MinTireTemp, v.config.MaxTireTemp)
	}
	return v.clamp(temp, v.config.MinAirTemp, v.config.MaxAirTemp)
}

// SanitizePressure cleans and corrects a pressure value
func (v *DataValidator) SanitizePressure(pressure float64) float64 {
	return v.clamp(pressure, v.config.MinTirePressure, v.config.MaxTirePressure)
}

// SanitizePercentage cleans and corrects a percentage value
func (v *DataValidator) SanitizePercentage(percentage float64) float64 {
	return v.clamp(percentage, v.config.MinPercentage, v.config.MaxPercentage)
}

// SanitizePosition cleans and corrects a position value
func (v *DataValidator) SanitizePosition(position int) int {
	if position < v.config.MinPosition {
		return v.config.MinPosition
	}
	if position > v.config.MaxPosition {
		return v.config.MaxPosition
	}
	return position
}

// Helper validation methods
func (v *DataValidator) isValidSimulatorType(simType SimulatorType) bool {
	return simType == SimulatorTypeIRacing || simType == SimulatorTypeACC || simType == SimulatorTypeLMU
}

func (v *DataValidator) isValidSessionType(sessionType SessionType) bool {
	return sessionType == SessionTypePractice || sessionType == SessionTypeQualifying ||
		sessionType == SessionTypeRace || sessionType == SessionTypeHotlap || sessionType == SessionTypeUnknown
}

func (v *DataValidator) isValidSessionFlag(flag SessionFlag) bool {
	return flag == SessionFlagNone || flag == SessionFlagGreen || flag == SessionFlagYellow ||
		flag == SessionFlagRed || flag == SessionFlagBlue || flag == SessionFlagWhite || flag == SessionFlagCheckered
}

func (v *DataValidator) isValidRaceFormat(format RaceFormat) bool {
	return format == RaceFormatSprint || format == RaceFormatEndurance || format == RaceFormatUnknown
}

func (v *DataValidator) isValidTireWearLevel(level TireWearLevel) bool {
	return level == TireWearFresh || level == TireWearGood || level == TireWearMedium ||
		level == TireWearWorn || level == TireWearCritical
}

func (v *DataValidator) isValidTireTempLevel(level TireTempLevel) bool {
	return level == TireTempCold || level == TireTempOptimal || level == TireTempHot ||
		level == TireTempOverheat
}

// Utility function to clamp values to a range
func (v *DataValidator) clamp(value, min, max float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
