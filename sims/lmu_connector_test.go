package sims

import (
	"context"
	"testing"
)

func TestNewLMUConnector(t *testing.T) {
	connector := NewLMUConnector()

	if connector == nil {
		t.Fatal("NewLMUConnector returned nil")
	}

	if connector.isConnected {
		t.Error("New connector should not be connected")
	}

	if connector.GetSimulatorType() != SimulatorTypeLMU {
		t.Errorf("GetSimulatorType() = %v, want %v", connector.GetSimulatorType(), SimulatorTypeLMU)
	}
}

func TestLMUConnectorConnect(t *testing.T) {
	connector := NewLMUConnector()
	ctx := context.Background()

	// Test connection attempt (should fail since LMU SDK is not implemented)
	err := connector.Connect(ctx)
	if err == nil {
		t.Error("Connect should return error when LMU SDK is not available")
	}

	expectedError := "LMU connector not fully implemented"
	if len(err.Error()) < len(expectedError) || err.Error()[:len(expectedError)] != expectedError {
		t.Errorf("Connect error = %q, should contain %q", err.Error(), expectedError)
	}

	// Verify connection state
	if connector.IsConnected() {
		t.Error("IsConnected should return false after failed connection")
	}
}

func TestLMUConnectorDisconnect(t *testing.T) {
	connector := NewLMUConnector()

	// Test disconnect when not connected
	err := connector.Disconnect()
	if err != nil {
		t.Errorf("Disconnect returned error when not connected: %v", err)
	}

	// Test disconnect when connected (simulate connection)
	connector.isConnected = true
	err = connector.Disconnect()
	if err != nil {
		t.Errorf("Disconnect returned error: %v", err)
	}

	if connector.IsConnected() {
		t.Error("IsConnected should return false after disconnect")
	}
}

func TestLMUConnectorGetTelemetryData(t *testing.T) {
	connector := NewLMUConnector()
	ctx := context.Background()

	// Test when not connected
	data, err := connector.GetTelemetryData(ctx)
	if err == nil {
		t.Error("GetTelemetryData should return error when not connected")
	}
	if data != nil {
		t.Error("GetTelemetryData should return nil data when not connected")
	}

	// Test when connected (simulate connection)
	connector.isConnected = true
	data, err = connector.GetTelemetryData(ctx)
	if err != nil {
		t.Errorf("GetTelemetryData returned error when connected: %v", err)
	}
	if data == nil {
		t.Error("GetTelemetryData should return data when connected")
	}

	// Validate mock data structure
	if data.SimulatorType != SimulatorTypeLMU {
		t.Errorf("SimulatorType = %v, want %v", data.SimulatorType, SimulatorTypeLMU)
	}

	if data.IsConnected != true {
		t.Error("IsConnected should be true in telemetry data")
	}
}
