package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestHandler_MalformedAdmissionReview(t *testing.T) {
	handler := NewAdmissionHandler()

	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "invalid JSON",
			body:    `{invalid json}`,
			wantErr: "invalid character",
		},
		{
			name:    "empty body",
			body:    "",
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "null body",
			body:    "null",
			wantErr: "admission review is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mutate", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantErr)
		})
	}
}

func TestHandler_MissingRequiredFields(t *testing.T) {
	handler := NewAdmissionHandler()

	tests := []struct {
		name    string
		review  *admissionv1.AdmissionReview
		wantErr string
	}{
		{
			name: "missing request",
			review: &admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "admission.k8s.io/v1",
					Kind:       "AdmissionReview",
				},
			},
			wantErr: "admission request is nil",
		},
		{
			name: "missing UID",
			review: &admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "admission.k8s.io/v1",
					Kind:       "AdmissionReview",
				},
				Request: &admissionv1.AdmissionRequest{},
			},
			wantErr: "admission request UID is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.review)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantErr)
		})
	}
}

func TestHandler_InvalidKubernetesAPIVersions(t *testing.T) {
	handler := NewAdmissionHandler()

	tests := []struct {
		name       string
		apiVersion string
		wantErr    string
	}{
		{
			name:       "unsupported API version",
			apiVersion: "admission.k8s.io/v2",
			wantErr:    "unsupported API version",
		},
		{
			name:       "empty API version",
			apiVersion: "",
			wantErr:    "API version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review := &admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: tt.apiVersion,
					Kind:       "AdmissionReview",
				},
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid",
				},
			}

			body, err := json.Marshal(review)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantErr)
		})
	}
}

func TestHandler_OversizedPayload(t *testing.T) {
	handler := NewAdmissionHandler()

	// Create a large payload (> 1MB)
	largeData := make([]byte, 2*1024*1024) // 2MB
	for i := range largeData {
		largeData[i] = 'a'
	}

	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(largeData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestHandler_ConcurrentRequests(t *testing.T) {
	handler := NewAdmissionHandler()

	review := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test"},"spec":{"containers":[{"name":"test","image":"nginx"}]}}`),
			},
		},
	}

	body, err := json.Marshal(review)
	require.NoError(t, err)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create unique UID for each request
			testReview := *review
			testReview.Request = &admissionv1.AdmissionRequest{
				UID: metav1.UID(fmt.Sprintf("test-uid-%d", id)),
				Object: runtime.RawExtension{
					Raw: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test"},"spec":{"containers":[{"name":"test","image":"nginx"}]}}`),
				},
			}

			testBody, err := json.Marshal(&testReview)
			if err != nil {
				errors <- err
				return
			}

			req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(testBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("request %d failed with status %d", id, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

func TestHandler_RequestTimeout(t *testing.T) {
	handler := NewAdmissionHandler()

	review := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test"},"spec":{"containers":[{"name":"test","image":"nginx"}]}}`),
			},
		},
	}

	body, err := json.Marshal(review)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set a very short timeout
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Nanosecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Request should complete normally since timeout is handled at server level
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_AdmissionResponseSerialization(t *testing.T) {
	handler := NewAdmissionHandler()

	review := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test"},"spec":{"containers":[{"name":"test","image":"nginx"}]}}`),
			},
		},
	}

	body, err := json.Marshal(review)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response.Response)
	assert.Equal(t, "test-uid", string(response.Response.UID))
	assert.True(t, response.Response.Allowed)
}

func TestHandler_WebhookFailurePolicy(t *testing.T) {
	handler := NewAdmissionHandler()

	// Test that webhook fails open (allows requests even on internal errors)
	review := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "test-uid",
			Object: runtime.RawExtension{
				Raw: []byte(`invalid pod spec`), // This should cause internal error
			},
		},
	}

	body, err := json.Marshal(review)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response admissionv1.AdmissionReview
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should fail open (allow the request)
	assert.True(t, response.Response.Allowed)
}