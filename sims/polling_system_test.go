package sims

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockConnector implements SimulatorConnector for testing
type MockConnector struct {
	isConnected        bool
	connectError       error
	disconnectError    error
	telemetryData      *TelemetryData
	telemetryError     error
	healthCheckError   error
	mutex              sync.RWMutex
	connectCallCount   int
	telemetryCallCount int
	healthCallCount    int
}

func NewMockConnector() *MockConnector {
	return &MockConnector{
		telemetryData: &TelemetryData{
			Player: PlayerData{
				Position:    1,
				CurrentLap:  5,
				LastLapTime: 88*time.Second + 234*time.Millisecond,
				BestLapTime: 87*time.Second + 892*time.Millisecond,
				Fuel: FuelData{
					Level:       45.6,
					Capacity:    65.0,
					UsagePerLap: 2.1,
				},
			},
			Session: SessionInfo{
				Type:          SessionTypeRace,
				Flag:          SessionFlagGreen,
				TimeRemaining: 1800 * time.Second,
				LapsRemaining: 25,
			},
		},
	}
}

func (mc *MockConnector) Connect(ctx context.Context) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.connectCallCount++

	if mc.connectError != nil {
		return mc.connectError
	}

	mc.isConnected = true
	return nil
}

func (mc *MockConnector) Disconnect() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.disconnectError != nil {
		return mc.disconnectError
	}

	mc.isConnected = false
	return nil
}

func (mc *MockConnector) IsConnected() bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.isConnected
}

func (mc *MockConnector) GetTelemetryData(ctx context.Context) (*TelemetryData, error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.telemetryCallCount++

	if mc.telemetryError != nil {
		return nil, mc.telemetryError
	}

	// Return a copy to avoid data races
	data := *mc.telemetryData
	return &data, nil
}

func (mc *MockConnector) HealthCheck(ctx context.Context) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.healthCallCount++

	if mc.healthCheckError != nil {
		return mc.healthCheckError
	}
	return nil
}

func (mc *MockConnector) GetSimulatorType() SimulatorType {
	return SimulatorTypeACC
}

func (mc *MockConnector) StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error) {
	// Simple mock implementation - not needed for these tests
	dataChan := make(chan *TelemetryData)
	errorChan := make(chan error)
	return dataChan, errorChan
}

func (mc *MockConnector) StopDataStream() {
	// Simple mock implementation - not needed for these tests
}

func (mc *MockConnector) SetConnectError(err error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.connectError = err
}

func (mc *MockConnector) SetTelemetryError(err error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.telemetryError = err
}

func (mc *MockConnector) SetHealthCheckError(err error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.healthCheckError = err
}

func (mc *MockConnector) GetCallCounts() (connect, telemetry, health int) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.connectCallCount, mc.telemetryCallCount, mc.healthCallCount
}

func TestDefaultPollingConfig(t *testing.T) {
	config := DefaultPollingConfig()

	if config.HighPriorityInterval != 16*time.Millisecond {
		t.Errorf("Expected HighPriorityInterval to be 16ms, got %v", config.HighPriorityInterval)
	}

	if config.MediumPriorityInterval != 100*time.Millisecond {
		t.Errorf("Expected MediumPriorityInterval to be 100ms, got %v", config.MediumPriorityInterval)
	}

	if config.LowPriorityInterval != 1000*time.Millisecond {
		t.Errorf("Expected LowPriorityInterval to be 1000ms, got %v", config.LowPriorityInterval)
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.BufferSize != 10 {
		t.Errorf("Expected BufferSize to be 10, got %d", config.BufferSize)
	}
}

func TestNewDataPollingSystem(t *testing.T) {
	// Test with nil config (should use default)
	dps1 := NewDataPollingSystem(nil)
	if dps1.config == nil {
		t.Error("Expected config to be set to default when nil provided")
	}

	// Test with custom config
	customConfig := &PollingConfig{
		HighPriorityInterval: 50 * time.Millisecond,
		BufferSize:           5,
	}
	dps2 := NewDataPollingSystem(customConfig)
	if dps2.config.HighPriorityInterval != 50*time.Millisecond {
		t.Error("Expected custom config to be used")
	}

	// Check initial state
	if dps1.isRunning {
		t.Error("Expected polling system to not be running initially")
	}

	if len(dps1.connectors) != 0 {
		t.Error("Expected no connectors initially")
	}
}

func TestRegisterUnregisterConnector(t *testing.T) {
	dps := NewDataPollingSystem(nil)
	mockConnector := NewMockConnector()

	// Register connector
	dps.RegisterConnector(SimulatorTypeACC, mockConnector)

	connectors := dps.GetRegisteredConnectors()
	if len(connectors) != 1 || connectors[0] != SimulatorTypeACC {
		t.Error("Expected ACC connector to be registered")
	}

	// Register another connector
	mockConnector2 := NewMockConnector()
	dps.RegisterConnector(SimulatorTypeIRacing, mockConnector2)

	connectors = dps.GetRegisteredConnectors()
	if len(connectors) != 2 {
		t.Error("Expected 2 connectors to be registered")
	}

	// Unregister connector
	dps.UnregisterConnector(SimulatorTypeACC)

	connectors = dps.GetRegisteredConnectors()
	if len(connectors) != 1 || connectors[0] != SimulatorTypeIRacing {
		t.Error("Expected only iRacing connector to remain")
	}

	// Unregister non-existent connector (should not error)
	dps.UnregisterConnector(SimulatorTypeLMU)
}

func TestSetActiveSimulator(t *testing.T) {
	dps := NewDataPollingSystem(nil)
	mockConnector := NewMockConnector()

	// Test setting active simulator without registering first
	err := dps.SetActiveSimulator(SimulatorTypeACC)
	if err == nil {
		t.Error("Expected error when setting unregistered simulator as active")
	}

	// Register and set active
	dps.RegisterConnector(SimulatorTypeACC, mockConnector)
	err = dps.SetActiveSimulator(SimulatorTypeACC)
	if err != nil {
		t.Errorf("Unexpected error setting active simulator: %v", err)
	}

	if dps.GetActiveSimulator() != SimulatorTypeACC {
		t.Error("Expected ACC to be active simulator")
	}

	if !mockConnector.IsConnected() {
		t.Error("Expected connector to be connected after setting as active")
	}

	// Test connection error
	mockConnector2 := NewMockConnector()
	mockConnector2.SetConnectError(errors.New("connection failed"))
	dps.RegisterConnector(SimulatorTypeIRacing, mockConnector2)

	err = dps.SetActiveSimulator(SimulatorTypeIRacing)
	if err == nil {
		t.Error("Expected error when connection fails")
	}
}

func TestIsConnected(t *testing.T) {
	dps := NewDataPollingSystem(nil)

	// No active simulator
	if dps.IsConnected() {
		t.Error("Expected not connected when no active simulator")
	}

	mockConnector := NewMockConnector()
	dps.RegisterConnector(SimulatorTypeACC, mockConnector)
	dps.SetActiveSimulator(SimulatorTypeACC)

	// Should be connected
	if !dps.IsConnected() {
		t.Error("Expected to be connected")
	}

	// Disconnect
	mockConnector.Disconnect()
	if dps.IsConnected() {
		t.Error("Expected not connected after disconnect")
	}
}

func TestStartStop(t *testing.T) {
	dps := NewDataPollingSystem(nil)
	ctx := context.Background()

	// Test start without active simulator
	err := dps.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting without active simulator")
	}

	// Setup mock connector
	mockConnector := NewMockConnector()
	dps.RegisterConnector(SimulatorTypeACC, mockConnector)
	dps.SetActiveSimulator(SimulatorTypeACC)

	// Start polling
	err = dps.Start(ctx)
	if err != nil {
		t.Errorf("Unexpected error starting polling: %v", err)
	}

	if !dps.IsRunning() {
		t.Error("Expected polling system to be running")
	}

	// Test starting when already running
	err = dps.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running system")
	}

	// Stop polling
	dps.Stop()
	if dps.IsRunning() {
		t.Error("Expected polling system to be stopped")
	}

	// Test stopping when already stopped (should not error)
	dps.Stop()
}

func TestDataPolling(t *testing.T) {
	// Use faster intervals for testing
	config := &PollingConfig{
		HighPriorityInterval:   10 * time.Millisecond,
		MediumPriorityInterval: 20 * time.Millisecond,
		LowPriorityInterval:    30 * time.Millisecond,
		MaxRetries:             1,
		RetryDelay:             10 * time.Millisecond,
		BufferSize:             10,
	}

	dps := NewDataPollingSystem(config)
	mockConnector := NewMockConnector()

	dps.RegisterConnector(SimulatorTypeACC, mockConnector)
	dps.SetActiveSimulator(SimulatorTypeACC)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := dps.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start polling: %v", err)
	}

	// Get data channels
	highChan, mediumChan, lowChan, errorChan := dps.GetDataChannels()

	// Wait for some data
	time.Sleep(50 * time.Millisecond)

	// Check that we received some data
	receivedHigh := false
	receivedMedium := false
	receivedLow := false

	// Drain channels
	for i := 0; i < 10; i++ {
		select {
		case <-highChan:
			receivedHigh = true
		case <-mediumChan:
			receivedMedium = true
		case <-lowChan:
			receivedLow = true
		case err := <-errorChan:
			t.Errorf("Unexpected error: %v", err)
		default:
			// No more data available
			break
		}
	}

	if !receivedHigh {
		t.Error("Expected to receive high priority data")
	}

	if !receivedMedium {
		t.Error("Expected to receive medium priority data")
	}

	if !receivedLow {
		t.Error("Expected to receive low priority data")
	}

	// Check latest data
	high, medium, low := dps.GetLatestData()
	if high == nil || medium == nil || low == nil {
		t.Error("Expected latest data to be available")
	}

	dps.Stop()
	// Check call counts
	connect, telemetry, _ := mockConnector.GetCallCounts()
	if connect == 0 {
		t.Error("Expected at least one connect call")
	}
	if telemetry == 0 {
		t.Error("Expected at least one telemetry call")
	}
	// Note: Health checks run every 5 seconds by default, so might not execute in short tests
	// This is expected behavior in a real scenario
}

func TestConfigUpdate(t *testing.T) {
	dps := NewDataPollingSystem(nil)

	originalConfig := dps.GetConfig()

	newConfig := &PollingConfig{
		HighPriorityInterval: 50 * time.Millisecond,
		BufferSize:           20,
	}

	dps.UpdateConfig(newConfig)

	updatedConfig := dps.GetConfig()
	if updatedConfig.HighPriorityInterval != 50*time.Millisecond {
		t.Error("Expected config to be updated")
	}

	// Ensure we get a copy (modification shouldn't affect internal config)
	updatedConfig.BufferSize = 999
	finalConfig := dps.GetConfig()
	if finalConfig.BufferSize == 999 {
		t.Error("Expected config copy to be returned (not reference)")
	}

	// Test nil config update (should not change anything)
	dps.UpdateConfig(nil)
	if dps.GetConfig().HighPriorityInterval != 50*time.Millisecond {
		t.Error("Expected config to remain unchanged when nil provided")
	}

	// Verify original config is unchanged
	if originalConfig.HighPriorityInterval == 50*time.Millisecond {
		t.Error("Original config should not have been modified")
	}
}
