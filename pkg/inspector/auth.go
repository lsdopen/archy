package inspector

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// BuildKeychain creates an authn.Keychain from Kubernetes imagePullSecrets
// This allows the inspector to authenticate with private registries using the same
// credentials that Kubernetes uses to pull pod images.
func BuildKeychain(ctx context.Context, client kubernetes.Interface, namespace string, imagePullSecrets []corev1.LocalObjectReference, serviceAccountName string) (authn.Keychain, error) {
	if client == nil {
		// If no client is provided, fall back to default keychain
		// This allows public registries to work without authentication
		// and will use ~/.docker/config.json if available
		return authn.DefaultKeychain, nil
	}

	// Create a k8schain keychain that mimics Kubernetes' image pull behavior
	// This will use the pod's imagePullSecrets and service account secrets

	// Convert LocalObjectReference to string slice
	secretNames := make([]string, len(imagePullSecrets))
	for i, secret := range imagePullSecrets {
		secretNames[i] = secret.Name
	}

	kc, err := k8schain.New(ctx, client, k8schain.Options{
		Namespace:          namespace,
		ServiceAccountName: serviceAccountName,
		ImagePullSecrets:   secretNames,
	})
	if err != nil {
		return nil, fmt.Errorf("creating k8schain: %w", err)
	}

	// Return the keychain which will be used to authenticate registry requests
	return kc, nil
}
