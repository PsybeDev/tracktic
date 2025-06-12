package sims

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ConnectionState represents the current state of a connection
type ConnectionState string

const (
	ConnectionStateHealthy    ConnectionState = "healthy"
	ConnectionStateUnhealthy  ConnectionState = "unhealthy"
	ConnectionStateFailed     ConnectionState = "failed"
	ConnectionStateRecovering ConnectionState = "recovering"
)

// ConnectionError represents a connection-specific error
type ConnectionError struct {
	ConnectorType SimulatorType
	Operation     string
	OriginalError error
	Timestamp     time.Time
	Retryable     bool
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error [%s:%s]: %v", e.ConnectorType, e.Operation, e.OriginalError)
}

// RetryConfig contains configuration for retry logic
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	Jitter          bool          `json:"jitter"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryConfig returns sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    5,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []string{
			"connection refused",
			"timeout",
			"temporary failure",
			"network unreachable",
			"shared memory not available",
		},
	}
}

// CircuitBreakerConfig contains configuration for circuit breaker pattern
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	HalfOpenMaxCalls int           `json:"half_open_max_calls"`
	SuccessThreshold int           `json:"success_threshold"`
}

// DefaultCircuitBreakerConfig returns sensible default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		HalfOpenMaxCalls: 3,
		SuccessThreshold: 2,
	}
}

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerStateClosed   CircuitBreakerState = "closed"    // Normal operation
	CircuitBreakerStateOpen     CircuitBreakerState = "open"      // Failing, rejecting calls
	CircuitBreakerStateHalfOpen CircuitBreakerState = "half_open" // Testing recovery
)

// CircuitBreaker implements the circuit breaker pattern for connection failures
type CircuitBreaker struct {
	config          *CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	halfOpenCalls   int
	mutex           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerStateClosed,
	}
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case CircuitBreakerStateClosed:
		return true
	case CircuitBreakerStateOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.lastFailureTime) >= cb.config.RecoveryTimeout {
			return true // Will transition to half-open
		}
		return false
	case CircuitBreakerStateHalfOpen:
		return cb.halfOpenCalls < cb.config.HalfOpenMaxCalls
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitBreakerStateClosed:
		cb.failureCount = 0
	case CircuitBreakerStateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = CircuitBreakerStateClosed
			cb.failureCount = 0
			cb.successCount = 0
			cb.halfOpenCalls = 0
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerStateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitBreakerStateOpen
		}
	case CircuitBreakerStateHalfOpen:
		cb.state = CircuitBreakerStateOpen
		cb.halfOpenCalls = 0
		cb.successCount = 0
	}
}

// Execute attempts to execute an operation with the circuit breaker
func (cb *CircuitBreaker) Execute(operation func() error) error {
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker is open, rejecting call")
	}

	// Transition to half-open if needed
	cb.mutex.Lock()
	if cb.state == CircuitBreakerStateOpen && time.Since(cb.lastFailureTime) >= cb.config.RecoveryTimeout {
		cb.state = CircuitBreakerStateHalfOpen
		cb.halfOpenCalls = 0
		cb.successCount = 0
	}
	if cb.state == CircuitBreakerStateHalfOpen {
		cb.halfOpenCalls++
	}
	cb.mutex.Unlock()

	err := operation()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"state":           cb.state,
		"failure_count":   cb.failureCount,
		"success_count":   cb.successCount,
		"half_open_calls": cb.halfOpenCalls,
		"last_failure":    cb.lastFailureTime,
	}
}

// RetryHandler handles retry logic with exponential backoff
type RetryHandler struct {
	config *RetryConfig
}

// NewRetryHandler creates a new retry handler with the given configuration
func NewRetryHandler(config *RetryConfig) *RetryHandler {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryHandler{config: config}
}

// Retry executes an operation with retry logic and exponential backoff
func (rh *RetryHandler) Retry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= rh.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !rh.isRetryableError(err) {
			return err
		}

		// Don't delay on last attempt
		if attempt == rh.config.MaxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := rh.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded, last error: %w", rh.config.MaxRetries, lastErr)
}

// isRetryableError checks if an error is retryable based on configuration
func (rh *RetryHandler) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := err.Error()
	for _, retryableError := range rh.config.RetryableErrors {
		if contains(errorMsg, retryableError) {
			return true
		}
	}

	// Check for connection-specific errors
	if connErr, ok := err.(*ConnectionError); ok {
		return connErr.Retryable
	}

	return false
}

// calculateDelay calculates the delay for the given attempt using exponential backoff
func (rh *RetryHandler) calculateDelay(attempt int) time.Duration {
	delay := float64(rh.config.InitialDelay) * math.Pow(rh.config.BackoffFactor, float64(attempt))

	// Apply maximum delay limit
	if delay > float64(rh.config.MaxDelay) {
		delay = float64(rh.config.MaxDelay)
	}
	// Add jitter if enabled
	if rh.config.Jitter {
		jitter := delay * 0.1 * (rand.Float64() - 0.5) // ±5% jitter
		delay += jitter
	}

	return time.Duration(delay)
}

// ConnectionHealthMonitor monitors the health of simulator connections
type ConnectionHealthMonitor struct {
	connectors          map[SimulatorType]SimulatorConnector
	retryHandlers       map[SimulatorType]*RetryHandler
	circuitBreakers     map[SimulatorType]*CircuitBreaker
	healthStatus        map[SimulatorType]ConnectionState
	lastHealthCheck     map[SimulatorType]time.Time
	validator           *DataValidator
	mutex               sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
	healthCheckInterval time.Duration
}

// NewConnectionHealthMonitor creates a new connection health monitor
func NewConnectionHealthMonitor(healthCheckInterval time.Duration) *ConnectionHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionHealthMonitor{
		connectors:          make(map[SimulatorType]SimulatorConnector),
		retryHandlers:       make(map[SimulatorType]*RetryHandler),
		circuitBreakers:     make(map[SimulatorType]*CircuitBreaker),
		healthStatus:        make(map[SimulatorType]ConnectionState),
		lastHealthCheck:     make(map[SimulatorType]time.Time),
		validator:           NewDataValidator(nil),
		ctx:                 ctx,
		cancel:              cancel,
		healthCheckInterval: healthCheckInterval,
	}
}

// RegisterConnector registers a simulator connector with the health monitor
func (chm *ConnectionHealthMonitor) RegisterConnector(simType SimulatorType, connector SimulatorConnector) {
	chm.mutex.Lock()
	defer chm.mutex.Unlock()

	chm.connectors[simType] = connector
	chm.retryHandlers[simType] = NewRetryHandler(nil)
	chm.circuitBreakers[simType] = NewCircuitBreaker(nil)
	chm.healthStatus[simType] = ConnectionStateHealthy
	chm.lastHealthCheck[simType] = time.Now()
}

// GetTelemetryWithRetry gets telemetry data with retry logic and circuit breaker protection
func (chm *ConnectionHealthMonitor) GetTelemetryWithRetry(simType SimulatorType) (*TelemetryData, error) {
	chm.mutex.RLock()
	connector, exists := chm.connectors[simType]
	retryHandler := chm.retryHandlers[simType]
	circuitBreaker := chm.circuitBreakers[simType]
	chm.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connector for simulator type %s not registered", simType)
	}

	var telemetryData *TelemetryData
	var operationError error
	operation := func() error {
		data, err := connector.GetTelemetryData(chm.ctx)
		if err != nil {
			operationError = &ConnectionError{
				ConnectorType: simType,
				Operation:     "GetTelemetryData",
				OriginalError: err,
				Timestamp:     time.Now(),
				Retryable:     true,
			}
			return operationError
		}

		// Validate the received data
		validationErrors := chm.validator.ValidateTelemetryData(data)
		if len(validationErrors) > 0 {
			operationError = fmt.Errorf("data validation failed: %d errors", len(validationErrors))
			return operationError
		}

		telemetryData = data
		return nil
	}

	// Execute with circuit breaker protection
	err := circuitBreaker.Execute(func() error {
		return retryHandler.Retry(chm.ctx, operation)
	})

	if err != nil {
		chm.updateHealthStatus(simType, ConnectionStateFailed)
		return nil, err
	}

	chm.updateHealthStatus(simType, ConnectionStateHealthy)
	return telemetryData, nil
}

// updateHealthStatus updates the health status of a connector
func (chm *ConnectionHealthMonitor) updateHealthStatus(simType SimulatorType, status ConnectionState) {
	chm.mutex.Lock()
	defer chm.mutex.Unlock()

	chm.healthStatus[simType] = status
	chm.lastHealthCheck[simType] = time.Now()
}

// GetHealthStatus returns the current health status of all connectors
func (chm *ConnectionHealthMonitor) GetHealthStatus() map[SimulatorType]ConnectionState {
	chm.mutex.RLock()
	defer chm.mutex.RUnlock()

	status := make(map[SimulatorType]ConnectionState)
	for simType, state := range chm.healthStatus {
		status[simType] = state
	}
	return status
}

// GetDetailedHealthMetrics returns detailed health metrics for all connectors
func (chm *ConnectionHealthMonitor) GetDetailedHealthMetrics() map[SimulatorType]map[string]interface{} {
	chm.mutex.RLock()
	defer chm.mutex.RUnlock()

	metrics := make(map[SimulatorType]map[string]interface{})
	for simType := range chm.connectors {
		circuitBreaker := chm.circuitBreakers[simType]
		cbMetrics := circuitBreaker.GetMetrics()

		metrics[simType] = map[string]interface{}{
			"health_status":     chm.healthStatus[simType],
			"last_health_check": chm.lastHealthCheck[simType],
			"circuit_breaker":   cbMetrics,
		}
	}
	return metrics
}

// StartHealthChecking starts periodic health checking for all registered connectors
func (chm *ConnectionHealthMonitor) StartHealthChecking() {
	go func() {
		ticker := time.NewTicker(chm.healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-chm.ctx.Done():
				return
			case <-ticker.C:
				chm.performHealthChecks()
			}
		}
	}()
}

// performHealthChecks performs health checks on all registered connectors
func (chm *ConnectionHealthMonitor) performHealthChecks() {
	chm.mutex.RLock()
	connectors := make(map[SimulatorType]SimulatorConnector)
	for simType, connector := range chm.connectors {
		connectors[simType] = connector
	}
	chm.mutex.RUnlock()

	for simType, connector := range connectors {
		go func(st SimulatorType, conn SimulatorConnector) {
			if conn.IsConnected() {
				chm.updateHealthStatus(st, ConnectionStateHealthy)
			} else {
				chm.updateHealthStatus(st, ConnectionStateUnhealthy)
			}
		}(simType, connector)
	}
}

// Stop stops the health monitoring
func (chm *ConnectionHealthMonitor) Stop() {
	chm.cancel()
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexOfSubstring(s, substr) >= 0)))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
