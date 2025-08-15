package metrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_CollectionUnderHighLoad(t *testing.T) {
	metrics := NewMetrics()
	
	// Simulate high load
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metrics.RecordMutation("nginx:latest", "amd64", true, 10*time.Millisecond)
		}()
	}
	
	wg.Wait()
	
	// Verify metrics are collected
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "archy_mutations_total")
}

func TestMetrics_AccuracyDuringFailures(t *testing.T) {
	metrics := NewMetrics()
	
	// Record successful mutations
	for i := 0; i < 10; i++ {
		metrics.RecordMutation("nginx:latest", "amd64", true, 5*time.Millisecond)
	}
	
	// Record failed mutations
	for i := 0; i < 5; i++ {
		metrics.RecordMutation("nginx:latest", "amd64", false, 100*time.Millisecond)
	}
	
	// Record cache operations
	for i := 0; i < 20; i++ {
		metrics.RecordCacheHit("nginx:latest")
	}
	for i := 0; i < 8; i++ {
		metrics.RecordCacheMiss("nginx:latest")
	}
	
	// Verify metrics accuracy
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)
	
	body := w.Body.String()
	
	// Check mutation counts
	assert.Contains(t, body, `archy_mutations_total{architecture="amd64",image="nginx:latest",success="true"} 10`)
	assert.Contains(t, body, `archy_mutations_total{architecture="amd64",image="nginx:latest",success="false"} 5`)
	
	// Check cache metrics
	assert.Contains(t, body, `archy_cache_hits_total{image="nginx:latest"} 20`)
	assert.Contains(t, body, `archy_cache_misses_total{image="nginx:latest"} 8`)
}

func TestMetrics_CardinalityExplosionPrevention(t *testing.T) {
	metrics := NewMetrics()
	
	// Try to create high cardinality by using many different image names
	for i := 0; i < 10000; i++ {
		image := strings.Repeat("a", i%100) + ":latest" // Varying length images
		metrics.RecordMutation(image, "amd64", true, 1*time.Millisecond)
	}
	
	// Metrics should still be collectable without memory explosion
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Response should not be excessively large (indicating cardinality explosion)
	assert.True(t, len(w.Body.String()) < 1024*1024) // Less than 1MB
}

func TestMetrics_ScrapingTimeout(t *testing.T) {
	metrics := NewMetrics()
	
	// Add many metrics to potentially slow down scraping
	for i := 0; i < 1000; i++ {
		metrics.RecordMutation("nginx:latest", "amd64", true, 1*time.Millisecond)
	}
	
	// Test scraping with timeout
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	start := time.Now()
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{
		Timeout: 1 * time.Second,
	})
	handler.ServeHTTP(w, req)
	duration := time.Since(start)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, duration < 2*time.Second) // Should complete quickly
}

func TestMetrics_MemoryUsageGrowth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}
	
	metrics := NewMetrics()
	
	// Record many metrics over time
	for i := 0; i < 10000; i++ {
		metrics.RecordMutation("nginx:latest", "amd64", true, 1*time.Millisecond)
		metrics.RecordCacheHit("nginx:latest")
		
		if i%1000 == 0 {
			// Force GC periodically
			runtime.GC()
		}
	}
	
	// Memory usage should be reasonable
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "archy_mutations_total")
}

func TestMetrics_ConcurrentScraping(t *testing.T) {
	metrics := NewMetrics()
	
	// Add some metrics
	metrics.RecordMutation("nginx:latest", "amd64", true, 5*time.Millisecond)
	
	// Concurrent scraping
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			
			handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
			handler.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("unexpected status code: %d", w.Code)
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	for err := range errors {
		t.Errorf("Concurrent scraping failed: %v", err)
	}
}

func TestMetrics_HistogramBuckets(t *testing.T) {
	metrics := NewMetrics()
	
	// Record mutations with different durations
	durations := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}
	
	for _, duration := range durations {
		metrics.RecordMutation("nginx:latest", "amd64", true, duration)
	}
	
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)
	
	body := w.Body.String()
	
	// Check histogram buckets are present
	assert.Contains(t, body, "archy_mutation_duration_seconds_bucket")
	assert.Contains(t, body, `le="0.001"`)
	assert.Contains(t, body, `le="0.01"`)
	assert.Contains(t, body, `le="0.1"`)
	assert.Contains(t, body, `le="1"`)
	assert.Contains(t, body, `le="+Inf"`)
}

func TestMetrics_LabelSanitization(t *testing.T) {
	metrics := NewMetrics()
	
	// Test with potentially problematic labels
	problematicImages := []string{
		"nginx:latest",
		"registry.example.com/nginx:v1.0",
		"nginx@sha256:abc123def456",
		"nginx:tag-with-dashes",
		"nginx:tag_with_underscores",
	}
	
	for _, image := range problematicImages {
		metrics.RecordMutation(image, "amd64", true, 1*time.Millisecond)
	}
	
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	handler := promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Should not contain invalid Prometheus label characters
	body := w.Body.String()
	assert.NotContains(t, body, "@") // Should be sanitized
	assert.NotContains(t, body, "/") // Should be sanitized
}