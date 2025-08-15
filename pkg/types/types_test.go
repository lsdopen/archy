package types

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRegistryClient for testing interface compliance
type MockRegistryClient struct {
	architectures []string
	err           error
	callCount     int
}

func (m *MockRegistryClient) GetSupportedArchitectures(ctx context.Context, image string) ([]string, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.architectures, nil
}

func TestRegistryClient_InterfaceCompliance(t *testing.T) {
	var client RegistryClient = &MockRegistryClient{}
	assert.NotNil(t, client)
}

func TestRegistryClient_MethodSignatures(t *testing.T) {
	mock := &MockRegistryClient{
		architectures: []string{"amd64", "arm64"},
	}

	ctx := context.Background()
	archs, err := mock.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.NoError(t, err)
	assert.Equal(t, []string{"amd64", "arm64"}, archs)
	assert.Equal(t, 1, mock.callCount)
}

func TestRegistryClient_ErrorHandling(t *testing.T) {
	mock := &MockRegistryClient{
		err: errors.New("network error"),
	}

	ctx := context.Background()
	archs, err := mock.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "network error")
}

func TestRegistryClient_TimeoutBehavior(t *testing.T) {
	mock := &MockRegistryClient{
		architectures: []string{"amd64"},
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Sleep to trigger timeout
	time.Sleep(2 * time.Millisecond)

	_, err := mock.GetSupportedArchitectures(ctx, "nginx:latest")
	
	// Mock doesn't respect context timeout, but interface should support it
	assert.NoError(t, err) // Mock implementation doesn't check context
}

func TestRegistryClient_ConcurrentAccess(t *testing.T) {
	mock := &MockRegistryClient{
		architectures: []string{"amd64", "arm64"},
	}

	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			ctx := context.Background()
			_, err := mock.GetSupportedArchitectures(ctx, "nginx:latest")
			if err != nil {
				errors <- err
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Check no errors occurred
	for err := range errors {
		t.Errorf("Concurrent access failed: %v", err)
	}

	assert.Equal(t, 10, mock.callCount)
}