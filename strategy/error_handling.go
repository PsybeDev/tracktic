package strategy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ErrorType represents different categories of errors that can occur
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeRateLimit
	ErrorTypeAuthentication
	ErrorTypeQuotaExceeded
	ErrorTypeInvalidRequest
	ErrorTypeInternalServer
	ErrorTypeTimeout
	ErrorTypeParsing
	ErrorTypeValidation
)

// String returns a human-readable representation of the error type
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeNetwork:
		return "network"
	case ErrorTypeRateLimit:
		return "rate_limit"
	case ErrorTypeAuthentication:
		return "authentication"
	case ErrorTypeQuotaExceeded:
		return "quota_exceeded"
	case ErrorTypeInvalidRequest:
		return "invalid_request"
	case ErrorTypeInternalServer:
		return "internal_server"
	case ErrorTypeTimeout:
		return "timeout"
	case ErrorTypeParsing:
		return "parsing"
	case ErrorTypeValidation:
		return "validation"
	default:
		return "unknown"
	}
}

// StrategyError represents a comprehensive error from the strategy engine
type StrategyError struct {
	Type       ErrorType
	Code       string
	Message    string
	Cause      error
	Retryable  bool
	RetryAfter time.Duration
	Context    map[string]interface{}
	Timestamp  time.Time
}

// Error implements the error interface
func (se *StrategyError) Error() string {
	if se.Cause != nil {
		return fmt.Sprintf("%s (%s): %s [caused by: %v]", se.Type.String(), se.Code, se.Message, se.Cause)
	}
	return fmt.Sprintf("%s (%s): %s", se.Type.String(), se.Code, se.Message)
}

// Unwrap returns the underlying cause error
func (se *StrategyError) Unwrap() error {
	return se.Cause
}

// IsRetryable returns whether this error suggests a retry might succeed
func (se *StrategyError) IsRetryable() bool {
	return se.Retryable
}

// GetRetryAfter returns the suggested delay before retrying
func (se *StrategyError) GetRetryAfter() time.Duration {
	if se.RetryAfter > 0 {
		return se.RetryAfter
	}

	// Default retry delays based on error type
	switch se.Type {
	case ErrorTypeRateLimit:
		return 60 * time.Second
	case ErrorTypeNetwork, ErrorTypeTimeout:
		return 5 * time.Second
	case ErrorTypeInternalServer:
		return 10 * time.Second
	default:
		return 1 * time.Second
	}
}

// ErrorClassifier analyzes errors and determines their type and retry characteristics
type ErrorClassifier struct{}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{}
}

// ClassifyError analyzes an error and returns a StrategyError with appropriate classification
func (ec *ErrorClassifier) ClassifyError(err error, contextData map[string]interface{}) *StrategyError {
	if err == nil {
		return nil
	}

	stratError := &StrategyError{
		Type:      ErrorTypeUnknown,
		Message:   err.Error(),
		Cause:     err,
		Retryable: false,
		Context:   contextData,
		Timestamp: time.Now(),
	}

	// Check for context cancellation/timeout
	if errors.Is(err, context.Canceled) {
		stratError.Type = ErrorTypeTimeout
		stratError.Code = "CONTEXT_CANCELED"
		stratError.Message = "Request was cancelled"
		stratError.Retryable = false
		return stratError
	}

	if errors.Is(err, context.DeadlineExceeded) {
		stratError.Type = ErrorTypeTimeout
		stratError.Code = "TIMEOUT"
		stratError.Message = "Request timed out"
		stratError.Retryable = true
		stratError.RetryAfter = 5 * time.Second
		return stratError
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		stratError.Type = ErrorTypeNetwork
		stratError.Retryable = true
		stratError.RetryAfter = 5 * time.Second

		if netErr.Timeout() {
			stratError.Code = "NETWORK_TIMEOUT"
			stratError.Message = "Network request timed out"
		} else {
			stratError.Code = "NETWORK_ERROR"
			stratError.Message = "Network connectivity issue"
		}
		return stratError
	}

	// Analyze error message for common API error patterns
	errMsg := strings.ToLower(err.Error())

	// Rate limiting errors
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many requests") || strings.Contains(errMsg, "429") {
		stratError.Type = ErrorTypeRateLimit
		stratError.Code = "RATE_LIMITED"
		stratError.Message = "API rate limit exceeded"
		stratError.Retryable = true
		stratError.RetryAfter = 60 * time.Second
		return stratError
	}

	// Authentication errors
	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "invalid api key") || strings.Contains(errMsg, "401") {
		stratError.Type = ErrorTypeAuthentication
		stratError.Code = "AUTH_FAILED"
		stratError.Message = "Authentication failed - check API key"
		stratError.Retryable = false
		return stratError
	}

	// Quota exceeded errors
	if strings.Contains(errMsg, "quota") || strings.Contains(errMsg, "billing") || strings.Contains(errMsg, "403") {
		stratError.Type = ErrorTypeQuotaExceeded
		stratError.Code = "QUOTA_EXCEEDED"
		stratError.Message = "API quota exceeded"
		stratError.Retryable = false
		return stratError
	}

	// Invalid request errors
	if strings.Contains(errMsg, "bad request") || strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "400") {
		stratError.Type = ErrorTypeInvalidRequest
		stratError.Code = "INVALID_REQUEST"
		stratError.Message = "Invalid request parameters"
		stratError.Retryable = false
		return stratError
	}

	// Server errors (usually retryable)
	if strings.Contains(errMsg, "internal server error") || strings.Contains(errMsg, "500") ||
		strings.Contains(errMsg, "bad gateway") || strings.Contains(errMsg, "502") ||
		strings.Contains(errMsg, "service unavailable") || strings.Contains(errMsg, "503") ||
		strings.Contains(errMsg, "gateway timeout") || strings.Contains(errMsg, "504") {
		stratError.Type = ErrorTypeInternalServer
		stratError.Code = "SERVER_ERROR"
		stratError.Message = "Server error"
		stratError.Retryable = true
		stratError.RetryAfter = 10 * time.Second
		return stratError
	}

	// JSON/parsing errors
	if strings.Contains(errMsg, "json") || strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "unmarshal") {
		stratError.Type = ErrorTypeParsing
		stratError.Code = "PARSE_ERROR"
		stratError.Message = "Failed to parse API response"
		stratError.Retryable = false
		return stratError
	}

	// Default: unknown error, not retryable unless it looks like a temporary issue
	if strings.Contains(errMsg, "temporary") || strings.Contains(errMsg, "retry") {
		stratError.Retryable = true
		stratError.RetryAfter = 5 * time.Second
	}

	return stratError
}

// RetryPolicy defines how retries should be handled for different error types
type RetryPolicy struct {
	MaxAttempts     map[ErrorType]int
	BackoffStrategy BackoffStrategy
	RetryableErrors map[ErrorType]bool
}

// BackoffStrategy defines how retry delays are calculated
type BackoffStrategy int

const (
	BackoffFixed BackoffStrategy = iota
	BackoffLinear
	BackoffExponential
	BackoffExponentialWithJitter
)

// NewDefaultRetryPolicy creates a sensible default retry policy
func NewDefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts: map[ErrorType]int{
			ErrorTypeNetwork:        3,
			ErrorTypeRateLimit:      1, // Don't retry rate limits, rely on rate limiter
			ErrorTypeTimeout:        2,
			ErrorTypeInternalServer: 3,
			ErrorTypeUnknown:        1,
		},
		BackoffStrategy: BackoffExponentialWithJitter,
		RetryableErrors: map[ErrorType]bool{
			ErrorTypeNetwork:        true,
			ErrorTypeRateLimit:      false, // Handled by rate limiter
			ErrorTypeAuthentication: false,
			ErrorTypeQuotaExceeded:  false,
			ErrorTypeInvalidRequest: false,
			ErrorTypeInternalServer: true,
			ErrorTypeTimeout:        true,
			ErrorTypeParsing:        false,
			ErrorTypeValidation:     false,
			ErrorTypeUnknown:        false,
		},
	}
}

// ShouldRetry determines if an error should be retried based on the policy
func (rp *RetryPolicy) ShouldRetry(err *StrategyError, attempt int) bool {
	if err == nil {
		return false
	}

	// Check if this error type is retryable
	retryable, exists := rp.RetryableErrors[err.Type]
	if !exists || !retryable {
		return false
	}

	// Check if we haven't exceeded max attempts for this error type
	maxAttempts, exists := rp.MaxAttempts[err.Type]
	if !exists {
		maxAttempts = 1 // Default to no retries for unknown error types
	}

	return attempt < maxAttempts
}

// CalculateBackoff calculates the delay before the next retry attempt
func (rp *RetryPolicy) CalculateBackoff(err *StrategyError, attempt int) time.Duration {
	baseDelay := err.GetRetryAfter()

	switch rp.BackoffStrategy {
	case BackoffFixed:
		return baseDelay
	case BackoffLinear:
		return baseDelay * time.Duration(attempt+1)
	case BackoffExponential:
		multiplier := 1
		for i := 0; i < attempt; i++ {
			multiplier *= 2
		}
		return baseDelay * time.Duration(multiplier)
	case BackoffExponentialWithJitter:
		multiplier := 1
		for i := 0; i < attempt; i++ {
			multiplier *= 2
		}
		delay := baseDelay * time.Duration(multiplier)
		// Add jitter (±25% random variation)
		jitter := time.Duration(float64(delay) * 0.25 * (2*randomFloat() - 1))
		return delay + jitter
	default:
		return baseDelay
	}
}

// Simple random float generator for jitter (replace with crypto/rand for production)
func randomFloat() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// ErrorReporter handles error reporting and metrics collection
type ErrorReporter struct {
	errorCounts map[ErrorType]int
	lastErrors  []*StrategyError
	maxHistory  int
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(maxHistory int) *ErrorReporter {
	return &ErrorReporter{
		errorCounts: make(map[ErrorType]int),
		lastErrors:  make([]*StrategyError, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

// ReportError records an error for metrics and debugging
func (er *ErrorReporter) ReportError(err *StrategyError) {
	if err == nil {
		return
	}

	// Update error counts
	er.errorCounts[err.Type]++

	// Add to error history
	er.lastErrors = append(er.lastErrors, err)
	if len(er.lastErrors) > er.maxHistory {
		er.lastErrors = er.lastErrors[1:]
	}
}

// GetErrorStats returns current error statistics
func (er *ErrorReporter) GetErrorStats() map[ErrorType]int {
	stats := make(map[ErrorType]int)
	for errorType, count := range er.errorCounts {
		stats[errorType] = count
	}
	return stats
}

// GetRecentErrors returns the most recent errors
func (er *ErrorReporter) GetRecentErrors(limit int) []*StrategyError {
	if limit <= 0 || limit > len(er.lastErrors) {
		limit = len(er.lastErrors)
	}

	start := len(er.lastErrors) - limit
	result := make([]*StrategyError, limit)
	copy(result, er.lastErrors[start:])
	return result
}
