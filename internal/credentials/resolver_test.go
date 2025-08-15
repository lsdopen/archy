package credentials

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestResolver_PodImagePullSecretsExtraction(t *testing.T) {
	client := fake.NewSimpleClientset()
	resolver := NewResolver(client)

	// Create test secret
	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			"registry.example.com": map[string]interface{}{
				"username": "testuser",
				"password": "testpass",
				"auth":     base64.StdEncoding.EncodeToString([]byte("testuser:testpass")),
			},
		},
	}
	configJSON, _ := json.Marshal(dockerConfig)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}
	client.CoreV1().Secrets("default").Create(nil, secret, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "test-secret"},
			},
		},
	}

	cred, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	require.NoError(t, err)
	require.NotNil(t, cred)
	assert.Equal(t, "testuser", cred.Username)
	assert.Equal(t, "testpass", cred.Password)
}

func TestResolver_ServiceAccountImagePullSecretsFallback(t *testing.T) {
	client := fake.NewSimpleClientset()
	resolver := NewResolver(client)

	// Create service account with imagePullSecrets
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "default",
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: "sa-secret"},
		},
	}
	client.CoreV1().ServiceAccounts("default").Create(nil, sa, metav1.CreateOptions{})

	// Create secret
	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			"registry.example.com": map[string]interface{}{
				"username": "sauser",
				"password": "sapass",
			},
		},
	}
	configJSON, _ := json.Marshal(dockerConfig)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}
	client.CoreV1().Secrets("default").Create(nil, secret, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "test-sa",
		},
	}

	cred, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	require.NoError(t, err)
	require.NotNil(t, cred)
	assert.Equal(t, "sauser", cred.Username)
	assert.Equal(t, "sapass", cred.Password)
}

func TestResolver_DockerConfigJsonParsing(t *testing.T) {
	tests := []struct {
		name       string
		configData string
		registry   string
		wantUser   string
		wantPass   string
		wantErr    bool
	}{
		{
			name: "valid config with auth",
			configData: `{
				"auths": {
					"registry.example.com": {
						"username": "user1",
						"password": "pass1",
						"auth": "dXNlcjE6cGFzczE="
					}
				}
			}`,
			registry: "registry.example.com",
			wantUser: "user1",
			wantPass: "pass1",
		},
		{
			name: "config with auth field only",
			configData: `{
				"auths": {
					"registry.example.com": {
						"auth": "dXNlcjI6cGFzczI="
					}
				}
			}`,
			registry: "registry.example.com",
			wantUser: "user2",
			wantPass: "pass2",
		},
		{
			name: "invalid JSON",
			configData: `{invalid json}`,
			registry:   "registry.example.com",
			wantErr:    true,
		},
		{
			name: "missing registry",
			configData: `{
				"auths": {
					"other-registry.com": {
						"username": "user",
						"password": "pass"
					}
				}
			}`,
			registry: "registry.example.com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			resolver := NewResolver(client)

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(tt.configData),
				},
			}

			cred, err := resolver.parseDockerConfigSecret(secret, tt.registry)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cred)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cred)
				assert.Equal(t, tt.wantUser, cred.Username)
				assert.Equal(t, tt.wantPass, cred.Password)
			}
		})
	}
}

func TestResolver_CredentialCaching(t *testing.T) {
	client := fake.NewSimpleClientset()
	resolver := NewResolver(client)

	// Create secret
	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			"registry.example.com": map[string]interface{}{
				"username": "cached-user",
				"password": "cached-pass",
			},
		},
	}
	configJSON, _ := json.Marshal(dockerConfig)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}
	client.CoreV1().Secrets("default").Create(nil, secret, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "cache-secret"},
			},
		},
	}

	// First call should cache the credential
	cred1, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	require.NoError(t, err)
	require.NotNil(t, cred1)

	// Second call should return cached credential
	cred2, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	require.NoError(t, err)
	require.NotNil(t, cred2)

	assert.Equal(t, cred1.Username, cred2.Username)
	assert.Equal(t, cred1.Password, cred2.Password)
}

func TestResolver_TTLExpiration(t *testing.T) {
	client := fake.NewSimpleClientset()
	resolver := NewResolverWithTTL(client, 10*time.Millisecond)

	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			"registry.example.com": map[string]interface{}{
				"username": "ttl-user",
				"password": "ttl-pass",
			},
		},
	}
	configJSON, _ := json.Marshal(dockerConfig)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ttl-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: configJSON,
		},
	}
	client.CoreV1().Secrets("default").Create(nil, secret, metav1.CreateOptions{})

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ttl-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "ttl-secret"},
			},
		},
	}

	// Cache credential
	cred1, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	require.NoError(t, err)
	require.NotNil(t, cred1)

	// Wait for TTL expiration
	time.Sleep(20 * time.Millisecond)

	// Should fetch fresh credential
	cred2, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	require.NoError(t, err)
	require.NotNil(t, cred2)
}

func TestResolver_MissingSecretHandling(t *testing.T) {
	client := fake.NewSimpleClientset()
	resolver := NewResolver(client)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "missing-secret-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "nonexistent-secret"},
			},
		},
	}

	cred, err := resolver.ResolveCredentials(pod, "registry.example.com/image:tag")
	assert.NoError(t, err) // Should not error, just return nil
	assert.Nil(t, cred)
}

func TestResolver_RegistryURLMatching(t *testing.T) {
	tests := []struct {
		name         string
		configHost   string
		imageRef     string
		shouldMatch  bool
	}{
		{
			name:        "exact match",
			configHost:  "registry.example.com",
			imageRef:    "registry.example.com/image:tag",
			shouldMatch: true,
		},
		{
			name:        "docker hub official",
			configHost:  "https://index.docker.io/v1/",
			imageRef:    "nginx:latest",
			shouldMatch: true,
		},
		{
			name:        "docker hub registry",
			configHost:  "registry-1.docker.io",
			imageRef:    "library/nginx:latest",
			shouldMatch: true,
		},
		{
			name:        "no match",
			configHost:  "registry.example.com",
			imageRef:    "other-registry.com/image:tag",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()
			resolver := NewResolver(client)

			matches := resolver.registryMatches(tt.configHost, tt.imageRef)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}