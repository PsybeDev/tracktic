package sims

import (
	"context"
	"time"
)

// SimulatorType represents the type of racing simulator
type SimulatorType string

const (
	SimulatorTypeIRacing SimulatorType = "iracing"
	SimulatorTypeACC     SimulatorType = "acc"
	SimulatorTypeLMU     SimulatorType = "lmu"
)

// SessionType represents the type of racing session
type SessionType string

const (
	SessionTypePractice   SessionType = "practice"
	SessionTypeQualifying SessionType = "qualifying"
	SessionTypeRace       SessionType = "race"
	SessionTypeHotlap     SessionType = "hotlap"
	SessionTypeUnknown    SessionType = "unknown"
)

// SessionFlag represents current session flags
type SessionFlag string

const (
	SessionFlagNone      SessionFlag = "none"
	SessionFlagGreen     SessionFlag = "green"
	SessionFlagYellow    SessionFlag = "yellow"
	SessionFlagRed       SessionFlag = "red"
	SessionFlagBlue      SessionFlag = "blue"
	SessionFlagWhite     SessionFlag = "white"
	SessionFlagCheckered SessionFlag = "checkered"
)

// RaceFormat represents the format of the race
type RaceFormat string

const (
	RaceFormatSprint    RaceFormat = "sprint"
	RaceFormatEndurance RaceFormat = "endurance"
	RaceFormatUnknown   RaceFormat = "unknown"
)

// TelemetryData represents standardized telemetry data from any simulator
type TelemetryData struct {
	// Metadata
	Timestamp     time.Time     `json:"timestamp"`
	SimulatorType SimulatorType `json:"simulator_type"`
	IsConnected   bool          `json:"is_connected"`

	// Session Information
	Session SessionInfo `json:"session"`

	// Player Data
	Player PlayerData `json:"player"`

	// Opponents Data
	Opponents []OpponentData `json:"opponents"`
}

// SessionInfo contains session-related data
type SessionInfo struct {
	Type             SessionType   `json:"type"`
	Format           RaceFormat    `json:"format"`
	Flag             SessionFlag   `json:"flag"`
	TimeRemaining    time.Duration `json:"time_remaining"`
	LapsRemaining    int           `json:"laps_remaining"`
	TotalLaps        int           `json:"total_laps"`
	SessionTime      time.Duration `json:"session_time"`
	IsTimedSession   bool          `json:"is_timed_session"`
	IsLappedSession  bool          `json:"is_lapped_session"`
	TrackName        string        `json:"track_name"`
	TrackLength      float64       `json:"track_length_km"`
	AirTemperature   float64       `json:"air_temperature_celsius"`
	TrackTemperature float64       `json:"track_temperature_celsius"`
}

// PlayerData contains player-specific telemetry
type PlayerData struct {
	// Position and Lap Data
	Position           int           `json:"position"`
	CurrentLap         int           `json:"current_lap"`
	LapDistancePercent float64       `json:"lap_distance_percent"`
	LastLapTime        time.Duration `json:"last_lap_time"`
	BestLapTime        time.Duration `json:"best_lap_time"`
	CurrentLapTime     time.Duration `json:"current_lap_time"`
	GapToLeader        time.Duration `json:"gap_to_leader"`
	GapToAhead         time.Duration `json:"gap_to_ahead"`
	GapToBehind        time.Duration `json:"gap_to_behind"`

	// Fuel Data
	Fuel FuelData `json:"fuel"`

	// Tire Data
	Tires TireData `json:"tires"`

	// Pit Data
	Pit PitData `json:"pit"`

	// Car State
	Speed    float64 `json:"speed_kmh"`
	RPM      float64 `json:"rpm"`
	Gear     int     `json:"gear"`
	Throttle float64 `json:"throttle_percent"`
	Brake    float64 `json:"brake_percent"`
	Clutch   float64 `json:"clutch_percent"`
	Steering float64 `json:"steering_angle"`
}

// FuelData contains fuel-related information
type FuelData struct {
	Level             float64       `json:"level_liters"`
	Capacity          float64       `json:"capacity_liters"`
	Percentage        float64       `json:"percentage"`
	UsagePerLap       float64       `json:"usage_per_lap_liters"`
	UsagePerHour      float64       `json:"usage_per_hour_liters"`
	EstimatedLapsLeft int           `json:"estimated_laps_left"`
	EstimatedTimeLeft time.Duration `json:"estimated_time_left"`
	LowFuelWarning    bool          `json:"low_fuel_warning"`
}

// TireData contains tire-related information
type TireData struct {
	Compound   string        `json:"compound"`
	FrontLeft  TireWheelData `json:"front_left"`
	FrontRight TireWheelData `json:"front_right"`
	RearLeft   TireWheelData `json:"rear_left"`
	RearRight  TireWheelData `json:"rear_right"`
	WearLevel  TireWearLevel `json:"wear_level"`
	TempLevel  TireTempLevel `json:"temp_level"`
}

// TireWheelData contains individual tire data
type TireWheelData struct {
	Temperature float64 `json:"temperature_celsius"`
	Pressure    float64 `json:"pressure_psi"`
	WearPercent float64 `json:"wear_percent"`
	DirtLevel   float64 `json:"dirt_level"`
}

// TireWearLevel represents overall tire condition
type TireWearLevel string

const (
	TireWearFresh    TireWearLevel = "fresh"
	TireWearGood     TireWearLevel = "good"
	TireWearMedium   TireWearLevel = "medium"
	TireWearWorn     TireWearLevel = "worn"
	TireWearCritical TireWearLevel = "critical"
)

// TireTempLevel represents overall tire temperature
type TireTempLevel string

const (
	TireTempCold     TireTempLevel = "cold"
	TireTempOptimal  TireTempLevel = "optimal"
	TireTempHot      TireTempLevel = "hot"
	TireTempOverheat TireTempLevel = "overheat"
)

// PitData contains pit-related information
type PitData struct {
	IsOnPitRoad       bool          `json:"is_on_pit_road"`
	IsInPitStall      bool          `json:"is_in_pit_stall"`
	PitWindowOpen     bool          `json:"pit_window_open"`
	PitWindowLapsLeft int           `json:"pit_window_laps_left"`
	LastPitLap        int           `json:"last_pit_lap"`
	LastPitTime       time.Duration `json:"last_pit_time"`
	EstimatedPitTime  time.Duration `json:"estimated_pit_time"`
	PitSpeedLimit     float64       `json:"pit_speed_limit_kmh"`
}

// OpponentData contains data about other cars
type OpponentData struct {
	CarIndex           int           `json:"car_index"`
	DriverName         string        `json:"driver_name"`
	CarNumber          string        `json:"car_number"`
	Position           int           `json:"position"`
	CurrentLap         int           `json:"current_lap"`
	LapDistancePercent float64       `json:"lap_distance_percent"`
	LastLapTime        time.Duration `json:"last_lap_time"`
	BestLapTime        time.Duration `json:"best_lap_time"`
	GapToPlayer        time.Duration `json:"gap_to_player"`
	IsOnPitRoad        bool          `json:"is_on_pit_road"`
	IsInPitStall       bool          `json:"is_in_pit_stall"`
	LastPitLap         int           `json:"last_pit_lap"`
	EstimatedPitTime   time.Duration `json:"estimated_pit_time"`
}

// SimulatorConnector defines the interface that all simulator connectors must implement
type SimulatorConnector interface {
	// Connection Management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetSimulatorType() SimulatorType

	// Data Collection
	GetTelemetryData(ctx context.Context) (*TelemetryData, error)
	StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error)
	StopDataStream()

	// Health Check
	HealthCheck(ctx context.Context) error
}

// ConnectorFactory creates simulator connectors
type ConnectorFactory interface {
	CreateConnector(simType SimulatorType) (SimulatorConnector, error)
	GetSupportedSimulators() []SimulatorType
}

// DataProcessor processes and validates telemetry data
type DataProcessor interface {
	ProcessTelemetryData(data *TelemetryData) (*TelemetryData, error)
	ValidateData(data *TelemetryData) error
	CalculateDerivedData(data *TelemetryData) error
}

// Helper functions for data processing

// CalculateRaceFormat determines if a race is sprint or endurance based on duration/laps
func CalculateRaceFormat(sessionInfo *SessionInfo) RaceFormat {
	// If timed session longer than 60 minutes, consider endurance
	if sessionInfo.IsTimedSession && sessionInfo.TimeRemaining > 60*time.Minute {
		return RaceFormatEndurance
	}

	// If lapped session with more than 50 laps, consider endurance
	if sessionInfo.IsLappedSession && sessionInfo.TotalLaps > 50 {
		return RaceFormatEndurance
	}

	// Default to sprint for shorter races
	if sessionInfo.IsTimedSession || sessionInfo.IsLappedSession {
		return RaceFormatSprint
	}

	return RaceFormatUnknown
}

// CalculateTireWearLevel determines overall tire condition
func CalculateTireWearLevel(tires *TireData) TireWearLevel {
	avgWear := (tires.FrontLeft.WearPercent + tires.FrontRight.WearPercent +
		tires.RearLeft.WearPercent + tires.RearRight.WearPercent) / 4.0

	switch {
	case avgWear < 10:
		return TireWearFresh
	case avgWear < 30:
		return TireWearGood
	case avgWear < 60:
		return TireWearMedium
	case avgWear < 85:
		return TireWearWorn
	default:
		return TireWearCritical
	}
}

// CalculateTireTempLevel determines overall tire temperature condition
func CalculateTireTempLevel(tires *TireData) TireTempLevel {
	avgTemp := (tires.FrontLeft.Temperature + tires.FrontRight.Temperature +
		tires.RearLeft.Temperature + tires.RearRight.Temperature) / 4.0

	switch {
	case avgTemp < 70:
		return TireTempCold
	case avgTemp < 100:
		return TireTempOptimal
	case avgTemp < 120:
		return TireTempHot
	default:
		return TireTempOverheat
	}
}

// CalculateFuelEstimates calculates fuel-related estimates
func CalculateFuelEstimates(fuel *FuelData, averageLapTime time.Duration) {
	if fuel.UsagePerLap > 0 {
		fuel.EstimatedLapsLeft = int(fuel.Level / fuel.UsagePerLap)
		if averageLapTime > 0 {
			fuel.EstimatedTimeLeft = time.Duration(fuel.EstimatedLapsLeft) * averageLapTime
		}
	}

	fuel.Percentage = (fuel.Level / fuel.Capacity) * 100
	fuel.LowFuelWarning = fuel.Percentage < 15.0 // Low fuel warning at 15%
}
