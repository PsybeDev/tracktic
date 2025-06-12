package sims

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DataPriority represents the priority level for different types of telemetry data
type DataPriority string

const (
	DataPriorityHigh   DataPriority = "high"   // 60Hz - Fuel, lap times, positions
	DataPriorityMedium DataPriority = "medium" // 10Hz - Tire data, session status
	DataPriorityLow    DataPriority = "low"    // 1Hz - Opponent data, session info
)

// PollingConfig contains configuration for the data polling system
type PollingConfig struct {
	HighPriorityInterval   time.Duration // Default: 16ms (60Hz)
	MediumPriorityInterval time.Duration // Default: 100ms (10Hz)
	LowPriorityInterval    time.Duration // Default: 1000ms (1Hz)
	MaxRetries             int           // Maximum connection retry attempts
	RetryDelay             time.Duration // Delay between retry attempts
	BufferSize             int           // Size of data channels
}

// DefaultPollingConfig returns the default polling configuration
func DefaultPollingConfig() *PollingConfig {
	return &PollingConfig{
		HighPriorityInterval:   16 * time.Millisecond,   // 60Hz
		MediumPriorityInterval: 100 * time.Millisecond,  // 10Hz
		LowPriorityInterval:    1000 * time.Millisecond, // 1Hz
		MaxRetries:             3,
		RetryDelay:             2 * time.Second,
		BufferSize:             10,
	}
}

// DataPollingSystem manages real-time data collection from multiple simulators
type DataPollingSystem struct {
	config     *PollingConfig
	connectors map[SimulatorType]SimulatorConnector
	activeType SimulatorType
	isRunning  bool
	ctx        context.Context
	cancel     context.CancelFunc
	mutex      sync.RWMutex

	// Data channels for different priorities
	highPriorityData   chan *TelemetryData
	mediumPriorityData chan *TelemetryData
	lowPriorityData    chan *TelemetryData
	errorChannel       chan error

	// Last received data for each priority level
	lastHighPriorityData   *TelemetryData
	lastMediumPriorityData *TelemetryData
	lastLowPriorityData    *TelemetryData
	lastDataMutex          sync.RWMutex
}

// NewDataPollingSystem creates a new data polling system
func NewDataPollingSystem(config *PollingConfig) *DataPollingSystem {
	if config == nil {
		config = DefaultPollingConfig()
	}

	return &DataPollingSystem{
		config:             config,
		connectors:         make(map[SimulatorType]SimulatorConnector),
		highPriorityData:   make(chan *TelemetryData, config.BufferSize),
		mediumPriorityData: make(chan *TelemetryData, config.BufferSize),
		lowPriorityData:    make(chan *TelemetryData, config.BufferSize),
		errorChannel:       make(chan error, config.BufferSize),
	}
}

// RegisterConnector registers a simulator connector with the polling system
func (dps *DataPollingSystem) RegisterConnector(simType SimulatorType, connector SimulatorConnector) {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	dps.connectors[simType] = connector
}

// UnregisterConnector removes a simulator connector from the polling system
func (dps *DataPollingSystem) UnregisterConnector(simType SimulatorType) {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	if connector, exists := dps.connectors[simType]; exists {
		if connector.IsConnected() {
			connector.Disconnect()
		}
		delete(dps.connectors, simType)
	}
}

// GetRegisteredConnectors returns a list of registered simulator types
func (dps *DataPollingSystem) GetRegisteredConnectors() []SimulatorType {
	dps.mutex.RLock()
	defer dps.mutex.RUnlock()

	types := make([]SimulatorType, 0, len(dps.connectors))
	for simType := range dps.connectors {
		types = append(types, simType)
	}
	return types
}

// SetActiveSimulator sets the active simulator for data collection
func (dps *DataPollingSystem) SetActiveSimulator(simType SimulatorType) error {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	connector, exists := dps.connectors[simType]
	if !exists {
		return fmt.Errorf("simulator type %s not registered", simType)
	}

	// Disconnect current active simulator if different
	if dps.activeType != "" && dps.activeType != simType {
		if currentConnector, exists := dps.connectors[dps.activeType]; exists && currentConnector.IsConnected() {
			currentConnector.Disconnect()
		}
	}

	// Connect to the new simulator
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	for i := 0; i <= dps.config.MaxRetries; i++ {
		err = connector.Connect(ctx)
		if err == nil {
			break
		}

		if i < dps.config.MaxRetries {
			time.Sleep(dps.config.RetryDelay)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to %s after %d retries: %w", simType, dps.config.MaxRetries, err)
	}

	dps.activeType = simType
	return nil
}

// GetActiveSimulator returns the currently active simulator type
func (dps *DataPollingSystem) GetActiveSimulator() SimulatorType {
	dps.mutex.RLock()
	defer dps.mutex.RUnlock()
	return dps.activeType
}

// IsConnected returns true if the active simulator is connected
func (dps *DataPollingSystem) IsConnected() bool {
	dps.mutex.RLock()
	defer dps.mutex.RUnlock()

	if dps.activeType == "" {
		return false
	}

	connector, exists := dps.connectors[dps.activeType]
	if !exists {
		return false
	}

	return connector.IsConnected()
}

// Start begins the data polling process
func (dps *DataPollingSystem) Start(ctx context.Context) error {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	if dps.isRunning {
		return fmt.Errorf("polling system is already running")
	}

	if dps.activeType == "" {
		return fmt.Errorf("no active simulator set")
	}

	connector, exists := dps.connectors[dps.activeType]
	if !exists {
		return fmt.Errorf("active simulator %s not found", dps.activeType)
	}

	if !connector.IsConnected() {
		return fmt.Errorf("active simulator %s is not connected", dps.activeType)
	}

	dps.ctx, dps.cancel = context.WithCancel(ctx)
	dps.isRunning = true

	// Start polling goroutines for different priorities
	go dps.pollHighPriority()
	go dps.pollMediumPriority()
	go dps.pollLowPriority()
	go dps.healthCheckLoop()

	return nil
}

// Stop stops the data polling process
func (dps *DataPollingSystem) Stop() {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	if !dps.isRunning {
		return
	}

	if dps.cancel != nil {
		dps.cancel()
	}

	dps.isRunning = false

	// Close all channels
	close(dps.highPriorityData)
	close(dps.mediumPriorityData)
	close(dps.lowPriorityData)
	close(dps.errorChannel)

	// Recreate channels for potential restart
	dps.highPriorityData = make(chan *TelemetryData, dps.config.BufferSize)
	dps.mediumPriorityData = make(chan *TelemetryData, dps.config.BufferSize)
	dps.lowPriorityData = make(chan *TelemetryData, dps.config.BufferSize)
	dps.errorChannel = make(chan error, dps.config.BufferSize)
}

// IsRunning returns true if the polling system is currently running
func (dps *DataPollingSystem) IsRunning() bool {
	dps.mutex.RLock()
	defer dps.mutex.RUnlock()
	return dps.isRunning
}

// GetDataChannels returns the data channels for different priorities
func (dps *DataPollingSystem) GetDataChannels() (high, medium, low <-chan *TelemetryData, errors <-chan error) {
	return dps.highPriorityData, dps.mediumPriorityData, dps.lowPriorityData, dps.errorChannel
}

// GetLatestData returns the latest received data for each priority level
func (dps *DataPollingSystem) GetLatestData() (high, medium, low *TelemetryData) {
	dps.lastDataMutex.RLock()
	defer dps.lastDataMutex.RUnlock()

	return dps.lastHighPriorityData, dps.lastMediumPriorityData, dps.lastLowPriorityData
}

// UpdateConfig updates the polling configuration (requires restart to take effect)
func (dps *DataPollingSystem) UpdateConfig(config *PollingConfig) {
	dps.mutex.Lock()
	defer dps.mutex.Unlock()

	if config != nil {
		dps.config = config
	}
}

// GetConfig returns the current polling configuration
func (dps *DataPollingSystem) GetConfig() *PollingConfig {
	dps.mutex.RLock()
	defer dps.mutex.RUnlock()

	// Return a copy to prevent modification
	configCopy := *dps.config
	return &configCopy
}

// Private methods for polling different priority levels

func (dps *DataPollingSystem) pollHighPriority() {
	ticker := time.NewTicker(dps.config.HighPriorityInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dps.collectAndSendData(DataPriorityHigh)
		case <-dps.ctx.Done():
			return
		}
	}
}

func (dps *DataPollingSystem) pollMediumPriority() {
	ticker := time.NewTicker(dps.config.MediumPriorityInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dps.collectAndSendData(DataPriorityMedium)
		case <-dps.ctx.Done():
			return
		}
	}
}

func (dps *DataPollingSystem) pollLowPriority() {
	ticker := time.NewTicker(dps.config.LowPriorityInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dps.collectAndSendData(DataPriorityLow)
		case <-dps.ctx.Done():
			return
		}
	}
}

func (dps *DataPollingSystem) collectAndSendData(priority DataPriority) {
	dps.mutex.RLock()
	connector, exists := dps.connectors[dps.activeType]
	dps.mutex.RUnlock()

	if !exists || !connector.IsConnected() {
		return
	}

	data, err := connector.GetTelemetryData(dps.ctx)
	if err != nil {
		select {
		case dps.errorChannel <- fmt.Errorf("failed to get %s priority data: %w", priority, err):
		default:
		}
		return
	}

	// Update last received data and send to appropriate channel
	dps.lastDataMutex.Lock()
	switch priority {
	case DataPriorityHigh:
		dps.lastHighPriorityData = data
		select {
		case dps.highPriorityData <- data:
		default:
		}
	case DataPriorityMedium:
		dps.lastMediumPriorityData = data
		select {
		case dps.mediumPriorityData <- data:
		default:
		}
	case DataPriorityLow:
		dps.lastLowPriorityData = data
		select {
		case dps.lowPriorityData <- data:
		default:
		}
	}
	dps.lastDataMutex.Unlock()
}

func (dps *DataPollingSystem) healthCheckLoop() {
	ticker := time.NewTicker(5 * time.Second) // Health check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dps.performHealthCheck()
		case <-dps.ctx.Done():
			return
		}
	}
}

func (dps *DataPollingSystem) performHealthCheck() {
	dps.mutex.RLock()
	connector, exists := dps.connectors[dps.activeType]
	activeType := dps.activeType
	dps.mutex.RUnlock()

	if !exists {
		return
	}

	err := connector.HealthCheck(dps.ctx)
	if err != nil {
		select {
		case dps.errorChannel <- fmt.Errorf("health check failed for %s: %w", activeType, err):
		default:
		}
	}
}
