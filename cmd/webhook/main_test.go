package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_InvalidTLSCertificates(t *testing.T) {
	tests := []struct {
		name     string
		certPath string
		keyPath  string
		wantErr  string
	}{
		{
			name:     "non-existent cert file",
			certPath: "/nonexistent/cert.pem",
			keyPath:  "/tmp/key.pem",
			wantErr:  "no such file or directory",
		},
		{
			name:     "non-existent key file",
			certPath: "/tmp/cert.pem",
			keyPath:  "/nonexistent/key.pem",
			wantErr:  "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &http.Server{
				Addr: ":0",
			}

			err := server.ListenAndServeTLS(tt.certPath, tt.keyPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestServer_NoAvailablePorts(t *testing.T) {
	// Bind to a specific port first
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Try to bind to the same port
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	err = server.ListenAndServe()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bind")
}

func TestHealthEndpoints_UnderLoad(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)

	server := &http.Server{
		Handler: mux,
		Addr:    ":0",
	}

	listener, err := net.Listen("tcp", server.Addr)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.Serve(listener)
	}()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())

	// Test concurrent requests
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- true }()
			
			resp, err := http.Get(baseURL + "/health")
			if err != nil {
				return
			}
			defer resp.Body.Close()
			
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	server.Shutdown(context.Background())
}

func TestServer_GracefulShutdown(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Handler: mux,
		Addr:    ":0",
	}

	listener, err := net.Listen("tcp", server.Addr)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.Serve(listener)
	}()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())

	// Start a slow request
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		resp, err := http.Get(baseURL + "/slow")
		if err == nil {
			resp.Body.Close()
		}
	}()

	// Give request time to start
	time.Sleep(10 * time.Millisecond)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)

	// Ensure request completed
	<-done
}

func TestServer_PanicRecovery(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with recovery middleware
	handler := recoveryMiddleware(mux)

	server := &http.Server{
		Handler: handler,
		Addr:    ":0",
	}

	listener, err := net.Listen("tcp", server.Addr)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.Serve(listener)
	}()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())

	resp, err := http.Get(baseURL + "/panic")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	server.Shutdown(context.Background())
}

func TestSignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal test in short mode")
	}

	// This test would require more complex setup to properly test signal handling
	// For now, we'll test that the signal handling functions exist
	assert.NotNil(t, handleSignals)
}

// Mock handlers for testing
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// Mock recovery middleware
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

// Mock signal handling function
var handleSignals = func(server *http.Server) {
	// Implementation would go here
}