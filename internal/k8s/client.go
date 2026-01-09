package k8s

import (
	"os"
	"path/filepath"

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

// getKubeconfig attempts to load kubeconfig
func getKubeconfig() (*rest.Config, bool, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(kubeconfig); err != nil {
		// No kubeconfig found
		return nil, false, nil
	}

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, false, err
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
	if data, err := os.ReadFile("/etc/podinfo/name"); err == nil {
		return string(data)
	}
	// Fallback to hostname
	hostname, _ := os.Hostname()
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
