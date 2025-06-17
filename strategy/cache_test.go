package strategy

import (
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	cache := NewStrategyCache(nil) // Use default config
	defer cache.Close()

	// Test Put and Get
	data := "test data"
	key := cache.Put(CacheTypeStrategy, data, []string{"test"})

	result, exists := cache.Get(key)
	if !exists {
		t.Error("Expected cache hit, got miss")
	}

	if result != data {
		t.Errorf("Expected %v, got %v", data, result)
	}

	// Test non-existent key
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Error("Expected cache miss, got hit")
	}
}

func TestCacheExpiration(t *testing.T) {
	config := DefaultCacheConfig()
	config.DefaultTTL = 100 * time.Millisecond
	config.TypeTTLs[CacheTypeStrategy] = 100 * time.Millisecond

	cache := NewStrategyCache(config)
	defer cache.Close()

	// Store data
	data := "expiring data"
	key := cache.Put(CacheTypeStrategy, data, nil)

	// Should exist immediately
	_, exists := cache.Get(key)
	if !exists {
		t.Error("Expected cache hit immediately after storing")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, exists = cache.Get(key)
	if exists {
		t.Error("Expected cache miss after expiration")
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewStrategyCache(nil)
	defer cache.Close()

	// Initial stats should be empty
	stats := cache.GetStats()
	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 entries, got %d", stats.TotalEntries)
	}

	// Add some data
	cache.Put(CacheTypeStrategy, "data1", nil)
	cache.Put(CacheTypeRecommendation, "data2", nil)

	stats = cache.GetStats()
	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.TotalEntries)
	}

	// Test hits and misses
	key := cache.Put(CacheTypeStrategy, "test", nil)
	cache.Get(key)           // Hit
	cache.Get("nonexistent") // Miss

	stats = cache.GetStats()
	if stats.HitCount == 0 {
		t.Error("Expected at least one hit")
	}
	if stats.MissCount == 0 {
		t.Error("Expected at least one miss")
	}
}

func TestCacheTagging(t *testing.T) {
	cache := NewStrategyCache(nil)
	defer cache.Close()

	// Add entries with tags
	cache.Put(CacheTypeStrategy, "data1", []string{"lap_5", "position_1"})
	cache.Put(CacheTypeStrategy, "data2", []string{"lap_5", "position_2"})
	cache.Put(CacheTypeStrategy, "data3", []string{"lap_6", "position_1"})

	stats := cache.GetStats()
	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 entries, got %d", stats.TotalEntries)
	}

	// Remove by tag
	removed := cache.RemoveByTag("lap_5")
	if removed != 2 {
		t.Errorf("Expected 2 entries removed, got %d", removed)
	}

	stats = cache.GetStats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry remaining, got %d", stats.TotalEntries)
	}
}

func TestCacheEviction(t *testing.T) {
	config := DefaultCacheConfig()
	config.MaxEntries = 3
	config.EvictionPolicy = "lru"

	cache := NewStrategyCache(config)
	defer cache.Close()
	// Fill cache to capacity
	key1 := cache.Put(CacheTypeStrategy, "data1", nil)
	key2 := cache.Put(CacheTypeStrategy, "data2", nil)
	_ = cache.Put(CacheTypeStrategy, "data3", nil)

	stats := cache.GetStats()
	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 entries, got %d", stats.TotalEntries)
	}

	// Access key1 to make it recently used
	cache.Get(key1)

	// Add one more entry, should evict key2 (least recently used)
	cache.Put(CacheTypeStrategy, "data4", nil)

	// key1 should still exist (recently accessed)
	_, exists := cache.Get(key1)
	if !exists {
		t.Error("Expected key1 to still exist (recently accessed)")
	}

	// key2 should be evicted
	_, exists = cache.Get(key2)
	if exists {
		t.Error("Expected key2 to be evicted")
	}
}

func TestCacheWithRefresh(t *testing.T) {
	config := DefaultCacheConfig()
	config.DefaultTTL = 100 * time.Millisecond
	config.StaleFactor = 0.5 // 50% of TTL

	cache := NewStrategyCache(config)
	defer cache.Close()

	refreshCalled := false
	refreshFunc := func() (interface{}, error) {
		refreshCalled = true
		return "refreshed data", nil
	}

	// Store initial data
	key := cache.Put(CacheTypeStrategy, "original data", nil)

	// Should get original data
	result, fromCache, err := cache.GetWithRefresh(key, refreshFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !fromCache {
		t.Error("Expected data from cache")
	}
	if result != "original data" {
		t.Errorf("Expected 'original data', got %v", result)
	}

	// Wait for data to become stale
	time.Sleep(60 * time.Millisecond) // More than 50% of 100ms TTL

	// Should still get cached data but trigger background refresh
	result, fromCache, err = cache.GetWithRefresh(key, refreshFunc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !fromCache {
		t.Error("Expected data from cache")
	}

	// Give background refresh time to complete
	time.Sleep(10 * time.Millisecond)

	// Refresh should have been called for stale data
	if !refreshCalled {
		t.Error("Expected refresh function to be called for stale data")
	}
}

func TestStrategyAnalysisCaching(t *testing.T) {
	// This tests integration with StrategyAnalysis objects
	cache := NewStrategyCache(nil)
	defer cache.Close()

	analysis := &StrategyAnalysis{
		CurrentSituation: "Test situation",
		PrimaryStrategy:  "Test strategy",
		Confidence:       0.8,
		RaceFormat:       "standard",
	}

	key := cache.Put(CacheTypeStrategy, analysis, []string{"test"})

	result, exists := cache.Get(key)
	if !exists {
		t.Error("Expected cache hit")
	}

	cachedAnalysis, ok := result.(*StrategyAnalysis)
	if !ok {
		t.Error("Expected StrategyAnalysis type")
	}

	if cachedAnalysis.CurrentSituation != analysis.CurrentSituation {
		t.Errorf("Expected %v, got %v", analysis.CurrentSituation, cachedAnalysis.CurrentSituation)
	}

	if cachedAnalysis.Confidence != analysis.Confidence {
		t.Errorf("Expected %v, got %v", analysis.Confidence, cachedAnalysis.Confidence)
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	cache := NewStrategyCache(nil)
	defer cache.Close()

	// Same data should generate same key
	data1 := map[string]interface{}{"test": "value"}
	data2 := map[string]interface{}{"test": "value"}

	key1 := cache.Put(CacheTypeStrategy, data1, nil)
	key2 := cache.generateKey(CacheTypeStrategy, data2)

	if key1 != key2 {
		t.Error("Same data should generate same cache key")
	}

	// Different data should generate different keys
	data3 := map[string]interface{}{"test": "different"}
	key3 := cache.generateKey(CacheTypeStrategy, data3)

	if key1 == key3 {
		t.Error("Different data should generate different cache keys")
	}
}

func BenchmarkCacheOperations(b *testing.B) {
	cache := NewStrategyCache(nil)
	defer cache.Close()

	// Benchmark Put operations
	b.Run("Put", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Put(CacheTypeStrategy, i, nil)
		}
	})

	// Setup for Get benchmark
	for i := 0; i < 1000; i++ {
		cache.Put(CacheTypeStrategy, i, nil)
	}

	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = cache.generateKey(CacheTypeStrategy, i)
	}

	// Benchmark Get operations
	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Get(keys[i%1000])
		}
	})
}
