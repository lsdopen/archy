package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const maxRequestSize = 1024 * 1024 // 1MB

// AdmissionHandler handles Kubernetes admission webhook requests
type AdmissionHandler struct {
	mutator *Mutator
}

// NewAdmissionHandler creates a new admission handler
func NewAdmissionHandler(kubeClient kubernetes.Interface) *AdmissionHandler {
	return &AdmissionHandler{
		mutator: NewMutator(kubeClient),
	}
}

func (h *AdmissionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "Request entity too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal admission review: %v", err), http.StatusBadRequest)
		return
	}

	if &admissionReview == nil {
		http.Error(w, "admission review is nil", http.StatusBadRequest)
		return
	}

	if admissionReview.Request == nil {
		http.Error(w, "admission request is nil", http.StatusBadRequest)
		return
	}

	if admissionReview.Request.UID == "" {
		http.Error(w, "admission request UID is empty", http.StatusBadRequest)
		return
	}

	if admissionReview.APIVersion == "" {
		http.Error(w, "API version is required", http.StatusBadRequest)
		return
	}

	if admissionReview.APIVersion != "admission.k8s.io/v1" {
		http.Error(w, fmt.Sprintf("unsupported API version: %s", admissionReview.APIVersion), http.StatusBadRequest)
		return
	}

	// Process the admission request
	response := h.processAdmissionRequest(admissionReview.Request)

	admissionResponse := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: response,
	}

	responseBytes, err := json.Marshal(admissionResponse)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal admission response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func (h *AdmissionHandler) processAdmissionRequest(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	// Always fail open - allow requests even if processing fails
	response := &admissionv1.AdmissionResponse{
		UID:     req.UID,
		Allowed: true,
	}

	// Try to process the request, but don't fail if it errors
	patches, err := h.mutator.Mutate(req)
	if err != nil {
		// Log error but allow request to proceed
		return response
	}

	if len(patches) > 0 {
		patchBytes, err := json.Marshal(patches)
		if err != nil {
			// Log error but allow request to proceed
			return response
		}

		patchType := admissionv1.PatchTypeJSONPatch
		response.Patch = patchBytes
		response.PatchType = &patchType
	}

	return response
}