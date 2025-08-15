package cache

import (
	"sync"
	"time"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value      []string
	Expiration time.Time
	AccessTime time.Time
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits      int
	Misses    int
	Evictions int
}

// MemoryCache implements an in-memory LRU cache with TTL
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*CacheEntry
	capacity int
	ttl      time.Duration
	stats    CacheStats
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(capacity int, ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		items:    make(map[string]*CacheEntry),
		capacity: capacity,
		ttl:      ttl,
	}
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) ([]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.items[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if c.ttl > 0 && time.Now().After(entry.Expiration) {
		delete(c.items, key)
		c.stats.Misses++
		return nil, false
	}

	// Update access time for LRU
	entry.AccessTime = time.Now()
	c.stats.Hits++
	return entry.Value, true
}

// Set stores a value in the cache
func (c *MemoryCache) Set(key string, value []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiration := now.Add(c.ttl)
	
	// Handle zero or negative TTL
	if c.ttl <= 0 {
		expiration = now // Immediately expired
	}

	entry := &CacheEntry{
		Value:      value,
		Expiration: expiration,
		AccessTime: now,
	}

	c.items[key] = entry

	// Evict if over capacity
	if len(c.items) > c.capacity {
		c.evictLRU()
	}
}

// evictLRU removes the least recently used item
func (c *MemoryCache) evictLRU() {
	if len(c.items) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range c.items {
		if first || entry.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessTime
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.stats.Evictions++
	}
}

// Len returns the number of items in the cache
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}