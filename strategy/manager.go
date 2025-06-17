package strategy

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// StrategyManager manages the strategy engine lifecycle and provides high-level operations
type StrategyManager struct {
	engine    *StrategyEngine
	config    *Config
	ctx       context.Context
	cancelFn  context.CancelFunc
	mutex     sync.RWMutex
	isRunning bool

	// Channels for communication
	analysisRequests chan AnalysisRequest
	analysisResults  chan AnalysisResult

	// Last analysis cache
	lastAnalysis   *StrategyAnalysis
	lastUpdateTime time.Time
}

// AnalysisRequest represents a request for strategy analysis
type AnalysisRequest struct {
	RaceData     *RaceData
	AnalysisType string
	Priority     int // Higher numbers = higher priority
	RequestTime  time.Time
	ResponseChan chan AnalysisResult
}

// AnalysisResult represents the result of a strategy analysis
type AnalysisResult struct {
	Analysis  *StrategyAnalysis
	Error     error
	Duration  time.Duration
	RequestID string
}

// NewStrategyManager creates a new strategy manager instance
func NewStrategyManager(config *Config) (*StrategyManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	engine, err := NewStrategyEngine(ctx, config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create strategy engine: %w", err)
	}

	manager := &StrategyManager{
		engine:           engine,
		config:           config,
		ctx:              ctx,
		cancelFn:         cancel,
		analysisRequests: make(chan AnalysisRequest, 10),
		analysisResults:  make(chan AnalysisResult, 10),
	}

	// Start the analysis worker
	go manager.analysisWorker()

	manager.isRunning = true
	log.Println("Strategy manager started successfully")

	return manager, nil
}

// Close shuts down the strategy manager and releases resources
func (sm *StrategyManager) Close() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.isRunning {
		return nil
	}

	sm.isRunning = false
	sm.cancelFn()

	// Close channels
	close(sm.analysisRequests)

	// Close the engine
	if err := sm.engine.Close(); err != nil {
		log.Printf("Warning: failed to close strategy engine: %v", err)
	}

	log.Println("Strategy manager stopped")
	return nil
}

// RequestAnalysis submits a request for strategy analysis
func (sm *StrategyManager) RequestAnalysis(raceData *RaceData, analysisType string) (*StrategyAnalysis, error) {
	sm.mutex.RLock()
	if !sm.isRunning {
		sm.mutex.RUnlock()
		return nil, fmt.Errorf("strategy manager is not running")
	}
	sm.mutex.RUnlock()

	responseChan := make(chan AnalysisResult, 1)

	request := AnalysisRequest{
		RaceData:     raceData,
		AnalysisType: analysisType,
		Priority:     getPriority(analysisType),
		RequestTime:  time.Now(),
		ResponseChan: responseChan,
	}

	// Send request with timeout
	select {
	case sm.analysisRequests <- request:
		// Request queued successfully
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("request queue is full, analysis request timed out")
	case <-sm.ctx.Done():
		return nil, fmt.Errorf("strategy manager is shutting down")
	}

	// Wait for response
	select {
	case result := <-responseChan:
		if result.Error != nil {
			return nil, result.Error
		}

		// Update last analysis cache
		sm.mutex.Lock()
		sm.lastAnalysis = result.Analysis
		sm.lastUpdateTime = time.Now()
		sm.mutex.Unlock()

		log.Printf("Strategy analysis completed in %v (confidence: %.2f)",
			result.Duration, result.Analysis.Confidence)

		return result.Analysis, nil

	case <-time.After(sm.config.RequestTimeout + 10*time.Second):
		return nil, fmt.Errorf("analysis request timed out")
	case <-sm.ctx.Done():
		return nil, fmt.Errorf("strategy manager is shutting down")
	}
}

// GetLastAnalysis returns the most recent strategy analysis
func (sm *StrategyManager) GetLastAnalysis() (*StrategyAnalysis, time.Time) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.lastAnalysis, sm.lastUpdateTime
}

// IsHealthy checks if the strategy manager and underlying services are healthy
func (sm *StrategyManager) IsHealthy() error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if !sm.isRunning {
		return fmt.Errorf("strategy manager is not running")
	}

	// Check Gemini API health
	if err := sm.engine.client.HealthCheck(); err != nil {
		return fmt.Errorf("Gemini API health check failed: %w", err)
	}

	return nil
}

// UpdateConfig updates the strategy manager configuration
func (sm *StrategyManager) UpdateConfig(newConfig *Config) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update engine config
	if err := sm.engine.client.UpdateConfig(newConfig); err != nil {
		return fmt.Errorf("failed to update engine config: %w", err)
	}

	sm.config = newConfig
	log.Println("Strategy manager configuration updated")

	return nil
}

// GetConfig returns the current configuration
func (sm *StrategyManager) GetConfig() *Config {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *sm.config
	return &configCopy
}

// analysisWorker runs in a goroutine to process analysis requests
func (sm *StrategyManager) analysisWorker() {
	log.Println("Strategy analysis worker started")

	for {
		select {
		case request, ok := <-sm.analysisRequests:
			if !ok {
				log.Println("Analysis worker stopping: request channel closed")
				return
			}

			sm.processAnalysisRequest(request)

		case <-sm.ctx.Done():
			log.Println("Analysis worker stopping: context cancelled")
			return
		}
	}
}

// processAnalysisRequest handles a single analysis request
func (sm *StrategyManager) processAnalysisRequest(request AnalysisRequest) {
	startTime := time.Now()
	requestID := generateRequestID()

	log.Printf("Processing strategy analysis request %s (type: %s, priority: %d)",
		requestID, request.AnalysisType, request.Priority)

	analysis, err := sm.engine.AnalyzeStrategy(request.RaceData, request.AnalysisType)
	duration := time.Since(startTime)

	result := AnalysisResult{
		Analysis:  analysis,
		Error:     err,
		Duration:  duration,
		RequestID: requestID,
	}

	// Send result back
	select {
	case request.ResponseChan <- result:
		// Result sent successfully
	case <-time.After(1 * time.Second):
		log.Printf("Warning: failed to send analysis result for request %s (timeout)", requestID)
	case <-sm.ctx.Done():
		// Manager is shutting down
		return
	}

	if err != nil {
		log.Printf("Strategy analysis request %s failed: %v", requestID, err)
	} else {
		log.Printf("Strategy analysis request %s completed successfully", requestID)
	}
}

// Helper functions

// getPriority returns the priority level for different analysis types
func getPriority(analysisType string) int {
	switch analysisType {
	case "critical":
		return 100
	case "pit_decision":
		return 80
	case "safety_car":
		return 70
	case "weather_change":
		return 60
	case "routine":
		return 30
	default:
		return 50
	}
}

// CreateSampleRaceData creates sample race data for testing purposes
func CreateSampleRaceData() *RaceData {
	return &RaceData{
		SessionType:      "race",
		SessionTime:      1800,
		SessionTimeLeft:  900,
		CurrentLap:       12,
		Position:         4,
		FuelLevel:        65.5,
		FuelConsumption:  2.8,
		TireWear:         45.0,
		TireCompound:     "medium",
		CurrentLapTime:   84.123,
		BestLapTime:      83.456,
		AverageLapTime:   84.789,
		RecentLapTimes:   []float64{84.123, 84.567, 83.998, 84.234, 84.456},
		TrackName:        "Silverstone",
		TrackTemp:        32.5,
		AirTemp:          24.8,
		Weather:          "dry",
		WeatherForecast:  "dry",
		TotalLaps:        25,
		RemainingLaps:    13,
		SafetyCarActive:  false,
		YellowFlagSector: 0,
		Opponents: []OpponentData{
			{
				Position:     3,
				Name:         "Hamilton",
				GapToPlayer:  2.456,
				LastLapTime:  83.789,
				TireAge:      8,
				RecentPitLap: 5,
			},
			{
				Position:     5,
				Name:         "Verstappen",
				GapToPlayer:  -1.234,
				LastLapTime:  84.567,
				TireAge:      12,
				RecentPitLap: 3,
			},
		},
	}
}

// ValidateRaceData performs basic validation on race data
func ValidateRaceData(data *RaceData) error {
	if data == nil {
		return fmt.Errorf("race data is nil")
	}

	if data.SessionType == "" {
		return fmt.Errorf("session type is required")
	}

	if data.TrackName == "" {
		return fmt.Errorf("track name is required")
	}

	if data.CurrentLap < 0 {
		return fmt.Errorf("current lap cannot be negative")
	}

	if data.Position < 1 {
		return fmt.Errorf("position must be at least 1")
	}

	if data.FuelLevel < 0 || data.FuelLevel > 100 {
		return fmt.Errorf("fuel level must be between 0 and 100")
	}

	if data.TireWear < 0 || data.TireWear > 100 {
		return fmt.Errorf("tire wear must be between 0 and 100")
	}

	return nil
}

// FormatAnalysisForDisplay formats a strategy analysis for user-friendly display
func FormatAnalysisForDisplay(analysis *StrategyAnalysis) string {
	if analysis == nil {
		return "No analysis available"
	}
	var result strings.Builder

	raceFormatEmoji := "🏁"
	if analysis.RaceFormat == "sprint" {
		raceFormatEmoji = "⚡"
	} else if analysis.RaceFormat == "endurance" {
		raceFormatEmoji = "⏱️"
	}

	result.WriteString(fmt.Sprintf("%s %s RACE STRATEGY (Confidence: %.0f%%)\n",
		raceFormatEmoji, strings.ToUpper(analysis.RaceFormat), analysis.Confidence*100))
	result.WriteString(fmt.Sprintf("📊 %s\n\n", analysis.CurrentSituation))
	result.WriteString(fmt.Sprintf("🎯 PRIMARY STRATEGY: %s\n\n", analysis.PrimaryStrategy))

	if analysis.PitWindowOpen {
		result.WriteString("🏁 PIT WINDOW OPEN\n")
		if analysis.RecommendedLap > 0 {
			result.WriteString(fmt.Sprintf("   Recommended pit lap: %d\n", analysis.RecommendedLap))
		}
		if analysis.TireRecommendation != "" {
			result.WriteString(fmt.Sprintf("   Tire recommendation: %s\n", analysis.TireRecommendation))
		}
		result.WriteString("\n")
	}

	if analysis.FuelStrategy != "" {
		result.WriteString(fmt.Sprintf("⛽ FUEL: %s\n\n", analysis.FuelStrategy))
	}

	if len(analysis.ImmediateActions) > 0 {
		result.WriteString("⚡ IMMEDIATE ACTIONS:\n")
		for _, action := range analysis.ImmediateActions {
			result.WriteString(fmt.Sprintf("   • %s\n", action))
		}
		result.WriteString("\n")
	}

	if len(analysis.LapTargets) > 0 {
		result.WriteString("🎯 LAP TARGETS:\n")
		for scenario, target := range analysis.LapTargets {
			result.WriteString(fmt.Sprintf("   %s: %.3fs\n", scenario, target))
		}
		result.WriteString("\n")
	}

	if len(analysis.RiskFactors) > 0 {
		result.WriteString("⚠️ RISKS:\n")
		for _, risk := range analysis.RiskFactors {
			result.WriteString(fmt.Sprintf("   • %s\n", risk))
		}
		result.WriteString("\n")
	}

	if len(analysis.Opportunities) > 0 {
		result.WriteString("💡 OPPORTUNITIES:\n")
		for _, opp := range analysis.Opportunities {
			result.WriteString(fmt.Sprintf("   • %s\n", opp))
		}
		result.WriteString("\n")
	}

	if analysis.EstimatedFinishPosition > 0 {
		result.WriteString(fmt.Sprintf("🏆 Estimated finish: P%d", analysis.EstimatedFinishPosition))
		if analysis.EstimatedFinishTime != "" {
			result.WriteString(fmt.Sprintf(" (%s)", analysis.EstimatedFinishTime))
		}
		result.WriteString("\n")
	}

	if analysis.StrategicTimeline != "" {
		result.WriteString("📅 STRATEGIC TIMELINE:\n")
		result.WriteString(fmt.Sprintf("   %s\n\n", analysis.StrategicTimeline))
	}

	return result.String()
}
