package strategy

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// CacheType represents different types of cache entries
type CacheType string

const (
	CacheTypeStrategy       CacheType = "strategy"
	CacheTypeRecommendation CacheType = "recommendation"
	CacheTypePitTiming      CacheType = "pit_timing"
	CacheTypeWeather        CacheType = "weather"
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Type        CacheType     `json:"type"`
	Key         string        `json:"key"`
	Data        interface{}   `json:"data"`
	Timestamp   time.Time     `json:"timestamp"`
	TTL         time.Duration `json:"ttl"`
	AccessCount int           `json:"access_count"`
	LastAccess  time.Time     `json:"last_access"`
	Size        int           `json:"size"` // Estimated size in bytes
	Tags        []string      `json:"tags"` // For organized cache invalidation
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Since(ce.Timestamp) > ce.TTL
}

// IsStale checks if the cache entry is stale (approaching expiration)
func (ce *CacheEntry) IsStale(staleFactor float64) bool {
	staleDuration := time.Duration(float64(ce.TTL) * staleFactor)
	return time.Since(ce.Timestamp) > staleDuration
}

// CacheStats provides statistics about cache performance
type CacheStats struct {
	TotalEntries     int               `json:"total_entries"`
	TotalSize        int               `json:"total_size_bytes"`
	HitCount         int64             `json:"hit_count"`
	MissCount        int64             `json:"miss_count"`
	EvictionCount    int64             `json:"eviction_count"`
	HitRatio         float64           `json:"hit_ratio"`
	AverageEntrySize int               `json:"average_entry_size"`
	OldestEntry      time.Time         `json:"oldest_entry"`
	NewestEntry      time.Time         `json:"newest_entry"`
	MemoryUsage      int               `json:"memory_usage_bytes"`
	EntriesByType    map[CacheType]int `json:"entries_by_type"`
}

// CacheConfig defines cache behavior and limits
type CacheConfig struct {
	MaxEntries      int           `json:"max_entries"`      // Maximum number of entries
	MaxMemoryBytes  int           `json:"max_memory_bytes"` // Maximum memory usage
	DefaultTTL      time.Duration `json:"default_ttl"`      // Default time-to-live
	StaleFactor     float64       `json:"stale_factor"`     // When to consider entries stale (0.8 = 80% of TTL)
	EnableMetrics   bool          `json:"enable_metrics"`   // Whether to collect detailed metrics
	CleanupInterval time.Duration `json:"cleanup_interval"` // How often to run cleanup
	EvictionPolicy  string        `json:"eviction_policy"`  // "lru", "lfu", "ttl"

	// Type-specific TTLs
	TypeTTLs map[CacheType]time.Duration `json:"type_ttls"`
}

// DefaultCacheConfig returns sensible default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxEntries:      1000,
		MaxMemoryBytes:  50 * 1024 * 1024, // 50MB
		DefaultTTL:      5 * time.Minute,
		StaleFactor:     0.8,
		EnableMetrics:   true,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  "lru",
		TypeTTLs: map[CacheType]time.Duration{
			CacheTypeStrategy:       5 * time.Minute,
			CacheTypeRecommendation: 3 * time.Minute,
			CacheTypePitTiming:      10 * time.Minute,
			CacheTypeWeather:        15 * time.Minute,
		},
	}
}

// StrategyCache provides intelligent caching for strategy analysis results
type StrategyCache struct {
	config    *CacheConfig
	entries   map[string]*CacheEntry
	mutex     sync.RWMutex
	stats     CacheStats
	stopChan  chan struct{}
	isRunning bool
}

// NewStrategyCache creates a new strategy cache with the given configuration
func NewStrategyCache(config *CacheConfig) *StrategyCache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cache := &StrategyCache{
		config:   config,
		entries:  make(map[string]*CacheEntry),
		stopChan: make(chan struct{}),
		stats: CacheStats{
			EntriesByType: make(map[CacheType]int),
		},
	}

	// Start background cleanup if enabled
	if config.CleanupInterval > 0 {
		go cache.backgroundCleanup()
		cache.isRunning = true
	}

	return cache
}

// Get retrieves a cached entry by key
func (sc *StrategyCache) Get(key string) (interface{}, bool) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	entry, exists := sc.entries[key]
	if !exists {
		sc.incrementMisses()
		return nil, false
	}

	if entry.IsExpired() {
		sc.mutex.RUnlock()
		sc.mutex.Lock()
		delete(sc.entries, key)
		sc.decrementTypeCount(entry.Type)
		sc.mutex.Unlock()
		sc.mutex.RLock()
		sc.incrementMisses()
		return nil, false
	}

	// Update access statistics
	sc.mutex.RUnlock()
	sc.mutex.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	sc.mutex.Unlock()
	sc.mutex.RLock()

	sc.incrementHits()
	return entry.Data, true
}

// Put stores a value in the cache with automatic key generation
func (sc *StrategyCache) Put(cacheType CacheType, data interface{}, tags []string) string {
	key := sc.generateKey(cacheType, data)
	sc.PutWithKey(key, cacheType, data, tags)
	return key
}

// PutWithKey stores a value in the cache with a specific key
func (sc *StrategyCache) PutWithKey(key string, cacheType CacheType, data interface{}, tags []string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Calculate TTL for this type
	ttl := sc.config.DefaultTTL
	if typeTTL, exists := sc.config.TypeTTLs[cacheType]; exists {
		ttl = typeTTL
	}

	// Estimate size
	size := sc.estimateSize(data)

	// Check if we need to evict entries
	sc.evictIfNecessary(size)

	// Create new entry
	entry := &CacheEntry{
		Type:        cacheType,
		Key:         key,
		Data:        data,
		Timestamp:   time.Now(),
		TTL:         ttl,
		AccessCount: 1,
		LastAccess:  time.Now(),
		Size:        size,
		Tags:        tags,
	}

	// Remove old entry if it exists
	if oldEntry, exists := sc.entries[key]; exists {
		sc.stats.TotalSize -= oldEntry.Size
		sc.decrementTypeCount(oldEntry.Type)
	}

	// Store new entry
	sc.entries[key] = entry
	sc.stats.TotalSize += size
	sc.stats.TotalEntries = len(sc.entries)
	sc.incrementTypeCount(cacheType)

	// Update newest entry timestamp
	if sc.stats.NewestEntry.IsZero() || entry.Timestamp.After(sc.stats.NewestEntry) {
		sc.stats.NewestEntry = entry.Timestamp
	}
}

// GetWithRefresh gets a cached entry and optionally refreshes it if stale
func (sc *StrategyCache) GetWithRefresh(key string, refreshFunc func() (interface{}, error)) (interface{}, bool, error) {
	sc.mutex.RLock()
	entry, exists := sc.entries[key]
	sc.mutex.RUnlock()

	if !exists {
		sc.incrementMisses()
		if refreshFunc != nil {
			data, err := refreshFunc()
			if err != nil {
				return nil, false, err
			}
			// Note: We can't determine cache type here, so refreshFunc should handle caching
			return data, false, nil
		}
		return nil, false, nil
	}

	if entry.IsExpired() {
		sc.Remove(key)
		sc.incrementMisses()
		if refreshFunc != nil {
			data, err := refreshFunc()
			if err != nil {
				return nil, false, err
			}
			return data, false, nil
		}
		return nil, false, nil
	}

	// Update access statistics
	sc.mutex.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	sc.mutex.Unlock()

	sc.incrementHits()

	// Check if stale and refresh in background if needed
	if entry.IsStale(sc.config.StaleFactor) && refreshFunc != nil {
		go func() {
			if newData, err := refreshFunc(); err == nil {
				sc.PutWithKey(key, entry.Type, newData, entry.Tags)
			}
		}()
	}

	return entry.Data, true, nil
}

// Remove deletes an entry from the cache
func (sc *StrategyCache) Remove(key string) bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	entry, exists := sc.entries[key]
	if !exists {
		return false
	}

	delete(sc.entries, key)
	sc.stats.TotalSize -= entry.Size
	sc.stats.TotalEntries = len(sc.entries)
	sc.decrementTypeCount(entry.Type)

	return true
}

// RemoveByTag removes all entries with the specified tag
func (sc *StrategyCache) RemoveByTag(tag string) int {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	var keysToRemove []string

	for key, entry := range sc.entries {
		for _, entryTag := range entry.Tags {
			if entryTag == tag {
				keysToRemove = append(keysToRemove, key)
				break
			}
		}
	}

	for _, key := range keysToRemove {
		entry := sc.entries[key]
		delete(sc.entries, key)
		sc.stats.TotalSize -= entry.Size
		sc.decrementTypeCount(entry.Type)
	}

	sc.stats.TotalEntries = len(sc.entries)
	return len(keysToRemove)
}

// Clear removes all entries from the cache
func (sc *StrategyCache) Clear() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.entries = make(map[string]*CacheEntry)
	sc.stats.TotalEntries = 0
	sc.stats.TotalSize = 0
	sc.stats.EntriesByType = make(map[CacheType]int)
}

// GetStats returns current cache statistics
func (sc *StrategyCache) GetStats() CacheStats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	stats := sc.stats

	// Calculate hit ratio
	totalRequests := stats.HitCount + stats.MissCount
	if totalRequests > 0 {
		stats.HitRatio = float64(stats.HitCount) / float64(totalRequests)
	}

	// Calculate average entry size
	if stats.TotalEntries > 0 {
		stats.AverageEntrySize = stats.TotalSize / stats.TotalEntries
	}

	// Find oldest entry
	stats.OldestEntry = time.Now()
	for _, entry := range sc.entries {
		if entry.Timestamp.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.Timestamp
		}
	}

	stats.MemoryUsage = stats.TotalSize

	return stats
}

// Close stops background processes and cleans up resources
func (sc *StrategyCache) Close() {
	if sc.isRunning {
		close(sc.stopChan)
		sc.isRunning = false
	}
}

// generateKey creates a consistent cache key from cache type and data
func (sc *StrategyCache) generateKey(cacheType CacheType, data interface{}) string {
	// Serialize data to JSON for consistent hashing
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Fallback to string representation
		jsonData = []byte(fmt.Sprintf("%+v", data))
	}

	// Create MD5 hash
	hash := md5.Sum(jsonData)
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("%s_%s", cacheType, hashStr)
}

// estimateSize estimates the memory size of data
func (sc *StrategyCache) estimateSize(data interface{}) int {
	// Simple estimation based on JSON serialization
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 1024 // Default 1KB if we can't serialize
	}

	// Add overhead for cache entry metadata
	return len(jsonData) + 200 // ~200 bytes for metadata
}

// evictIfNecessary removes old entries if cache limits are exceeded
func (sc *StrategyCache) evictIfNecessary(newEntrySize int) {
	// Check memory limit
	if sc.config.MaxMemoryBytes > 0 && sc.stats.TotalSize+newEntrySize > sc.config.MaxMemoryBytes {
		sc.evictEntries(sc.config.MaxMemoryBytes - newEntrySize)
	}

	// Check entry count limit
	if sc.config.MaxEntries > 0 && len(sc.entries) >= sc.config.MaxEntries {
		sc.evictEntries(sc.stats.TotalSize - newEntrySize - 1)
	}
}

// evictEntries removes entries based on eviction policy until target size is reached
func (sc *StrategyCache) evictEntries(targetSize int) {
	if len(sc.entries) == 0 {
		return
	}

	// Create sorted list of entries for eviction
	type evictionCandidate struct {
		key   string
		entry *CacheEntry
		score float64
	}

	candidates := make([]evictionCandidate, 0, len(sc.entries))

	for key, entry := range sc.entries {
		var score float64

		switch sc.config.EvictionPolicy {
		case "lru":
			// Score by last access time (older = higher score = more likely to evict)
			score = float64(time.Since(entry.LastAccess).Nanoseconds())
		case "lfu":
			// Score by access frequency (lower frequency = higher score)
			score = 1.0 / float64(entry.AccessCount+1)
		case "ttl":
			// Score by time to expiration (closer to expiry = higher score)
			score = float64(time.Since(entry.Timestamp).Nanoseconds()) / float64(entry.TTL.Nanoseconds())
		default:
			// Default to LRU
			score = float64(time.Since(entry.LastAccess).Nanoseconds())
		}

		candidates = append(candidates, evictionCandidate{
			key:   key,
			entry: entry,
			score: score,
		})
	}

	// Sort by eviction score (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Evict entries until we reach target size
	for _, candidate := range candidates {
		if sc.stats.TotalSize <= targetSize {
			break
		}

		delete(sc.entries, candidate.key)
		sc.stats.TotalSize -= candidate.entry.Size
		sc.decrementTypeCount(candidate.entry.Type)
		sc.stats.EvictionCount++
	}
}

// backgroundCleanup runs periodic maintenance on the cache
func (sc *StrategyCache) backgroundCleanup() {
	ticker := time.NewTicker(sc.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sc.cleanupExpiredEntries()
		case <-sc.stopChan:
			return
		}
	}
}

// cleanupExpiredEntries removes expired entries from the cache
func (sc *StrategyCache) cleanupExpiredEntries() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	var expiredKeys []string

	for key, entry := range sc.entries {
		if entry.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		entry := sc.entries[key]
		delete(sc.entries, key)
		sc.stats.TotalSize -= entry.Size
		sc.decrementTypeCount(entry.Type)
	}

	sc.stats.TotalEntries = len(sc.entries)
}

// Helper methods for statistics
func (sc *StrategyCache) incrementHits() {
	sc.stats.HitCount++
}

func (sc *StrategyCache) incrementMisses() {
	sc.stats.MissCount++
}

func (sc *StrategyCache) incrementTypeCount(cacheType CacheType) {
	sc.stats.EntriesByType[cacheType]++
}

func (sc *StrategyCache) decrementTypeCount(cacheType CacheType) {
	if count, exists := sc.stats.EntriesByType[cacheType]; exists && count > 0 {
		sc.stats.EntriesByType[cacheType]--
	}
}
