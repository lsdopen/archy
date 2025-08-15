package webhook

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

// JSONPatch represents a JSON Patch operation
type JSONPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Mutator handles pod mutations
type Mutator struct {
	defaultArch string
}

// NewMutator creates a new mutator
func NewMutator() *Mutator {
	return &Mutator{
		defaultArch: "amd64",
	}
}

// Mutate processes an admission request and returns JSON patches
func (m *Mutator) Mutate(req *admissionv1.AdmissionRequest) ([]JSONPatch, error) {
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		// Return empty patches on unmarshal error (fail open)
		return []JSONPatch{}, nil
	}

	// Check if pod already has architecture selector
	if pod.Spec.NodeSelector != nil {
		if _, exists := pod.Spec.NodeSelector["kubernetes.io/arch"]; exists {
			return []JSONPatch{}, nil // No mutation needed
		}
	}

	// Get all container images
	images := m.extractImages(&pod)
	if len(images) == 0 {
		return []JSONPatch{}, nil // No containers to process
	}

	// For now, use default architecture (Phase 3 will add registry detection)
	arch := m.defaultArch

	// Create patches to add node selector
	patches := m.createNodeSelectorPatches(&pod, arch)

	return patches, nil
}

func (m *Mutator) extractImages(pod *corev1.Pod) []string {
	var images []string

	// Extract from regular containers
	for _, container := range pod.Spec.Containers {
		if container.Image != "" {
			images = append(images, container.Image)
		}
	}

	// Extract from init containers
	for _, container := range pod.Spec.InitContainers {
		if container.Image != "" {
			images = append(images, container.Image)
		}
	}

	return images
}

func (m *Mutator) createNodeSelectorPatches(pod *corev1.Pod, arch string) []JSONPatch {
	var patches []JSONPatch

	if pod.Spec.NodeSelector == nil {
		// Create new nodeSelector
		nodeSelector := map[string]string{
			"kubernetes.io/arch": arch,
		}
		patches = append(patches, JSONPatch{
			Op:    "add",
			Path:  "/spec/nodeSelector",
			Value: nodeSelector,
		})
	} else {
		// Add to existing nodeSelector
		patches = append(patches, JSONPatch{
			Op:    "add",
			Path:  "/spec/nodeSelector/kubernetes.io~1arch",
			Value: arch,
		})
	}

	return patches
}