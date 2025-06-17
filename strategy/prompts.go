package strategy

import (
	"fmt"
	"strings"
)

// PromptTemplate defines the structure for race format-specific prompts
type PromptTemplate struct {
	RaceFormat       string
	SystemContext    string
	StrategyFocus    string
	KeyFactors       []string
	DecisionCriteria []string
	OutputGuidance   string
}

// RaceFormatDetector analyzes race data to determine the race format
type RaceFormatDetector struct {
	config *Config
}

// NewRaceFormatDetector creates a new race format detector
func NewRaceFormatDetector(config *Config) *RaceFormatDetector {
	return &RaceFormatDetector{config: config}
}

// DetectRaceFormat analyzes race data to determine if it's sprint, endurance, or standard
func (rfd *RaceFormatDetector) DetectRaceFormat(raceData *RaceData) string {
	// If manually specified in config, use that
	if rfd.config.AnalysisPreferences.RaceFormat != "auto" {
		return rfd.config.AnalysisPreferences.RaceFormat
	}

	// Auto-detection logic
	if raceData.TotalLaps > 0 {
		// Lap-based race
		if raceData.TotalLaps <= 15 {
			return "sprint"
		} else if raceData.TotalLaps >= 60 {
			return "endurance"
		} else {
			return "standard"
		}
	} else {
		// Time-based race
		totalSessionTime := raceData.SessionTime
		if totalSessionTime <= 30*60 { // 30 minutes or less
			return "sprint"
		} else if totalSessionTime >= 120*60 { // 2+ hours
			return "endurance"
		} else {
			return "standard"
		}
	}
}

// GetPromptTemplate returns the appropriate prompt template for the race format
func GetPromptTemplate(raceFormat string) *PromptTemplate {
	switch raceFormat {
	case "sprint":
		return getSprintRaceTemplate()
	case "endurance":
		return getEnduranceRaceTemplate()
	case "standard":
		return getStandardRaceTemplate()
	default:
		return getStandardRaceTemplate()
	}
}

// getSprintRaceTemplate returns the template for sprint race analysis
func getSprintRaceTemplate() *PromptTemplate {
	return &PromptTemplate{
		RaceFormat: "sprint",
		SystemContext: `You are an expert AI race strategist specializing in SPRINT RACE strategy for sim racing.
Sprint races are short, intense battles where every position matters and there's little room for error.
Focus on aggressive, high-reward strategies that maximize position gain in minimal time.`,

		StrategyFocus: `SPRINT RACE STRATEGY PRIORITIES:
- Aggressive positioning and overtaking opportunities
- Minimal pit stops (ideally no stops or single strategic stop)
- Risk vs reward analysis for bold moves
- Track position is CRITICAL - clean air often beats tire advantage
- Immediate tire performance over long-term degradation
- Capitalize on early race chaos and positioning mistakes`,

		KeyFactors: []string{
			"Current track position and overtaking difficulty",
			"Tire performance window and degradation rate",
			"Fuel requirements (usually sufficient for race distance)",
			"Opponent aggressive moves and defensive positioning",
			"Safety car probability and strategic impact",
			"Weather window changes (can create big opportunities)",
			"DRS zones and slipstream opportunities",
		},

		DecisionCriteria: []string{
			"Can we gain positions through strategy vs on-track overtaking?",
			"Is tire degradation severe enough to warrant a pit stop?",
			"Will a pit stop cycle put us in traffic or clean air?",
			"Can we undercut or overcut nearby competitors?",
			"Is the current pace sustainable to the finish?",
			"Should we prioritize track position or tire advantage?",
		},

		OutputGuidance: `For sprint races, emphasize:
- IMMEDIATE action items that can gain positions quickly
- Bold strategic moves with calculated risks
- Specific lap timing for any pit stops (if beneficial)
- Aggressive pace targets to maximize position
- Defensive strategies if leading or in points-paying positions
- Clear go/no-go decisions for strategic gambles`,
	}
}

// getEnduranceRaceTemplate returns the template for endurance race analysis
func getEnduranceRaceTemplate() *PromptTemplate {
	return &PromptTemplate{
		RaceFormat: "endurance",
		SystemContext: `You are an expert AI race strategist specializing in ENDURANCE RACE strategy for sim racing.
Endurance races are marathon events where consistency, tire management, and strategic patience win races.
Focus on long-term strategy, efficiency, and maintaining optimal performance over extended periods.`,

		StrategyFocus: `ENDURANCE RACE STRATEGY PRIORITIES:
- Long-term tire and fuel management for optimal stint lengths
- Consistent lap times and minimizing mistakes over time
- Strategic pit stop windows to avoid traffic and optimize track position
- Driver fatigue management and performance consistency
- Weather strategy for changing conditions over long periods
- Component reliability and avoiding unnecessary stress on the car
- Multiple pit stop coordination and strategic timing`,

		KeyFactors: []string{
			"Tire degradation curves and optimal stint lengths",
			"Fuel consumption rates and efficient pit stop strategies",
			"Track position evolution over multiple pit cycles",
			"Weather forecast impact over the full race duration",
			"Driver performance consistency and fatigue factors",
			"Car reliability limits and mechanical stress management",
			"Strategic pit windows and traffic avoidance",
			"Safety car statistics and strategic impact timing",
		},

		DecisionCriteria: []string{
			"What are the optimal stint lengths for tire compounds?",
			"How can we minimize total pit stop time while maintaining performance?",
			"Should we extend current stint or pit early for strategic advantage?",
			"Can we gain positions through superior pit strategy timing?",
			"Is current pace sustainable for the full stint duration?",
			"How will weather changes affect our long-term strategy?",
			"Should we prioritize track position now or strategic position later?",
		},

		OutputGuidance: `For endurance races, emphasize:
- LONG-TERM strategic planning with multiple stint scenarios
- Conservative pace recommendations to preserve tires and fuel
- Precise pit window timing for optimal track position
- Fuel saving techniques and efficiency targets
- Tire management advice for extended stint performance
- Strategic patience - positions gained/lost over time
- Risk mitigation to avoid race-ending incidents
- Adaptive strategy for changing conditions over hours`,
	}
}

// getStandardRaceTemplate returns the template for standard/feature race analysis
func getStandardRaceTemplate() *PromptTemplate {
	return &PromptTemplate{
		RaceFormat: "standard",
		SystemContext: `You are an expert AI race strategist for sim racing, analyzing a standard feature race.
These races balance sprint-like intensity with endurance strategy elements, requiring adaptive
decision-making based on evolving race conditions and strategic opportunities.`,

		StrategyFocus: `STANDARD RACE STRATEGY PRIORITIES:
- Balanced approach between aggressive moves and consistent performance
- Strategic pit stop timing for optimal track position and tire advantage
- Adaptive strategy based on race evolution and competitor actions
- Risk management with calculated aggressive moves when beneficial
- Tire compound strategy balancing performance and degradation
- Fuel management with strategic saving when needed`,

		KeyFactors: []string{
			"Race length and optimal pit stop timing",
			"Tire performance vs degradation balance",
			"Fuel strategy and consumption management",
			"Track position and overtaking opportunities",
			"Competitor strategies and counter-moves",
			"Weather evolution and strategic impact",
			"Safety car timing and strategic opportunities",
		},

		DecisionCriteria: []string{
			"Should we prioritize track position or tire advantage?",
			"Is this the optimal pit window for our strategy?",
			"Can we gain positions through strategic timing?",
			"Should we match competitor strategies or differentiate?",
			"Is aggressive pace worth the tire degradation cost?",
			"Can we capitalize on others' strategic mistakes?",
		},

		OutputGuidance: `For standard races, provide:
- BALANCED strategic recommendations adapting to race circumstances
- Clear pit stop timing with alternative scenarios
- Pace management advice balancing speed and tire life
- Strategic options for different competitive situations
- Risk assessment for aggressive vs conservative approaches
- Adaptive guidance as race conditions evolve`,
	}
}

// PromptBuilder creates specialized prompts based on race format and situation
type PromptBuilder struct {
	template *PromptTemplate
	detector *RaceFormatDetector
}

// NewPromptBuilder creates a new prompt builder with format detection
func NewPromptBuilder(config *Config) *PromptBuilder {
	return &PromptBuilder{
		detector: NewRaceFormatDetector(config),
	}
}

// BuildSpecializedPrompt constructs a race format-specific prompt
func (pb *PromptBuilder) BuildSpecializedPrompt(raceData *RaceData, analysisType string, config *Config) (string, error) {
	// Detect race format
	raceFormat := pb.detector.DetectRaceFormat(raceData)

	// Get appropriate template
	template := GetPromptTemplate(raceFormat)
	pb.template = template

	var promptBuilder strings.Builder

	// System context specific to race format
	promptBuilder.WriteString(template.SystemContext)
	promptBuilder.WriteString("\n\n")

	// Race format-specific strategy focus
	promptBuilder.WriteString(template.StrategyFocus)
	promptBuilder.WriteString("\n\n")

	// Current race situation (standard across all formats)
	promptBuilder.WriteString("=== CURRENT RACE SITUATION ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Race Format: %s\n", strings.ToUpper(raceFormat)))
	promptBuilder.WriteString(fmt.Sprintf("Session: %s\n", raceData.SessionType))
	promptBuilder.WriteString(fmt.Sprintf("Track: %s\n", raceData.TrackName))
	promptBuilder.WriteString(fmt.Sprintf("Current Position: P%d\n", raceData.Position))

	// Race progress with format-specific context
	if raceData.TotalLaps > 0 {
		remainingLaps := raceData.TotalLaps - raceData.CurrentLap
		progressPct := float64(raceData.CurrentLap) / float64(raceData.TotalLaps) * 100
		promptBuilder.WriteString(fmt.Sprintf("Race Progress: Lap %d of %d (%.1f%% complete, %d laps remaining)\n",
			raceData.CurrentLap, raceData.TotalLaps, progressPct, remainingLaps))
	} else {
		progressPct := (raceData.SessionTime - raceData.SessionTimeLeft) / raceData.SessionTime * 100
		promptBuilder.WriteString(fmt.Sprintf("Race Progress: %.1f minutes remaining (%.1f%% complete)\n",
			raceData.SessionTimeLeft/60, progressPct))
	}

	// Car status with format-specific emphasis
	promptBuilder.WriteString("\n=== CAR STATUS ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Fuel Level: %.1f%% (Consumption: %.2f per lap)\n", raceData.FuelLevel, raceData.FuelConsumption))

	// Add fuel strategy context based on race format
	if raceFormat == "sprint" {
		fuelNeeded := float64(raceData.RemainingLaps) * raceData.FuelConsumption
		if fuelNeeded <= raceData.FuelLevel {
			promptBuilder.WriteString("✅ Fuel sufficient for sprint distance - no fuel management needed\n")
		} else {
			fuelSavingNeeded := fuelNeeded - raceData.FuelLevel
			promptBuilder.WriteString(fmt.Sprintf("⚠️ Fuel deficit: %.1f%% - requires saving %.2f per lap\n", fuelSavingNeeded, fuelSavingNeeded/float64(raceData.RemainingLaps)))
		}
	} else if raceFormat == "endurance" {
		stintsRemaining := float64(raceData.RemainingLaps) / 25.0 // Estimate 25-lap stints
		promptBuilder.WriteString(fmt.Sprintf("Estimated pit stops remaining: %.1f (based on ~25 lap stints)\n", stintsRemaining))
	}

	promptBuilder.WriteString(fmt.Sprintf("Tires: %s compound, %.1f%% wear\n", raceData.TireCompound, raceData.TireWear))

	// Add tire strategy context based on format
	if raceFormat == "sprint" && raceData.TireWear < 80 {
		promptBuilder.WriteString("🏁 Sprint race: Consider staying out on current tires if performance adequate\n")
	} else if raceFormat == "endurance" && raceData.TireWear > 60 {
		promptBuilder.WriteString("⏱️ Endurance race: Plan pit stop before tire performance cliff\n")
	}

	promptBuilder.WriteString(fmt.Sprintf("Performance: Current %.3fs | Best %.3fs | Average %.3fs\n",
		raceData.CurrentLapTime, raceData.BestLapTime, raceData.AverageLapTime))

	// Track conditions
	promptBuilder.WriteString("\n=== TRACK CONDITIONS ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Weather: %s (Forecast: %s)\n", raceData.Weather, raceData.WeatherForecast))
	promptBuilder.WriteString(fmt.Sprintf("Temperature: Track %.1f°C | Air %.1f°C\n", raceData.TrackTemp, raceData.AirTemp))

	// Safety conditions with format-specific strategic implications
	if raceData.SafetyCarActive {
		promptBuilder.WriteString("🚨 SAFETY CAR ACTIVE\n")
		if raceFormat == "sprint" {
			promptBuilder.WriteString("   Sprint Strategy: Critical pit window opportunity - positions at stake!\n")
		} else if raceFormat == "endurance" {
			promptBuilder.WriteString("   Endurance Strategy: Evaluate if pit timing aligns with long-term plan\n")
		}
	}
	if raceData.YellowFlagSector > 0 {
		promptBuilder.WriteString(fmt.Sprintf("⚠️ YELLOW FLAG Sector %d\n", raceData.YellowFlagSector))
	}

	// Opponent analysis with format-specific considerations
	if config.AnalysisPreferences.IncludeOpponentData && len(raceData.Opponents) > 0 {
		promptBuilder.WriteString("\n=== COMPETITIVE SITUATION ===\n")
		for _, opponent := range raceData.Opponents {
			gapStr := "ahead"
			gapValue := opponent.GapToPlayer
			if opponent.GapToPlayer < 0 {
				gapStr = "behind"
				gapValue = -opponent.GapToPlayer
			}
			promptBuilder.WriteString(fmt.Sprintf("P%d %s: %.3fs %s (Last: %.3fs",
				opponent.Position, opponent.Name, gapValue, gapStr, opponent.LastLapTime))

			// Add strategic context based on gaps
			if gapValue < 1.0 {
				promptBuilder.WriteString(" - DIRECT BATTLE")
			} else if gapValue < 3.0 {
				promptBuilder.WriteString(" - STRATEGIC RANGE")
			} else if gapValue < 10.0 {
				promptBuilder.WriteString(" - PIT WINDOW OVERLAP")
			}
			promptBuilder.WriteString(")\n")
		}
	}
	// Format-specific key factors analysis
	promptBuilder.WriteString(fmt.Sprintf("\n=== %s RACE KEY FACTORS ===\n", strings.ToUpper(raceFormat)))
	for _, factor := range template.KeyFactors {
		promptBuilder.WriteString(fmt.Sprintf("• %s\n", factor))
	}

	// Analysis preferences
	promptBuilder.WriteString("\n=== STRATEGY PREFERENCES ===\n")
	if config.AnalysisPreferences.PrioritizeConsistency {
		promptBuilder.WriteString("Driver Preference: Consistency and reliability over aggressive moves\n")
	} else {
		promptBuilder.WriteString("Driver Preference: Aggressive strategy for maximum position gain\n")
	}
	promptBuilder.WriteString(fmt.Sprintf("Safety Margin: %.1f%% extra fuel/tire life buffer\n", (config.AnalysisPreferences.SafetyMargin-1)*100))
	// Format-specific decision criteria
	promptBuilder.WriteString(fmt.Sprintf("\n=== %s DECISION CRITERIA ===\n", strings.ToUpper(raceFormat)))
	for _, criteria := range template.DecisionCriteria {
		promptBuilder.WriteString(fmt.Sprintf("• %s\n", criteria))
	}

	// Analysis type specification
	promptBuilder.WriteString(fmt.Sprintf("\n=== ANALYSIS REQUEST ===\n"))
	switch analysisType {
	case "critical":
		promptBuilder.WriteString("🚨 CRITICAL SITUATION - Immediate strategic decision required!\n")
		if raceFormat == "sprint" {
			promptBuilder.WriteString("Sprint urgency: Every second counts - bold action may be needed.\n")
		} else if raceFormat == "endurance" {
			promptBuilder.WriteString("Endurance perspective: Consider long-term implications of immediate action.\n")
		}
	case "pit_decision":
		promptBuilder.WriteString("🏁 PIT STOP DECISION ANALYSIS\n")
		if raceFormat == "sprint" {
			promptBuilder.WriteString("Sprint context: Pit only if clear strategic advantage - track position is critical.\n")
		} else if raceFormat == "endurance" {
			promptBuilder.WriteString("Endurance context: Optimize pit timing for long-term stint strategy.\n")
		}
	case "routine":
		promptBuilder.WriteString("📊 ROUTINE STRATEGIC GUIDANCE\n")
		promptBuilder.WriteString(fmt.Sprintf("%s race management: ", strings.Title(raceFormat)))
		if raceFormat == "sprint" {
			promptBuilder.WriteString("Focus on immediate opportunities and position maximization.\n")
		} else if raceFormat == "endurance" {
			promptBuilder.WriteString("Focus on consistency and long-term strategic positioning.\n")
		} else {
			promptBuilder.WriteString("Balance immediate opportunities with long-term strategy.\n")
		}
	}

	// Format-specific output guidance
	promptBuilder.WriteString("\n=== OUTPUT REQUIREMENTS ===\n")
	promptBuilder.WriteString(template.OutputGuidance)
	promptBuilder.WriteString("\n\n")

	// JSON format specification (enhanced for race format)
	promptBuilder.WriteString("Provide your analysis in the following JSON format:\n")
	jsonFormat := `{
  "current_situation": "Brief summary emphasizing %s race context",
  "primary_strategy": "Main strategic recommendation for %s race format",
  "confidence": 0.85,
  "race_format": "%s",
  "pit_window_open": true/false,
  "recommended_lap": 15,
  "tire_recommendation": "soft/medium/hard/wet",
  "fuel_strategy": "Fuel management specific to %s races",
  "immediate_actions": ["Action 1 for %s context", "Action 2"],
  "lap_targets": {"current_stint": 1.23.456, "target_pace": 1.22.123},
  "risk_factors": ["Risk 1 in %s format", "Risk 2"],
  "opportunities": ["Opportunity 1 for %s races", "Opportunity 2"],
  "strategic_timeline": "Next 5-10 laps plan for %s race",
  "estimated_finish_position": 5,
  "estimated_finish_time": "1:35:42"
}`

	formattedJSON := fmt.Sprintf(jsonFormat,
		raceFormat, raceFormat, raceFormat, raceFormat,
		raceFormat, raceFormat, raceFormat, raceFormat)
	promptBuilder.WriteString(formattedJSON)

	return promptBuilder.String(), nil
}

// GetRaceFormatAnalysis provides format-specific strategic analysis
func (pb *PromptBuilder) GetRaceFormatAnalysis(raceFormat string) string {
	template := GetPromptTemplate(raceFormat)

	var analysis strings.Builder
	analysis.WriteString(fmt.Sprintf("=== %s RACE ANALYSIS ===\n", strings.ToUpper(raceFormat)))
	analysis.WriteString(template.StrategyFocus)
	analysis.WriteString("\n\nKey Strategic Factors:\n")
	for _, factor := range template.KeyFactors {
		analysis.WriteString(fmt.Sprintf("• %s\n", factor))
	}

	analysis.WriteString("\nDecision Framework:\n")
	for _, criteria := range template.DecisionCriteria {
		analysis.WriteString(fmt.Sprintf("• %s\n", criteria))
	}

	return analysis.String()
}
