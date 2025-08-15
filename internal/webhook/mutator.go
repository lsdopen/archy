package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"github.com/lsdopen/archy/internal/cache"
	"github.com/lsdopen/archy/internal/credentials"
	"github.com/lsdopen/archy/internal/metrics"
	"github.com/lsdopen/archy/internal/registry"
	"github.com/lsdopen/archy/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// JSONPatch represents a JSON Patch operation
type JSONPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Mutator handles pod mutations
type Mutator struct {
	defaultArch        string
	registryClient     types.RegistryClient
	cache              *cache.MemoryCache
	metrics            *metrics.Metrics
	credentialResolver *credentials.Resolver
}

// NewMutator creates a new mutator
func NewMutator(kubeClient kubernetes.Interface) *Mutator {
	// Create Docker Hub client as default
	client := registry.NewDockerHubClient()
	cache := cache.NewMemoryCache(1000, 5*time.Minute)
	metrics := metrics.NewMetrics()
	credResolver := credentials.NewResolver(kubeClient)
	
	return &Mutator{
		defaultArch:        "amd64",
		registryClient:     client,
		cache:              cache,
		metrics:            metrics,
		credentialResolver: credResolver,
	}
}

// Mutate processes an admission request and returns JSON patches
func (m *Mutator) Mutate(req *admissionv1.AdmissionRequest) ([]JSONPatch, error) {
	start := time.Now()
	var success bool
	var selectedArch string
	defer func() {
		if selectedArch != "" {
			m.metrics.RecordMutation("pod", selectedArch, success, time.Since(start))
		}
	}()

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

	// Detect architecture from first image
	arch := m.detectArchitecture(&pod, images[0])
	selectedArch = arch

	// Create patches to add node selector
	patches := m.createNodeSelectorPatches(&pod, arch)
	success = len(patches) > 0

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

// detectArchitecture detects the architecture for an image using cache and registry
func (m *Mutator) detectArchitecture(pod *corev1.Pod, image string) string {
	// Check cache first
	if archs, found := m.cache.Get(image); found {
		m.metrics.RecordCacheHit(image)
		if len(archs) > 0 {
			return archs[0] // Return first supported architecture
		}
	}

	m.metrics.RecordCacheMiss(image)

	// Resolve credentials for this image
	cred, _ := m.credentialResolver.ResolveCredentials(pod, image)

	// Create registry client with credentials if available
	client := m.registryClient
	if cred != nil {
		// Use authenticated client (implementation would create client with credentials)
		client = registry.NewDockerHubClientWithCredentials(cred.Username, cred.Password)
	}

	// Query registry
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	archs, err := client.GetSupportedArchitectures(ctx, image)
	if err != nil {
		// Fail open with default architecture
		return m.defaultArch
	}

	if len(archs) == 0 {
		return m.defaultArch
	}

	// Cache the result
	m.cache.Set(image, archs)

	// Return first supported architecture
	return archs[0]
}