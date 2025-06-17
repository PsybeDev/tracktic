package strategy

import (
	"math"
	"time"

	"changeme/sims"
)

// PitStopCalculator provides advanced pit stop timing and track position calculations
type PitStopCalculator struct {
	config          *Config
	trackDatabase   *TrackDatabase
	positionTracker *PositionTracker
	timingAnalyzer  *TimingAnalyzer
}

// TrackDatabase contains track-specific data for pit stop calculations
type TrackDatabase struct {
	tracks map[string]*TrackData
}

// TrackData contains track-specific information for calculations
type TrackData struct {
	Name             string           `json:"name"`
	Length           float64          `json:"length_km"`
	PitLaneLength    float64          `json:"pit_lane_length_km"`
	PitSpeedLimit    float64          `json:"pit_speed_limit_kmh"`
	PitEntry         float64          `json:"pit_entry_position"` // Position on track (0-1)
	PitExit          float64          `json:"pit_exit_position"`  // Position on track (0-1)
	TypicalPitTime   time.Duration    `json:"typical_pit_time"`   // Stationary time in pit
	PitLaneDelta     time.Duration    `json:"pit_lane_delta"`     // Time difference vs. racing line
	SafetyCarSectors []int            `json:"safety_car_sectors"` // Sectors where SC is likely
	DRSZones         []DRSZone        `json:"drs_zones"`          // DRS activation zones
	OvertakingZones  []OvertakingZone `json:"overtaking_zones"`   // Key overtaking opportunities
}

// DRSZone represents a DRS activation zone on track
type DRSZone struct {
	StartPosition float64 `json:"start_position"` // Track position 0-1
	EndPosition   float64 `json:"end_position"`   // Track position 0-1
	Name          string  `json:"name"`           // Zone name/description
}

// OvertakingZone represents areas where overtaking is common
type OvertakingZone struct {
	StartPosition float64 `json:"start_position"` // Track position 0-1
	EndPosition   float64 `json:"end_position"`   // Track position 0-1
	Name          string  `json:"name"`           // Zone name
	Difficulty    string  `json:"difficulty"`     // "easy", "medium", "hard"
}

// PositionTracker tracks and predicts track positions
type PositionTracker struct {
	playerHistory   []PositionSnapshot
	opponentHistory map[int][]PositionSnapshot // Indexed by car number/ID
	trackLength     float64
}

// PositionSnapshot captures position data at a specific time
type PositionSnapshot struct {
	Timestamp        time.Time
	TrackPosition    float64       // Position on track (0-1)
	Speed            float64       // Speed in km/h
	LapTime          time.Duration // Current/last lap time
	EstimatedLapTime time.Duration // Predicted lap time
	IsInPitLane      bool          // Currently in pit lane
	JustExitedPits   bool          // Just exited pits this lap
}

// TimingAnalyzer provides sophisticated timing calculations
type TimingAnalyzer struct {
	sectorTimes      map[int][]time.Duration // Sector times by sector number
	lapTimePatterns  []LapTimePattern
	degradationModel *TireDegradationModel
}

// LapTimePattern represents lap time progression patterns
type LapTimePattern struct {
	BaseLapTime     time.Duration
	TireDegradation time.Duration // Per lap degradation
	FuelEffect      time.Duration // Per lap fuel effect
	TrafficImpact   time.Duration // Average traffic delay
	WeatherImpact   time.Duration // Weather-based variation
}

// TireDegradationModel models tire performance degradation
type TireDegradationModel struct {
	Compound          string
	OptimalWindow     [2]int        // Lap range for optimal performance
	LinearDegradation time.Duration // Per lap time loss
	CliffEffect       struct {
		ThresholdLap int           // When cliff effect starts
		CliffPenalty time.Duration // Additional time loss per lap after cliff
	}
}

// PitStopAnalysis contains comprehensive pit stop timing analysis
type PitStopAnalysis struct {
	// Timing Analysis
	OptimalWindows     []PitWindow      `json:"optimal_windows"`
	CurrentPosition    TrackPosition    `json:"current_position"`
	EstimatedPositions []FuturePosition `json:"estimated_positions"`

	// Strategic Analysis
	UnderCutAnalysis UnderCutAnalysis `json:"undercut_analysis"`
	OverCutAnalysis  OverCutAnalysis  `json:"overcut_analysis"`
	TrafficAnalysis  TrafficAnalysis  `json:"traffic_analysis"`

	// Risk Assessment
	RiskFactors        []PitRiskFactor     `json:"risk_factors"`
	OpportunityWindows []OpportunityWindow `json:"opportunity_windows"`

	// Detailed Calculations
	PitLossCalculation PitLossCalculation `json:"pit_loss_calculation"`
	PositionChanges    []PositionChange   `json:"position_changes"`

	// Recommendations
	PrimaryRecommendation PitRecommendation `json:"primary_recommendation"`
	AlternativeOptions    []PitAlternative  `json:"alternative_options"`

	// Meta Information
	CalculationConfidence float64   `json:"calculation_confidence"`
	DataQuality           float64   `json:"data_quality"`
	LastUpdated           time.Time `json:"last_updated"`
}

// PitWindow represents an optimal pit stop window
type PitWindow struct {
	StartLap     int           `json:"start_lap"`
	EndLap       int           `json:"end_lap"`
	OptimalLap   int           `json:"optimal_lap"`
	WindowType   string        `json:"window_type"` // "strategic", "forced", "opportunity"
	Confidence   float64       `json:"confidence"`
	ExpectedGain time.Duration `json:"expected_gain"` // Net time gain/loss
	RiskLevel    string        `json:"risk_level"`    // "low", "medium", "high"
	Rationale    string        `json:"rationale"`     // Why this window exists
}

// TrackPosition represents detailed track position information
type TrackPosition struct {
	LapDistance      float64       `json:"lap_distance_percent"`
	EstimatedSpeed   float64       `json:"estimated_speed_kmh"`
	NextSector       int           `json:"next_sector"`
	DistanceToFinish float64       `json:"distance_to_finish_km"`
	TimeToFinish     time.Duration `json:"time_to_finish"`
}

// FuturePosition predicts future track positions
type FuturePosition struct {
	Lap                int           `json:"lap"`
	Position           int           `json:"position"`
	TrackPosition      float64       `json:"track_position"`
	EstimatedTime      time.Duration `json:"estimated_time"`
	Confidence         float64       `json:"confidence"`
	InfluencingFactors []string      `json:"influencing_factors"`
}

// UnderCutAnalysis analyzes undercut opportunities and threats
type UnderCutAnalysis struct {
	ThreatLevel     string           `json:"threat_level"`
	ThreateningCars []UnderCutThreat `json:"threatening_cars"`
	DefenseOptions  []DefenseOption  `json:"defense_options"`
	OptimalResponse string           `json:"optimal_response"`
}

// UnderCutThreat represents a specific undercut threat
type UnderCutThreat struct {
	CarPosition       int           `json:"car_position"`
	DriverName        string        `json:"driver_name"`
	GapBehind         time.Duration `json:"gap_behind"`
	TireAge           int           `json:"tire_age"`
	EstimatedGain     time.Duration `json:"estimated_gain"` // Potential gain from undercut
	ThreatProbability float64       `json:"threat_probability"`
}

// DefenseOption represents ways to defend against undercuts
type DefenseOption struct {
	Strategy        string   `json:"strategy"`
	Description     string   `json:"description"`
	Effectiveness   float64  `json:"effectiveness"` // 0-1 scale
	RiskLevel       string   `json:"risk_level"`
	RequiredActions []string `json:"required_actions"`
}

// OverCutAnalysis analyzes overcut opportunities
type OverCutAnalysis struct {
	OpportunityLevel string          `json:"opportunity_level"`
	TargetCars       []OverCutTarget `json:"target_cars"`
	RequiredStint    int             `json:"required_stint_extension"`
	ExpectedGain     time.Duration   `json:"expected_gain"`
	RiskFactors      []string        `json:"risk_factors"`
}

// OverCutTarget represents a car that could be overcutted
type OverCutTarget struct {
	CarPosition        int           `json:"car_position"`
	DriverName         string        `json:"driver_name"`
	GapAhead           time.Duration `json:"gap_ahead"`
	TireAge            int           `json:"tire_age"`
	DegradationRate    float64       `json:"degradation_rate"`
	EstimatedGain      time.Duration `json:"estimated_gain"`
	SuccessProbability float64       `json:"success_probability"`
}

// TrafficAnalysis analyzes traffic impact on pit stops
type TrafficAnalysis struct {
	TrafficDensity  float64          `json:"traffic_density"`  // Cars per sector
	ClearTrackLaps  []int            `json:"clear_track_laps"` // Laps with minimal traffic
	BackmarkerRisk  string           `json:"backmarker_risk"`  // Risk of getting stuck
	TrafficPatterns []TrafficPattern `json:"traffic_patterns"` // Predicted traffic flow
}

// TrafficPattern represents predicted traffic situations
type TrafficPattern struct {
	Lap               int           `json:"lap"`
	Sector            int           `json:"sector"`
	TrafficLevel      string        `json:"traffic_level"` // "clear", "light", "heavy"
	EstimatedDelay    time.Duration `json:"estimated_delay"`
	AffectedPositions []int         `json:"affected_positions"`
}

// PitRiskFactor represents risks associated with pit stops
type PitRiskFactor struct {
	RiskType    string  `json:"risk_type"`
	Severity    string  `json:"severity"`    // "low", "medium", "high", "critical"
	Probability float64 `json:"probability"` // 0-1
	Impact      string  `json:"impact"`      // Description of potential impact
	Mitigation  string  `json:"mitigation"`  // How to mitigate the risk
	TimeWindow  string  `json:"time_window"` // When this risk applies
}

// OpportunityWindow represents strategic opportunities for pit stops
type OpportunityWindow struct {
	WindowType      string        `json:"window_type"` // "safety_car", "vsc", "weather", "strategic"
	StartLap        int           `json:"start_lap"`
	EndLap          int           `json:"end_lap"`
	Probability     float64       `json:"probability"`      // Likelihood of opportunity
	PotentialGain   time.Duration `json:"potential_gain"`   // Possible time/position gain
	RequiredActions []string      `json:"required_actions"` // Actions needed to capitalize
}

// PitLossCalculation provides detailed breakdown of pit stop time costs
type PitLossCalculation struct {
	PitLaneEntry   time.Duration `json:"pit_lane_entry"`  // Time to enter pit lane
	PitLaneTravel  time.Duration `json:"pit_lane_travel"` // Travel time through pit lane
	StationaryTime time.Duration `json:"stationary_time"` // Time stopped for service
	PitLaneExit    time.Duration `json:"pit_lane_exit"`   // Time to exit and rejoin
	TotalPitTime   time.Duration `json:"total_pit_time"`  // Total time in pit lane
	TrackPosition  time.Duration `json:"track_position"`  // Time others gain on track
	NetTimeLoss    time.Duration `json:"net_time_loss"`   // Total net time lost
	PositionsLost  int           `json:"positions_lost"`  // Expected positions lost
	RecoveryLaps   int           `json:"recovery_laps"`   // Laps to recover positions
}

// PositionChange represents predicted position changes from pit stops
type PositionChange struct {
	FromPosition int           `json:"from_position"`
	ToPosition   int           `json:"to_position"`
	NetChange    int           `json:"net_change"` // Positive = gain, negative = loss
	Probability  float64       `json:"probability"`
	RecoveryTime time.Duration `json:"recovery_time"` // Time to recover lost positions
	DependsOn    []string      `json:"depends_on"`    // Factors this change depends on
}

// NewPitStopCalculator creates a new pit stop calculator instance
func NewPitStopCalculator(config *Config) *PitStopCalculator {
	calculator := &PitStopCalculator{
		config:          config,
		trackDatabase:   NewTrackDatabase(),
		positionTracker: NewPositionTracker(),
		timingAnalyzer:  NewTimingAnalyzer(),
	}

	return calculator
}

// NewTrackDatabase creates a track database with predefined track data
func NewTrackDatabase() *TrackDatabase {
	db := &TrackDatabase{
		tracks: make(map[string]*TrackData),
	}

	// Initialize with common tracks
	db.loadDefaultTracks()

	return db
}

// loadDefaultTracks loads predefined track data for major circuits
func (tdb *TrackDatabase) loadDefaultTracks() {
	// Spa-Francorchamps
	tdb.tracks["Spa-Francorchamps"] = &TrackData{
		Name:             "Spa-Francorchamps",
		Length:           7.004,
		PitLaneLength:    0.42,
		PitSpeedLimit:    60.0,
		PitEntry:         0.92, // After Blanchimont
		PitExit:          0.05, // Before Eau Rouge
		TypicalPitTime:   time.Second * 24,
		PitLaneDelta:     time.Second * 22,
		SafetyCarSectors: []int{2, 3}, // Sector 2 (Eau Rouge/Raidillon) and 3 (Bus Stop)
		DRSZones: []DRSZone{
			{StartPosition: 0.88, EndPosition: 0.05, Name: "Kemmel Straight"},
		},
		OvertakingZones: []OvertakingZone{
			{StartPosition: 0.02, EndPosition: 0.08, Name: "Eau Rouge/Raidillon", Difficulty: "hard"},
			{StartPosition: 0.88, EndPosition: 0.95, Name: "Kemmel Straight", Difficulty: "easy"},
			{StartPosition: 0.75, EndPosition: 0.82, Name: "Bus Stop Chicane", Difficulty: "medium"},
		},
	}

	// Silverstone
	tdb.tracks["Silverstone"] = &TrackData{
		Name:             "Silverstone",
		Length:           5.891,
		PitLaneLength:    0.38,
		PitSpeedLimit:    60.0,
		PitEntry:         0.85,
		PitExit:          0.12,
		TypicalPitTime:   time.Second * 22,
		PitLaneDelta:     time.Second * 20,
		SafetyCarSectors: []int{1, 3},
		DRSZones: []DRSZone{
			{StartPosition: 0.77, EndPosition: 0.12, Name: "Wellington Straight"},
			{StartPosition: 0.48, EndPosition: 0.58, Name: "Hangar Straight"},
		},
		OvertakingZones: []OvertakingZone{
			{StartPosition: 0.08, EndPosition: 0.15, Name: "Turn 3 (Village)", Difficulty: "medium"},
			{StartPosition: 0.55, EndPosition: 0.62, Name: "Stowe Corner", Difficulty: "medium"},
		},
	}

	// Monza
	tdb.tracks["Monza"] = &TrackData{
		Name:             "Monza",
		Length:           5.793,
		PitLaneLength:    0.35,
		PitSpeedLimit:    60.0,
		PitEntry:         0.92,
		PitExit:          0.05,
		TypicalPitTime:   time.Second * 21,
		PitLaneDelta:     time.Second * 19,
		SafetyCarSectors: []int{2},
		DRSZones: []DRSZone{
			{StartPosition: 0.85, EndPosition: 0.05, Name: "Main Straight"},
			{StartPosition: 0.45, EndPosition: 0.65, Name: "Back Straight"},
		},
		OvertakingZones: []OvertakingZone{
			{StartPosition: 0.02, EndPosition: 0.08, Name: "Turn 1 (Rettifilo)", Difficulty: "easy"},
			{StartPosition: 0.62, EndPosition: 0.68, Name: "Turn 4 (Roggia)", Difficulty: "medium"},
			{StartPosition: 0.82, EndPosition: 0.88, Name: "Parabolica", Difficulty: "hard"},
		},
	}
}

// GetTrackData retrieves track data for a specific track
func (tdb *TrackDatabase) GetTrackData(trackName string) *TrackData {
	if track, exists := tdb.tracks[trackName]; exists {
		return track
	}

	// Return generic track data if specific track not found
	return &TrackData{
		Name:             trackName,
		Length:           5.0, // Generic 5km track
		PitLaneLength:    0.4,
		PitSpeedLimit:    60.0,
		PitEntry:         0.90,
		PitExit:          0.10,
		TypicalPitTime:   time.Second * 25,
		PitLaneDelta:     time.Second * 23,
		SafetyCarSectors: []int{1, 2, 3},
		DRSZones:         []DRSZone{},
		OvertakingZones:  []OvertakingZone{},
	}
}

// NewPositionTracker creates a new position tracker
func NewPositionTracker() *PositionTracker {
	return &PositionTracker{
		playerHistory:   make([]PositionSnapshot, 0, 100),
		opponentHistory: make(map[int][]PositionSnapshot),
		trackLength:     5.0, // Default, will be updated
	}
}

// NewTimingAnalyzer creates a new timing analyzer
func NewTimingAnalyzer() *TimingAnalyzer {
	return &TimingAnalyzer{
		sectorTimes:     make(map[int][]time.Duration),
		lapTimePatterns: make([]LapTimePattern, 0),
	}
}

// CalculatePitStopTiming performs comprehensive pit stop timing analysis
func (psc *PitStopCalculator) CalculatePitStopTiming(data *sims.TelemetryData, raceAnalysis *RaceAnalysis) *PitStopAnalysis {
	// Get track-specific data
	trackData := psc.trackDatabase.GetTrackData(data.Session.TrackName)
	psc.positionTracker.trackLength = trackData.Length

	// Update position tracking
	psc.updatePositionTracking(data)

	// Update timing analysis
	psc.updateTimingAnalysis(data)

	analysis := &PitStopAnalysis{
		LastUpdated: time.Now(),
	}

	// Calculate current track position
	analysis.CurrentPosition = psc.calculateCurrentPosition(data, trackData)

	// Identify optimal pit windows
	analysis.OptimalWindows = psc.calculateOptimalWindows(data, trackData, raceAnalysis)

	// Predict future positions
	analysis.EstimatedPositions = psc.predictFuturePositions(data, trackData)

	// Analyze undercut threats and opportunities
	analysis.UnderCutAnalysis = psc.analyzeUnderCutScenarios(data, trackData)
	analysis.OverCutAnalysis = psc.analyzeOverCutOpportunities(data, trackData)

	// Analyze traffic impact
	analysis.TrafficAnalysis = psc.analyzeTrafficPatterns(data, trackData)

	// Calculate detailed pit loss
	analysis.PitLossCalculation = psc.calculateDetailedPitLoss(data, trackData)

	// Predict position changes
	analysis.PositionChanges = psc.predictPositionChanges(data, trackData)

	// Identify risk factors
	analysis.RiskFactors = psc.identifyRiskFactors(data, trackData, raceAnalysis)

	// Identify opportunity windows
	analysis.OpportunityWindows = psc.identifyOpportunityWindows(data, trackData, raceAnalysis)

	// Generate primary recommendation
	analysis.PrimaryRecommendation = psc.generateEnhancedPitRecommendation(analysis, data)

	// Generate alternative options
	analysis.AlternativeOptions = psc.generateAlternativeOptions(analysis, data)

	// Calculate confidence and data quality
	analysis.CalculationConfidence = psc.calculateConfidence(data)
	analysis.DataQuality = psc.assessDataQuality()

	return analysis
}

// updatePositionTracking updates the position tracking with new telemetry data
func (psc *PitStopCalculator) updatePositionTracking(data *sims.TelemetryData) {
	// Create position snapshot for player
	playerSnapshot := PositionSnapshot{
		Timestamp:        time.Now(),
		TrackPosition:    data.Player.LapDistancePercent / 100.0,
		Speed:            data.Player.Speed,
		LapTime:          data.Player.LastLapTime,
		EstimatedLapTime: data.Player.CurrentLapTime,
		IsInPitLane:      data.Player.Pit.IsOnPitRoad,
		JustExitedPits:   data.Player.Pit.IsOnPitRoad && data.Player.CurrentLap > data.Player.Pit.LastPitLap,
	}

	// Add to player history
	psc.positionTracker.playerHistory = append(psc.positionTracker.playerHistory, playerSnapshot)

	// Keep only last 100 snapshots
	if len(psc.positionTracker.playerHistory) > 100 {
		psc.positionTracker.playerHistory = psc.positionTracker.playerHistory[1:]
	}

	// Update opponent tracking
	for _, opponent := range data.Opponents {
		if psc.positionTracker.opponentHistory[opponent.CarIndex] == nil {
			psc.positionTracker.opponentHistory[opponent.CarIndex] = make([]PositionSnapshot, 0, 50)
		}

		opponentSnapshot := PositionSnapshot{
			Timestamp:      time.Now(),
			TrackPosition:  opponent.LapDistancePercent / 100.0,
			LapTime:        opponent.LastLapTime,
			IsInPitLane:    opponent.IsOnPitRoad,
			JustExitedPits: opponent.IsOnPitRoad && opponent.CurrentLap > opponent.LastPitLap,
		}

		history := psc.positionTracker.opponentHistory[opponent.CarIndex]
		history = append(history, opponentSnapshot)

		// Keep only last 50 snapshots per opponent
		if len(history) > 50 {
			history = history[1:]
		}

		psc.positionTracker.opponentHistory[opponent.CarIndex] = history
	}
}

// updateTimingAnalysis updates timing analysis with new data
func (psc *PitStopCalculator) updateTimingAnalysis(data *sims.TelemetryData) {
	// Update lap time patterns
	if data.Player.LastLapTime > 0 {
		pattern := LapTimePattern{
			BaseLapTime:     data.Player.BestLapTime,
			TireDegradation: psc.calculateCurrentDegradation(data),
			FuelEffect:      psc.calculateFuelEffect(data),
			TrafficImpact:   psc.estimateTrafficImpact(data),
		}

		psc.timingAnalyzer.lapTimePatterns = append(psc.timingAnalyzer.lapTimePatterns, pattern)

		// Keep only recent patterns
		if len(psc.timingAnalyzer.lapTimePatterns) > 20 {
			psc.timingAnalyzer.lapTimePatterns = psc.timingAnalyzer.lapTimePatterns[1:]
		}
	}
}

// calculateCurrentPosition determines current detailed track position
func (psc *PitStopCalculator) calculateCurrentPosition(data *sims.TelemetryData, trackData *TrackData) TrackPosition {
	position := TrackPosition{
		LapDistance:    data.Player.LapDistancePercent,
		EstimatedSpeed: data.Player.Speed,
		NextSector:     psc.calculateNextSector(data.Player.LapDistancePercent),
	}

	// Calculate distance and time to finish
	if data.Session.TotalLaps > 0 {
		lapsRemaining := float64(data.Session.TotalLaps - data.Player.CurrentLap)
		distanceRemaining := (1.0 - data.Player.LapDistancePercent/100.0) * trackData.Length
		position.DistanceToFinish = lapsRemaining*trackData.Length + distanceRemaining

		// Estimate time to finish based on average lap time
		if data.Player.LastLapTime > 0 {
			avgLapTime := data.Player.LastLapTime
			timeCurrentLap := time.Duration(float64(avgLapTime) * (1.0 - data.Player.LapDistancePercent/100.0))
			timeRemainingLaps := time.Duration(lapsRemaining) * avgLapTime
			position.TimeToFinish = timeCurrentLap + timeRemainingLaps
		}
	}

	return position
}

// calculateNextSector determines which sector the car will enter next
func (psc *PitStopCalculator) calculateNextSector(lapDistance float64) int {
	// Assuming 3 sectors, each 33.33% of track
	if lapDistance < 33.33 {
		return 1
	} else if lapDistance < 66.66 {
		return 2
	} else {
		return 3
	}
}

// Additional methods for the calculator will be implemented here...
// Due to space constraints, I'll implement the key methods and provide framework for others

// calculateCurrentDegradation estimates current tire degradation rate
func (psc *PitStopCalculator) calculateCurrentDegradation(data *sims.TelemetryData) time.Duration {
	currentWear := CalculateAverageWear(data.Player.Tires)
	stintLap := data.Player.CurrentLap - data.Player.Pit.LastPitLap
	if stintLap <= 0 {
		stintLap = data.Player.CurrentLap
	}

	// Simplified degradation calculation: 0.1s per 10% wear
	degradationSeconds := (currentWear / 10.0) * 0.1
	return time.Duration(degradationSeconds * float64(time.Second))
}

// calculateFuelEffect estimates lap time impact of current fuel load
func (psc *PitStopCalculator) calculateFuelEffect(data *sims.TelemetryData) time.Duration {
	// Simplified: assume 0.03s per liter of fuel
	fuelEffect := data.Player.Fuel.Level * 0.03
	return time.Duration(fuelEffect * float64(time.Second))
}

// estimateTrafficImpact estimates current traffic impact on lap times
func (psc *PitStopCalculator) estimateTrafficImpact(data *sims.TelemetryData) time.Duration {
	// Count nearby opponents
	nearbyCount := 0
	for _, opponent := range data.Opponents {
		gap := opponent.GapToPlayer
		if gap < 0 {
			gap = -gap
		}
		if gap < time.Second*5 {
			nearbyCount++
		}
	}

	// Estimate 0.5s per nearby car
	trafficDelay := float64(nearbyCount) * 0.5
	return time.Duration(trafficDelay * float64(time.Second))
}

// calculateOptimalWindows identifies optimal pit stop timing windows
func (psc *PitStopCalculator) calculateOptimalWindows(data *sims.TelemetryData, trackData *TrackData, raceAnalysis *RaceAnalysis) []PitWindow {
	windows := make([]PitWindow, 0)

	currentLap := data.Player.CurrentLap
	totalLaps := data.Session.TotalLaps

	if totalLaps <= 0 {
		return windows // Can't calculate windows for time-based sessions yet
	}

	// Strategic window based on tire degradation
	currentWear := CalculateAverageWear(data.Player.Tires)
	if currentWear > 40 {
		strategicWindow := PitWindow{
			StartLap:     currentLap + 1,
			EndLap:       currentLap + 8,
			OptimalLap:   currentLap + 3,
			WindowType:   "strategic",
			Confidence:   0.8,
			ExpectedGain: -trackData.PitLaneDelta,
			RiskLevel:    "medium",
			Rationale:    "Tire degradation reaching optimal pit window",
		}
		windows = append(windows, strategicWindow)
	}

	// Forced window based on fuel
	if data.Player.Fuel.EstimatedLapsLeft < 8 {
		forcedWindow := PitWindow{
			StartLap:     currentLap + 1,
			EndLap:       currentLap + data.Player.Fuel.EstimatedLapsLeft - 1,
			OptimalLap:   currentLap + data.Player.Fuel.EstimatedLapsLeft - 2,
			WindowType:   "forced",
			Confidence:   0.95,
			ExpectedGain: -trackData.PitLaneDelta,
			RiskLevel:    "high",
			Rationale:    "Fuel level requires pit stop",
		}
		windows = append(windows, forcedWindow)
	}

	// Opportunity window for end-of-race strategy
	if float64(totalLaps-currentLap) < float64(totalLaps)*0.3 { // Last 30% of race
		opportunityWindow := PitWindow{
			StartLap:     currentLap + 1,
			EndLap:       totalLaps - 3,
			OptimalLap:   totalLaps - 8,
			WindowType:   "opportunity",
			Confidence:   0.6,
			ExpectedGain: time.Second * 5, // Potential gain from fresh tires
			RiskLevel:    "medium",
			Rationale:    "Late-race fresh tire advantage",
		}
		windows = append(windows, opportunityWindow)
	}

	return windows
}

// predictFuturePositions predicts track positions for upcoming laps
func (psc *PitStopCalculator) predictFuturePositions(data *sims.TelemetryData, trackData *TrackData) []FuturePosition {
	positions := make([]FuturePosition, 0)

	currentPosition := data.Player.Position

	// Predict next 5 laps
	for lap := 1; lap <= 5; lap++ {
		futureLap := data.Player.CurrentLap + lap

		// Basic position prediction based on current trends
		predictedPosition := currentPosition
		confidence := 0.8 - float64(lap)*0.1 // Confidence decreases with distance

		factors := []string{"Current pace", "Tire degradation"}

		// Adjust for pit stops
		if psc.shouldPitInLap(futureLap, data) {
			predictedPosition += 2 // Lose ~2 positions from pit stop
			factors = append(factors, "Pit stop impact")
		}

		position := FuturePosition{
			Lap:                futureLap,
			Position:           predictedPosition,
			TrackPosition:      0.5, // Mid-lap estimate
			EstimatedTime:      time.Duration(lap) * data.Player.LastLapTime,
			Confidence:         confidence,
			InfluencingFactors: factors,
		}

		positions = append(positions, position)
	}

	return positions
}

// shouldPitInLap determines if a pit stop is likely in a given lap
func (psc *PitStopCalculator) shouldPitInLap(lap int, data *sims.TelemetryData) bool {
	currentWear := CalculateAverageWear(data.Player.Tires)
	stintLength := lap - data.Player.Pit.LastPitLap

	// Pit if wear will be > 80% or stint > 25 laps
	return currentWear > 80 || stintLength > 25
}

// analyzeUnderCutScenarios analyzes undercut threats and opportunities
func (psc *PitStopCalculator) analyzeUnderCutScenarios(data *sims.TelemetryData, trackData *TrackData) UnderCutAnalysis {
	analysis := UnderCutAnalysis{
		ThreatLevel:     "low",
		ThreateningCars: make([]UnderCutThreat, 0),
		DefenseOptions:  make([]DefenseOption, 0),
		OptimalResponse: "monitor",
	}

	// Check for threatening cars behind
	for _, opponent := range data.Opponents {
		if opponent.Position > data.Player.Position && opponent.GapToPlayer > -time.Second*25 {
			threat := UnderCutThreat{
				CarPosition:       opponent.Position,
				DriverName:        opponent.DriverName,
				GapBehind:         -opponent.GapToPlayer,
				TireAge:           data.Player.CurrentLap - opponent.LastPitLap,
				EstimatedGain:     time.Second * 8, // Typical undercut gain
				ThreatProbability: 0.7,
			}

			analysis.ThreateningCars = append(analysis.ThreateningCars, threat)
		}
	}

	if len(analysis.ThreateningCars) > 0 {
		analysis.ThreatLevel = "medium"
		analysis.OptimalResponse = "defensive_pit"

		defense := DefenseOption{
			Strategy:        "Defensive pit stop",
			Description:     "Pit immediately to cover undercut threat",
			Effectiveness:   0.8,
			RiskLevel:       "medium",
			RequiredActions: []string{"Pit next lap", "Optimize pit stop time"},
		}
		analysis.DefenseOptions = append(analysis.DefenseOptions, defense)
	}

	return analysis
}

// analyzeOverCutOpportunities analyzes overcut strategic opportunities
func (psc *PitStopCalculator) analyzeOverCutOpportunities(data *sims.TelemetryData, trackData *TrackData) OverCutAnalysis {
	analysis := OverCutAnalysis{
		OpportunityLevel: "low",
		TargetCars:       make([]OverCutTarget, 0),
		RequiredStint:    0,
		ExpectedGain:     0,
		RiskFactors:      make([]string, 0),
	}

	// Check for cars ahead that might pit soon
	for _, opponent := range data.Opponents {
		if opponent.Position < data.Player.Position && opponent.GapToPlayer < time.Second*20 {
			opponentStintLength := data.Player.CurrentLap - opponent.LastPitLap
			if opponentStintLength > 15 { // They might pit soon
				target := OverCutTarget{
					CarPosition:        opponent.Position,
					DriverName:         opponent.DriverName,
					GapAhead:           opponent.GapToPlayer,
					TireAge:            opponentStintLength,
					DegradationRate:    0.1, // Estimated
					EstimatedGain:      time.Second * 6,
					SuccessProbability: 0.6,
				}

				analysis.TargetCars = append(analysis.TargetCars, target)
			}
		}
	}

	if len(analysis.TargetCars) > 0 {
		analysis.OpportunityLevel = "medium"
		analysis.RequiredStint = 8 // Stay out 8 more laps
		analysis.ExpectedGain = time.Second * 6
		analysis.RiskFactors = []string{"Tire degradation", "Fuel consumption"}
	}

	return analysis
}

// analyzeTrafficPatterns analyzes traffic impact on pit timing
func (psc *PitStopCalculator) analyzeTrafficPatterns(data *sims.TelemetryData, trackData *TrackData) TrafficAnalysis {
	analysis := TrafficAnalysis{
		TrafficDensity:  float64(len(data.Opponents)) / 3.0, // Cars per sector
		ClearTrackLaps:  make([]int, 0),
		BackmarkerRisk:  "low",
		TrafficPatterns: make([]TrafficPattern, 0),
	}

	// Identify laps with minimal traffic
	currentLap := data.Player.CurrentLap
	for lap := currentLap + 1; lap <= currentLap+10; lap++ {
		if psc.isLapClear(lap, data) {
			analysis.ClearTrackLaps = append(analysis.ClearTrackLaps, lap)
		}
	}

	// Assess backmarker risk
	backmarkerCount := 0
	for _, opponent := range data.Opponents {
		if opponent.Position > data.Player.Position+5 {
			backmarkerCount++
		}
	}

	if backmarkerCount > 3 {
		analysis.BackmarkerRisk = "medium"
	}

	return analysis
}

// isLapClear checks if a lap will have minimal traffic
func (psc *PitStopCalculator) isLapClear(lap int, data *sims.TelemetryData) bool {
	// Simplified: assume laps with gaps > 30s ahead/behind are clear
	gapAhead := data.Player.GapToAhead
	gapBehind := data.Player.GapToBehind

	return gapAhead > time.Second*30 || gapBehind > time.Second*30
}

// calculateDetailedPitLoss calculates comprehensive pit stop time loss
func (psc *PitStopCalculator) calculateDetailedPitLoss(data *sims.TelemetryData, trackData *TrackData) PitLossCalculation {
	calc := PitLossCalculation{}

	// Calculate pit lane times based on track data
	pitLaneSpeed := trackData.PitSpeedLimit / 3.6     // Convert to m/s
	pitLaneDistance := trackData.PitLaneLength * 1000 // Convert to meters

	calc.PitLaneEntry = time.Second * 3
	calc.PitLaneTravel = time.Duration(float64(time.Second) * pitLaneDistance / pitLaneSpeed)
	calc.StationaryTime = trackData.TypicalPitTime
	calc.PitLaneExit = time.Second * 4

	calc.TotalPitTime = calc.PitLaneEntry + calc.PitLaneTravel + calc.StationaryTime + calc.PitLaneExit

	// Calculate time lost on track
	if data.Player.LastLapTime > 0 {
		raceSpeed := trackData.Length * 1000 / float64(data.Player.LastLapTime.Seconds()) // m/s
		trackTime := time.Duration(float64(time.Second) * pitLaneDistance / raceSpeed)
		calc.TrackPosition = trackTime
	}

	calc.NetTimeLoss = calc.TotalPitTime - calc.TrackPosition

	// Estimate positions lost (rough calculation)
	if data.Player.LastLapTime > 0 {
		lapsEquivalent := float64(calc.NetTimeLoss) / float64(data.Player.LastLapTime)
		calc.PositionsLost = int(lapsEquivalent * 20) // Assume 20 cars on lead lap
	}

	calc.RecoveryLaps = calc.PositionsLost / 2 // Optimistic recovery rate

	return calc
}

// predictPositionChanges predicts position changes from pit stops
func (psc *PitStopCalculator) predictPositionChanges(data *sims.TelemetryData, trackData *TrackData) []PositionChange {
	changes := make([]PositionChange, 0)

	currentPos := data.Player.Position

	// Immediate pit stop
	immediateChange := PositionChange{
		FromPosition: currentPos,
		ToPosition:   currentPos + 2,
		NetChange:    -2,
		Probability:  0.8,
		RecoveryTime: time.Minute * 5,
		DependsOn:    []string{"Clean pit stop", "Minimal traffic"},
	}
	changes = append(changes, immediateChange)

	// Delayed pit stop (overcut attempt)
	delayedChange := PositionChange{
		FromPosition: currentPos,
		ToPosition:   currentPos - 1,
		NetChange:    1,
		Probability:  0.6,
		RecoveryTime: time.Minute * 2,
		DependsOn:    []string{"Tire degradation management", "Others pitting first"},
	}
	changes = append(changes, delayedChange)

	return changes
}

// identifyRiskFactors identifies risks associated with pit stop timing
func (psc *PitStopCalculator) identifyRiskFactors(data *sims.TelemetryData, trackData *TrackData, raceAnalysis *RaceAnalysis) []PitRiskFactor {
	risks := make([]PitRiskFactor, 0)

	// Tire degradation risk
	currentWear := CalculateAverageWear(data.Player.Tires)
	if currentWear > 70 {
		risk := PitRiskFactor{
			RiskType:    "tire_degradation",
			Severity:    "high",
			Probability: 0.9,
			Impact:      "Lap time loss increasing rapidly",
			Mitigation:  "Pit within next 3 laps",
			TimeWindow:  "immediate",
		}
		risks = append(risks, risk)
	}

	// Fuel shortage risk
	if data.Player.Fuel.EstimatedLapsLeft < 5 {
		risk := PitRiskFactor{
			RiskType:    "fuel_shortage",
			Severity:    "critical",
			Probability: 1.0,
			Impact:      "DNF if not addressed",
			Mitigation:  "Mandatory pit stop for fuel",
			TimeWindow:  "next_2_laps",
		}
		risks = append(risks, risk)
	}

	// Undercut threat
	if len(data.Opponents) > 0 {
		for _, opponent := range data.Opponents {
			if opponent.Position > data.Player.Position && opponent.GapToPlayer > -time.Second*25 {
				risk := PitRiskFactor{
					RiskType:    "undercut_threat",
					Severity:    "medium",
					Probability: 0.7,
					Impact:      "Loss of track position",
					Mitigation:  "Defensive pit stop or extend stint",
					TimeWindow:  "next_5_laps",
				}
				risks = append(risks, risk)
				break
			}
		}
	}

	return risks
}

// identifyOpportunityWindows identifies strategic opportunities for pit stops
func (psc *PitStopCalculator) identifyOpportunityWindows(data *sims.TelemetryData, trackData *TrackData, raceAnalysis *RaceAnalysis) []OpportunityWindow {
	opportunities := make([]OpportunityWindow, 0)

	// Safety car opportunity (simplified prediction)
	if data.Session.Flag == sims.SessionFlagGreen {
		scOpportunity := OpportunityWindow{
			WindowType:      "safety_car",
			StartLap:        data.Player.CurrentLap + 3,
			EndLap:          data.Player.CurrentLap + 10,
			Probability:     0.3, // 30% chance of SC in next 7 laps
			PotentialGain:   time.Second * 15,
			RequiredActions: []string{"Monitor track conditions", "Ready pit crew"},
		}
		opportunities = append(opportunities, scOpportunity)
	}

	// Weather opportunity
	weatherOpportunity := OpportunityWindow{
		WindowType:      "weather",
		StartLap:        data.Player.CurrentLap + 5,
		EndLap:          data.Player.CurrentLap + 15,
		Probability:     0.2, // 20% chance of weather change
		PotentialGain:   time.Second * 20,
		RequiredActions: []string{"Monitor weather radar", "Prepare wet tires"},
	}
	opportunities = append(opportunities, weatherOpportunity)

	return opportunities
}

// generateEnhancedPitRecommendation creates enhanced pit recommendation
func (psc *PitStopCalculator) generateEnhancedPitRecommendation(analysis *PitStopAnalysis, data *sims.TelemetryData) PitRecommendation {
	recommendation := PitRecommendation{
		ShouldPit:     false,
		WindowOpen:    false,
		TireCompound:  "medium",
		EstimatedLoss: time.Second * 25,
		RiskFactors:   make([]string, 0),
		Alternatives:  make([]PitAlternative, 0),
	}

	// Determine if should pit based on optimal windows
	if len(analysis.OptimalWindows) > 0 {
		optimalWindow := analysis.OptimalWindows[0] // Use first/best window
		recommendation.ShouldPit = true
		recommendation.WindowOpen = true
		recommendation.OptimalLap = optimalWindow.OptimalLap
		recommendation.WindowCloseLap = optimalWindow.EndLap
		recommendation.EstimatedLoss = optimalWindow.ExpectedGain
	}

	// Set tire compound based on conditions
	currentWear := CalculateAverageWear(data.Player.Tires)
	if currentWear > 60 {
		recommendation.TireCompound = "soft" // Fresh grip needed
	} else {
		recommendation.TireCompound = "medium" // Balanced choice
	}

	// Calculate fuel load
	if data.Session.TotalLaps > 0 {
		lapsRemaining := data.Session.TotalLaps - data.Player.CurrentLap
		fuelNeeded := float64(lapsRemaining) * data.Player.Fuel.UsagePerLap * 1.1 // 10% safety margin
		recommendation.FuelLoad = fuelNeeded
	}

	// Add risk factors from analysis
	for _, risk := range analysis.RiskFactors {
		if risk.Severity == "high" || risk.Severity == "critical" {
			recommendation.RiskFactors = append(recommendation.RiskFactors, risk.Impact)
		}
	}

	return recommendation
}

// generateAlternativeOptions creates alternative pit stop options
func (psc *PitStopCalculator) generateAlternativeOptions(analysis *PitStopAnalysis, data *sims.TelemetryData) []PitAlternative {
	alternatives := make([]PitAlternative, 0)

	// Conservative option - pit now
	conservative := PitAlternative{
		Lap:          data.Player.CurrentLap + 1,
		TireCompound: "medium",
		FuelLoad:     50.0,
		Pros:         []string{"Safe option", "Avoids tire cliff", "Covers undercut"},
		Cons:         []string{"Gives up track position", "May be too early"},
		RiskLevel:    "low",
	}
	alternatives = append(alternatives, conservative)

	// Aggressive option - extend stint
	aggressive := PitAlternative{
		Lap:          data.Player.CurrentLap + 8,
		TireCompound: "soft",
		FuelLoad:     35.0,
		Pros:         []string{"Overcut opportunity", "Fresh tires at end", "Track position advantage"},
		Cons:         []string{"Tire degradation risk", "Fuel management required"},
		RiskLevel:    "high",
	}
	alternatives = append(alternatives, aggressive)

	return alternatives
}

// calculateConfidence calculates confidence in pit timing calculations
func (psc *PitStopCalculator) calculateConfidence(data *sims.TelemetryData) float64 {
	confidence := 0.7 // Base confidence

	// Higher confidence with more telemetry data
	if len(psc.positionTracker.playerHistory) > 10 {
		confidence += 0.1
	}

	// Lower confidence in changing conditions
	if data.Session.Flag != sims.SessionFlagGreen {
		confidence -= 0.2
	}

	// Higher confidence with clear track position
	if data.Player.GapToAhead > time.Second*10 && data.Player.GapToBehind > time.Second*10 {
		confidence += 0.1
	}

	return math.Max(0.1, math.Min(1.0, confidence))
}

// assessDataQuality assesses quality of data for calculations
func (psc *PitStopCalculator) assessDataQuality() float64 {
	quality := 0.7 // Base quality

	// Check position tracking history
	if len(psc.positionTracker.playerHistory) >= 5 {
		quality += 0.1
	}

	// Check timing analysis data
	if len(psc.timingAnalyzer.lapTimePatterns) >= 3 {
		quality += 0.1
	}

	// Check opponent data availability
	if len(psc.positionTracker.opponentHistory) >= 3 {
		quality += 0.1
	}

	return math.Max(0.1, math.Min(1.0, quality))
}
