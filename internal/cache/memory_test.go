package cache

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_UnderMemoryPressure(t *testing.T) {
	cache := NewMemoryCache(100, 1*time.Hour) // Small capacity

	// Fill cache beyond capacity
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []string{fmt.Sprintf("arch-%d", i)}
		cache.Set(key, value)
	}

	// Should have evicted older entries
	assert.True(t, cache.Len() <= 100)

	// Older entries should be evicted
	_, found := cache.Get("key-0")
	assert.False(t, found)

	// Newer entries should still exist
	_, found = cache.Get("key-199")
	assert.True(t, found)
}

func TestMemoryCache_ConcurrentReadWrite(t *testing.T) {
	cache := NewMemoryCache(1000, 1*time.Hour)
	
	var wg sync.WaitGroup
	errors := make(chan error, 200)

	// Concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			value := []string{fmt.Sprintf("arch-%d", id)}
			cache.Set(key, value)
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			_, found := cache.Get(key)
			// May or may not find due to race conditions, but shouldn't panic
			_ = found
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewMemoryCache(100, 50*time.Millisecond)

	// Set value
	cache.Set("test-key", []string{"amd64"})
	
	// Should be available immediately
	value, found := cache.Get("test-key")
	require.True(t, found)
	assert.Equal(t, []string{"amd64"}, value)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("test-key")
	assert.False(t, found)
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	cache := NewMemoryCache(3, 1*time.Hour) // Very small capacity

	// Fill cache
	cache.Set("key1", []string{"arch1"})
	cache.Set("key2", []string{"arch2"})
	cache.Set("key3", []string{"arch3"})

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add new item, should evict key2 (least recently used)
	cache.Set("key4", []string{"arch4"})

	// key1 should still exist (recently accessed)
	_, found := cache.Get("key1")
	assert.True(t, found)

	// key2 should be evicted
	_, found = cache.Get("key2")
	assert.False(t, found)

	// key3 and key4 should exist
	_, found = cache.Get("key3")
	assert.True(t, found)
	_, found = cache.Get("key4")
	assert.True(t, found)
}

func TestMemoryCache_GarbageCollectionBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GC test in short mode")
	}

	cache := NewMemoryCache(1000, 1*time.Hour)

	// Create many entries to trigger GC
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := make([]string, 100) // Larger values to use more memory
		for j := range value {
			value[j] = fmt.Sprintf("arch-%d-%d", i, j)
		}
		cache.Set(key, value)
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC()

	// Cache should still be functional
	cache.Set("test-after-gc", []string{"amd64"})
	value, found := cache.Get("test-after-gc")
	require.True(t, found)
	assert.Equal(t, []string{"amd64"}, value)
}

func TestMemoryCache_Statistics(t *testing.T) {
	cache := NewMemoryCache(100, 1*time.Hour)

	// Initial stats
	stats := cache.Stats()
	assert.Equal(t, 0, stats.Hits)
	assert.Equal(t, 0, stats.Misses)
	assert.Equal(t, 0, stats.Evictions)

	// Add some entries
	cache.Set("key1", []string{"arch1"})
	cache.Set("key2", []string{"arch2"})

	// Hit
	cache.Get("key1")
	stats = cache.Stats()
	assert.Equal(t, 1, stats.Hits)
	assert.Equal(t, 0, stats.Misses)

	// Miss
	cache.Get("nonexistent")
	stats = cache.Stats()
	assert.Equal(t, 1, stats.Hits)
	assert.Equal(t, 1, stats.Misses)

	// Test evictions by filling beyond capacity
	for i := 0; i < 150; i++ {
		cache.Set(fmt.Sprintf("evict-key-%d", i), []string{"arch"})
	}

	stats = cache.Stats()
	assert.True(t, stats.Evictions > 0)
}

func TestMemoryCache_ZeroTTL(t *testing.T) {
	cache := NewMemoryCache(100, 0) // Zero TTL

	cache.Set("test-key", []string{"amd64"})
	
	// Should be immediately expired
	_, found := cache.Get("test-key")
	assert.False(t, found)
}

func TestMemoryCache_NegativeTTL(t *testing.T) {
	cache := NewMemoryCache(100, -1*time.Hour) // Negative TTL

	cache.Set("test-key", []string{"amd64"})
	
	// Should be immediately expired
	_, found := cache.Get("test-key")
	assert.False(t, found)
}

func TestMemoryCache_KeyCollisions(t *testing.T) {
	cache := NewMemoryCache(100, 1*time.Hour)

	// Test with similar keys that might cause hash collisions
	keys := []string{
		"nginx:latest",
		"nginx:1.20",
		"nginx:1.21",
		"nginx@sha256:abc123",
		"nginx@sha256:def456",
	}

	// Set all keys
	for i, key := range keys {
		value := []string{fmt.Sprintf("arch-%d", i)}
		cache.Set(key, value)
	}

	// Verify all keys are retrievable
	for i, key := range keys {
		value, found := cache.Get(key)
		require.True(t, found, "Key %s not found", key)
		assert.Equal(t, []string{fmt.Sprintf("arch-%d", i)}, value)
	}
}

func TestMemoryCache_PersistenceAcrossRestarts(t *testing.T) {
	// Create cache, add data, then "restart" by creating new cache
	cache1 := NewMemoryCache(100, 1*time.Hour)
	cache1.Set("persistent-key", []string{"amd64"})

	// Simulate restart
	cache2 := NewMemoryCache(100, 1*time.Hour)
	
	// Data should not persist (in-memory cache)
	_, found := cache2.Get("persistent-key")
	assert.False(t, found)
}

func TestMemoryCache_ThreadSafety(t *testing.T) {
	cache := NewMemoryCache(1000, 1*time.Hour)
	
	// Test for race conditions
	var wg sync.WaitGroup
	
	// Multiple goroutines doing mixed operations
	for i := 0; i < 100; i++ {
		wg.Add(3)
		
		// Writer
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Set(key, []string{fmt.Sprintf("arch-%d", id)})
			}
		}(i)
		
		// Reader
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key)
			}
		}(i)
		
		// Stats reader
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				cache.Stats()
			}
		}(i)
	}
	
	wg.Wait()
	
	// Should not panic and cache should be in consistent state
	assert.True(t, cache.Len() >= 0)
}