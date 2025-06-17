// Cache Integration Demo
// This demo shows how the new caching system integrates with the strategy engine

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"changeme/strategy"
)

func main() {
	fmt.Println("=== Strategy Engine Cache Integration Demo ===")

	// Check if API key is available
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("WARNING: GEMINI_API_KEY not set - running cache demo without live API")
		runCacheOnlyDemo()
		return
	}

	runFullIntegrationDemo(apiKey)
}

func runCacheOnlyDemo() {
	fmt.Println("\n--- Cache-Only Demo ---")

	// Create cache with custom configuration
	cacheConfig := strategy.DefaultCacheConfig()
	cacheConfig.MaxEntries = 10
	cacheConfig.DefaultTTL = 30 * time.Second
	cacheConfig.CleanupInterval = 5 * time.Second

	cache := strategy.NewStrategyCache(cacheConfig)
	defer cache.Close()

	fmt.Printf("Created cache with config: MaxEntries=%d, TTL=%v\n",
		cacheConfig.MaxEntries, cacheConfig.DefaultTTL)

	// Create sample strategy analysis
	analysis := &strategy.StrategyAnalysis{
		CurrentSituation:   "Leading by 5 seconds on lap 10",
		PrimaryStrategy:    "Maintain pace, pit on lap 15",
		Confidence:         0.85,
		RaceFormat:         "standard",
		PitWindowOpen:      false,
		RecommendedLap:     15,
		TireRecommendation: "medium",
		FuelStrategy:       "conservative",
		ImmediateActions:   []string{"Monitor tire temperatures", "Maintain gap"},
		Timestamp:          time.Now(),
	}

	// Store analysis with tags
	tags := []string{"lap_10", "position_1", "leading"}
	key := cache.Put(strategy.CacheTypeStrategy, analysis, tags)
	fmt.Printf("Stored analysis with key: %s\n", key)

	// Retrieve from cache
	if cached, exists := cache.Get(key); exists {
		if cachedAnalysis, ok := cached.(*strategy.StrategyAnalysis); ok {
			fmt.Printf("Retrieved from cache: %s\n", cachedAnalysis.CurrentSituation)
			fmt.Printf("Strategy: %s\n", cachedAnalysis.PrimaryStrategy)
			fmt.Printf("Confidence: %.2f\n", cachedAnalysis.Confidence)
		}
	}

	// Show cache statistics
	stats := cache.GetStats()
	fmt.Printf("\nCache Stats:\n")
	fmt.Printf("  Total Entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Total Size: %d bytes\n", stats.TotalSize)
	fmt.Printf("  Hit Count: %d\n", stats.HitCount)
	fmt.Printf("  Miss Count: %d\n", stats.MissCount)
	fmt.Printf("  Hit Ratio: %.2f%%\n", stats.HitRatio*100)

	// Test cache invalidation by tag
	fmt.Printf("\nTesting tag-based invalidation...\n")
	removed := cache.RemoveByTag("lap_10")
	fmt.Printf("Removed %d entries with tag 'lap_10'\n", removed)

	// Verify removal
	if _, exists := cache.Get(key); !exists {
		fmt.Println("✓ Entry successfully removed from cache")
	} else {
		fmt.Println("✗ Entry still exists in cache")
	}

	stats = cache.GetStats()
	fmt.Printf("Cache now has %d entries\n", stats.TotalEntries)
}

func runFullIntegrationDemo(apiKey string) {
	fmt.Println("\n--- Full Integration Demo ---")

	// Create engine configuration with caching
	config := strategy.DefaultConfig()
	config.APIKey = apiKey
	config.EnableCaching = true
	config.CacheTTL = 2 * time.Minute

	// Create custom cache config
	cacheConfig := strategy.DefaultCacheConfig()
	cacheConfig.MaxEntries = 50
	cacheConfig.TypeTTLs[strategy.CacheTypeStrategy] = 3 * time.Minute
	cacheConfig.TypeTTLs[strategy.CacheTypeRecommendation] = 1 * time.Minute
	cacheConfig.TypeTTLs[strategy.CacheTypePitTiming] = 5 * time.Minute
	config.CacheConfig = cacheConfig

	// Create strategy engine
	ctx := context.Background()
	engine, err := strategy.NewStrategyEngine(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create strategy engine: %v", err)
	}
	defer engine.Close()

	fmt.Printf("Created strategy engine with caching enabled\n")

	// Create sample race data
	raceData := &strategy.RaceData{
		SessionType:     "race",
		SessionTime:     1800, // 30 minutes
		SessionTimeLeft: 1200, // 20 minutes left
		CurrentLap:      12,
		Position:        3,
		FuelLevel:       65.5,
		FuelConsumption: 2.8,
		TireWear:        35.0,
		TireCompound:    "medium",
		CurrentLapTime:  89.234,
		BestLapTime:     88.901,
		AverageLapTime:  89.156,
		TrackName:       "Silverstone",
		TrackTemp:       32.0,
		AirTemp:         26.0,
		Weather:         "dry",
		TotalLaps:       52,
		RemainingLaps:   40,
		SafetyCarActive: false,
	}

	fmt.Printf("Running analysis for lap %d, position %d\n", raceData.CurrentLap, raceData.Position)

	// First analysis call - should hit API
	fmt.Println("\n1. First analysis (should hit API)...")
	start := time.Now()
	analysis1, err := engine.AnalyzeStrategy(raceData, "routine")
	duration1 := time.Since(start)

	if err != nil {
		log.Printf("Analysis failed: %v", err)
		return
	}

	fmt.Printf("   ✓ Analysis completed in %v\n", duration1)
	fmt.Printf("   Situation: %s\n", analysis1.CurrentSituation)
	fmt.Printf("   Strategy: %s\n", analysis1.PrimaryStrategy)

	// Show cache stats after first call
	cacheStats := engine.GetCacheStats()
	fmt.Printf("   Cache entries: %d\n", cacheStats.TotalEntries)

	// Second analysis call with same data - should hit cache
	fmt.Println("\n2. Second analysis with same data (should hit cache)...")
	start = time.Now()
	analysis2, err := engine.AnalyzeStrategy(raceData, "routine")
	duration2 := time.Since(start)

	if err != nil {
		log.Printf("Analysis failed: %v", err)
		return
	}

	fmt.Printf("   ✓ Analysis completed in %v\n", duration2)
	fmt.Printf("   Cache speedup: %.1fx faster\n", float64(duration1)/float64(duration2))

	// Verify we got cached data
	if analysis1.Timestamp.Equal(analysis2.Timestamp) {
		fmt.Println("   ✓ Got cached analysis (timestamps match)")
	} else {
		fmt.Println("   ? Got fresh analysis (timestamps differ)")
	}

	// Test cache with different analysis type
	fmt.Println("\n3. Different analysis type (should hit API again)...")
	start = time.Now()
	analysis3, err := engine.AnalyzeStrategy(raceData, "pit_decision")
	duration3 := time.Since(start)

	if err != nil {
		log.Printf("Analysis failed: %v", err)
		return
	}

	fmt.Printf("   ✓ Pit analysis completed in %v\n", duration3)
	fmt.Printf("   Pit recommendation: lap %d\n", analysis3.RecommendedLap)

	// Final cache stats
	cacheStats = engine.GetCacheStats()
	fmt.Printf("\nFinal Cache Statistics:\n")
	fmt.Printf("  Total Entries: %d\n", cacheStats.TotalEntries)
	fmt.Printf("  Total Size: %d bytes\n", cacheStats.TotalSize)
	fmt.Printf("  Hit Count: %d\n", cacheStats.HitCount)
	fmt.Printf("  Miss Count: %d\n", cacheStats.MissCount)
	fmt.Printf("  Hit Ratio: %.2f%%\n", cacheStats.HitRatio*100)
	fmt.Printf("  Entries by type: %v\n", cacheStats.EntriesByType)

	// Test cache invalidation
	fmt.Println("\n4. Testing cache invalidation...")
	beforeCount := cacheStats.TotalEntries
	removed := engine.InvalidateCacheByTag("lap_12")
	afterStats := engine.GetCacheStats()

	fmt.Printf("   Removed %d entries for lap 12\n", removed)
	fmt.Printf("   Cache entries: %d → %d\n", beforeCount, afterStats.TotalEntries)
	// Test cache with regular analysis (already has caching built-in)
	fmt.Println("\n5. Testing cache with fresh analysis...")

	// Modify race data slightly to get fresh analysis
	raceData.CurrentLap = 13
	raceData.TireWear = 40.0

	start = time.Now()
	analysis4, err := engine.AnalyzeStrategy(raceData, "routine")
	duration4 := time.Since(start)

	if err != nil {
		log.Printf("Fresh analysis failed: %v", err)
		return
	}

	fmt.Printf("   ✓ Analysis for updated data completed in %v\n", duration4)
	fmt.Printf("   Strategy: %s\n", analysis4.PrimaryStrategy)

	fmt.Println("\n=== Demo Complete ===")
}
