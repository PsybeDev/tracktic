package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/genai"
)

// StrategyEngine provides AI-powered race strategy analysis using Google Gemini
type StrategyEngine struct {
	client        *GeminiClient
	config        *Config
	context       context.Context
	promptBuilder *PromptBuilder

	// Rate limiting and error handling
	rateLimiter     *RateLimiter
	errorClassifier *ErrorClassifier
	retryPolicy     *RetryPolicy
	errorReporter   *ErrorReporter

	// Advanced caching system for strategy results
	cache *StrategyCache
}

// RaceData represents the current race situation data
type RaceData struct {
	// Session Information
	SessionType     string  `json:"session_type"`      // "practice", "qualifying", "race"
	SessionTime     float64 `json:"session_time"`      // Current session time in seconds
	SessionTimeLeft float64 `json:"session_time_left"` // Time remaining in seconds

	// Car Status
	CurrentLap      int     `json:"current_lap"`
	Position        int     `json:"position"`
	FuelLevel       float64 `json:"fuel_level"`       // Current fuel level (liters or percentage)
	FuelConsumption float64 `json:"fuel_consumption"` // Average fuel per lap
	TireWear        float64 `json:"tire_wear"`        // Tire wear percentage (0-100)
	TireCompound    string  `json:"tire_compound"`    // Current tire compound

	// Performance Data
	CurrentLapTime float64   `json:"current_lap_time"` // Current lap time in seconds
	BestLapTime    float64   `json:"best_lap_time"`    // Personal best lap time
	AverageLapTime float64   `json:"average_lap_time"` // Average lap time over recent laps
	RecentLapTimes []float64 `json:"recent_lap_times"` // Last 5-10 lap times

	// Track Conditions
	TrackName       string  `json:"track_name"`
	TrackTemp       float64 `json:"track_temp"`       // Track temperature in Celsius
	AirTemp         float64 `json:"air_temp"`         // Air temperature in Celsius
	Weather         string  `json:"weather"`          // "dry", "light_rain", "heavy_rain"
	WeatherForecast string  `json:"weather_forecast"` // Weather prediction

	// Race Situation
	TotalLaps        int            `json:"total_laps"`     // Total race laps (0 for time-based)
	RemainingLaps    int            `json:"remaining_laps"` // Estimated laps remaining
	SafetyCarActive  bool           `json:"safety_car_active"`
	YellowFlagSector int            `json:"yellow_flag_sector"` // 0 = no yellow, 1-3 = sector
	Opponents        []OpponentData `json:"opponents"`
}

// OpponentData represents information about nearby competitors
type OpponentData struct {
	Position     int     `json:"position"`
	Name         string  `json:"name"`
	GapToPlayer  float64 `json:"gap_to_player"` // Gap in seconds (negative if behind)
	LastLapTime  float64 `json:"last_lap_time"`
	TireAge      int     `json:"tire_age"`       // Laps on current tires
	RecentPitLap int     `json:"recent_pit_lap"` // Lap of most recent pit stop (0 if none)
}

// StrategyAnalysis represents the AI's strategic recommendation
type StrategyAnalysis struct {
	// Overall Assessment
	CurrentSituation string  `json:"current_situation"` // Brief situation summary
	PrimaryStrategy  string  `json:"primary_strategy"`  // Main strategic recommendation
	Confidence       float64 `json:"confidence"`        // Confidence level (0-1)
	RaceFormat       string  `json:"race_format"`       // Detected race format (sprint/endurance/standard)

	// Specific Recommendations
	PitWindowOpen      bool   `json:"pit_window_open"`     // Should pit soon
	RecommendedLap     int    `json:"recommended_lap"`     // Recommended pit lap (0 if no pit needed)
	TireRecommendation string `json:"tire_recommendation"` // Recommended tire compound
	FuelStrategy       string `json:"fuel_strategy"`       // Fuel management recommendation

	// Tactical Advice
	ImmediateActions  []string           `json:"immediate_actions"`  // Actions to take right now
	LapTargets        map[string]float64 `json:"lap_targets"`        // Target lap times for different scenarios
	RiskFactors       []string           `json:"risk_factors"`       // Current risks to be aware of
	Opportunities     []string           `json:"opportunities"`      // Current opportunities to capitalize on
	StrategicTimeline string             `json:"strategic_timeline"` // Next 5-10 laps strategic plan

	// Predictions
	EstimatedFinishPosition int    `json:"estimated_finish_position"`
	EstimatedFinishTime     string `json:"estimated_finish_time"`

	// Context
	Timestamp    time.Time `json:"timestamp"`
	RequestID    string    `json:"request_id"`
	AnalysisType string    `json:"analysis_type"` // "routine", "critical", "pit_decision"
}

// NewStrategyEngine creates a new strategy engine instance
func NewStrategyEngine(ctx context.Context, config *Config) (*StrategyEngine, error) {
	client, err := NewGeminiClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Create cache with configuration
	cacheConfig := DefaultCacheConfig()
	if config.CacheConfig != nil {
		cacheConfig = config.CacheConfig
	}

	return &StrategyEngine{
		client:          client,
		config:          config,
		context:         ctx,
		promptBuilder:   NewPromptBuilder(config),
		rateLimiter:     NewRateLimiter(config.MaxRequestsPerMinute, config.BurstLimit),
		errorClassifier: NewErrorClassifier(),
		retryPolicy:     NewDefaultRetryPolicy(),
		errorReporter:   NewErrorReporter(50), // Keep last 50 errors
		cache:           NewStrategyCache(cacheConfig),
	}, nil
}

// Close releases resources used by the strategy engine
func (se *StrategyEngine) Close() error {
	// Close the cache to stop background processes
	if se.cache != nil {
		se.cache.Close()
	}

	return se.client.Close()
}

// GetRateLimiterStats returns current rate limiter statistics
func (se *StrategyEngine) GetRateLimiterStats() RateLimiterStats {
	return se.rateLimiter.GetStats()
}

// GetErrorStats returns current error statistics
func (se *StrategyEngine) GetErrorStats() map[ErrorType]int {
	return se.errorReporter.GetErrorStats()
}

// GetRecentErrors returns the most recent errors
func (se *StrategyEngine) GetRecentErrors(limit int) []*StrategyError {
	return se.errorReporter.GetRecentErrors(limit)
}

// GetCacheStats returns current cache statistics
func (se *StrategyEngine) GetCacheStats() CacheStats {
	return se.cache.GetStats()
}

// InvalidateCacheByTag removes cached entries with the specified tag
func (se *StrategyEngine) InvalidateCacheByTag(tag string) int {
	return se.cache.RemoveByTag(tag)
}

// ClearCache removes all cached entries
func (se *StrategyEngine) ClearCache() {
	se.cache.Clear()
}

// InvalidateOldLapData removes cached data from previous laps to ensure fresh analysis
func (se *StrategyEngine) InvalidateOldLapData(currentLap int) int {
	var removedCount int

	// Remove data from laps that are more than 2 laps old
	for lap := 1; lap < currentLap-2; lap++ {
		tag := fmt.Sprintf("lap_%d", lap)
		removedCount += se.cache.RemoveByTag(tag)
	}

	return removedCount
}

// UpdateConfig updates the strategy engine configuration
func (se *StrategyEngine) UpdateConfig(config *Config) error {
	if err := se.client.UpdateConfig(config); err != nil {
		return fmt.Errorf("failed to update client config: %w", err)
	}

	se.config = config

	// Update rate limiter if the limits changed
	if se.rateLimiter.maxRequests != config.MaxRequestsPerMinute ||
		se.rateLimiter.burstLimit != config.BurstLimit {
		se.rateLimiter = NewRateLimiter(config.MaxRequestsPerMinute, config.BurstLimit)
	}

	// Update prompt builder
	se.promptBuilder = NewPromptBuilder(config)

	return nil
}

// HealthCheck performs a comprehensive health check of the strategy engine
func (se *StrategyEngine) HealthCheck() error {
	// Check Gemini client health
	if err := se.client.HealthCheck(); err != nil {
		return fmt.Errorf("Gemini client health check failed: %w", err)
	}

	// Check rate limiter stats
	stats := se.rateLimiter.GetStats()
	if stats.AvailableTokens < 0 {
		return fmt.Errorf("rate limiter in invalid state: %d tokens available", stats.AvailableTokens)
	}

	// Check recent error rate
	recentErrors := se.errorReporter.GetRecentErrors(10)
	criticalErrors := 0
	for _, err := range recentErrors {
		if err.Type == ErrorTypeAuthentication || err.Type == ErrorTypeQuotaExceeded {
			criticalErrors++
		}
	}

	if criticalErrors > 5 {
		return fmt.Errorf("too many critical errors in recent history: %d", criticalErrors)
	}

	return nil
}

// AnalyzeStrategy performs a comprehensive race strategy analysis
func (se *StrategyEngine) AnalyzeStrategy(raceData *RaceData, analysisType string) (*StrategyAnalysis, error) {
	// Check cache first (if enabled and data is recent)
	if se.config.EnableCaching {
		cacheKey := se.generateCacheKey(raceData, analysisType)
		if cached, exists := se.cache.Get(cacheKey); exists {
			if analysis, ok := cached.(*StrategyAnalysis); ok {
				log.Printf("Returning cached strategy analysis for %s", analysisType)
				return analysis, nil
			}
		}
	}
	// Construct the prompt for Gemini using specialized prompt builder
	prompt, err := se.promptBuilder.BuildSpecializedPrompt(raceData, analysisType, se.config)
	if err != nil {
		return nil, fmt.Errorf("failed to construct specialized prompt: %w", err)
	}

	// Make the API request to Gemini
	response, err := se.requestAnalysis(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis from Gemini: %w", err)
	}

	// Parse the response into structured data
	analysis, err := se.parseResponse(response, raceData, analysisType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	// Cache the result
	if se.config.EnableCaching {
		se.cacheAnalysis(raceData, analysisType, analysis)
	}

	return analysis, nil
}

// constructPrompt builds a detailed prompt for the Gemini API
func (se *StrategyEngine) constructPrompt(raceData *RaceData, analysisType string) (string, error) {
	var promptBuilder strings.Builder

	// System context and role definition
	promptBuilder.WriteString("You are an expert AI race strategist for sim racing. ")
	promptBuilder.WriteString("Analyze the current race situation and provide strategic recommendations in JSON format.\n\n")

	// Current race situation
	promptBuilder.WriteString("=== RACE SITUATION ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Session: %s\n", raceData.SessionType))
	promptBuilder.WriteString(fmt.Sprintf("Track: %s\n", raceData.TrackName))
	promptBuilder.WriteString(fmt.Sprintf("Current Position: %d\n", raceData.Position))
	promptBuilder.WriteString(fmt.Sprintf("Current Lap: %d", raceData.CurrentLap))

	if raceData.TotalLaps > 0 {
		promptBuilder.WriteString(fmt.Sprintf(" of %d\n", raceData.TotalLaps))
	} else {
		promptBuilder.WriteString(fmt.Sprintf(" (Time remaining: %.1f minutes)\n", raceData.SessionTimeLeft/60))
	}

	// Car status
	promptBuilder.WriteString("\n=== CAR STATUS ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Fuel Level: %.1f%% (Consumption: %.2f per lap)\n", raceData.FuelLevel, raceData.FuelConsumption))
	promptBuilder.WriteString(fmt.Sprintf("Tires: %s compound, %.1f%% wear\n", raceData.TireCompound, raceData.TireWear))
	promptBuilder.WriteString(fmt.Sprintf("Current Lap Time: %.3fs (Best: %.3fs, Average: %.3fs)\n",
		raceData.CurrentLapTime, raceData.BestLapTime, raceData.AverageLapTime))

	// Track conditions
	promptBuilder.WriteString("\n=== CONDITIONS ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Weather: %s (Forecast: %s)\n", raceData.Weather, raceData.WeatherForecast))
	promptBuilder.WriteString(fmt.Sprintf("Track Temperature: %.1f°C, Air: %.1f°C\n", raceData.TrackTemp, raceData.AirTemp))

	// Safety conditions
	if raceData.SafetyCarActive {
		promptBuilder.WriteString("⚠️ SAFETY CAR ACTIVE\n")
	}
	if raceData.YellowFlagSector > 0 {
		promptBuilder.WriteString(fmt.Sprintf("⚠️ YELLOW FLAG in Sector %d\n", raceData.YellowFlagSector))
	}

	// Opponent information (if available and enabled)
	if se.config.AnalysisPreferences.IncludeOpponentData && len(raceData.Opponents) > 0 {
		promptBuilder.WriteString("\n=== NEARBY OPPONENTS ===\n")
		for _, opponent := range raceData.Opponents {
			gapStr := "ahead"
			if opponent.GapToPlayer < 0 {
				gapStr = "behind"
			}
			promptBuilder.WriteString(fmt.Sprintf("P%d %s: %.3fs %s, Last lap: %.3fs\n",
				opponent.Position, opponent.Name, abs(opponent.GapToPlayer), gapStr, opponent.LastLapTime))
		}
	}

	// Analysis preferences
	promptBuilder.WriteString("\n=== STRATEGY PREFERENCES ===\n")
	promptBuilder.WriteString(fmt.Sprintf("Race Format: %s\n", se.config.AnalysisPreferences.RaceFormat))
	if se.config.AnalysisPreferences.PrioritizeConsistency {
		promptBuilder.WriteString("Priority: Consistency over aggressive strategy\n")
	} else {
		promptBuilder.WriteString("Priority: Aggressive strategy for maximum position gain\n")
	}
	promptBuilder.WriteString(fmt.Sprintf("Safety Margin: %.1f%% extra fuel/tire life\n", (se.config.AnalysisPreferences.SafetyMargin-1)*100))

	// Specific analysis request
	promptBuilder.WriteString(fmt.Sprintf("\n=== ANALYSIS REQUEST ===\n"))
	switch analysisType {
	case "critical":
		promptBuilder.WriteString("This is a CRITICAL situation requiring immediate strategic decision.\n")
	case "pit_decision":
		promptBuilder.WriteString("Focus on pit stop timing and tire/fuel strategy.\n")
	case "routine":
		promptBuilder.WriteString("Provide routine strategic guidance and lap targets.\n")
	default:
		promptBuilder.WriteString("Provide comprehensive strategic analysis.\n")
	}

	// Response format specification
	promptBuilder.WriteString("\nProvide your analysis in the following JSON format:\n")
	promptBuilder.WriteString(`{
  "current_situation": "Brief summary of the current race situation",
  "primary_strategy": "Main strategic recommendation",
  "confidence": 0.85,
  "pit_window_open": true/false,
  "recommended_lap": 15,
  "tire_recommendation": "soft/medium/hard/wet",
  "fuel_strategy": "Fuel management recommendation",
  "immediate_actions": ["Action 1", "Action 2"],
  "lap_targets": {"current_stint": 1.23.456, "after_pit": 1.22.123},
  "risk_factors": ["Risk 1", "Risk 2"],
  "opportunities": ["Opportunity 1", "Opportunity 2"],
  "estimated_finish_position": 5,
  "estimated_finish_time": "1:35:42"
}`)

	return promptBuilder.String(), nil
}

// requestAnalysis sends the prompt to Gemini with rate limiting and comprehensive error handling
func (se *StrategyEngine) requestAnalysis(prompt string) (string, error) {
	// Apply rate limiting
	if err := se.rateLimiter.Wait(se.context); err != nil {
		classifiedError := se.errorClassifier.ClassifyError(err, map[string]interface{}{
			"operation":     "rate_limit_wait",
			"prompt_length": len(prompt),
		})
		se.errorReporter.ReportError(classifiedError)
		return "", classifiedError
	}

	ctx, cancel := context.WithTimeout(se.context, se.config.RequestTimeout)
	defer cancel()

	// Create the generation config
	temperature := float32(se.config.Temperature)
	topP := float32(se.config.TopP)
	topK := float32(se.config.TopK)
	maxTokens := int32(se.config.MaxTokens)

	genConfig := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		TopP:            &topP,
		TopK:            &topK,
		MaxOutputTokens: maxTokens,
	}

	// Enhanced retry logic with comprehensive error handling
	var lastStrategyError *StrategyError
	for attempt := 0; attempt <= se.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			log.Printf("Strategy analysis attempt %d/%d (previous error: %s)",
				attempt+1, se.config.RetryAttempts+1, lastStrategyError.Error())

			// Calculate backoff delay
			backoffDelay := se.retryPolicy.CalculateBackoff(lastStrategyError, attempt-1)

			select {
			case <-time.After(backoffDelay):
				// Continue with retry
			case <-ctx.Done():
				finalError := se.errorClassifier.ClassifyError(ctx.Err(), map[string]interface{}{
					"operation":      "backoff_wait",
					"attempt":        attempt,
					"total_attempts": se.config.RetryAttempts + 1,
				})
				se.errorReporter.ReportError(finalError)
				return "", finalError
			}
		}

		result, err := se.client.client.Models.GenerateContent(ctx, se.config.Model, []*genai.Content{
			{
				Parts: []*genai.Part{
					{Text: prompt},
				},
			},
		}, genConfig)

		if err != nil {
			// Classify the error
			strategyError := se.errorClassifier.ClassifyError(err, map[string]interface{}{
				"operation":      "gemini_api_request",
				"attempt":        attempt + 1,
				"total_attempts": se.config.RetryAttempts + 1,
				"model":          se.config.Model,
				"prompt_length":  len(prompt),
			})

			lastStrategyError = strategyError
			se.errorReporter.ReportError(strategyError)

			// Check if we should retry this error
			if !se.retryPolicy.ShouldRetry(strategyError, attempt) {
				log.Printf("Not retrying error of type %s after %d attempts", strategyError.Type.String(), attempt+1)
				return "", strategyError
			}

			log.Printf("Gemini API request failed (attempt %d): %v (will retry)", attempt+1, strategyError)
			continue
		}

		// Validate response structure
		if result == nil {
			strategyError := &StrategyError{
				Type:      ErrorTypeInvalidRequest,
				Code:      "EMPTY_RESPONSE",
				Message:   "Received nil response from Gemini API",
				Retryable: true,
				Context: map[string]interface{}{
					"operation": "response_validation",
					"attempt":   attempt + 1,
				},
				Timestamp: time.Now(),
			}

			lastStrategyError = strategyError
			se.errorReporter.ReportError(strategyError)

			if !se.retryPolicy.ShouldRetry(strategyError, attempt) {
				return "", strategyError
			}
			continue
		}

		if len(result.Candidates) == 0 {
			strategyError := &StrategyError{
				Type:      ErrorTypeInvalidRequest,
				Code:      "NO_CANDIDATES",
				Message:   "No candidates in Gemini API response",
				Retryable: true,
				Context: map[string]interface{}{
					"operation": "response_validation",
					"attempt":   attempt + 1,
				},
				Timestamp: time.Now(),
			}

			lastStrategyError = strategyError
			se.errorReporter.ReportError(strategyError)

			if !se.retryPolicy.ShouldRetry(strategyError, attempt) {
				return "", strategyError
			}
			continue
		}

		candidate := result.Candidates[0]
		if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
			strategyError := &StrategyError{
				Type:      ErrorTypeInvalidRequest,
				Code:      "NO_CONTENT",
				Message:   "No content in Gemini response candidate",
				Retryable: true,
				Context: map[string]interface{}{
					"operation": "response_validation",
					"attempt":   attempt + 1,
				},
				Timestamp: time.Now(),
			}

			lastStrategyError = strategyError
			se.errorReporter.ReportError(strategyError)

			if !se.retryPolicy.ShouldRetry(strategyError, attempt) {
				return "", strategyError
			}
			continue
		}

		// Extract text from the response
		var responseText strings.Builder
		for _, part := range candidate.Content.Parts {
			responseText.WriteString(part.Text)
		}

		responseString := responseText.String()
		if responseString == "" {
			strategyError := &StrategyError{
				Type:      ErrorTypeInvalidRequest,
				Code:      "EMPTY_CONTENT",
				Message:   "Empty text content in Gemini response",
				Retryable: true,
				Context: map[string]interface{}{
					"operation": "response_validation",
					"attempt":   attempt + 1,
				},
				Timestamp: time.Now(),
			}

			lastStrategyError = strategyError
			se.errorReporter.ReportError(strategyError)

			if !se.retryPolicy.ShouldRetry(strategyError, attempt) {
				return "", strategyError
			}
			continue
		}

		// Success - log attempt count if retries were needed
		if attempt > 0 {
			log.Printf("Strategy analysis succeeded on attempt %d/%d", attempt+1, se.config.RetryAttempts+1)
		}

		return responseString, nil
	}

	// If we get here, all retry attempts failed
	if lastStrategyError != nil {
		return "", fmt.Errorf("failed to get response from Gemini after %d attempts: %w", se.config.RetryAttempts+1, lastStrategyError)
	}

	// This should never happen, but provide a fallback
	finalError := &StrategyError{
		Type:      ErrorTypeUnknown,
		Code:      "RETRY_EXHAUSTED",
		Message:   fmt.Sprintf("All %d retry attempts exhausted", se.config.RetryAttempts+1),
		Retryable: false,
		Context: map[string]interface{}{
			"operation":      "request_analysis",
			"total_attempts": se.config.RetryAttempts + 1,
		},
		Timestamp: time.Now(),
	}
	se.errorReporter.ReportError(finalError)
	return "", finalError
}

// parseResponse converts the Gemini JSON response into a StrategyAnalysis struct
func (se *StrategyEngine) parseResponse(response string, raceData *RaceData, analysisType string) (*StrategyAnalysis, error) {
	// Try to extract JSON from the response (in case there's extra text)
	response = strings.TrimSpace(response)

	// Find JSON object boundaries
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonResponse := response[startIdx : endIdx+1]

	// Parse the JSON response
	var rawAnalysis map[string]interface{}
	if err := json.Unmarshal([]byte(jsonResponse), &rawAnalysis); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to StrategyAnalysis struct with validation
	analysis := &StrategyAnalysis{
		Timestamp:    time.Now(),
		RequestID:    generateRequestID(),
		AnalysisType: analysisType,
	}

	// Extract and validate fields
	if val, ok := rawAnalysis["current_situation"].(string); ok {
		analysis.CurrentSituation = val
	}

	if val, ok := rawAnalysis["primary_strategy"].(string); ok {
		analysis.PrimaryStrategy = val
	}

	if val, ok := rawAnalysis["confidence"].(float64); ok {
		analysis.Confidence = val
	}

	if val, ok := rawAnalysis["race_format"].(string); ok {
		analysis.RaceFormat = val
	}

	if val, ok := rawAnalysis["pit_window_open"].(bool); ok {
		analysis.PitWindowOpen = val
	}

	if val, ok := rawAnalysis["recommended_lap"].(float64); ok {
		analysis.RecommendedLap = int(val)
	}

	if val, ok := rawAnalysis["tire_recommendation"].(string); ok {
		analysis.TireRecommendation = val
	}

	if val, ok := rawAnalysis["fuel_strategy"].(string); ok {
		analysis.FuelStrategy = val
	}

	// Parse arrays
	if val, ok := rawAnalysis["immediate_actions"].([]interface{}); ok {
		for _, action := range val {
			if actionStr, ok := action.(string); ok {
				analysis.ImmediateActions = append(analysis.ImmediateActions, actionStr)
			}
		}
	}

	if val, ok := rawAnalysis["risk_factors"].([]interface{}); ok {
		for _, risk := range val {
			if riskStr, ok := risk.(string); ok {
				analysis.RiskFactors = append(analysis.RiskFactors, riskStr)
			}
		}
	}

	if val, ok := rawAnalysis["opportunities"].([]interface{}); ok {
		for _, opp := range val {
			if oppStr, ok := opp.(string); ok {
				analysis.Opportunities = append(analysis.Opportunities, oppStr)
			}
		}
	}

	// Parse lap targets map
	if val, ok := rawAnalysis["lap_targets"].(map[string]interface{}); ok {
		analysis.LapTargets = make(map[string]float64)
		for key, target := range val {
			if targetFloat, ok := target.(float64); ok {
				analysis.LapTargets[key] = targetFloat
			}
		}
	}

	if val, ok := rawAnalysis["strategic_timeline"].(string); ok {
		analysis.StrategicTimeline = val
	}

	if val, ok := rawAnalysis["estimated_finish_position"].(float64); ok {
		analysis.EstimatedFinishPosition = int(val)
	}

	if val, ok := rawAnalysis["estimated_finish_time"].(string); ok {
		analysis.EstimatedFinishTime = val
	}

	// Validate essential fields
	if analysis.CurrentSituation == "" {
		analysis.CurrentSituation = "Unable to assess current situation"
	}
	if analysis.PrimaryStrategy == "" {
		analysis.PrimaryStrategy = "Continue current strategy"
	}
	if analysis.Confidence == 0 {
		analysis.Confidence = 0.5 // Default moderate confidence
	}

	return analysis, nil
}

// Helper functions

func (se *StrategyEngine) generateCacheKey(raceData *RaceData, analysisType string) string {
	return fmt.Sprintf("%s_%d_%d_%.1f_%.1f",
		analysisType, raceData.CurrentLap, raceData.Position,
		raceData.FuelLevel, raceData.TireWear)
}

func (se *StrategyEngine) cacheAnalysis(raceData *RaceData, analysisType string, analysis *StrategyAnalysis) {
	cacheKey := se.generateCacheKey(raceData, analysisType)

	// Determine cache type based on analysis type
	var cacheType CacheType
	switch analysisType {
	case "pit_decision":
		cacheType = CacheTypePitTiming
	case "recommendation":
		cacheType = CacheTypeRecommendation
	default:
		cacheType = CacheTypeStrategy
	}

	// Generate tags for organized cache invalidation
	tags := []string{
		fmt.Sprintf("lap_%d", raceData.CurrentLap),
		fmt.Sprintf("position_%d", raceData.Position),
		fmt.Sprintf("session_%s", raceData.SessionType),
	}

	// Store in cache
	se.cache.PutWithKey(cacheKey, cacheType, analysis, tags)
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
