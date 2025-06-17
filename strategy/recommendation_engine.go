package strategy

import (
	"math"
	"sort"
	"time"

	"changeme/sims" // Import simulator data types
)

// RecommendationEngine provides advanced data analysis and strategic recommendations
type RecommendationEngine struct {
	config           *Config
	telemetryHistory []TelemetrySnapshot
	lapAnalysis      *LapAnalysis
	fuelAnalysis     *FuelAnalysis
	tireAnalysis     *TireAnalysis
	raceAnalysis     *RaceAnalysis
}

// TelemetrySnapshot captures telemetry data at a specific moment
type TelemetrySnapshot struct {
	Timestamp time.Time
	Data      *sims.TelemetryData
}

// LapAnalysis provides lap time analysis and predictions
type LapAnalysis struct {
	ConsistencyScore float64 // 0-1, higher is more consistent
	TrendDirection   string  // "improving", "stable", "degrading"
	PredictedLapTime time.Duration
	OptimalLapTime   time.Duration
	LapTimeVariance  time.Duration
	RecentLapTimes   []time.Duration
	AverageLapTime   time.Duration
	MedianLapTime    time.Duration
}

// FuelAnalysis provides fuel consumption analysis and predictions
type FuelAnalysis struct {
	AverageConsumption     float64 // Liters per lap
	TrendingConsumption    float64 // Recent consumption trend
	RemainingLaps          int     // Estimated laps with current fuel
	FuelToFinish           float64 // Fuel needed to finish the race
	SaveRequired           float64 // Fuel save required per lap (if negative = excess)
	SafetyMargin           float64 // Safety margin in liters
	WeatherImpact          float64 // Consumption change due to weather (-1 to 1)
	StrategyRecommendation string  // "conservative", "balanced", "aggressive"
}

// TireAnalysis provides tire degradation analysis and pit strategy
type TireAnalysis struct {
	DegradationRate        float64       // Wear per lap (0-1)
	OptimalStintLength     int           // Optimal laps on current compound
	CurrentStintLap        int           // Laps on current tires
	PerformanceDelta       time.Duration // Performance loss due to wear
	PitWindowOpen          bool          // Whether pit window is optimal
	CompoundRecommendation string        // Recommended tire compound
	EstimatedPitLoss       time.Duration // Time lost in pit stop
	UnderCutThreat         bool          // Risk of being undercut
	OverCutOpportunity     bool          // Opportunity to overcut
}

// RaceAnalysis provides overall race situation analysis
type RaceAnalysis struct {
	RaceFormat               string          // Detected race format
	StrategicPhase           string          // "early", "middle", "late", "critical"
	PositionTrend            string          // "gaining", "stable", "losing"
	CompetitiveGaps          map[int]float64 // Position -> gap analysis
	SafetyCarProbability     float64         // Likelihood of SC (0-1)
	WeatherChangeProbability float64         // Likelihood of weather change (0-1)
	RiskLevel                string          // "low", "medium", "high", "critical"
	OpportunityScore         float64         // Overall opportunity rating (0-1)
	KeyStrategicFactors      []string        // Most important strategic considerations
}

// StrategicRecommendation represents a comprehensive strategic recommendation
type StrategicRecommendation struct {
	// Core Recommendation
	PrimaryStrategy string  `json:"primary_strategy"`
	ConfidenceLevel float64 `json:"confidence_level"`
	RiskAssessment  string  `json:"risk_assessment"`

	// Immediate Actions
	ImmediateActions []ActionRecommendation   `json:"immediate_actions"`
	LapTargets       map[string]time.Duration `json:"lap_targets"`

	// Pit Strategy
	PitRecommendation     PitRecommendation     `json:"pit_recommendation"`
	AlternativeStrategies []AlternativeStrategy `json:"alternative_strategies"`

	// Performance Optimization
	DrivingRecommendations []string `json:"driving_recommendations"`
	SetupSuggestions       []string `json:"setup_suggestions"`

	// Race Management
	FuelManagement FuelManagementPlan `json:"fuel_management"`
	TireManagement TireManagementPlan `json:"tire_management"`

	// Situational Awareness
	ThreatsAndOpportunities ThreatOpportunityAnalysis `json:"threats_opportunities"`
	WeatherConsiderations   WeatherStrategy           `json:"weather_considerations"`

	// Predictions
	FinishPrediction FinishPrediction `json:"finish_prediction"`

	// Analysis Meta
	AnalysisDepth string    `json:"analysis_depth"`
	DataQuality   float64   `json:"data_quality"`
	Timestamp     time.Time `json:"timestamp"`
}

// ActionRecommendation represents a specific action to take
type ActionRecommendation struct {
	Action     string  `json:"action"`
	Priority   string  `json:"priority"` // "immediate", "high", "medium", "low"
	Timing     string  `json:"timing"`   // "now", "next_lap", "pit_window"
	Confidence float64 `json:"confidence"`
	Rationale  string  `json:"rationale"`
}

// PitRecommendation provides pit stop strategy
type PitRecommendation struct {
	ShouldPit      bool             `json:"should_pit"`
	OptimalLap     int              `json:"optimal_lap"`
	WindowOpen     bool             `json:"window_open"`
	WindowCloseLap int              `json:"window_close_lap"`
	TireCompound   string           `json:"tire_compound"`
	FuelLoad       float64          `json:"fuel_load"`
	EstimatedLoss  time.Duration    `json:"estimated_loss"`
	StrategicGain  time.Duration    `json:"strategic_gain"`
	RiskFactors    []string         `json:"risk_factors"`
	Alternatives   []PitAlternative `json:"alternatives"`
}

// PitAlternative represents alternative pit strategies
type PitAlternative struct {
	Lap          int      `json:"lap"`
	TireCompound string   `json:"tire_compound"`
	FuelLoad     float64  `json:"fuel_load"`
	Pros         []string `json:"pros"`
	Cons         []string `json:"cons"`
	RiskLevel    string   `json:"risk_level"`
}

// AlternativeStrategy represents alternative race strategies
type AlternativeStrategy struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	PitStops      []PitStop `json:"pit_stops"`
	RiskLevel     string    `json:"risk_level"`
	Probability   float64   `json:"probability"`
	Advantages    []string  `json:"advantages"`
	Disadvantages []string  `json:"disadvantages"`
}

// PitStop represents a planned pit stop
type PitStop struct {
	Lap          int     `json:"lap"`
	TireCompound string  `json:"tire_compound"`
	FuelLoad     float64 `json:"fuel_load"`
	Reason       string  `json:"reason"`
}

// FuelManagementPlan provides fuel strategy details
type FuelManagementPlan struct {
	CurrentConsumption float64  `json:"current_consumption"`
	TargetConsumption  float64  `json:"target_consumption"`
	SaveRequired       float64  `json:"save_required"`
	MarginAvailable    float64  `json:"margin_available"`
	LiftAndCoastZones  []string `json:"lift_coast_zones"`
	ShortShiftPoints   []string `json:"short_shift_points"`
	WeatherContingency float64  `json:"weather_contingency"`
}

// TireManagementPlan provides tire strategy details
type TireManagementPlan struct {
	CurrentDegradation   float64            `json:"current_degradation"`
	OptimalStintLength   int                `json:"optimal_stint_length"`
	ManagementTechniques []string           `json:"management_techniques"`
	TemperatureTargets   map[string]float64 `json:"temperature_targets"`
	PressureTargets      map[string]float64 `json:"pressure_targets"`
	CompoundStrategy     string             `json:"compound_strategy"`
}

// ThreatOpportunityAnalysis identifies threats and opportunities
type ThreatOpportunityAnalysis struct {
	ImmediateThreats       []Threat      `json:"immediate_threats"`
	ImmediateOpportunities []Opportunity `json:"immediate_opportunities"`
	StrategicThreats       []Threat      `json:"strategic_threats"`
	StrategicOpportunities []Opportunity `json:"strategic_opportunities"`
	OverallRiskLevel       string        `json:"overall_risk_level"`
	OpportunityScore       float64       `json:"opportunity_score"`
}

// Threat represents a strategic threat
type Threat struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Probability float64 `json:"probability"`
	Impact      string  `json:"impact"`
	Mitigation  string  `json:"mitigation"`
}

// Opportunity represents a strategic opportunity
type Opportunity struct {
	Type         string   `json:"type"`
	Potential    string   `json:"potential"`
	Probability  float64  `json:"probability"`
	Requirements []string `json:"requirements"`
	Timeline     string   `json:"timeline"`
}

// WeatherStrategy provides weather-related recommendations
type WeatherStrategy struct {
	CurrentConditions    string   `json:"current_conditions"`
	Forecast             string   `json:"forecast"`
	ChangeProability     float64  `json:"change_probability"`
	TireRecommendations  []string `json:"tire_recommendations"`
	TimingConsiderations string   `json:"timing_considerations"`
	RiskFactors          []string `json:"risk_factors"`
}

// FinishPrediction provides race finish predictions
type FinishPrediction struct {
	EstimatedPosition int           `json:"estimated_position"`
	PositionRange     [2]int        `json:"position_range"` // [min, max]
	FinishTime        time.Duration `json:"finish_time"`
	GapToWinner       time.Duration `json:"gap_to_winner"`
	Confidence        float64       `json:"confidence"`
	KeyFactors        []string      `json:"key_factors"`
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine(config *Config) *RecommendationEngine {
	return &RecommendationEngine{
		config:           config,
		telemetryHistory: make([]TelemetrySnapshot, 0, 1000), // Store last 1000 samples
		lapAnalysis:      &LapAnalysis{},
		fuelAnalysis:     &FuelAnalysis{},
		tireAnalysis:     &TireAnalysis{},
		raceAnalysis:     &RaceAnalysis{},
	}
}

// AddTelemetrySnapshot adds a new telemetry sample for analysis
func (re *RecommendationEngine) AddTelemetrySnapshot(data *sims.TelemetryData) {
	snapshot := TelemetrySnapshot{
		Timestamp: time.Now(),
		Data:      data,
	}

	re.telemetryHistory = append(re.telemetryHistory, snapshot)

	// Keep only the last 1000 samples to prevent memory issues
	if len(re.telemetryHistory) > 1000 {
		re.telemetryHistory = re.telemetryHistory[1:]
	}

	// Update all analyses with new data
	re.updateAnalyses()
}

// updateAnalyses updates all analysis components with latest data
func (re *RecommendationEngine) updateAnalyses() {
	if len(re.telemetryHistory) == 0 {
		return
	}

	latest := re.telemetryHistory[len(re.telemetryHistory)-1]

	re.updateLapAnalysis(latest)
	re.updateFuelAnalysis(latest)
	re.updateTireAnalysis(latest)
	re.updateRaceAnalysis(latest)
}

// updateLapAnalysis analyzes lap time data for consistency and trends
func (re *RecommendationEngine) updateLapAnalysis(snapshot TelemetrySnapshot) {
	data := snapshot.Data

	// Collect recent lap times (last 10 laps or available data)
	recentLaps := make([]time.Duration, 0, 10)
	for i := len(re.telemetryHistory) - 1; i >= 0 && len(recentLaps) < 10; i-- {
		if re.telemetryHistory[i].Data.Player.LastLapTime > 0 {
			recentLaps = append(recentLaps, re.telemetryHistory[i].Data.Player.LastLapTime)
		}
	}

	if len(recentLaps) < 3 {
		return // Not enough data for meaningful analysis
	}

	re.lapAnalysis.RecentLapTimes = recentLaps

	// Calculate average and median lap times
	total := time.Duration(0)
	sortedLaps := make([]time.Duration, len(recentLaps))
	copy(sortedLaps, recentLaps)
	sort.Slice(sortedLaps, func(i, j int) bool {
		return sortedLaps[i] < sortedLaps[j]
	})

	for _, lapTime := range recentLaps {
		total += lapTime
	}

	re.lapAnalysis.AverageLapTime = total / time.Duration(len(recentLaps))
	re.lapAnalysis.MedianLapTime = sortedLaps[len(sortedLaps)/2]

	// Calculate consistency score (lower variance = higher consistency)
	variance := time.Duration(0)
	for _, lapTime := range recentLaps {
		diff := lapTime - re.lapAnalysis.AverageLapTime
		if diff < 0 {
			diff = -diff
		}
		variance += diff
	}
	re.lapAnalysis.LapTimeVariance = variance / time.Duration(len(recentLaps))

	// Consistency score: 1 - (variance as percentage of average lap time)
	if re.lapAnalysis.AverageLapTime > 0 {
		variancePercent := float64(re.lapAnalysis.LapTimeVariance) / float64(re.lapAnalysis.AverageLapTime)
		re.lapAnalysis.ConsistencyScore = math.Max(0, 1-variancePercent*10) // Scale variance impact
	}

	// Determine trend direction
	if len(recentLaps) >= 5 {
		earlyAvg := (recentLaps[0] + recentLaps[1] + recentLaps[2]) / 3
		recentAvg := (recentLaps[len(recentLaps)-3] + recentLaps[len(recentLaps)-2] + recentLaps[len(recentLaps)-1]) / 3

		diff := recentAvg - earlyAvg
		threshold := time.Millisecond * 200 // 0.2 second threshold

		if diff < -threshold {
			re.lapAnalysis.TrendDirection = "improving"
		} else if diff > threshold {
			re.lapAnalysis.TrendDirection = "degrading"
		} else {
			re.lapAnalysis.TrendDirection = "stable"
		}
	}

	// Predict next lap time based on trend and consistency
	if data.Player.BestLapTime > 0 {
		re.lapAnalysis.OptimalLapTime = data.Player.BestLapTime

		// Predicted lap time considers current form and tire degradation
		trendAdjustment := time.Duration(0)
		switch re.lapAnalysis.TrendDirection {
		case "improving":
			trendAdjustment = -time.Millisecond * 100
		case "degrading":
			trendAdjustment = time.Millisecond * 200
		}

		re.lapAnalysis.PredictedLapTime = re.lapAnalysis.AverageLapTime + trendAdjustment
	}
}

// updateFuelAnalysis analyzes fuel consumption patterns and predicts fuel needs
func (re *RecommendationEngine) updateFuelAnalysis(snapshot TelemetrySnapshot) {
	data := snapshot.Data

	if len(re.telemetryHistory) < 2 {
		return // Need at least 2 samples for consumption calculation
	}

	// Calculate fuel consumption over recent laps
	consumptionSamples := make([]float64, 0, 10)
	for i := len(re.telemetryHistory) - 1; i >= 1 && len(consumptionSamples) < 10; i-- {
		currentPlayer := re.telemetryHistory[i].Data.Player
		previousPlayer := re.telemetryHistory[i-1].Data.Player

		// Only calculate if we completed a lap (lap number increased)
		if currentPlayer.CurrentLap > previousPlayer.CurrentLap && previousPlayer.Fuel.Level > currentPlayer.Fuel.Level {
			consumption := previousPlayer.Fuel.Level - currentPlayer.Fuel.Level
			if consumption > 0 && consumption < 10 { // Sanity check
				consumptionSamples = append(consumptionSamples, consumption)
			}
		}
	}

	if len(consumptionSamples) > 0 {
		// Calculate average consumption
		total := 0.0
		for _, consumption := range consumptionSamples {
			total += consumption
		}
		re.fuelAnalysis.AverageConsumption = total / float64(len(consumptionSamples))

		// Calculate trending consumption (recent vs older samples)
		if len(consumptionSamples) >= 4 {
			recentAvg := (consumptionSamples[0] + consumptionSamples[1]) / 2
			olderAvg := (consumptionSamples[len(consumptionSamples)-2] + consumptionSamples[len(consumptionSamples)-1]) / 2
			re.fuelAnalysis.TrendingConsumption = recentAvg

			// Adjust average based on trend
			if math.Abs(recentAvg-olderAvg) > 0.1 {
				re.fuelAnalysis.AverageConsumption = (re.fuelAnalysis.AverageConsumption + recentAvg) / 2
			}
		}
	}

	// Calculate remaining laps with current fuel
	if re.fuelAnalysis.AverageConsumption > 0 {
		re.fuelAnalysis.RemainingLaps = int(data.Player.Fuel.Level / re.fuelAnalysis.AverageConsumption)
	}

	// Calculate fuel needed to finish race
	if data.Session.TotalLaps > 0 {
		lapsRemaining := data.Session.TotalLaps - data.Player.CurrentLap
		re.fuelAnalysis.FuelToFinish = float64(lapsRemaining) * re.fuelAnalysis.AverageConsumption
		re.fuelAnalysis.SaveRequired = (re.fuelAnalysis.FuelToFinish - data.Player.Fuel.Level) / float64(lapsRemaining)
	} else if data.Session.TimeRemaining > 0 {
		// For time-based sessions, estimate laps remaining
		if re.lapAnalysis.AverageLapTime > 0 {
			lapsRemaining := float64(data.Session.TimeRemaining) / float64(re.lapAnalysis.AverageLapTime)
			re.fuelAnalysis.FuelToFinish = lapsRemaining * re.fuelAnalysis.AverageConsumption
			re.fuelAnalysis.SaveRequired = (re.fuelAnalysis.FuelToFinish - data.Player.Fuel.Level) / lapsRemaining
		}
	}

	// Add safety margin based on configuration
	re.fuelAnalysis.SafetyMargin = re.fuelAnalysis.FuelToFinish * (re.config.AnalysisPreferences.SafetyMargin - 1)

	// Weather impact on fuel consumption
	re.fuelAnalysis.WeatherImpact = re.calculateWeatherFuelImpact(data)

	// Determine strategy recommendation
	if re.fuelAnalysis.SaveRequired > 0.3 {
		re.fuelAnalysis.StrategyRecommendation = "aggressive_save"
	} else if re.fuelAnalysis.SaveRequired > 0.1 {
		re.fuelAnalysis.StrategyRecommendation = "conservative"
	} else if re.fuelAnalysis.SaveRequired < -0.2 {
		re.fuelAnalysis.StrategyRecommendation = "aggressive"
	} else {
		re.fuelAnalysis.StrategyRecommendation = "balanced"
	}
}

// calculateWeatherFuelImpact estimates fuel consumption change due to weather
func (re *RecommendationEngine) calculateWeatherFuelImpact(data *sims.TelemetryData) float64 {
	// This is a simplified model - in practice would be more sophisticated
	baseImpact := 0.0

	// Rain increases fuel consumption due to lower speeds and more throttle application
	switch data.Session.Flag {
	case sims.SessionFlagYellow:
		baseImpact = -0.1 // Safety car saves fuel
	case sims.SessionFlagRed:
		baseImpact = -0.3 // Red flag saves significant fuel
	}

	// Temperature impacts (very simplified)
	if data.Session.AirTemperature > 30 {
		baseImpact += 0.05 // Hot weather increases consumption slightly
	} else if data.Session.AirTemperature < 10 {
		baseImpact += 0.03 // Cold weather affects engine efficiency
	}

	return baseImpact
}

// updateTireAnalysis analyzes tire degradation and pit strategy
func (re *RecommendationEngine) updateTireAnalysis(snapshot TelemetrySnapshot) {
	data := snapshot.Data
	// Calculate degradation rate based on tire wear progression
	if len(re.telemetryHistory) >= 2 {
		currentStintLap := data.Player.CurrentLap - data.Player.Pit.LastPitLap
		if currentStintLap < 0 {
			currentStintLap = data.Player.CurrentLap // If no previous pit stop
		}
		re.tireAnalysis.CurrentStintLap = currentStintLap

		// Track tire wear progression over recent laps
		wearSamples := make([]float64, 0, 10)
		lapSamples := make([]int, 0, 10)

		for i := len(re.telemetryHistory) - 1; i >= 0 && len(wearSamples) < 10; i-- {
			tireData := re.telemetryHistory[i].Data.Player.Tires
			avgWear := CalculateAverageWear(tireData)
			if avgWear > 0 {
				wearSamples = append(wearSamples, avgWear)
				currentStint := re.telemetryHistory[i].Data.Player.CurrentLap - re.telemetryHistory[i].Data.Player.Pit.LastPitLap
				if currentStint < 0 {
					currentStint = re.telemetryHistory[i].Data.Player.CurrentLap
				}
				lapSamples = append(lapSamples, currentStint)
			}
		}

		if len(wearSamples) >= 3 {
			// Calculate wear rate per lap
			wearDiff := wearSamples[0] - wearSamples[len(wearSamples)-1]
			lapDiff := lapSamples[0] - lapSamples[len(lapSamples)-1]

			if lapDiff > 0 {
				re.tireAnalysis.DegradationRate = wearDiff / float64(lapDiff)
			}
		}
	}
	// Calculate optimal stint length based on degradation
	if re.tireAnalysis.DegradationRate > 0 {
		// Assume optimal wear limit is around 80%
		currentWear := CalculateAverageWear(data.Player.Tires)
		remainingWear := 80.0 - currentWear
		re.tireAnalysis.OptimalStintLength = int(remainingWear / re.tireAnalysis.DegradationRate)
	}

	// Calculate performance delta due to tire wear
	currentWear := CalculateAverageWear(data.Player.Tires)
	if currentWear > 20 {
		// Simplified model: 0.1 seconds per 10% wear beyond 20%
		wearPenalty := (currentWear - 20) / 10 * 0.1
		re.tireAnalysis.PerformanceDelta = time.Duration(wearPenalty * float64(time.Second))
	}

	// Determine if pit window is open
	re.tireAnalysis.PitWindowOpen = re.calculatePitWindow(data)

	// Estimate pit stop time loss (track-specific, simplified here)
	re.tireAnalysis.EstimatedPitLoss = time.Second * 25 // Typical pit loss

	// Analyze undercut/overcut opportunities
	re.analyzeUnderOverCutOpportunities(data)

	// Determine compound recommendation
	re.tireAnalysis.CompoundRecommendation = re.recommendTireCompound(data)
}

// calculatePitWindow determines if the pit window is strategically open
func (re *RecommendationEngine) calculatePitWindow(data *sims.TelemetryData) bool {
	// Consider multiple factors:
	// 1. Tire wear level
	// 2. Fuel level
	// 3. Track position
	// 4. Race phase

	wearCritical := CalculateAverageWear(data.Player.Tires) > 60
	fuelCritical := re.fuelAnalysis.RemainingLaps < 10

	// Don't pit if very early or very late in race
	if data.Session.TotalLaps > 0 {
		raceProgress := float64(data.Player.CurrentLap) / float64(data.Session.TotalLaps)
		if raceProgress < 0.2 || raceProgress > 0.9 {
			return false
		}
	}

	return wearCritical || fuelCritical
}

// analyzeUnderOverCutOpportunities analyzes pit timing relative to competitors
func (re *RecommendationEngine) analyzeUnderOverCutOpportunities(data *sims.TelemetryData) {
	// Simplified analysis - in practice would analyze opponent pit strategies

	// Look for cars within pit stop delta
	pitDelta := time.Second * 25

	re.tireAnalysis.UnderCutThreat = false
	re.tireAnalysis.OverCutOpportunity = false

	for _, opponent := range data.Opponents {
		gap := opponent.GapToPlayer
		if gap < 0 {
			gap = -gap
		}

		// If someone close behind hasn't pitted recently, undercut threat
		if gap < pitDelta && opponent.Position > data.Player.Position {
			re.tireAnalysis.UnderCutThreat = true
		}
		// If someone close ahead has older tires, overcut opportunity
		if gap < pitDelta && opponent.Position < data.Player.Position {
			// Calculate opponent stint length based on their last pit lap
			opponentStintLap := data.Player.CurrentLap - opponent.LastPitLap
			playerStintLap := data.Player.CurrentLap - data.Player.Pit.LastPitLap
			if playerStintLap < 0 {
				playerStintLap = data.Player.CurrentLap
			}
			if opponentStintLap > playerStintLap+5 {
				re.tireAnalysis.OverCutOpportunity = true
			}
		}
	}
}

// recommendTireCompound suggests optimal tire compound
func (re *RecommendationEngine) recommendTireCompound(data *sims.TelemetryData) string {
	// Simplified compound recommendation based on conditions

	// Weather-based selection
	if data.Session.TrackTemperature > 40 {
		return "hard" // Hot track favors harder compounds
	} else if data.Session.TrackTemperature < 20 {
		return "soft" // Cold track needs softer compounds
	}

	// Race phase considerations
	if data.Session.TotalLaps > 0 {
		raceProgress := float64(data.Player.CurrentLap) / float64(data.Session.TotalLaps)
		if raceProgress > 0.7 {
			return "soft" // Push hard at race end
		}
	}

	return "medium" // Default balanced choice
}

// updateRaceAnalysis analyzes overall race situation and strategic context
func (re *RecommendationEngine) updateRaceAnalysis(snapshot TelemetrySnapshot) {
	data := snapshot.Data

	// Detect race format
	re.raceAnalysis.RaceFormat = re.detectRaceFormat(data)

	// Determine strategic phase
	re.raceAnalysis.StrategicPhase = re.determineStrategicPhase(data)

	// Analyze position trends
	re.analyzePositionTrends(data)

	// Calculate competitive gaps
	re.calculateCompetitiveGaps(data)

	// Assess overall risk level
	re.assessRiskLevel(data)

	// Calculate opportunity score
	re.calculateOpportunityScore(data)

	// Identify key strategic factors
	re.identifyKeyStrategicFactors(data)
}

// detectRaceFormat determines the race format based on session data
func (re *RecommendationEngine) detectRaceFormat(data *sims.TelemetryData) string {
	if data.Session.TotalLaps > 0 {
		if data.Session.TotalLaps <= 15 {
			return "sprint"
		} else if data.Session.TotalLaps >= 50 {
			return "endurance"
		}
	} else if data.Session.TimeRemaining > 0 {
		if data.Session.TimeRemaining < time.Hour {
			return "sprint"
		} else if data.Session.TimeRemaining > time.Hour*2 {
			return "endurance"
		}
	}
	return "standard"
}

// determineStrategicPhase determines what phase of the race we're in
func (re *RecommendationEngine) determineStrategicPhase(data *sims.TelemetryData) string {
	progress := 0.0

	if data.Session.TotalLaps > 0 {
		progress = float64(data.Player.CurrentLap) / float64(data.Session.TotalLaps)
	} else if data.Session.SessionTime > 0 {
		elapsed := data.Session.SessionTime - data.Session.TimeRemaining
		progress = float64(elapsed) / float64(data.Session.SessionTime)
	}

	switch {
	case progress < 0.25:
		return "early"
	case progress < 0.75:
		return "middle"
	case progress < 0.9:
		return "late"
	default:
		return "critical"
	}
}

// analyzePositionTrends determines if position is improving, stable, or declining
func (re *RecommendationEngine) analyzePositionTrends(data *sims.TelemetryData) {
	if len(re.telemetryHistory) < 5 {
		re.raceAnalysis.PositionTrend = "stable"
		return
	}

	// Look at position over last 5 samples
	positions := make([]int, 0, 5)
	for i := len(re.telemetryHistory) - 5; i < len(re.telemetryHistory); i++ {
		positions = append(positions, re.telemetryHistory[i].Data.Player.Position)
	}

	startPos := positions[0]
	endPos := positions[len(positions)-1]

	if endPos < startPos-1 {
		re.raceAnalysis.PositionTrend = "gaining"
	} else if endPos > startPos+1 {
		re.raceAnalysis.PositionTrend = "losing"
	} else {
		re.raceAnalysis.PositionTrend = "stable"
	}
}

// calculateCompetitiveGaps analyzes gaps to key competitors
func (re *RecommendationEngine) calculateCompetitiveGaps(data *sims.TelemetryData) {
	re.raceAnalysis.CompetitiveGaps = make(map[int]float64)

	for _, opponent := range data.Opponents {
		// Focus on nearby competitors (within 3 positions)
		positionDiff := opponent.Position - data.Player.Position
		positionDiffAbs := positionDiff
		if positionDiffAbs < 0 {
			positionDiffAbs = -positionDiffAbs
		}
		if positionDiffAbs <= 3 {
			gapSeconds := float64(opponent.GapToPlayer) / float64(time.Second)
			re.raceAnalysis.CompetitiveGaps[opponent.Position] = gapSeconds
		}
	}
}

// assessRiskLevel determines overall risk level of current situation
func (re *RecommendationEngine) assessRiskLevel(data *sims.TelemetryData) {
	riskFactors := 0

	// Check various risk factors
	if data.Player.Fuel.Level < re.fuelAnalysis.FuelToFinish*1.1 {
		riskFactors++
	}

	if CalculateAverageWear(data.Player.Tires) > 70 {
		riskFactors++
	}

	if data.Session.Flag == sims.SessionFlagYellow || data.Session.Flag == sims.SessionFlagRed {
		riskFactors++
	}

	if re.tireAnalysis.UnderCutThreat {
		riskFactors++
	}

	switch riskFactors {
	case 0:
		re.raceAnalysis.RiskLevel = "low"
	case 1:
		re.raceAnalysis.RiskLevel = "medium"
	case 2:
		re.raceAnalysis.RiskLevel = "high"
	default:
		re.raceAnalysis.RiskLevel = "critical"
	}
}

// calculateOpportunityScore rates current strategic opportunities
func (re *RecommendationEngine) calculateOpportunityScore(data *sims.TelemetryData) {
	score := 0.5 // Base score

	// Positive factors
	if re.tireAnalysis.OverCutOpportunity {
		score += 0.2
	}

	if re.lapAnalysis.TrendDirection == "improving" {
		score += 0.1
	}

	if data.Session.Flag == sims.SessionFlagYellow {
		score += 0.2 // Safety car creates opportunities
	}

	if re.fuelAnalysis.StrategyRecommendation == "aggressive" {
		score += 0.1 // Fuel advantage
	}

	// Negative factors
	if re.tireAnalysis.UnderCutThreat {
		score -= 0.2
	}

	if re.lapAnalysis.TrendDirection == "degrading" {
		score -= 0.1
	}

	if CalculateAverageWear(data.Player.Tires) > 80 {
		score -= 0.2 // Tire disadvantage
	}

	re.raceAnalysis.OpportunityScore = math.Max(0, math.Min(1, score))
}

// identifyKeyStrategicFactors determines the most important current considerations
func (re *RecommendationEngine) identifyKeyStrategicFactors(data *sims.TelemetryData) {
	factors := make([]string, 0)

	// Critical fuel situation
	if re.fuelAnalysis.SaveRequired > 0.2 {
		factors = append(factors, "Critical fuel management required")
	}

	// Tire strategy
	if CalculateAverageWear(data.Player.Tires) > 60 {
		factors = append(factors, "Tire degradation reaching critical level")
	}

	// Position battles
	if len(re.raceAnalysis.CompetitiveGaps) > 0 {
		factors = append(factors, "Close position battles ongoing")
	}

	// Weather considerations
	if data.Session.Flag != sims.SessionFlagGreen {
		factors = append(factors, "Track conditions affecting strategy")
	}

	// Race phase
	switch re.raceAnalysis.StrategicPhase {
	case "late":
		factors = append(factors, "Late race phase - aggressive strategy window")
	case "critical":
		factors = append(factors, "Final phase - every decision critical")
	}

	re.raceAnalysis.KeyStrategicFactors = factors
}

// GenerateRecommendation creates a comprehensive strategic recommendation
func (re *RecommendationEngine) GenerateRecommendation(data *sims.TelemetryData) *StrategicRecommendation {
	// Ensure analyses are up to date
	re.AddTelemetrySnapshot(data)

	recommendation := &StrategicRecommendation{
		Timestamp:     time.Now(),
		AnalysisDepth: "comprehensive",
		DataQuality:   re.calculateDataQuality(),
	}

	// Generate core strategy recommendation
	recommendation.PrimaryStrategy = re.generatePrimaryStrategy(data)
	recommendation.ConfidenceLevel = re.calculateConfidenceLevel(data)
	recommendation.RiskAssessment = re.raceAnalysis.RiskLevel

	// Generate immediate actions
	recommendation.ImmediateActions = re.generateImmediateActions(data)
	recommendation.LapTargets = re.generateLapTargets(data)

	// Generate pit strategy
	recommendation.PitRecommendation = re.generatePitRecommendation(data)
	recommendation.AlternativeStrategies = re.generateAlternativeStrategies(data)

	// Generate performance recommendations
	recommendation.DrivingRecommendations = re.generateDrivingRecommendations(data)
	recommendation.SetupSuggestions = re.generateSetupSuggestions(data)

	// Generate management plans
	recommendation.FuelManagement = re.generateFuelManagementPlan(data)
	recommendation.TireManagement = re.generateTireManagementPlan(data) // Generate situational analysis
	recommendation.ThreatsAndOpportunities = re.generateThreatOpportunityAnalysis(data)
	recommendation.WeatherConsiderations = re.generateWeatherStrategy(data)

	// Generate predictions
	recommendation.FinishPrediction = re.generateFinishPrediction(data)
	return recommendation
}

// calculateDataQuality assesses the quality of available data
func (re *RecommendationEngine) calculateDataQuality() float64 {
	score := 0.0
	maxScore := 0.0

	// Check telemetry history depth
	maxScore += 0.3
	if len(re.telemetryHistory) >= 10 {
		score += 0.3
	} else if len(re.telemetryHistory) >= 5 {
		score += 0.2
	} else if len(re.telemetryHistory) >= 2 {
		score += 0.1
	}

	// Check lap time data quality
	maxScore += 0.3
	if len(re.lapAnalysis.RecentLapTimes) >= 5 {
		score += 0.3
	} else if len(re.lapAnalysis.RecentLapTimes) >= 3 {
		score += 0.2
	}

	// Check fuel analysis quality
	maxScore += 0.2
	if re.fuelAnalysis.AverageConsumption > 0 {
		score += 0.2
	}

	// Check tire analysis quality
	maxScore += 0.2
	if re.tireAnalysis.DegradationRate > 0 {
		score += 0.2
	}

	return score / maxScore
}

// Tire data helper functions

// CalculateAverageWear calculates the average tire wear percentage across all four tires
func CalculateAverageWear(tires sims.TireData) float64 {
	return (tires.FrontLeft.WearPercent + tires.FrontRight.WearPercent +
		tires.RearLeft.WearPercent + tires.RearRight.WearPercent) / 4.0
}

// CalculateAverageTireTemp calculates the average tire temperature across all four tires
func CalculateAverageTireTemp(tires sims.TireData) float64 {
	return (tires.FrontLeft.Temperature + tires.FrontRight.Temperature +
		tires.RearLeft.Temperature + tires.RearRight.Temperature) / 4.0
}

// CalculateAverageTirePressure calculates the average tire pressure across all four tires
func CalculateAverageTirePressure(tires sims.TireData) float64 {
	return (tires.FrontLeft.Pressure + tires.FrontRight.Pressure +
		tires.RearLeft.Pressure + tires.RearRight.Pressure) / 4.0
}

// FindMaxTireWear finds the maximum tire wear percentage among all four tires
func FindMaxTireWear(tires sims.TireData) float64 {
	maxWear := tires.FrontLeft.WearPercent
	if tires.FrontRight.WearPercent > maxWear {
		maxWear = tires.FrontRight.WearPercent
	}
	if tires.RearLeft.WearPercent > maxWear {
		maxWear = tires.RearLeft.WearPercent
	}
	if tires.RearRight.WearPercent > maxWear {
		maxWear = tires.RearRight.WearPercent
	}
	return maxWear
}

// FindMinTireWear finds the minimum tire wear percentage among all four tires
func FindMinTireWear(tires sims.TireData) float64 {
	minWear := tires.FrontLeft.WearPercent
	if tires.FrontRight.WearPercent < minWear {
		minWear = tires.FrontRight.WearPercent
	}
	if tires.RearLeft.WearPercent < minWear {
		minWear = tires.RearLeft.WearPercent
	}
	if tires.RearRight.WearPercent < minWear {
		minWear = tires.RearRight.WearPercent
	}
	return minWear
}

// CalculateTireWearVariance calculates the variance in tire wear across all four tires
func CalculateTireWearVariance(tires sims.TireData) float64 {
	avgWear := CalculateAverageWear(tires)
	wears := []float64{
		tires.FrontLeft.WearPercent,
		tires.FrontRight.WearPercent,
		tires.RearLeft.WearPercent,
		tires.RearRight.WearPercent,
	}

	variance := 0.0
	for _, wear := range wears {
		diff := wear - avgWear
		variance += diff * diff
	}
	return variance / 4.0
}

// generatePrimaryStrategy creates the main strategic recommendation
func (re *RecommendationEngine) generatePrimaryStrategy(data *sims.TelemetryData) string {
	switch re.raceAnalysis.StrategicPhase {
	case "early":
		if re.fuelAnalysis.StrategyRecommendation == "aggressive" {
			return "Push hard early to build gap, manage fuel mid-race"
		}
		return "Focus on consistency and tire management for long-term strategy"
	case "middle":
		if re.tireAnalysis.PitWindowOpen {
			return "Execute pit stop strategy, optimize tire compound choice"
		}
		return "Maintain position while monitoring competitors' strategies"
	case "late":
		if re.raceAnalysis.OpportunityScore > 0.7 {
			return "Aggressive push phase - maximize performance for position gain"
		}
		return "Defensive strategy - protect position and manage to finish"
	case "critical":
		return "All-out attack phase - every tenth counts for final positions"
	default:
		return "Balanced approach focusing on consistency and opportunistic gains"
	}
}

// calculateConfidenceLevel determines confidence in the recommendation
func (re *RecommendationEngine) calculateConfidenceLevel(data *sims.TelemetryData) float64 {
	confidence := 0.5

	// Higher confidence with more data
	if len(re.telemetryHistory) >= 10 {
		confidence += 0.2
	}

	// Higher confidence with consistent lap times
	if re.lapAnalysis.ConsistencyScore > 0.8 {
		confidence += 0.1
	}

	// Lower confidence in changing conditions
	if data.Session.Flag != sims.SessionFlagGreen {
		confidence -= 0.2
	}

	// Higher confidence in familiar race phases
	switch re.raceAnalysis.StrategicPhase {
	case "early", "middle":
		confidence += 0.1
	case "critical":
		confidence -= 0.1
	}

	return math.Max(0.1, math.Min(1.0, confidence))
}

// generateImmediateActions creates specific actions to take now
func (re *RecommendationEngine) generateImmediateActions(data *sims.TelemetryData) []ActionRecommendation {
	actions := make([]ActionRecommendation, 0)

	// Critical fuel save
	if re.fuelAnalysis.SaveRequired > 0.3 {
		actions = append(actions, ActionRecommendation{
			Action:     "Immediate fuel saving required",
			Priority:   "immediate",
			Timing:     "now",
			Confidence: 0.9,
			Rationale:  "Critical fuel shortage detected",
		})
	}

	// Pit stop recommendation
	if re.tireAnalysis.PitWindowOpen && CalculateAverageWear(data.Player.Tires) > 70 {
		actions = append(actions, ActionRecommendation{
			Action:     "Prepare for pit stop",
			Priority:   "high",
			Timing:     "next_lap",
			Confidence: 0.8,
			Rationale:  "Optimal pit window with tire degradation",
		})
	}

	// Safety car opportunity
	if data.Session.Flag == sims.SessionFlagYellow && !re.tireAnalysis.PitWindowOpen {
		actions = append(actions, ActionRecommendation{
			Action:     "Consider safety car pit opportunity",
			Priority:   "high",
			Timing:     "pit_window",
			Confidence: 0.7,
			Rationale:  "Safety car reduces pit stop penalty",
		})
	}

	return actions
}

// generateLapTargets creates target lap times for different scenarios
func (re *RecommendationEngine) generateLapTargets(data *sims.TelemetryData) map[string]time.Duration {
	targets := make(map[string]time.Duration)

	if re.lapAnalysis.OptimalLapTime > 0 {
		targets["optimal"] = re.lapAnalysis.OptimalLapTime

		// Fuel saving target
		if re.fuelAnalysis.SaveRequired > 0.1 {
			targets["fuel_save"] = re.lapAnalysis.OptimalLapTime + time.Millisecond*500
		}

		// Attack target
		if re.raceAnalysis.OpportunityScore > 0.6 {
			targets["attack"] = re.lapAnalysis.OptimalLapTime - time.Millisecond*200
		}

		// Current stint target
		targets["current_stint"] = re.lapAnalysis.PredictedLapTime
	}

	return targets
}

// generatePitRecommendation creates detailed pit strategy
func (re *RecommendationEngine) generatePitRecommendation(data *sims.TelemetryData) PitRecommendation {
	recommendation := PitRecommendation{
		ShouldPit:     re.tireAnalysis.PitWindowOpen,
		WindowOpen:    re.tireAnalysis.PitWindowOpen,
		TireCompound:  re.tireAnalysis.CompoundRecommendation,
		EstimatedLoss: re.tireAnalysis.EstimatedPitLoss,
		RiskFactors:   make([]string, 0),
		Alternatives:  make([]PitAlternative, 0),
	}

	if data.Session.TotalLaps > 0 {
		lapsRemaining := data.Session.TotalLaps - data.Player.CurrentLap
		recommendation.OptimalLap = data.Player.CurrentLap + min(5, lapsRemaining/3)
		recommendation.WindowCloseLap = data.Player.CurrentLap + 10
	}

	// Calculate fuel load
	recommendation.FuelLoad = re.fuelAnalysis.FuelToFinish + re.fuelAnalysis.SafetyMargin

	// Add risk factors
	if re.tireAnalysis.UnderCutThreat {
		recommendation.RiskFactors = append(recommendation.RiskFactors, "Undercut threat from competitors")
	}

	return recommendation
}

// generateAlternativeStrategies creates alternative strategic options
func (re *RecommendationEngine) generateAlternativeStrategies(data *sims.TelemetryData) []AlternativeStrategy {
	strategies := make([]AlternativeStrategy, 0)

	// Conservative strategy
	strategies = append(strategies, AlternativeStrategy{
		Name:          "Conservative",
		Description:   "Focus on consistency and finishing position",
		RiskLevel:     "low",
		Probability:   0.8,
		Advantages:    []string{"Lower risk", "Consistent finishing"},
		Disadvantages: []string{"Limited position gain opportunity"},
	})

	// Aggressive strategy
	if re.raceAnalysis.OpportunityScore > 0.5 {
		strategies = append(strategies, AlternativeStrategy{
			Name:          "Aggressive",
			Description:   "Push hard for maximum position gain",
			RiskLevel:     "high",
			Probability:   0.6,
			Advantages:    []string{"High position gain potential", "Capitalize on opportunities"},
			Disadvantages: []string{"Higher tire wear", "Fuel consumption risk"},
		})
	}

	return strategies
}

// generateDrivingRecommendations creates driving advice
func (re *RecommendationEngine) generateDrivingRecommendations(data *sims.TelemetryData) []string {
	recommendations := make([]string, 0)

	// Consistency recommendations
	if re.lapAnalysis.ConsistencyScore < 0.7 {
		recommendations = append(recommendations, "Focus on consistent braking points and racing line")
	}

	// Fuel saving
	if re.fuelAnalysis.SaveRequired > 0.1 {
		recommendations = append(recommendations, "Implement lift-and-coast technique in slow corners")
		recommendations = append(recommendations, "Short-shift by 500-1000 RPM to save fuel")
	}

	// Tire management
	if CalculateAverageWear(data.Player.Tires) > 50 {
		recommendations = append(recommendations, "Avoid aggressive curb usage to preserve tires")
		recommendations = append(recommendations, "Smooth throttle application to manage tire temperatures")
	}

	return recommendations
}

// generateSetupSuggestions creates setup recommendations
func (re *RecommendationEngine) generateSetupSuggestions(data *sims.TelemetryData) []string {
	suggestions := make([]string, 0)

	// Weather-based suggestions
	if data.Session.TrackTemperature > 35 {
		suggestions = append(suggestions, "Consider softer front anti-roll bar for hot conditions")
	} else if data.Session.TrackTemperature < 20 {
		suggestions = append(suggestions, "Increase tire pressures for cold track conditions")
	}

	// Tire wear suggestions
	if re.tireAnalysis.DegradationRate > 0.05 {
		suggestions = append(suggestions, "Reduce camber to improve tire longevity")
	}

	return suggestions
}

// generateFuelManagementPlan creates detailed fuel strategy
func (re *RecommendationEngine) generateFuelManagementPlan(data *sims.TelemetryData) FuelManagementPlan {
	plan := FuelManagementPlan{
		CurrentConsumption: re.fuelAnalysis.AverageConsumption,
		TargetConsumption:  re.fuelAnalysis.AverageConsumption - re.fuelAnalysis.SaveRequired,
		SaveRequired:       re.fuelAnalysis.SaveRequired,
		MarginAvailable:    re.fuelAnalysis.SafetyMargin,
		LiftAndCoastZones:  []string{"Turn 1 braking zone", "Final sector slow corners"},
		ShortShiftPoints:   []string{"Exit of slow corners", "Long straights"},
		WeatherContingency: re.fuelAnalysis.WeatherImpact,
	}

	return plan
}

// generateTireManagementPlan creates detailed tire management strategy
func (re *RecommendationEngine) generateTireManagementPlan(data *sims.TelemetryData) TireManagementPlan {
	plan := TireManagementPlan{
		CurrentDegradation:   CalculateAverageWear(data.Player.Tires),
		OptimalStintLength:   re.tireAnalysis.OptimalStintLength,
		ManagementTechniques: make([]string, 0),
		TemperatureTargets:   make(map[string]float64),
		PressureTargets:      make(map[string]float64),
		CompoundStrategy:     re.tireAnalysis.CompoundRecommendation,
	}

	// Management techniques based on current wear
	if plan.CurrentDegradation > 70 {
		plan.ManagementTechniques = append(plan.ManagementTechniques,
			"Gentle cornering", "Avoid kerbs", "Smooth throttle application")
	} else if plan.CurrentDegradation > 50 {
		plan.ManagementTechniques = append(plan.ManagementTechniques,
			"Avoid sliding", "Maintain optimal temperatures")
	} else {
		plan.ManagementTechniques = append(plan.ManagementTechniques,
			"Maximize performance", "Heat tires when needed")
	}

	// Temperature targets
	plan.TemperatureTargets["front_left"] = 85.0
	plan.TemperatureTargets["front_right"] = 85.0
	plan.TemperatureTargets["rear_left"] = 90.0
	plan.TemperatureTargets["rear_right"] = 90.0

	// Pressure targets
	plan.PressureTargets["front_left"] = 27.5
	plan.PressureTargets["front_right"] = 27.5
	plan.PressureTargets["rear_left"] = 27.0
	plan.PressureTargets["rear_right"] = 27.0

	return plan
}

// TireStrategicPlan provides comprehensive tire strategy recommendations
type TireStrategicPlan struct {
	BaseRecommendation TireManagementPlan          `json:"base_recommendation"`
	CompoundStrategy   TireCompoundStrategy        `json:"compound_strategy"`
	DegradationMgmt    TireDegradationManagement   `json:"degradation_management"`
	PerformanceOpt     TirePerformanceOptimization `json:"performance_optimization"`
	PitTimingStrategy  TirePitTimingStrategy       `json:"pit_timing_strategy"`
	RiskAssessment     TireRiskAssessment          `json:"risk_assessment"`
	AdaptiveStrategies []TireAdaptiveStrategy      `json:"adaptive_strategies"`
}

// TireCompoundStrategy provides compound selection and usage strategy
type TireCompoundStrategy struct {
	PrimaryCompound      string            `json:"primary_compound"`      // Main compound recommendation
	AlternativeCompounds []string          `json:"alternative_compounds"` // Alternative options
	CompoundSequence     []string          `json:"compound_sequence"`     // Planned compound sequence
	WeatherContingency   map[string]string `json:"weather_contingency"`   // Weather-based alternatives
	PerformanceProfile   string            `json:"performance_profile"`   // Expected performance characteristics
	StrategicAdvantage   string            `json:"strategic_advantage"`   // Why this strategy is optimal
}

// TireDegradationManagement provides tire wear management techniques
type TireDegradationManagement struct {
	CurrentWearRate       float64            `json:"current_wear_rate"`      // Current degradation rate
	OptimalWearRate       float64            `json:"optimal_wear_rate"`      // Target degradation rate
	ManagementTechniques  []string           `json:"management_techniques"`  // Specific techniques to use
	CriticalLaps          []int              `json:"critical_laps"`          // Laps where degradation becomes critical
	PerformanceThresholds map[string]float64 `json:"performance_thresholds"` // Performance degradation thresholds
	AdaptationStrategies  []string           `json:"adaptation_strategies"`  // Strategies to adapt to degradation
}

// TirePerformanceOptimization provides performance optimization strategies
type TirePerformanceOptimization struct {
	TemperatureTargets map[string]float64 `json:"temperature_targets"` // Optimal temperature ranges
	PressureTargets    map[string]float64 `json:"pressure_targets"`    // Optimal pressure settings
	CamberOptimization string             `json:"camber_optimization"` // Camber adjustment recommendations
	HeatingStrategies  []string           `json:"heating_strategies"`  // Tire heating techniques
	CoolingStrategies  []string           `json:"cooling_strategies"`  // Tire cooling techniques
	GripMaximization   []string           `json:"grip_maximization"`   // Techniques for maximum grip
}

// TirePitTimingStrategy provides pit timing strategy for tires
type TirePitTimingStrategy struct {
	OptimalPitLap       int      `json:"optimal_pit_lap"`      // Optimal lap to pit
	EarliestPitLap      int      `json:"earliest_pit_lap"`     // Earliest practical pit lap
	LatestPitLap        int      `json:"latest_pit_lap"`       // Latest safe pit lap
	WindowFactors       []string `json:"window_factors"`       // Factors affecting pit window
	CompetitorInfluence string   `json:"competitor_influence"` // How competitors affect timing
	StrategicTiming     string   `json:"strategic_timing"`     // Overall timing strategy
}

// TireRiskAssessment evaluates tire-related risks
type TireRiskAssessment struct {
	DegradationRisk     float64  `json:"degradation_risk"`      // Risk of excessive degradation
	PerformanceRisk     float64  `json:"performance_risk"`      // Risk of performance loss
	StrategicRisk       float64  `json:"strategic_risk"`        // Risk of strategic disadvantage
	SafetyRisk          float64  `json:"safety_risk"`           // Safety risk from tire condition
	RiskMitigationPlans []string `json:"risk_mitigation_plans"` // Plans to mitigate risks
	ContingencyOptions  []string `json:"contingency_options"`   // Backup options available
}

// TireAdaptiveStrategy provides adaptive strategies for changing conditions
type TireAdaptiveStrategy struct {
	TriggerCondition   string `json:"trigger_condition"`   // Condition that triggers this strategy
	StrategyAdjustment string `json:"strategy_adjustment"` // How to adjust strategy
	ImplementationTime string `json:"implementation_time"` // When to implement
	ExpectedBenefit    string `json:"expected_benefit"`    // Expected benefit
	RiskLevel          string `json:"risk_level"`          // Risk level of this strategy
	MonitoringRequired bool   `json:"monitoring_required"` // Whether monitoring is needed
}

// Enhanced fuel strategy data structures

// FuelStrategicPlan provides comprehensive fuel strategy recommendations
type FuelStrategicPlan struct {
	BaseRecommendation  FuelManagementPlan             `json:"base_recommendation"`
	ScenarioPlans       map[string]FuelManagementPlan  `json:"scenario_plans"`
	RiskAssessment      FuelRiskAssessment             `json:"risk_assessment"`
	OptimizationTips    []FuelOptimizationTip          `json:"optimization_tips"`
	PitStrategyImpact   FuelPitStrategyAnalysis        `json:"pit_strategy_impact"`
	RealTimeAdjustments []FuelAdjustmentRecommendation `json:"real_time_adjustments"`
}

// FuelRiskAssessment evaluates fuel-related risks and contingencies
type FuelRiskAssessment struct {
	ShortageRisk       float64  `json:"shortage_risk"`       // 0-1 risk of running out of fuel
	OverageRisk        float64  `json:"overage_risk"`        // 0-1 risk of carrying too much fuel
	WeatherRisk        float64  `json:"weather_risk"`        // 0-1 risk from weather changes
	TrafficRisk        float64  `json:"traffic_risk"`        // 0-1 risk from traffic impact
	CriticalLaps       []int    `json:"critical_laps"`       // Laps where fuel becomes critical
	ContingencyOptions []string `json:"contingency_options"` // Available backup strategies
	MonitoringRequired bool     `json:"monitoring_required"` // Whether active monitoring is needed
}

// FuelOptimizationTip provides specific fuel saving advice
type FuelOptimizationTip struct {
	Technique          string  `json:"technique"`           // The optimization technique
	PotentialSaving    float64 `json:"potential_saving"`    // Fuel saved per lap (liters)
	PerformanceImpact  string  `json:"performance_impact"`  // Impact on lap times
	ImplementationEase string  `json:"implementation_ease"` // How easy to implement
	Priority           string  `json:"priority"`            // Implementation priority
	ContextSpecific    bool    `json:"context_specific"`    // Whether context-dependent
}

// FuelPitStrategyAnalysis analyzes fuel impact on pit strategy
type FuelPitStrategyAnalysis struct {
	OptimalFuelLoad      float64 `json:"optimal_fuel_load"`     // Optimal fuel for next stint
	MinimumFuelLoad      float64 `json:"minimum_fuel_load"`     // Minimum fuel to finish
	MaximumFuelLoad      float64 `json:"maximum_fuel_load"`     // Maximum practical fuel load
	StrategicFlexibility float64 `json:"strategic_flexibility"` // Flexibility for strategy changes
	FuelVsTime           string  `json:"fuel_vs_time"`          // Trade-off analysis
	RecommendedLoad      float64 `json:"recommended_load"`      // Final recommendation
}

// FuelAdjustmentRecommendation provides real-time fuel adjustments
type FuelAdjustmentRecommendation struct {
	TriggerCondition string  `json:"trigger_condition"` // When to apply this adjustment
	AdjustmentType   string  `json:"adjustment_type"`   // Type of adjustment needed
	TargetSaving     float64 `json:"target_saving"`     // Target fuel saving
	Duration         string  `json:"duration"`          // How long to maintain adjustment
	MonitoringMetric string  `json:"monitoring_metric"` // What to monitor for success
	ExitCondition    string  `json:"exit_condition"`    // When to stop the adjustment
}

// generateThreatOpportunityAnalysis analyzes current threats and opportunities
func (re *RecommendationEngine) generateThreatOpportunityAnalysis(data *sims.TelemetryData) ThreatOpportunityAnalysis {
	threats := []Threat{}
	opportunities := []Opportunity{}

	// Analyze fuel threats and opportunities
	if data.Player.Fuel.Percentage < 20.0 {
		threats = append(threats, Threat{
			Type:     "fuel",
			Severity: "medium",
			Impact:   "0.6",
		})
	}

	// Analyze tire threats and opportunities
	avgWear := (data.Player.Tires.FrontLeft.WearPercent + data.Player.Tires.FrontRight.WearPercent +
		data.Player.Tires.RearLeft.WearPercent + data.Player.Tires.RearRight.WearPercent) / 4.0
	if avgWear > 80.0 {
		threats = append(threats, Threat{
			Type:     "tire",
			Severity: "high",
			Impact:   "0.8",
		})
	}

	// Position opportunities
	if data.Player.Position > 1 {
		opportunities = append(opportunities, Opportunity{
			Type:      "position",
			Potential: "high",
		})
	}

	return ThreatOpportunityAnalysis{
		ImmediateThreats:       threats,
		ImmediateOpportunities: opportunities,
		OverallRiskLevel:       "medium",
		OpportunityScore:       0.6,
	}
}

// generateWeatherStrategy provides weather-specific strategic recommendations
func (re *RecommendationEngine) generateWeatherStrategy(data *sims.TelemetryData) WeatherStrategy {
	strategy := WeatherStrategy{
		CurrentConditions: "dry",
		Forecast:          "stable",
	}

	if data.Session.TrackTemperature > 40.0 {
		strategy.TireRecommendations = []string{"Conservative compound selection due to high temperatures"}
		strategy.TimingConsiderations = "Consider earlier pit stops"
	} else {
		strategy.TireRecommendations = []string{"Standard compound selection"}
		strategy.TimingConsiderations = "Maintain planned strategy"
	}

	return strategy
}

// generateFinishPrediction estimates finishing position and time
func (re *RecommendationEngine) generateFinishPrediction(data *sims.TelemetryData) FinishPrediction {
	// Simple prediction based on current pace and position
	estimatedPosition := data.Player.Position

	return FinishPrediction{
		EstimatedPosition: estimatedPosition,
		FinishTime:        data.Session.TimeRemaining,
		Confidence:        0.75,
		KeyFactors: []string{
			"Based on current pace",
			"Assuming no major incidents",
		},
	}
}
