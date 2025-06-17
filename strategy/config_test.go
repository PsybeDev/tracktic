package strategy

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	if config.Model != "gemini-2.0-flash" {
		t.Errorf("Expected model 'gemini-2.0-flash', got %s", config.Model)
	}

	if config.MaxTokens != 8192 {
		t.Errorf("Expected MaxTokens 8192, got %d", config.MaxTokens)
	}

	if config.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", config.Temperature)
	}

	if config.MaxRequestsPerMinute != 10 {
		t.Errorf("Expected MaxRequestsPerMinute 10, got %d", config.MaxRequestsPerMinute)
	}

	if config.EnableCaching != true {
		t.Error("Expected EnableCaching to be true")
	}

	if config.CacheTTL != 5*time.Minute {
		t.Errorf("Expected CacheTTL 5 minutes, got %v", config.CacheTTL)
	}

	// Test update intervals
	if config.UpdateIntervals.LapInterval != 3 {
		t.Errorf("Expected LapInterval 3, got %d", config.UpdateIntervals.LapInterval)
	}

	if config.UpdateIntervals.FuelWarningThreshold != 15.0 {
		t.Errorf("Expected FuelWarningThreshold 15.0, got %f", config.UpdateIntervals.FuelWarningThreshold)
	}

	// Test analysis preferences
	if config.AnalysisPreferences.RaceFormat != "auto" {
		t.Errorf("Expected RaceFormat 'auto', got %s", config.AnalysisPreferences.RaceFormat)
	}

	if config.AnalysisPreferences.SafetyMargin != 1.1 {
		t.Errorf("Expected SafetyMargin 1.1, got %f", config.AnalysisPreferences.SafetyMargin)
	}
}

func TestLoadConfig(t *testing.T) {
	// Test with no environment variables
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Should use defaults when no env vars are set
	if config.Model != "gemini-2.0-flash" {
		t.Errorf("Expected default model, got %s", config.Model)
	}

	// Test with API key environment variable
	testAPIKey := "test-api-key-12345"
	oldAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Setenv("GEMINI_API_KEY", testAPIKey)
	defer func() {
		if oldAPIKey == "" {
			os.Unsetenv("GEMINI_API_KEY")
		} else {
			os.Setenv("GEMINI_API_KEY", oldAPIKey)
		}
	}()

	config, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig with API key failed: %v", err)
	}

	if config.APIKey != testAPIKey {
		t.Errorf("Expected API key %s, got %s", testAPIKey, config.APIKey)
	}
}

func TestConfigValidation(t *testing.T) {
	config := DefaultConfig()

	// Test valid configuration
	if err := config.Validate(); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// Test invalid API key
	config.APIKey = ""
	if err := config.Validate(); err == nil {
		t.Error("Empty API key should return validation error")
	}

	// Test invalid model
	config.APIKey = "valid-key"
	config.Model = ""
	if err := config.Validate(); err == nil {
		t.Error("Empty model should return validation error")
	}

	// Test invalid max tokens
	config.Model = "gemini-2.0-flash"
	config.MaxTokens = 0
	if err := config.Validate(); err == nil {
		t.Error("Zero MaxTokens should return validation error")
	}

	// Test invalid temperature range
	config.MaxTokens = 8192
	config.Temperature = -0.1
	if err := config.Validate(); err == nil {
		t.Error("Negative temperature should return validation error")
	}

	config.Temperature = 2.1
	if err := config.Validate(); err == nil {
		t.Error("Temperature > 2.0 should return validation error")
	}

	// Test invalid rate limiting values
	config.Temperature = 0.7
	config.MaxRequestsPerMinute = 0
	if err := config.Validate(); err == nil {
		t.Error("Zero MaxRequestsPerMinute should return validation error")
	}

	config.MaxRequestsPerMinute = 10
	config.BurstLimit = 0
	if err := config.Validate(); err == nil {
		t.Error("Zero BurstLimit should return validation error")
	}
}

func TestUpdateConfigValidation(t *testing.T) {
	updateConfig := UpdateConfig{
		LapInterval:          3,
		CriticalEventDelay:   30 * time.Second,
		PitWindowNotice:      2 * time.Minute,
		FuelWarningThreshold: 15.0,
		TireWearThreshold:    80.0,
	}

	// Test valid update config
	if err := updateConfig.Validate(); err != nil {
		t.Errorf("Valid update config should not return error: %v", err)
	}

	// Test invalid lap interval
	updateConfig.LapInterval = 0
	if err := updateConfig.Validate(); err == nil {
		t.Error("Zero LapInterval should return validation error")
	}

	// Test invalid fuel threshold
	updateConfig.LapInterval = 3
	updateConfig.FuelWarningThreshold = -1.0
	if err := updateConfig.Validate(); err == nil {
		t.Error("Negative FuelWarningThreshold should return validation error")
	}

	updateConfig.FuelWarningThreshold = 101.0
	if err := updateConfig.Validate(); err == nil {
		t.Error("FuelWarningThreshold > 100 should return validation error")
	}

	// Test invalid tire wear threshold
	updateConfig.FuelWarningThreshold = 15.0
	updateConfig.TireWearThreshold = -1.0
	if err := updateConfig.Validate(); err == nil {
		t.Error("Negative TireWearThreshold should return validation error")
	}

	updateConfig.TireWearThreshold = 101.0
	if err := updateConfig.Validate(); err == nil {
		t.Error("TireWearThreshold > 100 should return validation error")
	}
}

func TestAnalysisConfigValidation(t *testing.T) {
	analysisConfig := AnalysisConfig{
		RaceFormat:            "auto",
		PrioritizeConsistency: false,
		IncludeOpponentData:   true,
		WeatherConsideration:  true,
		SafetyMargin:          1.1,
	}

	// Test valid analysis config
	if err := analysisConfig.Validate(); err != nil {
		t.Errorf("Valid analysis config should not return error: %v", err)
	}

	// Test invalid race format
	analysisConfig.RaceFormat = "invalid"
	if err := analysisConfig.Validate(); err == nil {
		t.Error("Invalid RaceFormat should return validation error")
	}

	// Test invalid safety margin
	analysisConfig.RaceFormat = "auto"
	analysisConfig.SafetyMargin = 0.5
	if err := analysisConfig.Validate(); err == nil {
		t.Error("SafetyMargin < 1.0 should return validation error")
	}

	analysisConfig.SafetyMargin = 3.0
	if err := analysisConfig.Validate(); err == nil {
		t.Error("SafetyMargin > 2.0 should return validation error")
	}
}

func TestConfigClone(t *testing.T) {
	original := DefaultConfig()
	original.APIKey = "test-key"
	original.MaxTokens = 4096

	cloned := original.Clone()

	// Test that values are copied
	if cloned.APIKey != original.APIKey {
		t.Error("APIKey not cloned correctly")
	}

	if cloned.MaxTokens != original.MaxTokens {
		t.Error("MaxTokens not cloned correctly")
	}

	// Test that modifying clone doesn't affect original
	cloned.APIKey = "different-key"
	cloned.MaxTokens = 2048

	if original.APIKey == cloned.APIKey {
		t.Error("Clone should be independent of original")
	}

	if original.MaxTokens == cloned.MaxTokens {
		t.Error("Clone should be independent of original")
	}
}

func TestConfigJSON(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	// Test marshaling to JSON
	jsonData, err := config.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Test unmarshaling from JSON
	newConfig := &Config{}
	if err := newConfig.FromJSON(jsonData); err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Verify values are preserved
	if newConfig.APIKey != config.APIKey {
		t.Error("APIKey not preserved in JSON roundtrip")
	}

	if newConfig.Model != config.Model {
		t.Error("Model not preserved in JSON roundtrip")
	}

	if newConfig.MaxTokens != config.MaxTokens {
		t.Error("MaxTokens not preserved in JSON roundtrip")
	}
}

func TestCacheConfigIntegration(t *testing.T) {
	config := DefaultConfig()

	// Test that CacheConfig can be set
	cacheConfig := DefaultCacheConfig()
	cacheConfig.MaxEntries = 500
	config.CacheConfig = cacheConfig

	if config.CacheConfig.MaxEntries != 500 {
		t.Error("CacheConfig not properly integrated")
	}

	// Test backward compatibility
	if config.EnableCaching != true {
		t.Error("Backward compatibility with EnableCaching broken")
	}

	if config.CacheTTL != 5*time.Minute {
		t.Error("Backward compatibility with CacheTTL broken")
	}
}

func BenchmarkConfigCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultConfig()
	config.APIKey = "test-key"

	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

func BenchmarkConfigClone(b *testing.B) {
	config := DefaultConfig()

	for i := 0; i < b.N; i++ {
		_ = config.Clone()
	}
}
