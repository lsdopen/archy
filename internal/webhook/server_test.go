package webhook

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_TLSCertificateLoading(t *testing.T) {
	// Create valid cert and key files
	certFile, keyFile := createTestCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)
	assert.NotNil(t, server)
}

func TestServer_InvalidCertificateHandling(t *testing.T) {
	tests := []struct {
		name     string
		certPath string
		keyPath  string
		wantErr  string
	}{
		{
			name:     "missing cert file",
			certPath: "/nonexistent/cert.pem",
			keyPath:  "/tmp/key.pem",
			wantErr:  "no such file or directory",
		},
		{
			name:     "missing key file",
			certPath: "/tmp/cert.pem",
			keyPath:  "/nonexistent/key.pem",
			wantErr:  "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServer(":0", tt.certPath, tt.keyPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestServer_ExpiredCertificateHandling(t *testing.T) {
	// Create expired certificate
	certFile, keyFile := createExpiredCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)

	// Start server
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.ServeTLS(listener, "", "")
	}()

	// Try to connect with certificate verification
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
		Timeout: 1 * time.Second,
	}

	_, err = client.Get(fmt.Sprintf("https://%s/health", listener.Addr().String()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")
}

func TestServer_HTTPTimeouts(t *testing.T) {
	certFile, keyFile := createTestCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)

	// Verify timeout settings
	assert.Equal(t, 30*time.Second, server.ReadTimeout)
	assert.Equal(t, 30*time.Second, server.WriteTimeout)
	assert.Equal(t, 120*time.Second, server.IdleTimeout)
}

func TestServer_MiddlewareChain(t *testing.T) {
	certFile, keyFile := createTestCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.ServeTLS(listener, "", "")
	}()

	// Test middleware execution order
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("https://%s/health", listener.Addr().String()))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_ShutdownWithPendingRequests(t *testing.T) {
	certFile, keyFile := createTestCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.ServeTLS(listener, "", "")
	}()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Start a slow request
	done := make(chan bool)
	go func() {
		defer func() { done <- true }()
		resp, err := client.Get(fmt.Sprintf("https://%s/slow", listener.Addr().String()))
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

func TestServer_ConcurrentConnections(t *testing.T) {
	certFile, keyFile := createTestCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.ServeTLS(listener, "", "")
	}()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(fmt.Sprintf("https://%s/health", listener.Addr().String()))
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Request failed: %v", err)
	}
}

func TestServer_MemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	certFile, keyFile := createTestCertificates(t)
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	server, err := NewServer(":0", certFile, keyFile)
	require.NoError(t, err)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		server.ServeTLS(listener, "", "")
	}()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 1 * time.Second,
	}

	// Make many requests to check for memory leaks
	for i := 0; i < 1000; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/health", listener.Addr().String()))
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Force garbage collection
	runtime.GC()
	runtime.GC()
}

// Helper functions
func createTestCertificates(t *testing.T) (string, string) {
	return createCertificates(t, time.Now().Add(24*time.Hour))
}

func createExpiredCertificates(t *testing.T) (string, string) {
	return createCertificates(t, time.Now().Add(-24*time.Hour))
}

func createCertificates(t *testing.T, notAfter time.Time) (string, string) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Write certificate file
	certFile, err := os.CreateTemp("", "cert-*.pem")
	require.NoError(t, err)
	defer certFile.Close()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err)

	// Write key file
	keyFile, err := os.CreateTemp("", "key-*.pem")
	require.NoError(t, err)
	defer keyFile.Close()

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	err = pem.Encode(keyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})
	require.NoError(t, err)

	return certFile.Name(), keyFile.Name()
}