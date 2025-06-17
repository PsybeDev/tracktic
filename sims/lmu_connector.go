package sims

import (
	"context"
	"fmt"
	"time"
)

// LMUConnector implements the SimulatorConnector interface for Le Mans Ultimate
type LMUConnector struct {
	isConnected           bool
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

// NewLMUConnector creates a new LMU connector instance
func NewLMUConnector() *LMUConnector {
	return &LMUConnector{
		isConnected:           false,
		stopStream:            make(chan bool),
		validator:             NewDataValidator(nil),
		retryHandler:          NewRetryHandler(nil),
		circuitBreaker:        NewCircuitBreaker(nil),
		maxConnectionAttempts: 5,
	}
}

// Connect implements SimulatorConnector.Connect
func (c *LMUConnector) Connect(ctx context.Context) error {
	// LMU support is not fully implemented yet
	return fmt.Errorf("LMU connector not fully implemented - requires rFactor 2 SDK integration")
}

// Disconnect implements SimulatorConnector.Disconnect
func (c *LMUConnector) Disconnect() error {
	if c.isConnected {
		c.StopDataStream()
		c.isConnected = false
	}
	return nil
}

// IsConnected implements SimulatorConnector.IsConnected
func (c *LMUConnector) IsConnected() bool {
	return c.isConnected
}

// GetSimulatorType implements SimulatorConnector.GetSimulatorType
func (c *LMUConnector) GetSimulatorType() SimulatorType {
	return SimulatorTypeLMU
}

// GetTelemetryData implements SimulatorConnector.GetTelemetryData
func (c *LMUConnector) GetTelemetryData(ctx context.Context) (*TelemetryData, error) {
	if !c.isConnected {
		return nil, fmt.Errorf("not connected to LMU")
	}

	// Return mock data for now
	return c.createMockTelemetryData(), nil
}

// StartDataStream implements SimulatorConnector.StartDataStream
func (c *LMUConnector) StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error) {
	c.dataStream = make(chan *TelemetryData, 10)
	c.errorStream = make(chan error, 10)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if c.isConnected {
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
func (c *LMUConnector) StopDataStream() {
	select {
	case c.stopStream <- true:
	default:
	}
}

// HealthCheck implements SimulatorConnector.HealthCheck
func (c *LMUConnector) HealthCheck(ctx context.Context) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to LMU")
	}
	return nil
}

// createMockTelemetryData creates mock telemetry data for testing
func (c *LMUConnector) createMockTelemetryData() *TelemetryData {
	now := time.Now()

	// Create mock session info
	sessionInfo := SessionInfo{
		Type:             SessionTypeRace,
		Format:           RaceFormatEndurance,
		Flag:             SessionFlagGreen,
		TimeRemaining:    45 * time.Minute,
		LapsRemaining:    0,
		TotalLaps:        0,
		SessionTime:      15 * time.Minute,
		IsTimedSession:   true,
		IsLappedSession:  false,
		TrackName:        "Le Mans",
		TrackLength:      13.626,
		AirTemperature:   22.0,
		TrackTemperature: 28.0,
	}

	sessionInfo.Format = CalculateRaceFormat(&sessionInfo)

	// Create mock player data
	playerData := PlayerData{
		Position:           5,
		CurrentLap:         12,
		LapDistancePercent: 35.5,
		LastLapTime:        3*time.Minute + 25*time.Second,
		BestLapTime:        3*time.Minute + 20*time.Second,
		CurrentLapTime:     1*time.Minute + 15*time.Second,
		GapToLeader:        25 * time.Second,
		GapToAhead:         8 * time.Second,
		GapToBehind:        12 * time.Second,

		Fuel: FuelData{
			Level:             65.0,
			Capacity:          90.0,
			UsagePerLap:       4.2,
			EstimatedLapsLeft: 15,
			Percentage:        72.2,
			LowFuelWarning:    false,
		},

		Tires: TireData{
			Compound: "Medium",
			FrontLeft: TireWheelData{
				Temperature: 85.0,
				Pressure:    28.5,
				WearPercent: 25.0,
				DirtLevel:   0.1,
			},
			FrontRight: TireWheelData{
				Temperature: 87.0,
				Pressure:    28.3,
				WearPercent: 26.0,
				DirtLevel:   0.1,
			},
			RearLeft: TireWheelData{
				Temperature: 82.0,
				Pressure:    27.8,
				WearPercent: 28.0,
				DirtLevel:   0.2,
			},
			RearRight: TireWheelData{
				Temperature: 83.0,
				Pressure:    27.9,
				WearPercent: 27.0,
				DirtLevel:   0.2,
			},
		},

		Pit: PitData{
			IsOnPitRoad:       false,
			IsInPitStall:      false,
			PitWindowOpen:     true,
			PitWindowLapsLeft: 8,
			LastPitLap:        0,
			EstimatedPitTime:  35 * time.Second,
			PitSpeedLimit:     60.0,
		},

		Speed:    285.5,
		RPM:      7200.0,
		Gear:     6,
		Throttle: 95.0,
		Brake:    0.0,
		Clutch:   0.0,
		Steering: -2.5,
	}

	// Calculate derived data
	playerData.Tires.WearLevel = CalculateTireWearLevel(&playerData.Tires)
	playerData.Tires.TempLevel = CalculateTireTempLevel(&playerData.Tires)
	CalculateFuelEstimates(&playerData.Fuel, playerData.LastLapTime)

	// Mock opponents data
	opponents := []OpponentData{
		{
			CarIndex:           1,
			DriverName:         "Driver 1",
			CarNumber:          "1",
			Position:           1,
			CurrentLap:         12,
			LapDistancePercent: 45.2,
			LastLapTime:        3*time.Minute + 18*time.Second,
			BestLapTime:        3*time.Minute + 15*time.Second,
			GapToPlayer:        -25 * time.Second,
			IsOnPitRoad:        false,
			EstimatedPitTime:   35 * time.Second,
		},
	}

	return &TelemetryData{
		Timestamp:     now,
		SimulatorType: SimulatorTypeLMU,
		IsConnected:   c.isConnected,
		Session:       sessionInfo,
		Player:        playerData,
		Opponents:     opponents,
	}
}
