package strategy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestErrorClassifierBasic(t *testing.T) {
	classifier := NewErrorClassifier()

	// Test basic error classification
	quotaErr := errors.New("quota exceeded")
	stratErr := classifier.ClassifyError(quotaErr, nil)
	if stratErr == nil {
		t.Fatal("Expected classified error, got nil")
	}

	if stratErr.Type != ErrorTypeQuotaExceeded {
		t.Errorf("Expected ErrorTypeQuotaExceeded, got %v", stratErr.Type)
	}

	// Test authentication error
	authErr := errors.New("authentication failed")
	stratErr = classifier.ClassifyError(authErr, nil)
	if stratErr.Type != ErrorTypeAuthentication {
		t.Errorf("Expected ErrorTypeAuthentication, got %v", stratErr.Type)
	}

	// Test nil error
	stratErr = classifier.ClassifyError(nil, nil)
	if stratErr != nil {
		t.Error("Expected nil for nil error input")
	}
}

func TestRetryPolicyBasic(t *testing.T) {
	policy := NewDefaultRetryPolicy()

	// Create test strategy errors
	networkErr := &StrategyError{
		Type:      ErrorTypeNetwork,
		Message:   "Network error",
		Retryable: true,
	}

	authErr := &StrategyError{
		Type:      ErrorTypeAuthentication,
		Message:   "Auth error",
		Retryable: false,
	}

	// Test retry decisions
	if !policy.ShouldRetry(networkErr, 1) {
		t.Error("Should retry network errors")
	}

	if policy.ShouldRetry(authErr, 1) {
		t.Error("Should not retry authentication errors")
	}

	// Test attempt limits
	if policy.ShouldRetry(networkErr, 10) {
		t.Error("Should not retry after many attempts")
	}
}

func TestRetryBackoffCalculation(t *testing.T) {
	policy := NewDefaultRetryPolicy()

	networkErr := &StrategyError{
		Type:      ErrorTypeNetwork,
		Message:   "Network error",
		Retryable: true,
	}

	// Test backoff calculation
	delay1 := policy.CalculateBackoff(networkErr, 1)
	delay2 := policy.CalculateBackoff(networkErr, 2)

	if delay1 <= 0 {
		t.Error("Delay should be positive")
	}

	if delay2 <= 0 {
		t.Error("Delay should be positive")
	}

	if delay1 > 30*time.Second || delay2 > 30*time.Second {
		t.Error("Delays should be reasonable")
	}
}

func TestErrorReporterBasic(t *testing.T) {
	reporter := NewErrorReporter(5)

	// Test error reporting
	err1 := &StrategyError{
		Type:    ErrorTypeNetwork,
		Message: "Network error",
		Context: map[string]interface{}{"test": "context1"},
	}

	err2 := &StrategyError{
		Type:    ErrorTypeAuthentication,
		Message: "Auth error",
		Context: map[string]interface{}{"test": "context2"},
	}

	reporter.ReportError(err1)
	reporter.ReportError(err2)

	// Test getting recent errors
	recent := reporter.GetRecentErrors(10)
	if len(recent) != 2 {
		t.Errorf("Expected 2 recent errors, got %d", len(recent))
	}

	// Test error statistics
	stats := reporter.GetErrorStats()
	if stats[ErrorTypeNetwork] != 1 {
		t.Errorf("Expected 1 network error, got %d", stats[ErrorTypeNetwork])
	}

	if stats[ErrorTypeAuthentication] != 1 {
		t.Errorf("Expected 1 auth error, got %d", stats[ErrorTypeAuthentication])
	}
}

func TestStrategyErrorMethods(t *testing.T) {
	err := &StrategyError{
		Type:    ErrorTypeAuthentication,
		Code:    "AUTH_FAILED",
		Message: "Invalid API key",
		Context: map[string]interface{}{
			"endpoint": "/analyze",
			"attempt":  1,
		},
		Timestamp: time.Now(),
	}

	// Test error string representation
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error string should not be empty")
	}

	// Test retry methods
	retryAfter := err.GetRetryAfter()
	if retryAfter < 0 {
		t.Error("Retry after should not be negative")
	}

	// Test unwrap
	cause := errors.New("underlying cause")
	err.Cause = cause
	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestErrorTypeStringRepresentation(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrorTypeAuthentication, "authentication"},
		{ErrorTypeRateLimit, "rate_limit"},
		{ErrorTypeQuotaExceeded, "quota_exceeded"},
		{ErrorTypeNetwork, "network"},
		{ErrorTypeInternalServer, "internal_server"},
		{ErrorTypeUnknown, "unknown"},
	}

	for _, test := range tests {
		if test.errorType.String() != test.expected {
			t.Errorf("Expected ErrorType %v to string as '%s', got '%s'",
				test.errorType, test.expected, test.errorType.String())
		}
	}
}

func TestContextBasedErrors(t *testing.T) {
	classifier := NewErrorClassifier()

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stratErr := classifier.ClassifyError(ctx.Err(), nil)
	if stratErr.Type != ErrorTypeTimeout {
		t.Errorf("Expected ErrorTypeTimeout for context cancellation, got %v", stratErr.Type)
	}

	// Test context deadline exceeded
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout

	stratErr = classifier.ClassifyError(ctx.Err(), nil)
	if stratErr.Type != ErrorTypeTimeout {
		t.Errorf("Expected ErrorTypeTimeout for deadline exceeded, got %v", stratErr.Type)
	}
}

func TestErrorReporterCapacity(t *testing.T) {
	reporter := NewErrorReporter(3) // Small capacity for testing

	// Add more errors than capacity
	for i := 0; i < 5; i++ {
		err := &StrategyError{
			Type:    ErrorTypeNetwork,
			Message: "Network error",
			Context: map[string]interface{}{"index": i},
		}
		reporter.ReportError(err)
	}

	// Should only keep the latest 3 errors
	recent := reporter.GetRecentErrors(10)
	if len(recent) != 3 {
		t.Errorf("Expected 3 errors (capacity limit), got %d", len(recent))
	}

	// Check that we kept the most recent ones
	if recent[0].Context["index"] != 4 { // Most recent first
		t.Error("Should keep the most recent errors")
	}
}

func BenchmarkErrorClassification(b *testing.B) {
	classifier := NewErrorClassifier()
	testErr := errors.New("quota exceeded for this project")

	for i := 0; i < b.N; i++ {
		_ = classifier.ClassifyError(testErr, nil)
	}
}

func BenchmarkErrorReporting(b *testing.B) {
	reporter := NewErrorReporter(100)
	testErr := &StrategyError{
		Type:    ErrorTypeNetwork,
		Message: "Test error",
		Context: make(map[string]interface{}),
	}

	for i := 0; i < b.N; i++ {
		reporter.ReportError(testErr)
	}
}

func BenchmarkRetryDelayCalculation(b *testing.B) {
	policy := NewDefaultRetryPolicy()
	testErr := &StrategyError{
		Type:    ErrorTypeNetwork,
		Message: "Test error",
	}

	for i := 0; i < b.N; i++ {
		_ = policy.CalculateBackoff(testErr, 2)
	}
}
