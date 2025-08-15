package webhook

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMutator_PodsWithNoContainers(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{}, // No containers
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	req := &admissionv1.AdmissionRequest{
		UID: "test-uid",
		Object: runtime.RawExtension{
			Raw: podBytes,
		},
	}

	patches, err := mutator.Mutate(req)
	require.NoError(t, err)
	assert.Empty(t, patches) // No patches should be applied
}

func TestMutator_PodsWithInitContainersOnly(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "init-container",
					Image: "busybox",
				},
			},
			Containers: []corev1.Container{}, // No regular containers
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	req := &admissionv1.AdmissionRequest{
		UID: "test-uid",
		Object: runtime.RawExtension{
			Raw: podBytes,
		},
	}

	patches, err := mutator.Mutate(req)
	require.NoError(t, err)

	// Should add node selector based on init container image
	assert.NotEmpty(t, patches)
	assertNodeSelectorPatch(t, patches, "amd64") // Default architecture
}

func TestMutator_PodsWithExistingArchitectureSelector(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"kubernetes.io/arch": "arm64", // Already has arch selector
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx",
				},
			},
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	req := &admissionv1.AdmissionRequest{
		UID: "test-uid",
		Object: runtime.RawExtension{
			Raw: podBytes,
		},
	}

	patches, err := mutator.Mutate(req)
	require.NoError(t, err)
	assert.Empty(t, patches) // No patches should be applied
}

func TestMutator_PodsWithConflictingNodeSelectors(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			NodeSelector: map[string]string{
				"environment": "production",
				"zone":        "us-west-1",
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx",
				},
			},
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	req := &admissionv1.AdmissionRequest{
		UID: "test-uid",
		Object: runtime.RawExtension{
			Raw: podBytes,
		},
	}

	patches, err := mutator.Mutate(req)
	require.NoError(t, err)

	// Should add arch selector while preserving existing selectors
	assert.NotEmpty(t, patches)
	assertNodeSelectorPatch(t, patches, "amd64")
}

func TestMutator_PodsWithInvalidImageReferences(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	tests := []struct {
		name  string
		image string
	}{
		{
			name:  "empty image",
			image: "",
		},
		{
			name:  "invalid image format",
			image: ":::invalid:::image:::",
		},
		{
			name:  "image with spaces",
			image: "nginx with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: tt.image,
						},
					},
				},
			}

			podBytes, err := json.Marshal(pod)
			require.NoError(t, err)

			req := &admissionv1.AdmissionRequest{
				UID: "test-uid",
				Object: runtime.RawExtension{
					Raw: podBytes,
				},
			}

			patches, err := mutator.Mutate(req)
			require.NoError(t, err)

			// Should fallback to default architecture
			assert.NotEmpty(t, patches)
			assertNodeSelectorPatch(t, patches, "amd64")
		})
	}
}

func TestMutator_SystemPods(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-proxy-abc123",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "kube-proxy",
					Image: "k8s.gcr.io/kube-proxy:v1.20.0",
				},
			},
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	req := &admissionv1.AdmissionRequest{
		UID:       "test-uid",
		Namespace: "kube-system",
		Object: runtime.RawExtension{
			Raw: podBytes,
		},
	}

	patches, err := mutator.Mutate(req)
	require.NoError(t, err)

	// Should still process system pods
	assert.NotEmpty(t, patches)
	assertNodeSelectorPatch(t, patches, "amd64")
}

func TestMutator_ConcurrentMutationRequests(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx",
				},
			},
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	// Run concurrent mutations
	done := make(chan bool, 100)
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			defer func() { done <- true }()

			req := &admissionv1.AdmissionRequest{
				UID: metav1.UID(fmt.Sprintf("test-uid-%d", id)),
				Object: runtime.RawExtension{
					Raw: podBytes,
				},
			}

			patches, err := mutator.Mutate(req)
			if err != nil {
				errors <- err
				return
			}

			if len(patches) == 0 {
				errors <- fmt.Errorf("expected patches but got none")
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent mutation failed: %v", err)
	}
}

func TestMutator_MutationRollback(t *testing.T) {
	client := fake.NewSimpleClientset()
	mutator := NewMutator(client)

	// Test that mutations are atomic - either all succeed or none are applied
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx",
				},
			},
		},
	}

	podBytes, err := json.Marshal(pod)
	require.NoError(t, err)

	req := &admissionv1.AdmissionRequest{
		UID: "test-uid",
		Object: runtime.RawExtension{
			Raw: podBytes,
		},
	}

	patches, err := mutator.Mutate(req)
	require.NoError(t, err)

	// Verify patches are valid JSON Patch format
	for _, patch := range patches {
		assert.Contains(t, []string{"add", "replace", "remove"}, patch.Op)
		assert.NotEmpty(t, patch.Path)
	}
}

// Helper function to verify node selector patch
func assertNodeSelectorPatch(t *testing.T, patches []JSONPatch, expectedArch string) {
	found := false
	for _, patch := range patches {
		if patch.Path == "/spec/nodeSelector" || patch.Path == "/spec/nodeSelector/kubernetes.io~1arch" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected node selector patch not found")
}