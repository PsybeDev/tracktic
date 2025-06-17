package sims

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the enhanced connection handling and validation system
func ExampleUsage() {
	// Create a connection health monitor
	healthMonitor := NewConnectionHealthMonitor(5 * time.Second)
	defer healthMonitor.Stop()

	// Register simulator connectors
	iracingConnector := NewIRacingConnector()
	accConnector := NewACCConnector()
	lmuConnector := NewLMUConnector()

	healthMonitor.RegisterConnector(SimulatorTypeIRacing, iracingConnector)
	healthMonitor.RegisterConnector(SimulatorTypeACC, accConnector)
	healthMonitor.RegisterConnector(SimulatorTypeLMU, lmuConnector)

	// Start health monitoring
	healthMonitor.StartHealthChecking()

	// Create context for operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: Get telemetry data with automatic retry and circuit breaker protection
	fmt.Println("=== Example 1: Getting telemetry data with error handling ===")

	for _, simType := range []SimulatorType{SimulatorTypeIRacing, SimulatorTypeACC, SimulatorTypeLMU} {
		data, err := healthMonitor.GetTelemetryWithRetry(simType)
		if err != nil {
			log.Printf("Failed to get telemetry from %s: %v", simType, err)
			continue
		}

		fmt.Printf("Successfully retrieved telemetry from %s\n", simType)
		fmt.Printf("  - Timestamp: %v\n", data.Timestamp)
		fmt.Printf("  - Connected: %v\n", data.IsConnected)
		fmt.Printf("  - Session Type: %s\n", data.Session.Type)
		fmt.Printf("  - Player Position: %d\n", data.Player.Position)
	}

	// Example 2: Monitor health status
	fmt.Println("\n=== Example 2: Health status monitoring ===")

	healthStatus := healthMonitor.GetHealthStatus()
	for simType, status := range healthStatus {
		fmt.Printf("%s: %s\n", simType, status)
	}

	// Example 3: Detailed health metrics including circuit breaker state
	fmt.Println("\n=== Example 3: Detailed health metrics ===")

	detailedMetrics := healthMonitor.GetDetailedHealthMetrics()
	for simType, metrics := range detailedMetrics {
		fmt.Printf("%s:\n", simType)
		fmt.Printf("  - Health Status: %v\n", metrics["health_status"])
		fmt.Printf("  - Last Health Check: %v\n", metrics["last_health_check"])

		if cbMetrics, ok := metrics["circuit_breaker"].(map[string]interface{}); ok {
			fmt.Printf("  - Circuit Breaker State: %v\n", cbMetrics["state"])
			fmt.Printf("  - Failure Count: %v\n", cbMetrics["failure_count"])
		}
	}

	// Example 4: Manual connection with retry logic
	fmt.Println("\n=== Example 4: Manual connection attempts ===")

	retryHandler := NewRetryHandler(&RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	})

	err := retryHandler.Retry(ctx, func() error {
		return iracingConnector.Connect(ctx)
	})

	if err != nil {
		log.Printf("Failed to connect to iRacing after retries: %v", err)
	} else {
		fmt.Println("Successfully connected to iRacing")
	}

	// Example 5: Data validation and sanitization
	fmt.Println("\n=== Example 5: Data validation and sanitization ===")

	validator := NewDataValidator(nil)

	// Create some invalid test data
	invalidData := &TelemetryData{
		Timestamp:     time.Now(),
		SimulatorType: SimulatorTypeIRacing,
		Session: SessionInfo{
			TrackLength:    -1.0,  // Invalid
			AirTemperature: 100.0, // Too hot
		}, Player: PlayerData{
			Position:           0,     // Invalid
			LapDistancePercent: 150.0, // Too high
			Fuel: FuelData{
				Level: -10.0, // Negative
			},
		},
	}

	// Validate the data
	validationErrors := validator.ValidateTelemetryData(invalidData)
	fmt.Printf("Found %d validation errors\n", len(validationErrors))
	for _, err := range validationErrors {
		fmt.Printf("  - %v\n", err)
	}

	// Sanitize the data
	sanitizedData := validator.SanitizeTelemetryData(invalidData)

	// Validate again after sanitization
	validationErrors = validator.ValidateTelemetryData(sanitizedData)
	fmt.Printf("After sanitization: %d validation errors\n", len(validationErrors))

	fmt.Printf("Sanitized values:\n")
	fmt.Printf("  - Track Length: %.2f km\n", sanitizedData.Session.TrackLength)
	fmt.Printf("  - Air Temperature: %.1f°C\n", sanitizedData.Session.AirTemperature)
	fmt.Printf("  - Position: %d\n", sanitizedData.Player.Position)
	fmt.Printf("  - Lap Distance: %.1f%%\n", sanitizedData.Player.LapDistancePercent)
	fmt.Printf("  - Fuel Level: %.1fL\n", sanitizedData.Player.Fuel.Level)
}

// ExampleCircuitBreakerUsage demonstrates circuit breaker pattern usage
func ExampleCircuitBreakerUsage() {
	fmt.Println("\n=== Circuit Breaker Example ===")

	// Create circuit breaker with quick failure threshold for demo
	circuitBreaker := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  2 * time.Second,
		HalfOpenMaxCalls: 2,
		SuccessThreshold: 2,
	})

	// Simulate a failing operation
	failingOperation := func() error {
		return fmt.Errorf("simulated failure")
	}

	// Execute failing operation multiple times to trigger circuit breaker
	for i := 1; i <= 5; i++ {
		err := circuitBreaker.Execute(failingOperation)
		state := circuitBreaker.GetState()

		fmt.Printf("Attempt %d: State=%s, Error=%v\n", i, state, err != nil)

		if state == CircuitBreakerStateOpen {
			fmt.Println("Circuit breaker is now OPEN - rejecting calls")
			break
		}
	}

	// Wait for recovery timeout
	fmt.Println("Waiting for recovery timeout...")
	time.Sleep(3 * time.Second)

	// Try a successful operation to recover
	successfulOperation := func() error {
		return nil
	}

	for i := 1; i <= 3; i++ {
		err := circuitBreaker.Execute(successfulOperation)
		state := circuitBreaker.GetState()

		fmt.Printf("Recovery attempt %d: State=%s, Error=%v\n", i, state, err != nil)

		if state == CircuitBreakerStateClosed {
			fmt.Println("Circuit breaker recovered to CLOSED state")
			break
		}
	}
}

// ExamplePollingSystemIntegration shows how to integrate with the polling system
func ExamplePollingSystemIntegration() {
	fmt.Println("\n=== Polling System Integration Example ===")

	// Create a health monitor
	healthMonitor := NewConnectionHealthMonitor(10 * time.Second)
	defer healthMonitor.Stop()

	// Register an iRacing connector
	connector := NewIRacingConnector()
	healthMonitor.RegisterConnector(SimulatorTypeIRacing, connector) // Create a polling system with custom config
	config := &PollingConfig{
		HighPriorityInterval:   time.Second,
		MediumPriorityInterval: time.Second,
		LowPriorityInterval:    time.Second,
		BufferSize:             10,
		MaxRetries:             3,
		RetryDelay:             2 * time.Second,
	}
	pollingSystem := NewDataPollingSystem(config)

	// Configure polling with error-resilient connector wrapper
	resilientConnector := &ResilientConnectorWrapper{
		healthMonitor: healthMonitor,
		simType:       SimulatorTypeIRacing,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Register the connector with the polling system
	pollingSystem.RegisterConnector(SimulatorTypeIRacing, resilientConnector)

	// Set the active simulator
	err := pollingSystem.SetActiveSimulator(SimulatorTypeIRacing)
	if err != nil {
		fmt.Printf("Failed to set active simulator: %v\n", err)
		return
	}

	// Start the polling system
	err = pollingSystem.Start(ctx)
	if err != nil {
		fmt.Printf("Failed to start polling: %v\n", err)
		return
	}

	// Get data channels
	highChan, mediumChan, lowChan, errorStream := pollingSystem.GetDataChannels()

	// Monitor the streams
	for i := 0; i < 5; i++ {
		select {
		case data := <-highChan:
			if data != nil {
				fmt.Printf("Received high priority telemetry: Position=%d, Speed=%.1f km/h\n",
					data.Player.Position, data.Player.Speed)
			}
		case data := <-mediumChan:
			if data != nil {
				fmt.Printf("Received medium priority telemetry: Fuel=%.1fL\n",
					data.Player.Fuel.Level)
			}
		case data := <-lowChan:
			if data != nil {
				fmt.Printf("Received low priority telemetry: Lap=%d\n",
					data.Player.CurrentLap)
			}
		case err := <-errorStream:
			fmt.Printf("Polling error: %v\n", err)
		case <-time.After(5 * time.Second):
			fmt.Println("No data received in 5 seconds")
		}
	}

	pollingSystem.Stop()
}

// ResilientConnectorWrapper wraps a health monitor to provide a resilient connector
type ResilientConnectorWrapper struct {
	healthMonitor *ConnectionHealthMonitor
	simType       SimulatorType
}

func (w *ResilientConnectorWrapper) Connect(ctx context.Context) error {
	// The health monitor handles connection internally
	return nil
}

func (w *ResilientConnectorWrapper) Disconnect() error {
	// The health monitor handles disconnection internally
	return nil
}

func (w *ResilientConnectorWrapper) IsConnected() bool {
	status := w.healthMonitor.GetHealthStatus()
	if connStatus, exists := status[w.simType]; exists {
		return connStatus == ConnectionStateHealthy
	}
	return false
}

func (w *ResilientConnectorWrapper) GetSimulatorType() SimulatorType {
	return w.simType
}

func (w *ResilientConnectorWrapper) GetTelemetryData(ctx context.Context) (*TelemetryData, error) {
	return w.healthMonitor.GetTelemetryWithRetry(w.simType)
}

func (w *ResilientConnectorWrapper) StartDataStream(ctx context.Context, interval time.Duration) (<-chan *TelemetryData, <-chan error) {
	// This could be implemented to provide streaming data using the health monitor
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
				data, err := w.GetTelemetryData(ctx)
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

func (w *ResilientConnectorWrapper) StopDataStream() {
	// Implementation would depend on the specific streaming approach
}

func (w *ResilientConnectorWrapper) HealthCheck(ctx context.Context) error {
	if w.IsConnected() {
		return nil
	}
	return fmt.Errorf("connector is not healthy")
}
