package sims

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mpapenbr/goirsdk/irsdk"
)

// IRacingConnector implements the SimulatorConnector interface for iRacing
type IRacingConnector struct {
	isConnected           bool
	api                   *irsdk.Irsdk
	client                *http.Client
	dataStream            chan *TelemetryData
	errorStream           chan error
	stopStream            chan bool
	validator             *DataValidator
	retryHandler          *RetryHandler
	circuitBreaker        *CircuitBreaker
	lastValidData         *TelemetryData
	connectionAttempts    int
	maxConnectionAttempts int
}

// NewIRacingConnector creates a new iRacing connector instance
func NewIRacingConnector() *IRacingConnector {
	return &IRacingConnector{
		isConnected:           false,
		client:                &http.Client{Timeout: 10 * time.Second},
		stopStream:            make(chan bool),
		validator:             NewDataValidator(nil),
		retryHandler:          NewRetryHandler(nil),
		circuitBreaker:        NewCircuitBreaker(nil),
		maxConnectionAttempts: 5,
	}
}

// Connect implements SimulatorConnector.Connect
func (c *IRacingConnector) Connect(ctx context.Context) error {
	// Use circuit breaker to prevent repeated connection attempts
	return c.circuitBreaker.Execute(func() error {
		return c.retryHandler.Retry(ctx, func() error {
			return c.attemptConnection(ctx)
		})
	})
}

// attemptConnection performs a single connection attempt
func (c *IRacingConnector) attemptConnection(ctx context.Context) error {
	c.connectionAttempts++

	// Check if iRacing is running
	simIsRunning, err := irsdk.IsSimRunning(ctx, c.client)
	if err != nil {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "CheckSimRunning",
			OriginalError: err,
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	if !simIsRunning {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "SimNotRunning",
			OriginalError: fmt.Errorf("iRacing is not running"),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	// Initialize the iRacing SDK
	c.api = irsdk.NewIrsdk()
	if !c.api.WaitForValidData() {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "WaitForValidData",
			OriginalError: fmt.Errorf("failed to get valid data from iRacing"),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	// Test that we can actually read data
	_, err = c.readRawTelemetryData()
	if err != nil {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "ReadTestData",
			OriginalError: err,
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	c.isConnected = true
	c.connectionAttempts = 0 // Reset on successful connection
	return nil
}

// Disconnect implements SimulatorConnector.Disconnect
func (c *IRacingConnector) Disconnect() error {
	if c.isConnected {
		c.StopDataStream()
		c.api = nil
		c.isConnected = false
	}
	return nil
}

// IsConnected implements SimulatorConnector.IsConnected
func (c *IRacingConnector) IsConnected() bool {
	return c.isConnected
}

// GetSimulatorType implements SimulatorConnector.GetSimulatorType
func (c *IRacingConnector) GetSimulatorType() SimulatorType {
	return SimulatorTypeIRacing
}

// GetTelemetryData implements SimulatorConnector.GetTelemetryData
func (c *IRacingConnector) GetTelemetryData(ctx context.Context) (*TelemetryData, error) {
	if !c.isConnected || c.api == nil {
		return c.lastValidData, &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "GetTelemetryData",
			OriginalError: fmt.Errorf("not connected to iRacing"),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	var telemetryData *TelemetryData
	var err error

	// Use circuit breaker to protect against repeated failures
	err = c.circuitBreaker.Execute(func() error {
		return c.retryHandler.Retry(ctx, func() error {
			data, readErr := c.readRawTelemetryData()
			if readErr != nil {
				return &ConnectionError{
					ConnectorType: SimulatorTypeIRacing,
					Operation:     "ReadTelemetryData",
					OriginalError: readErr,
					Timestamp:     time.Now(),
					Retryable:     true,
				}
			}

			// Validate the received data
			validationErrors := c.validator.ValidateTelemetryData(data)
			if len(validationErrors) > 0 {
				// Log validation errors but try to sanitize and use the data
				sanitizedData := c.validator.SanitizeTelemetryData(data)

				// If sanitization produces valid data, use it
				validationErrors = c.validator.ValidateTelemetryData(sanitizedData)
				if len(validationErrors) > 0 {
					return fmt.Errorf("data validation failed after sanitization: %d errors", len(validationErrors))
				}
				data = sanitizedData
			}

			telemetryData = data
			c.lastValidData = data // Store as last known good data
			return nil
		})
	})

	if err != nil {
		// Return last valid data if available during errors
		if c.lastValidData != nil {
			staleData := *c.lastValidData
			staleData.IsConnected = false
			return &staleData, err
		}
		return nil, err
	}

	return telemetryData, nil
}

// readRawTelemetryData reads and converts raw iRacing data to TelemetryData
func (c *IRacingConnector) readRawTelemetryData() (*TelemetryData, error) {
	// Wait for valid data and update
	if !c.api.WaitForValidData() {
		return nil, fmt.Errorf("failed to get valid data from iRacing")
	}

	c.api.GetData()

	return c.convertToTelemetryData()
}

// StartDataStream implements SimulatorConnector.StartDataStream
func (c *IRacingConnector) StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error) {
	c.dataStream = make(chan *TelemetryData, 10)
	c.errorStream = make(chan error, 10)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				data, err := c.GetTelemetryData(ctx)
				if err != nil {
					select {
					case c.errorStream <- err:
					default:
					}
				} else {
					select {
					case c.dataStream <- data:
					default:
					}
				}
			case <-c.stopStream:
				close(c.dataStream)
				close(c.errorStream)
				return
			case <-ctx.Done():
				close(c.dataStream)
				close(c.errorStream)
				return
			}
		}
	}()

	return c.dataStream, c.errorStream
}

// StopDataStream implements SimulatorConnector.StopDataStream
func (c *IRacingConnector) StopDataStream() {
	select {
	case c.stopStream <- true:
	default:
	}
}

// HealthCheck implements SimulatorConnector.HealthCheck
func (c *IRacingConnector) HealthCheck(ctx context.Context) error {
	if !c.isConnected {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "HealthCheck",
			OriginalError: fmt.Errorf("not connected to iRacing"),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	// Check if iRacing is still running
	simIsRunning, err := irsdk.IsSimRunning(ctx, c.client)
	if err != nil {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "HealthCheck",
			OriginalError: fmt.Errorf("health check failed: %w", err),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	if !simIsRunning {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "HealthCheck",
			OriginalError: fmt.Errorf("iRacing is no longer running"),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	// Try to get some data to verify connection
	if c.api == nil || !c.api.WaitForValidData() {
		return &ConnectionError{
			ConnectorType: SimulatorTypeIRacing,
			Operation:     "HealthCheck",
			OriginalError: fmt.Errorf("failed to get valid data from iRacing"),
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	return nil
}

// Helper methods for data conversion

func (c *IRacingConnector) convertToTelemetryData() (*TelemetryData, error) {
	now := time.Now()

	// Get session information
	sessionInfo, err := c.getSessionInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get session info: %w", err)
	}

	// Get player data
	playerData, err := c.getPlayerData()
	if err != nil {
		return nil, fmt.Errorf("failed to get player data: %w", err)
	}

	// Get opponents data
	opponents, err := c.getOpponentsData()
	if err != nil {
		return nil, fmt.Errorf("failed to get opponents data: %w", err)
	}

	// Create telemetry data
	telemetry := &TelemetryData{
		Timestamp:     now,
		SimulatorType: SimulatorTypeIRacing,
		IsConnected:   true,
		Session:       *sessionInfo,
		Player:        *playerData,
		Opponents:     opponents,
	}

	return telemetry, nil
}

func (c *IRacingConnector) getSessionInfo() (*SessionInfo, error) {
	// Get session time
	sessionTime, err := c.api.GetDoubleValue("SessionTime")
	if err != nil {
		return nil, fmt.Errorf("failed to get session time: %w", err)
	}

	// Get session time remaining
	sessionTimeRemain, err := c.api.GetDoubleValue("SessionTimeRemain")
	if err != nil {
		return nil, fmt.Errorf("failed to get session time remaining: %w", err)
	}

	// Get session laps remaining
	sessionLapsRemain, err := c.api.GetIntValue("SessionLapsRemain")
	if err != nil {
		sessionLapsRemain = 0 // Default if not available
	}

	// Get session state
	sessionState, err := c.api.GetIntValue("SessionState")
	if err != nil {
		return nil, fmt.Errorf("failed to get session state: %w", err)
	}

	// Get session flags
	sessionFlags, err := c.api.GetIntValue("SessionFlags")
	if err != nil {
		sessionFlags = 0 // Default if not available
	}
	// Get track info
	trackDisplayName := "Unknown Track" // Default value, iRacing doesn't easily provide track display name

	trackLength, err := c.api.GetFloatValue("TrackLength")
	if err != nil {
		trackLength = 0.0
	}

	// Get weather info
	airTemp, err := c.api.GetFloatValue("AirTemp")
	if err != nil {
		airTemp = 20.0 // Default temperature
	}

	trackTemp, err := c.api.GetFloatValue("TrackTemp")
	if err != nil {
		trackTemp = 25.0 // Default temperature
	}

	// Convert session state to session type
	sessionType := c.convertSessionState(sessionState)

	// Convert session flags
	sessionFlag := c.convertSessionFlags(sessionFlags)

	// Determine if session is timed or lapped
	isTimedSession := sessionTimeRemain > 0
	isLappedSession := sessionLapsRemain > 0

	sessionInfo := &SessionInfo{
		Type:             sessionType,
		Flag:             sessionFlag,
		TimeRemaining:    time.Duration(sessionTimeRemain) * time.Second,
		LapsRemaining:    int(sessionLapsRemain),
		TotalLaps:        0, // iRacing doesn't always provide this directly
		SessionTime:      time.Duration(sessionTime) * time.Second,
		IsTimedSession:   isTimedSession,
		IsLappedSession:  isLappedSession,
		TrackName:        trackDisplayName,
		TrackLength:      float64(trackLength), // Already in km
		AirTemperature:   float64(airTemp),
		TrackTemperature: float64(trackTemp),
	}

	// Calculate race format
	sessionInfo.Format = CalculateRaceFormat(sessionInfo)

	return sessionInfo, nil
}

func (c *IRacingConnector) getPlayerData() (*PlayerData, error) {
	// Get position data
	position, err := c.api.GetIntValue("Position")
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	// Get lap data
	lap, err := c.api.GetIntValue("Lap")
	if err != nil {
		return nil, fmt.Errorf("failed to get lap: %w", err)
	}

	lapDistPct, err := c.api.GetFloatValue("LapDistPct")
	if err != nil {
		return nil, fmt.Errorf("failed to get lap distance percentage: %w", err)
	}

	lastLapTime, err := c.api.GetFloatValue("LapLastLapTime")
	if err != nil {
		lastLapTime = 0.0
	}

	bestLapTime, err := c.api.GetFloatValue("LapBestLapTime")
	if err != nil {
		bestLapTime = 0.0
	}

	currentLapTime, err := c.api.GetFloatValue("LapCurrentLapTime")
	if err != nil {
		currentLapTime = 0.0
	}

	// Get fuel data
	fuelLevel, err := c.api.GetFloatValue("FuelLevel")
	if err != nil {
		return nil, fmt.Errorf("failed to get fuel level: %w", err)
	}

	fuelUsePerHour, err := c.api.GetFloatValue("FuelUsePerHour")
	if err != nil {
		fuelUsePerHour = 0.0
	}

	// Get tire data
	tireData, err := c.getTireData()
	if err != nil {
		return nil, fmt.Errorf("failed to get tire data: %w", err)
	}

	// Get pit data
	pitData, err := c.getPitData()
	if err != nil {
		return nil, fmt.Errorf("failed to get pit data: %w", err)
	}

	// Get car state
	speed, err := c.api.GetFloatValue("Speed")
	if err != nil {
		speed = 0.0
	}

	rpm, err := c.api.GetFloatValue("RPM")
	if err != nil {
		rpm = 0.0
	}

	gear, err := c.api.GetIntValue("Gear")
	if err != nil {
		gear = 0
	}

	throttle, err := c.api.GetFloatValue("Throttle")
	if err != nil {
		throttle = 0.0
	}

	brake, err := c.api.GetFloatValue("Brake")
	if err != nil {
		brake = 0.0
	}

	clutch, err := c.api.GetFloatValue("Clutch")
	if err != nil {
		clutch = 0.0
	}

	steerAngle, err := c.api.GetFloatValue("SteeringWheelAngle")
	if err != nil {
		steerAngle = 0.0
	}

	// Calculate fuel estimates
	fuelData := FuelData{
		Level:        float64(fuelLevel),
		Capacity:     100.0, // Default capacity, iRacing doesn't always provide this
		UsagePerHour: float64(fuelUsePerHour),
	}

	// Calculate fuel usage per lap (approximate based on usage per hour and lap time)
	if lastLapTime > 0 && fuelUsePerHour > 0 {
		lapTimeHours := float64(lastLapTime) / 3600.0
		fuelData.UsagePerLap = float64(fuelUsePerHour) * lapTimeHours
	}

	avgLapTime := time.Duration(lastLapTime) * time.Second
	CalculateFuelEstimates(&fuelData, avgLapTime)

	playerData := &PlayerData{
		Position:           int(position),
		CurrentLap:         int(lap),
		LapDistancePercent: float64(lapDistPct * 100), // Convert to percentage
		LastLapTime:        time.Duration(lastLapTime) * time.Second,
		BestLapTime:        time.Duration(bestLapTime) * time.Second,
		CurrentLapTime:     time.Duration(currentLapTime) * time.Second,
		GapToLeader:        0, // Will be calculated from opponent data
		GapToAhead:         0, // Will be calculated from opponent data
		GapToBehind:        0, // Will be calculated from opponent data
		Fuel:               fuelData,
		Tires:              *tireData,
		Pit:                *pitData,
		Speed:              float64(speed * 3.6), // Convert m/s to km/h
		RPM:                float64(rpm),
		Gear:               int(gear),
		Throttle:           float64(throttle * 100), // Convert to percentage
		Brake:              float64(brake * 100),    // Convert to percentage
		Clutch:             float64(clutch * 100),   // Convert to percentage
		Steering:           float64(steerAngle),
	}

	return playerData, nil
}

func (c *IRacingConnector) getTireData() (*TireData, error) {
	// Get tire temperatures
	lfTemp, err := c.api.GetFloatValue("LFtempCM")
	if err != nil {
		lfTemp = 80.0 // Default temperature
	}

	rfTemp, err := c.api.GetFloatValue("RFtempCM")
	if err != nil {
		rfTemp = 80.0
	}

	lrTemp, err := c.api.GetFloatValue("LRtempCM")
	if err != nil {
		lrTemp = 80.0
	}

	rrTemp, err := c.api.GetFloatValue("RRtempCM")
	if err != nil {
		rrTemp = 80.0
	}

	// Get tire wear
	lfWear, err := c.api.GetFloatValue("LFwearM")
	if err != nil {
		lfWear = 0.0
	}

	rfWear, err := c.api.GetFloatValue("RFwearM")
	if err != nil {
		rfWear = 0.0
	}

	lrWear, err := c.api.GetFloatValue("LRwearM")
	if err != nil {
		lrWear = 0.0
	}

	rrWear, err := c.api.GetFloatValue("RRwearM")
	if err != nil {
		rrWear = 0.0
	}

	// Create tire data
	tireData := &TireData{
		Compound: "Unknown", // iRacing doesn't provide tire compound directly
		FrontLeft: TireWheelData{
			Temperature: float64(lfTemp),
			Pressure:    30.0,                  // Default pressure, iRacing doesn't always provide this
			WearPercent: float64(lfWear * 100), // Convert to percentage
			DirtLevel:   0.0,                   // iRacing doesn't provide dirt level
		},
		FrontRight: TireWheelData{
			Temperature: float64(rfTemp),
			Pressure:    30.0,
			WearPercent: float64(rfWear * 100),
			DirtLevel:   0.0,
		},
		RearLeft: TireWheelData{
			Temperature: float64(lrTemp),
			Pressure:    30.0,
			WearPercent: float64(lrWear * 100),
			DirtLevel:   0.0,
		},
		RearRight: TireWheelData{
			Temperature: float64(rrTemp),
			Pressure:    30.0,
			WearPercent: float64(rrWear * 100),
			DirtLevel:   0.0,
		},
	}

	// Calculate derived data
	tireData.WearLevel = CalculateTireWearLevel(tireData)
	tireData.TempLevel = CalculateTireTempLevel(tireData)

	return tireData, nil
}

func (c *IRacingConnector) getPitData() (*PitData, error) {
	// Get pit road status
	onPitRoad, err := c.api.GetBoolValue("OnPitRoad")
	if err != nil {
		onPitRoad = false
	}

	// Get pit stall status (approximate)
	speed, err := c.api.GetFloatValue("Speed")
	if err != nil {
		speed = 0.0
	}

	isInPitStall := onPitRoad && speed < 1.0 // If on pit road and not moving

	pitData := &PitData{
		IsOnPitRoad:       onPitRoad,
		IsInPitStall:      isInPitStall,
		PitWindowOpen:     true,             // iRacing doesn't provide this directly
		PitWindowLapsLeft: 0,                // iRacing doesn't provide this directly
		LastPitLap:        0,                // Would need to track this separately
		LastPitTime:       0,                // Would need to track this separately
		EstimatedPitTime:  30 * time.Second, // Default estimate
		PitSpeedLimit:     56.0,             // Default iRacing pit speed limit (35 mph = ~56 km/h)
	}

	return pitData, nil
}

func (c *IRacingConnector) getOpponentsData() ([]OpponentData, error) {
	// For now, return empty opponents array since the goirsdk library doesn't
	// provide simple array access methods and we need to complete the basic functionality first
	// TODO: Implement opponent data collection using the correct iRacing variable names
	return []OpponentData{}, nil
}

func (c *IRacingConnector) convertSessionState(sessionState int32) SessionType {
	// iRacing session states:
	// 0 = irsdk_StateInvalid
	// 1 = irsdk_StateGetInCar
	// 2 = irsdk_StateWarmup
	// 3 = irsdk_StateParadeLaps
	// 4 = irsdk_StateRacing
	// 5 = irsdk_StateCheckered
	// 6 = irsdk_StateCoolDown

	switch sessionState {
	case 2: // Warmup
		return SessionTypePractice
	case 3, 4, 5: // Parade, Racing, Checkered
		return SessionTypeRace
	default:
		// Default to practice for other states
		return SessionTypePractice
	}
}

func (c *IRacingConnector) convertSessionFlags(sessionFlags int32) SessionFlag {
	// iRacing session flags (bitfield):
	// 0x00000001 = checkered flag
	// 0x00000002 = white flag
	// 0x00000004 = green flag
	// 0x00000008 = yellow flag
	// 0x00000010 = red flag
	// 0x00000020 = blue flag
	// 0x00000040 = debris flag
	// 0x00000080 = crossed flag
	// etc.

	if sessionFlags&0x00000010 != 0 { // Red flag
		return SessionFlagRed
	}
	if sessionFlags&0x00000008 != 0 { // Yellow flag
		return SessionFlagYellow
	}
	if sessionFlags&0x00000020 != 0 { // Blue flag
		return SessionFlagBlue
	}
	if sessionFlags&0x00000002 != 0 { // White flag
		return SessionFlagWhite
	}
	if sessionFlags&0x00000001 != 0 { // Checkered flag
		return SessionFlagCheckered
	}
	if sessionFlags&0x00000004 != 0 { // Green flag
		return SessionFlagGreen
	}

	return SessionFlagNone
}
