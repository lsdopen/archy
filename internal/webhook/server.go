package webhook

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// NewServer creates a new HTTPS server with TLS configuration
func NewServer(addr, certPath, keyPath string) (*http.Server, error) {
	// Load TLS certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)
	mux.HandleFunc("/slow", slowHandler)

	server := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(recoveryMiddleware(mux)),
		TLSConfig:    tlsConfig,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(100 * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Slow"))
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Log request (implementation would go here)
		_ = time.Since(start)
	})
}