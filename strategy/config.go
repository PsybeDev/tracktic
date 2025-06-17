package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/genai"
)

// Config holds configuration for the strategy engine
type Config struct {
	// Gemini API Configuration
	APIKey         string        `json:"api_key"`
	Model          string        `json:"model"`           // Default: "gemini-2.0-flash"
	MaxTokens      int           `json:"max_tokens"`      // Default: 8192
	Temperature    float64       `json:"temperature"`     // Default: 0.7
	TopP           float64       `json:"top_p"`           // Default: 0.95
	TopK           int           `json:"top_k"`           // Default: 40
	RequestTimeout time.Duration `json:"request_timeout"` // Default: 30s

	// Rate Limiting Configuration
	MaxRequestsPerMinute int           `json:"max_requests_per_minute"` // Default: 10
	BurstLimit           int           `json:"burst_limit"`             // Default: 3
	RetryAttempts        int           `json:"retry_attempts"`          // Default: 3
	RetryDelay           time.Duration `json:"retry_delay"`             // Default: 1s
	// Caching Configuration
	EnableCaching bool          `json:"enable_caching"` // Default: true
	CacheTTL      time.Duration `json:"cache_ttl"`      // Default: 5 minutes
	MaxCacheSize  int           `json:"max_cache_size"` // Default: 100 entries
	CacheConfig   *CacheConfig  `json:"cache_config"`   // Advanced cache configuration

	// Strategy Configuration
	EnableVoiceUpdates  bool           `json:"enable_voice_updates"` // Default: true
	UpdateIntervals     UpdateConfig   `json:"update_intervals"`
	AnalysisPreferences AnalysisConfig `json:"analysis_preferences"`
}

// UpdateConfig defines when and how to provide strategy updates
type UpdateConfig struct {
	LapInterval          int           `json:"lap_interval"`           // Default: 3 (every 3rd lap)
	CriticalEventDelay   time.Duration `json:"critical_event_delay"`   // Default: 30s
	PitWindowNotice      time.Duration `json:"pit_window_notice"`      // Default: 2 minutes
	FuelWarningThreshold float64       `json:"fuel_warning_threshold"` // Default: 15% (percentage)
	TireWearThreshold    float64       `json:"tire_wear_threshold"`    // Default: 80% (percentage)
}

// AnalysisConfig defines strategy analysis preferences
type AnalysisConfig struct {
	RaceFormat            string  `json:"race_format"`            // "auto", "sprint", "endurance"
	PrioritizeConsistency bool    `json:"prioritize_consistency"` // vs aggressive strategy
	IncludeOpponentData   bool    `json:"include_opponent_data"`  // Default: true
	WeatherConsideration  bool    `json:"weather_consideration"`  // Default: true
	SafetyMargin          float64 `json:"safety_margin"`          // Default: 1.1 (10% safety margin)
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Model:          "gemini-2.0-flash",
		MaxTokens:      8192,
		Temperature:    0.7,
		TopP:           0.95,
		TopK:           40,
		RequestTimeout: 30 * time.Second,

		MaxRequestsPerMinute: 10,
		BurstLimit:           3,
		RetryAttempts:        3,
		RetryDelay:           1 * time.Second,

		EnableCaching: true,
		CacheTTL:      5 * time.Minute,
		MaxCacheSize:  100,

		EnableVoiceUpdates: true,
		UpdateIntervals: UpdateConfig{
			LapInterval:          3,
			CriticalEventDelay:   30 * time.Second,
			PitWindowNotice:      2 * time.Minute,
			FuelWarningThreshold: 15.0,
			TireWearThreshold:    80.0,
		},
		AnalysisPreferences: AnalysisConfig{
			RaceFormat:            "auto",
			PrioritizeConsistency: false,
			IncludeOpponentData:   true,
			WeatherConsideration:  true,
			SafetyMargin:          1.1,
		},
	}
}

// LoadConfig loads configuration from environment variables and applies defaults
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	// Load API key from environment
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		// Also check for alternative environment variable names
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not found: set GOOGLE_API_KEY or GEMINI_API_KEY environment variable")
	}

	config.APIKey = apiKey

	// TODO: Add support for loading other config values from environment or config file
	// For now, we'll use defaults and allow runtime configuration

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	if c.Model == "" {
		return fmt.Errorf("model name is required")
	}

	if c.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	if c.Temperature < 0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0 and 2.0")
	}

	if c.TopP < 0 || c.TopP > 1.0 {
		return fmt.Errorf("top_p must be between 0 and 1.0")
	}

	if c.TopK <= 0 {
		return fmt.Errorf("top_k must be positive")
	}

	if c.MaxRequestsPerMinute <= 0 {
		return fmt.Errorf("max_requests_per_minute must be positive")
	}

	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts cannot be negative")
	}

	if err := c.UpdateIntervals.Validate(); err != nil {
		return fmt.Errorf("invalid update intervals: %w", err)
	}

	if err := c.AnalysisPreferences.Validate(); err != nil {
		return fmt.Errorf("invalid analysis preferences: %w", err)
	}

	return nil
}

// Validate checks if the UpdateConfig is valid
func (uc *UpdateConfig) Validate() error {
	if uc.LapInterval <= 0 {
		return fmt.Errorf("lap_interval must be positive")
	}

	if uc.FuelWarningThreshold < 0 || uc.FuelWarningThreshold > 100 {
		return fmt.Errorf("fuel_warning_threshold must be between 0 and 100")
	}

	if uc.TireWearThreshold < 0 || uc.TireWearThreshold > 100 {
		return fmt.Errorf("tire_wear_threshold must be between 0 and 100")
	}

	return nil
}

// Validate checks if the AnalysisConfig is valid
func (ac *AnalysisConfig) Validate() error {
	validFormats := map[string]bool{
		"auto":      true,
		"sprint":    true,
		"endurance": true,
		"standard":  true,
	}

	if !validFormats[ac.RaceFormat] {
		return fmt.Errorf("race_format must be one of: auto, sprint, endurance, standard")
	}

	if ac.SafetyMargin < 1.0 || ac.SafetyMargin > 2.0 {
		return fmt.Errorf("safety_margin must be between 1.0 and 2.0")
	}

	return nil
}

// Clone creates a deep copy of the Config
func (c *Config) Clone() *Config {
	clone := *c

	// Deep copy nested structs
	clone.UpdateIntervals = c.UpdateIntervals
	clone.AnalysisPreferences = c.AnalysisPreferences

	// Deep copy cache config if it exists
	if c.CacheConfig != nil {
		cacheClone := *c.CacheConfig
		clone.CacheConfig = &cacheClone

		// Deep copy the TypeTTLs map
		if c.CacheConfig.TypeTTLs != nil {
			clone.CacheConfig.TypeTTLs = make(map[CacheType]time.Duration)
			for k, v := range c.CacheConfig.TypeTTLs {
				clone.CacheConfig.TypeTTLs[k] = v
			}
		}
	}

	return &clone
}

// ToJSON serializes the Config to JSON
func (c *Config) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON deserializes JSON into the Config
func (c *Config) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// GeminiClient wraps the Google Gen AI client with our configuration
type GeminiClient struct {
	client *genai.Client
	config *Config
	ctx    context.Context
}

// NewGeminiClient creates a new Gemini API client with the given configuration
func NewGeminiClient(ctx context.Context, config *Config) (*GeminiClient, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create the Gen AI client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiClient{
		client: client,
		config: config,
		ctx:    ctx,
	}, nil
}

// Close closes the Gemini client and releases resources
func (gc *GeminiClient) Close() error {
	// The genai.Client doesn't appear to have a Close method
	// We'll just set the client to nil for cleanup
	gc.client = nil
	return nil
}

// GetConfig returns the current configuration
func (gc *GeminiClient) GetConfig() *Config {
	return gc.config
}

// UpdateConfig updates the client configuration
func (gc *GeminiClient) UpdateConfig(config *Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// If API key changed, we need to recreate the client
	if config.APIKey != gc.config.APIKey {
		if err := gc.Close(); err != nil {
			log.Printf("Warning: failed to close existing client: %v", err)
		}

		client, err := genai.NewClient(gc.ctx, &genai.ClientConfig{
			APIKey:  config.APIKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return fmt.Errorf("failed to create new Gemini client: %w", err)
		}

		gc.client = client
	}

	gc.config = config
	return nil
}

// HealthCheck verifies that the Gemini API is accessible
func (gc *GeminiClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(gc.ctx, gc.config.RequestTimeout)
	defer cancel()

	// Make a simple request to verify connectivity
	result, err := gc.client.Models.GenerateContent(ctx, gc.config.Model, []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: "Hello"},
			},
		},
	}, nil)

	if err != nil {
		return fmt.Errorf("Gemini API health check failed: %w", err)
	}

	if result == nil {
		return fmt.Errorf("Gemini API returned nil result")
	}

	return nil
}
