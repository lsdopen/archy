package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/lsdopen/archy/pkg/inspector"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	admissionv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
}

type Handler struct {
	inspector inspector.Inspector
	k8sClient kubernetes.Interface
}

func NewHandler(inspector inspector.Inspector, k8sClient kubernetes.Interface) *Handler {
	return &Handler{
		inspector: inspector,
		k8sClient: k8sClient,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Decode AdmissionReview
	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Request could not be decoded: %v", err), http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case admissionv1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*admissionv1.AdmissionReview)
		if !ok {
			http.Error(w, "Expected v1.AdmissionReview", http.StatusBadRequest)
			return
		}
		responseAdmissionReview := &admissionv1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = h.mutate(r.Context(), requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		http.Error(w, fmt.Sprintf("Unsupported group version kind: %v", gvk), http.StatusBadRequest)
		return
	}

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func (h *Handler) mutate(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// 1. Check if nodeSelector is already present
	if len(pod.Spec.NodeSelector) > 0 {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Message: "Pod already has nodeSelector, skipping",
			},
		}
	}

	// 2. Collect all images
	images := []string{}
	for _, c := range pod.Spec.Containers {
		images = append(images, c.Image)
	}
	for _, c := range pod.Spec.InitContainers {
		images = append(images, c.Image)
	}

	// 3. Inspect images and find intersection
	// Pass pod authentication details for private registry access
	commonPlatforms, err := h.getCommonPlatforms(ctx, images, req.Namespace, pod.Spec.ImagePullSecrets, pod.Spec.ServiceAccountName)
	if err != nil {
		// Fail open or closed? Usually fail closed if we can't determine arch to be safe,
		// but for now let's return an error status.
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to inspect images: %v", err),
			},
		}
	}

	if len(commonPlatforms) == 0 {
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: "Images have no common supported platform (OS/Arch)",
			},
		}
	}

	// 4. If multiple platforms are supported (e.g. Multi-arch), allow scheduler to decide
	if len(commonPlatforms) > 1 {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Message: "Multiple common platforms found, leaving scheduling to Kubernetes",
			},
		}
	}

	// 5. Exactly one common platform -> Patch
	target := commonPlatforms[0]

	nodeSelector := map[string]string{
		"kubernetes.io/arch": target.Architecture,
	}

	// We need to construct the patch carefully.
	// Since we checked pod.Spec.NodeSelector is empty, we can just "add" the map.
	// However, "add" to a non-existent map field in JSONPatch can be tricky if the parent struct exists but field is null.
	// In k8s, if nodeSelector is nil, we need to initialize it.
	// A safer way is to use a map for the value in the patch.

	patchBytes, err := json.Marshal([]map[string]interface{}{
		{
			"op":    "add",
			"path":  "/spec/nodeSelector",
			"value": nodeSelector,
		},
	})

	if err != nil {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to marshal patch: %v", err),
			},
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (h *Handler) getCommonPlatforms(ctx context.Context, images []string, namespace string, imagePullSecrets []corev1.LocalObjectReference, serviceAccountName string) ([]inspector.Platform, error) {
	if len(images) == 0 {
		return nil, nil
	}

	// Build authentication keychain from pod's imagePullSecrets
	keychain, err := inspector.BuildKeychain(ctx, h.k8sClient, namespace, imagePullSecrets, serviceAccountName)
	if err != nil {
		return nil, fmt.Errorf("building keychain: %w", err)
	}

	// Create remote options with authentication
	opts := []remote.Option{remote.WithAuthFromKeychain(keychain)}

	// Get platforms for first image
	firstImagePlatforms, err := h.inspector.GetSupportedPlatforms(ctx, images[0], opts...)
	if err != nil {
		return nil, err
	}

	common := firstImagePlatforms

	// Intersect with rest
	for _, img := range images[1:] {
		platforms, err := h.inspector.GetSupportedPlatforms(ctx, img, opts...)
		if err != nil {
			return nil, err
		}
		common = intersect(common, platforms)
	}

	return common, nil
}

func intersect(a, b []inspector.Platform) []inspector.Platform {
	var result []inspector.Platform
	for _, pa := range a {
		for _, pb := range b {
			if isCompatible(pa, pb) {
				result = append(result, pa)
			}
		}
	}
	return result
}

func isCompatible(a, b inspector.Platform) bool {
	return a.Architecture == b.Architecture
}
