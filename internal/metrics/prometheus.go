package metrics

import (
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	registry         *prometheus.Registry
	mutationsTotal   *prometheus.CounterVec
	mutationDuration *prometheus.HistogramVec
	cacheHits        *prometheus.CounterVec
	cacheMisses      *prometheus.CounterVec
}

var labelSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()

	mutationsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "archy_mutations_total",
			Help: "Total number of pod mutations performed",
		},
		[]string{"image", "architecture", "success"},
	)

	mutationDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "archy_mutation_duration_seconds",
			Help:    "Duration of pod mutations in seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 1, 5, 10},
		},
		[]string{"image", "architecture"},
	)

	cacheHits := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "archy_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"image"},
	)

	cacheMisses := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "archy_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"image"},
	)

	registry.MustRegister(mutationsTotal)
	registry.MustRegister(mutationDuration)
	registry.MustRegister(cacheHits)
	registry.MustRegister(cacheMisses)

	return &Metrics{
		registry:         registry,
		mutationsTotal:   mutationsTotal,
		mutationDuration: mutationDuration,
		cacheHits:        cacheHits,
		cacheMisses:      cacheMisses,
	}
}

// RecordMutation records a pod mutation
func (m *Metrics) RecordMutation(image, architecture string, success bool, duration time.Duration) {
	sanitizedImage := sanitizeLabel(image)
	successStr := "false"
	if success {
		successStr = "true"
	}

	m.mutationsTotal.WithLabelValues(sanitizedImage, architecture, successStr).Inc()
	m.mutationDuration.WithLabelValues(sanitizedImage, architecture).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(image string) {
	sanitizedImage := sanitizeLabel(image)
	m.cacheHits.WithLabelValues(sanitizedImage).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(image string) {
	sanitizedImage := sanitizeLabel(image)
	m.cacheMisses.WithLabelValues(sanitizedImage).Inc()
}

// Registry returns the Prometheus registry
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// sanitizeLabel sanitizes label values for Prometheus
func sanitizeLabel(label string) string {
	// Replace invalid characters with underscores
	sanitized := labelSanitizer.ReplaceAllString(label, "_")
	
	// Limit length to prevent cardinality explosion
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	// Remove consecutive underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}
	
	// Trim underscores from start and end
	sanitized = strings.Trim(sanitized, "_")
	
	if sanitized == "" {
		sanitized = "unknown"
	}
	
	return sanitized
}