package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/lsdopen/archy/pkg/inspector"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MockInspector implements inspector.Inspector for testing
type MockInspector struct {
	platforms map[string][]inspector.Platform
}

func (m *MockInspector) GetSupportedPlatforms(ctx context.Context, image string, opts ...remote.Option) ([]inspector.Platform, error) {
	return m.platforms[image], nil
}

func TestMutate(t *testing.T) {
	tests := []struct {
		name           string
		pod            corev1.Pod
		mockImages     map[string][]inspector.Platform
		expectAllowed  bool
		expectPatch    bool
		expectedPatch  string // Partial match or key check
		expectedStatus string
	}{
		{
			name: "Single image, single arch (ARM64)",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Image: "app:arm64"}},
				},
			},
			mockImages: map[string][]inspector.Platform{
				"app:arm64": {{Architecture: "arm64"}},
			},
			expectAllowed: true,
			expectPatch:   true,
			expectedPatch: `"kubernetes.io/arch":"arm64"`,
		},
		{
			name: "Single image, multi-arch (AMD64/ARM64)",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Image: "app:multi"}},
				},
			},
			mockImages: map[string][]inspector.Platform{
				"app:multi": {
					{Architecture: "amd64"},
					{Architecture: "arm64"},
				},
			},
			expectAllowed: true,
			expectPatch:   false, // Should not patch, let scheduler decide
		},
		{
			name: "Mixed images, compatible (ARM64)",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Image: "app:arm64"},
						{Image: "sidecar:multi"},
					},
				},
			},
			mockImages: map[string][]inspector.Platform{
				"app:arm64": {{Architecture: "arm64"}},
				"sidecar:multi": {
					{Architecture: "amd64"},
					{Architecture: "arm64"},
				},
			},
			expectAllowed: true,
			expectPatch:   true,
			expectedPatch: `"kubernetes.io/arch":"arm64"`,
		},
		{
			name: "Mixed images, incompatible",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Image: "app:arm64"},
						{Image: "app:amd64"},
					},
				},
			},
			mockImages: map[string][]inspector.Platform{
				"app:arm64": {{Architecture: "arm64"}},
				"app:amd64": {{Architecture: "amd64"}},
			},
			expectAllowed: false,
			expectPatch:   false,
		},
		{
			name: "Existing NodeSelector",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					NodeSelector: map[string]string{"foo": "bar"},
					Containers:   []corev1.Container{{Image: "app:arm64"}},
				},
			},
			mockImages: map[string][]inspector.Platform{
				"app:arm64": {{Architecture: "arm64"}},
			},
			expectAllowed: true,
			expectPatch:   false, // Should be ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspector := &MockInspector{platforms: tt.mockImages}
			// Pass nil for k8sClient since we're using a mock inspector
			// that doesn't need to authenticate with real registries
			handler := NewHandler(inspector, nil)

			podBytes, _ := json.Marshal(tt.pod)
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "123",
					Object: runtime.RawExtension{
						Raw: podBytes,
					},
				},
			}

			resp := handler.mutate(context.Background(), ar)

			if resp.Allowed != tt.expectAllowed {
				t.Errorf("Expected Allowed=%v, got %v. Message: %v", tt.expectAllowed, resp.Allowed, resp.Result.Message)
			}

			if tt.expectPatch && resp.Patch == nil {
				t.Error("Expected patch, got nil")
			} else if !tt.expectPatch && resp.Patch != nil {
				t.Errorf("Expected no patch, got %s", string(resp.Patch))
			}

			if tt.expectPatch && tt.expectedPatch != "" {
				patchStr := string(resp.Patch)
				// Simple substring check for now
				if !contains(patchStr, tt.expectedPatch) {
					t.Errorf("Expected patch to contain %s, got %s", tt.expectedPatch, patchStr)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	// Simple helper, strings.Contains is fine but just to be explicit
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
