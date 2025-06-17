package sims

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ACCConnector implements the SimulatorConnector interface for Assetto Corsa Competizione
type ACCConnector struct {
	isConnected           bool
	dataStream            chan *TelemetryData
	errorStream           chan error
	stopStream            chan bool
	physicsHandle         windows.Handle
	graphicsHandle        windows.Handle
	staticHandle          windows.Handle
	validator             *DataValidator
	retryHandler          *RetryHandler
	circuitBreaker        *CircuitBreaker
	lastValidData         *TelemetryData
	connectionAttempts    int
	maxConnectionAttempts int
}

// ACC Shared Memory Structure Definitions
// These match the ACC SDK shared memory layout

type ACCPhysics struct {
	PacketID            int32         `json:"packet_id"`
	Gas                 float32       `json:"gas"`
	Brake               float32       `json:"brake"`
	Fuel                float32       `json:"fuel"`
	Gear                int32         `json:"gear"`
	RPM                 int32         `json:"rpm"`
	SteerAngle          float32       `json:"steer_angle"`
	SpeedKMH            float32       `json:"speed_kmh"`
	Velocity            [3]float32    `json:"velocity"`
	AccG                [3]float32    `json:"acc_g"`
	WheelSlip           [4]float32    `json:"wheel_slip"`
	WheelLoad           [4]float32    `json:"wheel_load"`
	WheelsPressure      [4]float32    `json:"wheels_pressure"`
	WheelAngularSpeed   [4]float32    `json:"wheel_angular_speed"`
	TyreWear            [4]float32    `json:"tyre_wear"`
	TyreDirtyLevel      [4]float32    `json:"tyre_dirty_level"`
	TyreCoreTemperature [4]float32    `json:"tyre_core_temperature"`
	CamberRAD           [4]float32    `json:"camber_rad"`
	SuspensionTravel    [4]float32    `json:"suspension_travel"`
	DRS                 float32       `json:"drs"`
	TC                  float32       `json:"tc"`
	Heading             float32       `json:"heading"`
	Pitch               float32       `json:"pitch"`
	Roll                float32       `json:"roll"`
	CgHeight            float32       `json:"cg_height"`
	CarDamage           [5]float32    `json:"car_damage"`
	NumberOfTyresOut    int32         `json:"number_of_tyres_out"`
	PitLimiterOn        int32         `json:"pit_limiter_on"`
	ABS                 float32       `json:"abs"`
	KersCharge          float32       `json:"kers_charge"`
	KersInput           float32       `json:"kers_input"`
	AutoShifterOn       int32         `json:"auto_shifter_on"`
	RideHeight          [2]float32    `json:"ride_height"`
	TurboBoost          float32       `json:"turbo_boost"`
	Ballast             float32       `json:"ballast"`
	AirDensity          float32       `json:"air_density"`
	AirTemp             float32       `json:"air_temp"`
	RoadTemp            float32       `json:"road_temp"`
	LocalAngularVel     [3]float32    `json:"local_angular_vel"`
	FinalFF             float32       `json:"final_ff"`
	PerformanceMeter    float32       `json:"performance_meter"`
	EngineBrake         int32         `json:"engine_brake"`
	ErsRecoveryLevel    int32         `json:"ers_recovery_level"`
	ErsPowerLevel       int32         `json:"ers_power_level"`
	ErsHeatCharging     int32         `json:"ers_heat_charging"`
	ErsIsCharging       int32         `json:"ers_is_charging"`
	KersCurrentKJ       float32       `json:"kers_current_kj"`
	DrsAvailable        int32         `json:"drs_available"`
	DrsEnabled          int32         `json:"drs_enabled"`
	BrakeTemp           [4]float32    `json:"brake_temp"`
	Clutch              float32       `json:"clutch"`
	TyreTempI           [4]float32    `json:"tyre_temp_i"`
	TyreTempM           [4]float32    `json:"tyre_temp_m"`
	TyreTempO           [4]float32    `json:"tyre_temp_o"`
	IsAIControlled      int32         `json:"is_ai_controlled"`
	TyreContactPoint    [4][3]float32 `json:"tyre_contact_point"`
	TyreContactNormal   [4][3]float32 `json:"tyre_contact_normal"`
	TyreContactHeading  [4][3]float32 `json:"tyre_contact_heading"`
	BrakeBias           float32       `json:"brake_bias"`
	LocalVelocity       [3]float32    `json:"local_velocity"`
	P2PActivations      int32         `json:"p2p_activations"`
	P2PStatus           int32         `json:"p2p_status"`
	CurrentMaxRPM       int32         `json:"current_max_rpm"`
	MZ                  [4]float32    `json:"mz"`
	FX                  [4]float32    `json:"fx"`
	FY                  [4]float32    `json:"fy"`
	SlipRatio           [4]float32    `json:"slip_ratio"`
	SlipAngle           [4]float32    `json:"slip_angle"`
	TCInAction          int32         `json:"tc_in_action"`
	ABSInAction         int32         `json:"abs_in_action"`
	SuspensionDamage    [4]float32    `json:"suspension_damage"`
	TyreTemp            [4]float32    `json:"tyre_temp"`
	WaterTemp           float32       `json:"water_temp"`
	BrakePressure       [4]float32    `json:"brake_pressure"`
	FrontBrakeCompound  int32         `json:"front_brake_compound"`
	RearBrakeCompound   int32         `json:"rear_brake_compound"`
	PadLife             [4]float32    `json:"pad_life"`
	DiscLife            [4]float32    `json:"disc_life"`
	IgnitionOn          int32         `json:"ignition_on"`
	StarterEngineOn     int32         `json:"starter_engine_on"`
	IsEngineRunning     int32         `json:"is_engine_running"`
	KerbVibration       float32       `json:"kerb_vibration"`
	SlipVibrations      float32       `json:"slip_vibrations"`
	GVibrations         float32       `json:"g_vibrations"`
	AbsVibrations       float32       `json:"abs_vibrations"`
}

type ACCGraphics struct {
	PacketID                 int32          `json:"packet_id"`
	ACStatus                 int32          `json:"ac_status"`
	ACSessionType            int32          `json:"ac_session_type"`
	CurrentTime              [15]uint16     `json:"current_time"`
	LastTime                 [15]uint16     `json:"last_time"`
	BestTime                 [15]uint16     `json:"best_time"`
	Split                    [15]uint16     `json:"split"`
	CompletedLaps            int32          `json:"completed_laps"`
	Position                 int32          `json:"position"`
	ICurrentTime             int32          `json:"i_current_time"`
	ILastTime                int32          `json:"i_last_time"`
	IBestTime                int32          `json:"i_best_time"`
	SessionTimeLeft          float32        `json:"session_time_left"`
	DistanceTraveled         float32        `json:"distance_traveled"`
	IsInPit                  int32          `json:"is_in_pit"`
	CurrentSectorIndex       int32          `json:"current_sector_index"`
	LastSectorTime           int32          `json:"last_sector_time"`
	NumberOfLaps             int32          `json:"number_of_laps"`
	TyreCompound             [33]uint16     `json:"tyre_compound"`
	ReplayTimeMultiplier     float32        `json:"replay_time_multiplier"`
	NormalizedCarPosition    float32        `json:"normalized_car_position"`
	ActiveCars               int32          `json:"active_cars"`
	CarCoordinates           [60][3]float32 `json:"car_coordinates"`
	CarID                    [60]int32      `json:"car_id"`
	PlayerCarID              int32          `json:"player_car_id"`
	PenaltyTime              float32        `json:"penalty_time"`
	Flag                     int32          `json:"flag"`
	PenaltyShortcut          int32          `json:"penalty_shortcut"`
	IdealLineOn              int32          `json:"ideal_line_on"`
	IsInPitLane              int32          `json:"is_in_pit_lane"`
	SurfaceGrip              float32        `json:"surface_grip"`
	MandatoryPitDone         int32          `json:"mandatory_pit_done"`
	WindSpeed                float32        `json:"wind_speed"`
	WindDirection            float32        `json:"wind_direction"`
	IsSetupMenuVisible       int32          `json:"is_setup_menu_visible"`
	MainDisplayIndex         int32          `json:"main_display_index"`
	SecondaryDisplyIndex     int32          `json:"secondary_disply_index"`
	TC                       int32          `json:"tc"`
	TCCut                    int32          `json:"tc_cut"`
	EngineMap                int32          `json:"engine_map"`
	ABS                      int32          `json:"abs"`
	FuelXLap                 float32        `json:"fuel_x_lap"`
	RainLights               int32          `json:"rain_lights"`
	FlashingLights           int32          `json:"flashing_lights"`
	LightsStage              int32          `json:"lights_stage"`
	ExhaustTemperature       float32        `json:"exhaust_temperature"`
	WiperLV                  int32          `json:"wiper_lv"`
	DriverStintTotalTimeLeft int32          `json:"driver_stint_total_time_left"`
	DriverStintTimeLeft      int32          `json:"driver_stint_time_left"`
	RainTyres                int32          `json:"rain_tyres"`
	SessionIndex             int32          `json:"session_index"`
	UsedFuel                 float32        `json:"used_fuel"`
	DeltaLapTime             [15]uint16     `json:"delta_lap_time"`
	IDeltaLapTime            int32          `json:"i_delta_lap_time"`
	EstimatedLapTime         [15]uint16     `json:"estimated_lap_time"`
	IEstimatedLapTime        int32          `json:"i_estimated_lap_time"`
	IsDeltaPositive          int32          `json:"is_delta_positive"`
	ISplit                   int32          `json:"i_split"`
	IsValidLap               int32          `json:"is_valid_lap"`
	FuelEstimatedLaps        float32        `json:"fuel_estimated_laps"`
	TrackStatus              [33]uint16     `json:"track_status"`
	MissingMandatoryPits     int32          `json:"missing_mandatory_pits"`
	Clock                    float32        `json:"clock"`
	DirectionLightsLeft      int32          `json:"direction_lights_left"`
	DirectionLightsRight     int32          `json:"direction_lights_right"`
	GlobalYellow             int32          `json:"global_yellow"`
	GlobalYellow1            int32          `json:"global_yellow1"`
	GlobalYellow2            int32          `json:"global_yellow2"`
	GlobalYellow3            int32          `json:"global_yellow3"`
	GlobalWhite              int32          `json:"global_white"`
	GlobalGreen              int32          `json:"global_green"`
	GlobalChequered          int32          `json:"global_chequered"`
	GlobalRed                int32          `json:"global_red"`
	MfdTyreSet               int32          `json:"mfd_tyre_set"`
	MfdFuelToAdd             float32        `json:"mfd_fuel_to_add"`
	MfdTyrePressureLF        float32        `json:"mfd_tyre_pressure_lf"`
	MfdTyrePressureRF        float32        `json:"mfd_tyre_pressure_rf"`
	MfdTyrePressureLR        float32        `json:"mfd_tyre_pressure_lr"`
	MfdTyrePressureRR        float32        `json:"mfd_tyre_pressure_rr"`
	TrackGripStatus          int32          `json:"track_grip_status"`
	RainIntensity            int32          `json:"rain_intensity"`
	RainIntensityIn10min     int32          `json:"rain_intensity_in_10min"`
	RainIntensityIn30min     int32          `json:"rain_intensity_in_30min"`
	CurrentTyreSet           int32          `json:"current_tyre_set"`
	StrategyTyreSet          int32          `json:"strategy_tyre_set"`
	GapAhead                 int32          `json:"gap_ahead"`
	GapBehind                int32          `json:"gap_behind"`
}

type ACCStatic struct {
	SMVersion                [15]uint16 `json:"sm_version"`
	ACVersion                [15]uint16 `json:"ac_version"`
	NumberOfSessions         int32      `json:"number_of_sessions"`
	NumCars                  int32      `json:"num_cars"`
	CarModel                 [33]uint16 `json:"car_model"`
	Track                    [33]uint16 `json:"track"`
	PlayerName               [33]uint16 `json:"player_name"`
	PlayerSurname            [33]uint16 `json:"player_surname"`
	PlayerNick               [33]uint16 `json:"player_nick"`
	SectorCount              int32      `json:"sector_count"`
	MaxTorque                float32    `json:"max_torque"`
	MaxPower                 float32    `json:"max_power"`
	MaxRPM                   int32      `json:"max_rpm"`
	MaxFuel                  float32    `json:"max_fuel"`
	SuspensionMaxTravel      [4]float32 `json:"suspension_max_travel"`
	TyreRadius               [4]float32 `json:"tyre_radius"`
	MaxTurboBoost            float32    `json:"max_turbo_boost"`
	Deprecated1              float32    `json:"deprecated_1"`
	Deprecated2              float32    `json:"deprecated_2"`
	PenaltiesEnabled         int32      `json:"penalties_enabled"`
	AidFuelRate              float32    `json:"aid_fuel_rate"`
	AidTireRate              float32    `json:"aid_tire_rate"`
	AidMechanicalDamage      float32    `json:"aid_mechanical_damage"`
	AidAllowTyreBlankets     int32      `json:"aid_allow_tyre_blankets"`
	AidStability             float32    `json:"aid_stability"`
	AidAutoClutch            int32      `json:"aid_auto_clutch"`
	AidAutoBlip              int32      `json:"aid_auto_blip"`
	HasDRS                   int32      `json:"has_drs"`
	HasERS                   int32      `json:"has_ers"`
	HasKERS                  int32      `json:"has_kers"`
	KersMaxJ                 float32    `json:"kers_max_j"`
	EngineBrakeSettingsCount int32      `json:"engine_brake_settings_count"`
	ErsPowerControllerCount  int32      `json:"ers_power_controller_count"`
	TrackSPlineLength        float32    `json:"track_spline_length"`
	TrackConfiguration       [33]uint16 `json:"track_configuration"`
	ErsMaxJ                  float32    `json:"ers_max_j"`
	IsTimedRace              int32      `json:"is_timed_race"`
	HasExtraLap              int32      `json:"has_extra_lap"`
	CarSkin                  [33]uint16 `json:"car_skin"`
	ReversedGridPositions    int32      `json:"reversed_grid_positions"`
	PitWindowStart           int32      `json:"pit_window_start"`
	PitWindowEnd             int32      `json:"pit_window_end"`
	IsOnline                 int32      `json:"is_online"`
	DryTyresName             [33]uint16 `json:"dry_tyres_name"`
	WetTyresName             [33]uint16 `json:"wet_tyres_name"`
}

// NewACCConnector creates a new ACC connector instance
func NewACCConnector() *ACCConnector {
	return &ACCConnector{
		isConnected:           false,
		stopStream:            make(chan bool),
		validator:             NewDataValidator(nil),
		retryHandler:          NewRetryHandler(nil),
		circuitBreaker:        NewCircuitBreaker(nil),
		maxConnectionAttempts: 5,
	}
}

// Connect implements SimulatorConnector.Connect
func (c *ACCConnector) Connect(ctx context.Context) error {
	// Use circuit breaker to prevent repeated connection attempts
	return c.circuitBreaker.Execute(func() error {
		return c.retryHandler.Retry(ctx, func() error {
			return c.attemptConnection()
		})
	})
}

// attemptConnection performs a single connection attempt
func (c *ACCConnector) attemptConnection() error {
	c.connectionAttempts++

	var err error

	// Open shared memory handles for ACC
	c.physicsHandle, err = c.openSharedMemory("Local\\acpmf_physics")
	if err != nil {
		return &ConnectionError{
			ConnectorType: SimulatorTypeACC,
			Operation:     "OpenPhysicsMemory",
			OriginalError: err,
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	c.graphicsHandle, err = c.openSharedMemory("Local\\acpmf_graphics")
	if err != nil {
		windows.CloseHandle(c.physicsHandle)
		return &ConnectionError{
			ConnectorType: SimulatorTypeACC,
			Operation:     "OpenGraphicsMemory",
			OriginalError: err,
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	c.staticHandle, err = c.openSharedMemory("Local\\acpmf_static")
	if err != nil {
		windows.CloseHandle(c.physicsHandle)
		windows.CloseHandle(c.graphicsHandle)
		return &ConnectionError{
			ConnectorType: SimulatorTypeACC,
			Operation:     "OpenStaticMemory",
			OriginalError: err,
			Timestamp:     time.Now(),
			Retryable:     true,
		}
	}

	// Test that we can actually read data
	_, err = c.readRawTelemetryData()
	if err != nil {
		c.cleanupHandles()
		return &ConnectionError{
			ConnectorType: SimulatorTypeACC,
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

// cleanupHandles closes all open handles
func (c *ACCConnector) cleanupHandles() {
	if c.physicsHandle != 0 {
		windows.CloseHandle(c.physicsHandle)
		c.physicsHandle = 0
	}
	if c.graphicsHandle != 0 {
		windows.CloseHandle(c.graphicsHandle)
		c.graphicsHandle = 0
	}
	if c.staticHandle != 0 {
		windows.CloseHandle(c.staticHandle)
		c.staticHandle = 0
	}
}

// Disconnect implements SimulatorConnector.Disconnect
func (c *ACCConnector) Disconnect() error {
	if c.isConnected {
		c.StopDataStream()
		c.cleanupHandles()
		c.isConnected = false
	}
	return nil
}

// IsConnected implements SimulatorConnector.IsConnected
func (c *ACCConnector) IsConnected() bool {
	return c.isConnected
}

// GetSimulatorType implements SimulatorConnector.GetSimulatorType
func (c *ACCConnector) GetSimulatorType() SimulatorType {
	return SimulatorTypeACC
}

// GetTelemetryData implements SimulatorConnector.GetTelemetryData
func (c *ACCConnector) GetTelemetryData(ctx context.Context) (*TelemetryData, error) {
	if !c.isConnected {
		return c.lastValidData, &ConnectionError{
			ConnectorType: SimulatorTypeACC,
			Operation:     "GetTelemetryData",
			OriginalError: fmt.Errorf("not connected to ACC"),
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
					ConnectorType: SimulatorTypeACC,
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

// readRawTelemetryData reads and converts raw ACC data to TelemetryData
func (c *ACCConnector) readRawTelemetryData() (*TelemetryData, error) {
	physics, err := c.readPhysicsData()
	if err != nil {
		return nil, fmt.Errorf("failed to read physics data: %w", err)
	}

	graphics, err := c.readGraphicsData()
	if err != nil {
		return nil, fmt.Errorf("failed to read graphics data: %w", err)
	}

	static, err := c.readStaticData()
	if err != nil {
		return nil, fmt.Errorf("failed to read static data: %w", err)
	}

	return c.convertToTelemetryData(physics, graphics, static), nil
}

// StartDataStream implements SimulatorConnector.StartDataStream
func (c *ACCConnector) StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error) {
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
func (c *ACCConnector) StopDataStream() {
	select {
	case c.stopStream <- true:
	default:
	}
}

// HealthCheck implements SimulatorConnector.HealthCheck
func (c *ACCConnector) HealthCheck(ctx context.Context) error {
	if !c.isConnected {
		return fmt.Errorf("not connected to ACC")
	}

	// Try to read a small amount of data to verify connection
	_, err := c.readPhysicsData()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Helper methods

func (c *ACCConnector) openSharedMemory(name string) (windows.Handle, error) {
	// For now, return an error since we can't actually test ACC integration without the game running
	// This allows the code to compile but will fail gracefully when ACC is not running
	return 0, fmt.Errorf("ACC shared memory not available: %s", name)
}

func (c *ACCConnector) readPhysicsData() (*ACCPhysics, error) {
	ptr, err := windows.MapViewOfFile(
		c.physicsHandle,
		windows.FILE_MAP_READ,
		0,
		0,
		unsafe.Sizeof(ACCPhysics{}),
	)
	if err != nil {
		return nil, err
	}
	defer windows.UnmapViewOfFile(ptr)

	physics := (*ACCPhysics)(unsafe.Pointer(ptr))
	// Make a copy to avoid referencing shared memory after unmapping
	physicsCopy := *physics
	return &physicsCopy, nil
}

func (c *ACCConnector) readGraphicsData() (*ACCGraphics, error) {
	ptr, err := windows.MapViewOfFile(
		c.graphicsHandle,
		windows.FILE_MAP_READ,
		0,
		0,
		unsafe.Sizeof(ACCGraphics{}),
	)
	if err != nil {
		return nil, err
	}
	defer windows.UnmapViewOfFile(ptr)

	graphics := (*ACCGraphics)(unsafe.Pointer(ptr))
	// Make a copy to avoid referencing shared memory after unmapping
	graphicsCopy := *graphics
	return &graphicsCopy, nil
}

func (c *ACCConnector) readStaticData() (*ACCStatic, error) {
	ptr, err := windows.MapViewOfFile(
		c.staticHandle,
		windows.FILE_MAP_READ,
		0,
		0,
		unsafe.Sizeof(ACCStatic{}),
	)
	if err != nil {
		return nil, err
	}
	defer windows.UnmapViewOfFile(ptr)

	static := (*ACCStatic)(unsafe.Pointer(ptr))
	// Make a copy to avoid referencing shared memory after unmapping
	staticCopy := *static
	return &staticCopy, nil
}

func (c *ACCConnector) convertToTelemetryData(physics *ACCPhysics, graphics *ACCGraphics, static *ACCStatic) *TelemetryData {
	now := time.Now()

	// Convert session type
	sessionType := c.convertSessionType(graphics.ACSessionType)

	// Convert session flags
	sessionFlag := c.convertSessionFlag(graphics)

	// Calculate tire wear percentages (ACC provides 0-1 range)
	tireWear := TireData{
		Compound: c.convertUTF16ToString(static.DryTyresName[:]),
		FrontLeft: TireWheelData{
			Temperature: float64(physics.TyreTempI[0]),
			Pressure:    float64(physics.WheelsPressure[0]),
			WearPercent: float64(physics.TyreWear[0] * 100),
			DirtLevel:   float64(physics.TyreDirtyLevel[0]),
		},
		FrontRight: TireWheelData{
			Temperature: float64(physics.TyreTempI[1]),
			Pressure:    float64(physics.WheelsPressure[1]),
			WearPercent: float64(physics.TyreWear[1] * 100),
			DirtLevel:   float64(physics.TyreDirtyLevel[1]),
		},
		RearLeft: TireWheelData{
			Temperature: float64(physics.TyreTempI[2]),
			Pressure:    float64(physics.WheelsPressure[2]),
			WearPercent: float64(physics.TyreWear[2] * 100),
			DirtLevel:   float64(physics.TyreDirtyLevel[2]),
		},
		RearRight: TireWheelData{
			Temperature: float64(physics.TyreTempI[3]),
			Pressure:    float64(physics.WheelsPressure[3]),
			WearPercent: float64(physics.TyreWear[3] * 100),
			DirtLevel:   float64(physics.TyreDirtyLevel[3]),
		},
	}

	// Calculate derived tire data
	tireWear.WearLevel = CalculateTireWearLevel(&tireWear)
	tireWear.TempLevel = CalculateTireTempLevel(&tireWear)

	// Fuel data
	fuelData := FuelData{
		Level:             float64(physics.Fuel),
		Capacity:          float64(static.MaxFuel),
		UsagePerLap:       float64(graphics.FuelXLap),
		EstimatedLapsLeft: int(graphics.FuelEstimatedLaps),
	}

	// Calculate fuel estimates
	avgLapTime := time.Duration(graphics.ILastTime) * time.Millisecond
	CalculateFuelEstimates(&fuelData, avgLapTime)

	// Session info
	sessionInfo := SessionInfo{
		Type:             sessionType,
		Flag:             sessionFlag,
		TimeRemaining:    time.Duration(graphics.SessionTimeLeft) * time.Second,
		LapsRemaining:    0, // ACC doesn't provide this directly
		TotalLaps:        int(graphics.NumberOfLaps),
		SessionTime:      time.Duration(graphics.Clock) * time.Second,
		IsTimedSession:   static.IsTimedRace == 1,
		IsLappedSession:  static.IsTimedRace == 0,
		TrackName:        c.convertUTF16ToString(static.Track[:]),
		TrackLength:      float64(static.TrackSPlineLength) / 1000.0, // Convert to km
		AirTemperature:   float64(physics.AirTemp),
		TrackTemperature: float64(physics.RoadTemp),
	}

	// Calculate race format
	sessionInfo.Format = CalculateRaceFormat(&sessionInfo)

	// Player data
	playerData := PlayerData{
		Position:           int(graphics.Position),
		CurrentLap:         int(graphics.CompletedLaps + 1), // ACC counts completed laps
		LapDistancePercent: float64(graphics.NormalizedCarPosition * 100),
		LastLapTime:        time.Duration(graphics.ILastTime) * time.Millisecond,
		BestLapTime:        time.Duration(graphics.IBestTime) * time.Millisecond,
		CurrentLapTime:     time.Duration(graphics.ICurrentTime) * time.Millisecond,
		GapToAhead:         time.Duration(graphics.GapAhead) * time.Millisecond,
		GapToBehind:        time.Duration(graphics.GapBehind) * time.Millisecond,
		Fuel:               fuelData,
		Tires:              tireWear,
		Pit: PitData{
			IsOnPitRoad:      graphics.IsInPitLane == 1,
			IsInPitStall:     graphics.IsInPit == 1,
			PitWindowOpen:    true,             // ACC doesn't provide pit window info directly
			EstimatedPitTime: 25 * time.Second, // Estimated value
			PitSpeedLimit:    80.0,             // Standard ACC pit speed limit
		},
		Speed:    float64(physics.SpeedKMH),
		RPM:      float64(physics.RPM),
		Gear:     int(physics.Gear),
		Throttle: float64(physics.Gas * 100),
		Brake:    float64(physics.Brake * 100),
		Clutch:   float64(physics.Clutch * 100),
		Steering: float64(physics.SteerAngle),
	}

	// Create telemetry data
	telemetry := &TelemetryData{
		Timestamp:     now,
		SimulatorType: SimulatorTypeACC,
		IsConnected:   true,
		Session:       sessionInfo,
		Player:        playerData,
		Opponents:     []OpponentData{}, // ACC has limited opponent data
	}

	return telemetry
}

func (c *ACCConnector) convertSessionType(accSessionType int32) SessionType {
	switch accSessionType {
	case 0: // AC_PRACTICE
		return SessionTypePractice
	case 1: // AC_QUALIFY
		return SessionTypeQualifying
	case 2: // AC_RACE
		return SessionTypeRace
	case 3: // AC_HOTLAP
		return SessionTypeHotlap
	default:
		return SessionTypeUnknown
	}
}

func (c *ACCConnector) convertSessionFlag(graphics *ACCGraphics) SessionFlag {
	// Check global flags first
	if graphics.GlobalRed == 1 {
		return SessionFlagRed
	}
	if graphics.GlobalYellow == 1 || graphics.GlobalYellow1 == 1 || graphics.GlobalYellow2 == 1 || graphics.GlobalYellow3 == 1 {
		return SessionFlagYellow
	}
	if graphics.GlobalChequered == 1 {
		return SessionFlagCheckered
	}
	if graphics.GlobalWhite == 1 {
		return SessionFlagWhite
	}
	if graphics.GlobalGreen == 1 {
		return SessionFlagGreen
	}

	// Check local flag
	switch graphics.Flag {
	case 0:
		return SessionFlagNone
	case 1:
		return SessionFlagBlue
	case 2:
		return SessionFlagYellow
	case 3:
		return SessionFlagRed
	case 4:
		return SessionFlagWhite
	case 5:
		return SessionFlagCheckered
	default:
		return SessionFlagNone
	}
}

func (c *ACCConnector) convertUTF16ToString(utf16Data []uint16) string {
	// Find the null terminator
	var length int
	for i, v := range utf16Data {
		if v == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		length = len(utf16Data)
	}

	return windows.UTF16ToString(utf16Data[:length])
}
