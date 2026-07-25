package k8s

import (
	"os"
	"strings"

	"github.com/platformfuzz/k8s-inspector/internal/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset and provides helper methods
type Client struct {
	clientset kubernetes.Interface
	inCluster bool
	podName   string
	namespace string
	config    *rest.Config
}

// NewClient creates a new Kubernetes client with auto-detection
func NewClient(cfg *config.Config) (*Client, error) {
	k8sConfig, inCluster, err := getK8sConfig(cfg)
	if err != nil {
		return nil, err
	}

	// If no config available, return standalone client
	if k8sConfig == nil {
		return newStandaloneClient(cfg), nil
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	// Get pod name and namespace
	podName, namespace := getPodInfo(cfg, inCluster)

	return &Client{
		clientset: clientset,
		inCluster: inCluster,
		podName:   podName,
		namespace: namespace,
		config:    k8sConfig,
	}, nil
}

// getK8sConfig attempts to get Kubernetes config (in-cluster or kubeconfig)
func getK8sConfig(cfg *config.Config) (*rest.Config, bool, error) {
	// Try in-cluster config first if detected
	if cfg.InCluster {
		k8sConfig, err := rest.InClusterConfig()
		if err == nil {
			return k8sConfig, true, nil
		}
	}

	// Fall back to kubeconfig
	return getKubeconfig()
}

// getKubeconfig attempts to load kubeconfig via client-go loading rules.
func getKubeconfig() (*rest.Config, bool, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	k8sConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		// No usable kubeconfig — standalone mode
		return nil, false, nil
	}

	return k8sConfig, false, nil
}

// newStandaloneClient creates a minimal client for standalone mode
func newStandaloneClient(cfg *config.Config) *Client {
	return &Client{
		clientset: nil,
		inCluster: false,
		podName:   cfg.PodName,
		namespace: cfg.Namespace,
		config:    nil,
	}
}

// getPodInfo retrieves pod name and namespace
func getPodInfo(cfg *config.Config, inCluster bool) (string, string) {
	podName := cfg.PodName
	namespace := cfg.Namespace

	if !inCluster {
		return podName, namespace
	}

	// Get pod name from downward API
	if podName == "" {
		podName = getPodNameFromAPI()
	}

	// Get namespace from downward API
	if namespace == "default" {
		namespace = getNamespaceFromAPI()
	}

	return podName, namespace
}

// getPodNameFromAPI attempts to get pod name from downward API
func getPodNameFromAPI() string {
	// Try multiple downward API paths
	paths := []string{
		"/etc/podinfo/name",
		"/var/run/secrets/kubernetes.io/serviceaccount/namespace", // Sometimes pod name is here
	}

	for _, path := range paths {
		// gosec G304: path is from a hardcoded list, safe to read
		// #nosec G304
		if data, err := os.ReadFile(path); err == nil {
			name := strings.TrimSpace(string(data))
			if name != "" {
				return name
			}
		}
	}

	// Fallback to hostname (in K8s, hostname usually equals pod name)
	hostname, _ := os.Hostname()
	// For StatefulSets, pod name might be base-0, but we want the full name
	// So we keep the full hostname as-is
	return hostname
}

// getNamespaceFromAPI attempts to get namespace from downward API
func getNamespaceFromAPI() string {
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return string(data)
	}
	return "default"
}

// GetClientset returns the Kubernetes clientset
func (c *Client) GetClientset() kubernetes.Interface {
	return c.clientset
}

// IsInCluster returns whether the client is configured for in-cluster mode
func (c *Client) IsInCluster() bool {
	return c.inCluster
}

// GetPodName returns the current pod name
func (c *Client) GetPodName() string {
	return c.podName
}

// GetNamespace returns the current namespace
func (c *Client) GetNamespace() string {
	return c.namespace
}

// IsAvailable returns whether the Kubernetes API is available
func (c *Client) IsAvailable() bool {
	return c.clientset != nil
}

// GetConfig returns the REST config
func (c *Client) GetConfig() *rest.Config {
	return c.config
}
