package sims

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// MockSimulatorConnector for testing
type MockSimulatorConnector struct {
	connected     bool
	shouldFail    bool
	failureCount  int
	maxFailures   int
	telemetryData *TelemetryData
	mutex         sync.Mutex
}

func NewMockSimulatorConnector(connected bool) *MockSimulatorConnector {
	return &MockSimulatorConnector{
		connected: connected,
		telemetryData: &TelemetryData{
			Timestamp:     time.Now(),
			SimulatorType: SimulatorTypeIRacing,
			IsConnected:   connected,
			Session: SessionInfo{
				Type:        SessionTypeRace,
				TrackLength: 5.0,
			}, Player: PlayerData{
				Position:   1,
				CurrentLap: 1,
				Fuel: FuelData{
					Level:    50.0,
					Capacity: 100.0,
				},
			},
		},
	}
}

func (m *MockSimulatorConnector) Connect(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.shouldFail {
		m.failureCount++
		if m.failureCount <= m.maxFailures {
			return errors.New("connection refused")
		}
	}

	m.connected = true
	return nil
}

func (m *MockSimulatorConnector) Disconnect() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.connected = false
	return nil
}

func (m *MockSimulatorConnector) IsConnected() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.connected
}

func (m *MockSimulatorConnector) GetTelemetryData(ctx context.Context) (*TelemetryData, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.connected {
		return nil, errors.New("not connected")
	}

	if m.shouldFail {
		m.failureCount++
		if m.failureCount <= m.maxFailures {
			return nil, errors.New("temporary failure")
		}
	}

	// Update timestamp for fresh data
	data := *m.telemetryData
	data.Timestamp = time.Now()
	return &data, nil
}

func (m *MockSimulatorConnector) SetShouldFail(shouldFail bool, maxFailures int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.shouldFail = shouldFail
	m.maxFailures = maxFailures
	m.failureCount = 0
}

func (m *MockSimulatorConnector) GetSimulatorType() SimulatorType {
	return SimulatorTypeIRacing // Default to iRacing for mock
}

func (m *MockSimulatorConnector) StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error) {
	dataStream := make(chan *TelemetryData)
	errorStream := make(chan error)

	go func() {
		defer close(dataStream)
		defer close(errorStream)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := m.GetTelemetryData(ctx)
				if err != nil {
					select {
					case errorStream <- err:
					default:
					}
				} else {
					select {
					case dataStream <- data:
					default:
					}
				}
			}
		}
	}()

	return dataStream, errorStream
}

func (m *MockSimulatorConnector) StopDataStream() {
	// Mock implementation - nothing to stop
}

func (m *MockSimulatorConnector) HealthCheck(ctx context.Context) error {
	if m.IsConnected() {
		return nil
	}
	return errors.New("mock connector not connected")
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config == nil {
		t.Fatal("DefaultRetryConfig returned nil")
	}

	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", config.MaxRetries)
	}

	if config.InitialDelay != 500*time.Millisecond {
		t.Errorf("Expected InitialDelay 500ms, got %v", config.InitialDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor 2.0, got %f", config.BackoffFactor)
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config == nil {
		t.Fatal("DefaultCircuitBreakerConfig returned nil")
	}

	if config.FailureThreshold != 5 {
		t.Errorf("Expected FailureThreshold 5, got %d", config.FailureThreshold)
	}

	if config.RecoveryTimeout != 30*time.Second {
		t.Errorf("Expected RecoveryTimeout 30s, got %v", config.RecoveryTimeout)
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	// Test with nil config (should use defaults)
	cb := NewCircuitBreaker(nil)
	if cb == nil {
		t.Fatal("NewCircuitBreaker returned nil")
	}

	if cb.state != CircuitBreakerStateClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.state)
	}

	// Test with custom config
	customConfig := &CircuitBreakerConfig{FailureThreshold: 3}
	cb2 := NewCircuitBreaker(customConfig)
	if cb2.config.FailureThreshold != 3 {
		t.Errorf("Expected custom FailureThreshold 3, got %d", cb2.config.FailureThreshold)
	}
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 2,
	}

	cb := NewCircuitBreaker(config)

	// Initial state should be Closed
	if cb.GetState() != CircuitBreakerStateClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.GetState())
	}

	// Should allow execution when closed
	if !cb.CanExecute() {
		t.Error("Should allow execution when closed")
	}

	// Record failures to trigger open state
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Should be open now
	if cb.GetState() != CircuitBreakerStateOpen {
		t.Errorf("Expected state Open after failures, got %v", cb.GetState())
	}

	// Should not allow execution when open
	if cb.CanExecute() {
		t.Error("Should not allow execution when open")
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Should allow execution again (will transition to half-open)
	if !cb.CanExecute() {
		t.Error("Should allow execution after recovery timeout")
	}

	// Execute to transition to half-open
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("Expected successful execution, got %v", err)
	}

	// Should be half-open now or closed (depending on success threshold)
	state := cb.GetState()
	if state != CircuitBreakerStateHalfOpen && state != CircuitBreakerStateClosed {
		t.Errorf("Expected state HalfOpen or Closed, got %v", state)
	}
}

func TestCircuitBreakerExecute(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	// Test successful execution
	callCount := 0
	err := cb.Execute(func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected successful execution, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}

	// Test failed execution
	err = cb.Execute(func() error {
		return errors.New("test error")
	})

	if err == nil {
		t.Error("Expected error from failed execution")
	}
}

func TestRetryHandler(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		BackoffFactor:   2.0,
		Jitter:          false,
		RetryableErrors: []string{"temporary failure", "connection refused"},
	}

	retryHandler := NewRetryHandler(config)
	ctx := context.Background()

	// Test successful operation (no retries needed)
	callCount := 0
	err := retryHandler.Retry(ctx, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Test operation that succeeds after retries
	callCount = 0
	err = retryHandler.Retry(ctx, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Test operation that fails with non-retryable error
	callCount = 0
	err = retryHandler.Retry(ctx, func() error {
		callCount++
		return errors.New("non-retryable error")
	})

	if err == nil {
		t.Error("Expected error for non-retryable failure")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call for non-retryable error, got %d", callCount)
	}

	// Test operation that exhausts retries
	callCount = 0
	err = retryHandler.Retry(ctx, func() error {
		callCount++
		return errors.New("temporary failure")
	})

	if err == nil {
		t.Error("Expected error after exhausting retries")
	}

	if callCount != 4 { // Initial call + 3 retries
		t.Errorf("Expected 4 calls (1 + 3 retries), got %d", callCount)
	}
}

func TestRetryHandlerWithContext(t *testing.T) {
	retryHandler := NewRetryHandler(nil)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryHandler.Retry(ctx, func() error {
		callCount++
		time.Sleep(100 * time.Millisecond) // Longer than cancellation
		return errors.New("temporary failure")
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestConnectionHealthMonitor(t *testing.T) {
	monitor := NewConnectionHealthMonitor(100 * time.Millisecond)
	defer monitor.Stop()

	// Register a mock connector
	mockConnector := NewMockSimulatorConnector(true)
	monitor.RegisterConnector(SimulatorTypeIRacing, mockConnector)

	// Test getting telemetry data
	data, err := monitor.GetTelemetryWithRetry(SimulatorTypeIRacing)
	if err != nil {
		t.Errorf("Expected successful telemetry retrieval, got %v", err)
	}

	if data == nil {
		t.Error("Expected telemetry data, got nil")
	}

	if data.SimulatorType != SimulatorTypeIRacing {
		t.Errorf("Expected simulator type iRacing, got %v", data.SimulatorType)
	}

	// Test health status
	healthStatus := monitor.GetHealthStatus()
	if status, exists := healthStatus[SimulatorTypeIRacing]; !exists || status != ConnectionStateHealthy {
		t.Errorf("Expected healthy status for iRacing, got %v", status)
	}

	// Test with failing connector
	mockConnector.SetShouldFail(true, 10) // Fail more than retry limit

	_, err = monitor.GetTelemetryWithRetry(SimulatorTypeIRacing)
	if err == nil {
		t.Error("Expected error with failing connector")
	}

	// Check that health status is updated
	healthStatus = monitor.GetHealthStatus()
	if status := healthStatus[SimulatorTypeIRacing]; status != ConnectionStateFailed {
		t.Errorf("Expected failed status after errors, got %v", status)
	}
}

func TestConnectionHealthMonitorWithUnregisteredConnector(t *testing.T) {
	monitor := NewConnectionHealthMonitor(100 * time.Millisecond)
	defer monitor.Stop()

	// Try to get telemetry from unregistered connector
	_, err := monitor.GetTelemetryWithRetry(SimulatorTypeACC)
	if err == nil {
		t.Error("Expected error for unregistered connector")
	}

	expectedError := "connector for simulator type acc not registered"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestConnectionHealthMonitorHealthChecking(t *testing.T) {
	monitor := NewConnectionHealthMonitor(50 * time.Millisecond)
	defer monitor.Stop()

	// Register connectors
	connectedMock := NewMockSimulatorConnector(true)
	disconnectedMock := NewMockSimulatorConnector(false)

	monitor.RegisterConnector(SimulatorTypeIRacing, connectedMock)
	monitor.RegisterConnector(SimulatorTypeACC, disconnectedMock)

	// Start health checking
	monitor.StartHealthChecking()

	// Wait for a few health check cycles
	time.Sleep(150 * time.Millisecond)

	// Check health status
	healthStatus := monitor.GetHealthStatus()

	if status := healthStatus[SimulatorTypeIRacing]; status != ConnectionStateHealthy {
		t.Errorf("Expected healthy status for connected connector, got %v", status)
	}

	if status := healthStatus[SimulatorTypeACC]; status != ConnectionStateUnhealthy {
		t.Errorf("Expected unhealthy status for disconnected connector, got %v", status)
	}
}

func TestConnectionHealthMonitorDetailedMetrics(t *testing.T) {
	monitor := NewConnectionHealthMonitor(100 * time.Millisecond)
	defer monitor.Stop()

	mockConnector := NewMockSimulatorConnector(true)
	monitor.RegisterConnector(SimulatorTypeIRacing, mockConnector)

	// Get detailed metrics
	metrics := monitor.GetDetailedHealthMetrics()

	iracingMetrics, exists := metrics[SimulatorTypeIRacing]
	if !exists {
		t.Error("Expected metrics for iRacing connector")
	}

	// Check that circuit breaker metrics are included
	if _, exists := iracingMetrics["circuit_breaker"]; !exists {
		t.Error("Expected circuit breaker metrics")
	}

	if _, exists := iracingMetrics["health_status"]; !exists {
		t.Error("Expected health status in metrics")
	}

	if _, exists := iracingMetrics["last_health_check"]; !exists {
		t.Error("Expected last health check time in metrics")
	}
}

func TestConnectionError(t *testing.T) {
	originalErr := errors.New("connection refused")
	connErr := &ConnectionError{
		ConnectorType: SimulatorTypeIRacing,
		Operation:     "GetTelemetryData",
		OriginalError: originalErr,
		Timestamp:     time.Now(),
		Retryable:     true,
	}

	expectedMsg := "connection error [iracing:GetTelemetryData]: connection refused"
	if connErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, connErr.Error())
	}
}

func TestIsRetryableError(t *testing.T) {
	retryHandler := NewRetryHandler(nil)

	// Test retryable errors
	retryableErrors := []error{
		errors.New("connection refused"),
		errors.New("temporary failure"),
		errors.New("network unreachable"),
		&ConnectionError{Retryable: true},
	}

	for _, err := range retryableErrors {
		if !retryHandler.isRetryableError(err) {
			t.Errorf("Expected error to be retryable: %v", err)
		}
	}

	// Test non-retryable errors
	nonRetryableErrors := []error{
		errors.New("invalid credentials"),
		errors.New("access denied"),
		&ConnectionError{Retryable: false},
	}

	for _, err := range nonRetryableErrors {
		if retryHandler.isRetryableError(err) {
			t.Errorf("Expected error to be non-retryable: %v", err)
		}
	}
}

func TestCalculateDelay(t *testing.T) {
	config := &RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
	}

	retryHandler := NewRetryHandler(config)

	// Test delay calculation
	delays := []time.Duration{
		retryHandler.calculateDelay(0), // Should be 100ms
		retryHandler.calculateDelay(1), // Should be 200ms
		retryHandler.calculateDelay(2), // Should be 400ms
		retryHandler.calculateDelay(3), // Should be 800ms
		retryHandler.calculateDelay(4), // Should be 1000ms (capped at MaxDelay)
	}

	expectedDelays := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1000 * time.Millisecond,
	}

	for i, delay := range delays {
		if delay != expectedDelays[i] {
			t.Errorf("Expected delay %v for attempt %d, got %v", expectedDelays[i], i, delay)
		}
	}
}
