package main

import (
	"changeme/strategy"
	"fmt"
)

func main() {
	fmt.Println("Testing cache functionality...")

	cache := strategy.NewStrategyCache(nil)
	defer cache.Close()

	// Test basic operations
	key := cache.Put(strategy.CacheTypeStrategy, "test data", []string{"test"})

	if data, exists := cache.Get(key); exists {
		fmt.Printf("✓ Cache stored and retrieved: %v\n", data)
	} else {
		fmt.Println("✗ Cache failed")
	}

	// Test stats
	stats := cache.GetStats()
	fmt.Printf("✓ Cache stats: %d entries, %d bytes\n", stats.TotalEntries, stats.TotalSize)

	fmt.Println("Cache test completed successfully!")
}
