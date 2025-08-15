package credentials

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// RegistryCredential holds registry authentication information
type RegistryCredential struct {
	Username string
	Password string
	Registry string
}

// Resolver handles credential resolution for container registries
type Resolver struct {
	client kubernetes.Interface
	cache  map[string]*cacheEntry
	mu     sync.RWMutex
	ttl    time.Duration
}

type cacheEntry struct {
	credential *RegistryCredential
	expiry     time.Time
}

type dockerConfig struct {
	Auths map[string]dockerAuth `json:"auths"`
}

type dockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

// NewResolver creates a new credential resolver
func NewResolver(client kubernetes.Interface) *Resolver {
	return &Resolver{
		client: client,
		cache:  make(map[string]*cacheEntry),
		ttl:    5 * time.Minute,
	}
}

// NewResolverWithTTL creates a resolver with custom TTL
func NewResolverWithTTL(client kubernetes.Interface, ttl time.Duration) *Resolver {
	return &Resolver{
		client: client,
		cache:  make(map[string]*cacheEntry),
		ttl:    ttl,
	}
}

// ResolveCredentials resolves registry credentials using hybrid priority chain
func (r *Resolver) ResolveCredentials(pod *corev1.Pod, imageRef string) (*RegistryCredential, error) {
	registry := extractRegistry(imageRef)
	cacheKey := fmt.Sprintf("%s/%s:%s", pod.Namespace, pod.Name, registry)

	// Check cache first
	if cred := r.getFromCache(cacheKey); cred != nil {
		return cred, nil
	}

	// 1. Try pod imagePullSecrets
	if cred := r.getPodCredentials(pod, registry); cred != nil {
		r.setCache(cacheKey, cred)
		return cred, nil
	}

	// 2. Try service account imagePullSecrets
	if cred := r.getServiceAccountCredentials(pod, registry); cred != nil {
		r.setCache(cacheKey, cred)
		return cred, nil
	}

	// 3. Return nil for anonymous access
	return nil, nil
}

func (r *Resolver) getPodCredentials(pod *corev1.Pod, registry string) *RegistryCredential {
	for _, secretRef := range pod.Spec.ImagePullSecrets {
		if cred := r.getSecretCredential(pod.Namespace, secretRef.Name, registry); cred != nil {
			return cred
		}
	}
	return nil
}

func (r *Resolver) getServiceAccountCredentials(pod *corev1.Pod, registry string) *RegistryCredential {
	saName := pod.Spec.ServiceAccountName
	if saName == "" {
		saName = "default"
	}

	sa, err := r.client.CoreV1().ServiceAccounts(pod.Namespace).Get(context.TODO(), saName, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	for _, secretRef := range sa.ImagePullSecrets {
		if cred := r.getSecretCredential(pod.Namespace, secretRef.Name, registry); cred != nil {
			return cred
		}
	}
	return nil
}

func (r *Resolver) getSecretCredential(namespace, secretName, registry string) *RegistryCredential {
	secret, err := r.client.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	cred, err := r.parseDockerConfigSecret(secret, registry)
	if err != nil {
		return nil
	}

	return cred
}

func (r *Resolver) parseDockerConfigSecret(secret *corev1.Secret, registry string) (*RegistryCredential, error) {
	var configData []byte
	var ok bool

	// Try new format first
	if configData, ok = secret.Data[corev1.DockerConfigJsonKey]; !ok {
		// Try legacy format
		if configData, ok = secret.Data[corev1.DockerConfigKey]; !ok {
			return nil, fmt.Errorf("secret does not contain docker config")
		}
	}

	var config dockerConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse docker config: %w", err)
	}

	for host, auth := range config.Auths {
		if r.registryMatches(host, registry) {
			cred := &RegistryCredential{
				Registry: registry,
			}

			// Try username/password first
			if auth.Username != "" && auth.Password != "" {
				cred.Username = auth.Username
				cred.Password = auth.Password
				return cred, nil
			}

			// Try auth field
			if auth.Auth != "" {
				decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
				if err != nil {
					continue
				}
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 {
					cred.Username = parts[0]
					cred.Password = parts[1]
					return cred, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no credentials found for registry %s", registry)
}

func (r *Resolver) registryMatches(configHost, imageRegistry string) bool {
	// Normalize hosts
	configHost = strings.TrimPrefix(configHost, "https://")
	configHost = strings.TrimPrefix(configHost, "http://")
	configHost = strings.TrimSuffix(configHost, "/")
	configHost = strings.TrimSuffix(configHost, "/v1")

	// Handle Docker Hub special cases
	dockerHubHosts := []string{
		"index.docker.io",
		"registry-1.docker.io",
		"docker.io",
	}

	isConfigDockerHub := false
	isImageDockerHub := imageRegistry == "docker.io" || imageRegistry == ""

	for _, host := range dockerHubHosts {
		if strings.Contains(configHost, host) {
			isConfigDockerHub = true
			break
		}
	}

	if isConfigDockerHub && isImageDockerHub {
		return true
	}

	// Direct match
	return configHost == imageRegistry
}

func (r *Resolver) getFromCache(key string) *RegistryCredential {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.cache[key]
	if !exists || time.Now().After(entry.expiry) {
		return nil
	}

	return entry.credential
}

func (r *Resolver) setCache(key string, cred *RegistryCredential) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[key] = &cacheEntry{
		credential: cred,
		expiry:     time.Now().Add(r.ttl),
	}
}

func extractRegistry(imageRef string) string {
	// Handle Docker Hub images
	if !strings.Contains(imageRef, "/") || (!strings.Contains(imageRef, ".") && !strings.Contains(imageRef, ":")) {
		return "docker.io"
	}

	parts := strings.Split(imageRef, "/")
	if len(parts) > 0 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		return parts[0]
	}

	return "docker.io"
}