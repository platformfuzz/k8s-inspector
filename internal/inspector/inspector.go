package inspector

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/platformfuzz/k8s-inspector/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Inspector provides inspection capabilities for Kubernetes pods
type Inspector struct {
	k8sClient *k8s.Client
}

// PodInfo contains comprehensive pod information
type PodInfo struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	UID               string            `json:"uid"`
	CreationTimestamp time.Time         `json:"creationTimestamp"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	Phase             string            `json:"phase"`
	IP                string            `json:"ip"`
	HostIP            string            `json:"hostIP"`
	NodeName          string            `json:"nodeName"`
	Containers        []ContainerInfo   `json:"containers"`
	Conditions        []PodCondition    `json:"conditions"`
	InCluster         bool              `json:"inCluster"`
}

// ContainerInfo contains container-specific information
type ContainerInfo struct {
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	ImageID      string            `json:"imageID"`
	Ready        bool              `json:"ready"`
	RestartCount int32             `json:"restartCount"`
	State        ContainerState    `json:"state"`
	Resources    ResourceLimits    `json:"resources"`
	Ports        []ContainerPort   `json:"ports"`
}

// ContainerState represents the current state of a container
type ContainerState struct {
	Waiting    *ContainerStateWaiting    `json:"waiting,omitempty"`
	Running    *ContainerStateRunning    `json:"running,omitempty"`
	Terminated *ContainerStateTerminated `json:"terminated,omitempty"`
}

// ContainerStateWaiting represents a waiting container state
type ContainerStateWaiting struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ContainerStateRunning represents a running container state
type ContainerStateRunning struct {
	StartedAt time.Time `json:"startedAt"`
}

// ContainerStateTerminated represents a terminated container state
type ContainerStateTerminated struct {
	ExitCode    int32     `json:"exitCode"`
	Reason      string    `json:"reason"`
	Message     string    `json:"message"`
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt"`
}

// ContainerPort represents a container port
type ContainerPort struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
	HostPort      int32  `json:"hostPort"`
}

// ResourceLimits contains resource limits and requests
type ResourceLimits struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

// PodCondition represents a pod condition
type PodCondition struct {
	Type               string      `json:"type"`
	Status             string      `json:"status"`
	LastProbeTime      *time.Time  `json:"lastProbeTime,omitempty"`
	LastTransitionTime *time.Time  `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// Metrics contains resource usage metrics
type Metrics struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   int    `json:"pods"`
}

// InspectorData contains all data needed for the inspector page
type InspectorData struct {
	PodInfo  *PodInfo            `json:"podInfo"`
	EnvVars  map[string]string    `json:"envVars"`
	Secrets  map[string]string    `json:"secrets"`
	Metrics  *Metrics             `json:"metrics"`
	Events   []Event              `json:"events"`
}

// Event represents a Kubernetes event
type Event struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int32     `json:"count"`
	FirstTime time.Time `json:"firstTime"`
	LastTime  time.Time `json:"lastTime"`
	Source    string    `json:"source"`
}

// NewInspector creates a new Inspector instance
func NewInspector(k8sClient *k8s.Client) *Inspector {
	return &Inspector{
		k8sClient: k8sClient,
	}
}

// GetK8sClient returns the Kubernetes client (for diagnostics)
func (i *Inspector) GetK8sClient() *k8s.Client {
	return i.k8sClient
}

// GetPodInfo retrieves comprehensive information about the current pod
func (i *Inspector) GetPodInfo() (*PodInfo, error) {
	if !i.k8sClient.IsAvailable() {
		// Standalone mode - return basic info
		hostname, _ := os.Hostname()
		return &PodInfo{
			Name:      hostname,
			Namespace: i.k8sClient.GetNamespace(),
			Phase:     "Running",
			InCluster: false,
			Containers: []ContainerInfo{
				{
					Name:  "standalone",
					Image: "standalone",
					Ready: true,
				},
			},
		}, nil
	}

	clientset := i.k8sClient.GetClientset()
	podName := i.k8sClient.GetPodName()
	namespace := i.k8sClient.GetNamespace()

	if podName == "" {
		hostname, _ := os.Hostname()
		podName = hostname
	}

	// Try to get pod by name first
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		// If direct lookup fails, try fallback methods
		pod, err = i.findPodFallback(i.k8sClient, namespace, podName, err)
		if err != nil {
			return nil, fmt.Errorf("failed to get pod %s/%s (may need RBAC permissions or pod name mismatch): %w", namespace, podName, err)
		}
	}

	containers := i.buildContainerInfos(pod)
	conditions := i.buildPodConditions(pod)

	return &PodInfo{
		Name:              pod.Name,
		Namespace:         pod.Namespace,
		UID:               string(pod.UID),
		CreationTimestamp: pod.CreationTimestamp.Time,
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		Phase:             string(pod.Status.Phase),
		IP:                pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		NodeName:          pod.Spec.NodeName,
		Containers:        containers,
		Conditions:        conditions,
		InCluster:         i.k8sClient.IsInCluster(),
	}, nil
}

// GetLogs retrieves container logs
func (i *Inspector) GetLogs(containerName string, tailLines int) ([]string, error) {
	if !i.k8sClient.IsAvailable() {
		// Standalone mode - return empty logs or read from stdout/stderr
		return []string{"[Standalone mode] Logs not available"}, nil
	}

	clientset := i.k8sClient.GetClientset()
	podName := i.k8sClient.GetPodName()
	namespace := i.k8sClient.GetNamespace()

	if podName == "" {
		hostname, _ := os.Hostname()
		podName = hostname
	}

	// If container name not specified, use first container
	if containerName == "" {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get pod: %w", err)
		}
		if len(pod.Spec.Containers) > 0 {
			containerName = pod.Spec.Containers[0].Name
		}
	}

	// Get logs
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		TailLines: int64Ptr(int64(tailLines)),
	})

	stream, err := req.Stream(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to stream logs: %w", err)
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			log.Printf("Failed to close log stream: %v", closeErr)
		}
	}()

	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	logs := strings.Split(string(data), "\n")
	// Remove empty last line if present
	if len(logs) > 0 && logs[len(logs)-1] == "" {
		logs = logs[:len(logs)-1]
	}

	return logs, nil
}

// GetEnvVars retrieves environment variables from the pod
func (i *Inspector) GetEnvVars() (map[string]string, error) {
	if !i.k8sClient.IsAvailable() {
		// Standalone mode - return current process env vars
		envVars := make(map[string]string)
		for _, env := range os.Environ() {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envVars[parts[0]] = parts[1]
			}
		}
		return envVars, nil
	}

	clientset := i.k8sClient.GetClientset()
	podName := i.k8sClient.GetPodName()
	namespace := i.k8sClient.GetNamespace()

	if podName == "" {
		hostname, _ := os.Hostname()
		podName = hostname
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	envVars := make(map[string]string)

	// Collect env vars from all containers
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if env.Value != "" {
				envVars[env.Name] = env.Value
			} else if env.ValueFrom != nil {
				// Try to resolve the actual value
				resolvedValue, err := i.resolveEnvVarReference(env.ValueFrom, pod)
				if err == nil && resolvedValue != "" {
					envVars[env.Name] = resolvedValue
				} else {
					// Fallback to reference description
					envVars[env.Name] = i.getEnvVarReference(env.ValueFrom)
				}
			}
		}
	}

	return envVars, nil
}

// GetSecrets retrieves secret values referenced by the pod
func (i *Inspector) GetSecrets() (map[string]string, error) {
	if !i.k8sClient.IsAvailable() {
		return make(map[string]string), nil
	}

	clientset := i.k8sClient.GetClientset()
	podName := i.k8sClient.GetPodName()
	namespace := i.k8sClient.GetNamespace()

	if podName == "" {
		hostname, _ := os.Hostname()
		podName = hostname
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	secrets := make(map[string]string)
	secretNames := make(map[string]bool)

	// Find all secret references
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				secretName := env.ValueFrom.SecretKeyRef.Name
				if !secretNames[secretName] {
					secretNames[secretName] = true
				}
			}
		}
	}

	// Also check volume mounts
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			secretNames[volume.Secret.SecretName] = true
		}
	}

	// Retrieve secret values
	for secretName := range secretNames {
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			continue // Skip secrets we can't access
		}

		for key, value := range secret.Data {
			fullKey := fmt.Sprintf("%s/%s", secretName, key)
			secrets[fullKey] = string(value)
		}
	}

	return secrets, nil
}

// GetMetrics retrieves resource usage metrics
func (i *Inspector) GetMetrics() (*Metrics, error) {
	if !i.k8sClient.IsAvailable() {
		return &Metrics{
			CPU:    "N/A",
			Memory: "N/A",
			Pods:   0,
		}, nil
	}

	clientset := i.k8sClient.GetClientset()
	namespace := i.k8sClient.GetNamespace()

	// Try to get metrics from metrics-server
	// Note: This requires metrics-server to be installed in the cluster
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	return &Metrics{
		CPU:    "N/A", // Would need metrics-server API
		Memory: "N/A", // Would need metrics-server API
		Pods:   len(pods.Items),
	}, nil
}

// GetEvents retrieves recent events for the pod
func (i *Inspector) GetEvents() ([]Event, error) {
	if !i.k8sClient.IsAvailable() {
		return []Event{}, nil
	}

	clientset := i.k8sClient.GetClientset()
	podName := i.k8sClient.GetPodName()
	namespace := i.k8sClient.GetNamespace()

	if podName == "" {
		hostname, _ := os.Hostname()
		podName = hostname
	}

	events, err := clientset.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	result := make([]Event, 0, len(events.Items))
	for _, event := range events.Items {
		result = append(result, Event{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Count:     event.Count,
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Source:    fmt.Sprintf("%s/%s", event.Source.Component, event.Source.Host),
		})
	}

	return result, nil
}

// GetAll retrieves all inspection data at once
func (i *Inspector) GetAll() (*InspectorData, error) {
	data := &InspectorData{
		EnvVars: make(map[string]string),
		Secrets: make(map[string]string),
		Events:  []Event{},
	}

	// Get pod info (always try, even if it fails)
	podInfo, err := i.GetPodInfo()
	if err == nil && podInfo != nil {
		data.PodInfo = podInfo
	}

	// Get env vars (non-critical, continue on error)
	envVars, err := i.GetEnvVars()
	if err == nil && envVars != nil {
		data.EnvVars = envVars
	}

	// Get secrets (non-critical, continue on error)
	secrets, err := i.GetSecrets()
	if err == nil && secrets != nil {
		data.Secrets = secrets
	}

	// Get metrics (non-critical, continue on error)
	metrics, err := i.GetMetrics()
	if err == nil && metrics != nil {
		data.Metrics = metrics
	}

	// Get events (non-critical, continue on error)
	events, err := i.GetEvents()
	if err == nil && events != nil {
		data.Events = events
	}

	// Always return valid data, even if some operations failed
	return data, nil
}

// Helper function
func int64Ptr(i int64) *int64 {
	return &i
}

// buildContainerInfos builds container information from pod spec and status
// findPodFallback attempts to find a pod using fallback methods when direct lookup fails
func (i *Inspector) findPodFallback(k8sClient *k8s.Client, namespace, podName string, originalErr error) (*corev1.Pod, error) {
	log.Printf("Failed to get pod %s/%s: %v, attempting fallback", namespace, podName, originalErr)

	// Try to list pods and find one matching hostname
	hostname, _ := os.Hostname()
	clientset := k8sClient.GetClientset()
	pods, listErr := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if listErr != nil || len(pods.Items) == 0 {
		return nil, originalErr
	}

	// Try to find pod by hostname or name match
	for idx := range pods.Items {
		p := &pods.Items[idx]
		if p.Spec.Hostname == hostname || p.Name == hostname || strings.HasPrefix(p.Name, hostname) {
			log.Printf("Found pod %s/%s via fallback (hostname: %s)", p.Namespace, p.Name, hostname)
			return p, nil
		}
	}

	// If no match found but we have pods, use the first one as last resort
	if len(pods.Items) == 1 {
		pod := &pods.Items[0]
		log.Printf("Using first available pod %s/%s as fallback", pod.Namespace, pod.Name)
		return pod, nil
	}

	return nil, originalErr
}

// buildPodConditions converts pod conditions to our format
func (i *Inspector) buildPodConditions(pod *corev1.Pod) []PodCondition {
	conditions := make([]PodCondition, 0, len(pod.Status.Conditions))
	for _, cond := range pod.Status.Conditions {
		var lastProbeTime *time.Time
		if !cond.LastProbeTime.IsZero() {
			t := cond.LastProbeTime.Time
			lastProbeTime = &t
		}

		var lastTransitionTime *time.Time
		if !cond.LastTransitionTime.IsZero() {
			t := cond.LastTransitionTime.Time
			lastTransitionTime = &t
		}

		conditions = append(conditions, PodCondition{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			LastProbeTime:      lastProbeTime,
			LastTransitionTime: lastTransitionTime,
			Reason:             cond.Reason,
			Message:            cond.Message,
		})
	}
	return conditions
}

func (i *Inspector) buildContainerInfos(pod *corev1.Pod) []ContainerInfo {
	containers := make([]ContainerInfo, 0, len(pod.Spec.Containers))
	for idx, container := range pod.Spec.Containers {
		containerInfo := ContainerInfo{
			Name:  container.Name,
			Image: container.Image,
			Ports: make([]ContainerPort, 0, len(container.Ports)),
		}

		// Get container status
		if idx < len(pod.Status.ContainerStatuses) {
			i.populateContainerStatus(&containerInfo, &pod.Status.ContainerStatuses[idx])
		}

		// Ports
		for _, port := range container.Ports {
			containerInfo.Ports = append(containerInfo.Ports, ContainerPort{
				Name:          port.Name,
				ContainerPort: port.ContainerPort,
				Protocol:      string(port.Protocol),
				HostPort:      port.HostPort,
			})
		}

		// Resources
		containerInfo.Resources = i.buildResourceLimits(container.Resources)

		containers = append(containers, containerInfo)
	}
	return containers
}

// populateContainerStatus populates container status information
func (i *Inspector) populateContainerStatus(containerInfo *ContainerInfo, status *corev1.ContainerStatus) {
	containerInfo.Ready = status.Ready
	containerInfo.RestartCount = status.RestartCount
	containerInfo.ImageID = status.ImageID

	// Container state
	if status.State.Waiting != nil {
		containerInfo.State.Waiting = &ContainerStateWaiting{
			Reason:  status.State.Waiting.Reason,
			Message: status.State.Waiting.Message,
		}
	}
	if status.State.Running != nil {
		containerInfo.State.Running = &ContainerStateRunning{
			StartedAt: status.State.Running.StartedAt.Time,
		}
	}
	if status.State.Terminated != nil {
		containerInfo.State.Terminated = &ContainerStateTerminated{
			ExitCode:   status.State.Terminated.ExitCode,
			Reason:     status.State.Terminated.Reason,
			Message:    status.State.Terminated.Message,
			StartedAt:  status.State.Terminated.StartedAt.Time,
			FinishedAt: status.State.Terminated.FinishedAt.Time,
		}
	}
}

// buildResourceLimits builds resource limits from container resources
func (i *Inspector) buildResourceLimits(resources corev1.ResourceRequirements) ResourceLimits {
	rl := ResourceLimits{
		Requests: make(map[string]string),
		Limits:   make(map[string]string),
	}
	if resources.Requests != nil {
		for k, v := range resources.Requests {
			rl.Requests[string(k)] = v.String()
		}
	}
	if resources.Limits != nil {
		for k, v := range resources.Limits {
			rl.Limits[string(k)] = v.String()
		}
	}
	return rl
}

// resolveEnvVarReference attempts to resolve the actual value of an environment variable reference
func (i *Inspector) resolveEnvVarReference(valueFrom *corev1.EnvVarSource, pod *corev1.Pod) (string, error) {
	if valueFrom.FieldRef != nil {
		return i.resolveFieldRef(valueFrom.FieldRef, pod)
	}

	if valueFrom.SecretKeyRef != nil {
		return i.resolveSecretKeyRef(valueFrom.SecretKeyRef)
	}

	if valueFrom.ConfigMapKeyRef != nil {
		return i.resolveConfigMapKeyRef(valueFrom.ConfigMapKeyRef)
	}

	return "", fmt.Errorf("unable to resolve reference")
}

// resolveFieldRef handles downward API field references
func (i *Inspector) resolveFieldRef(fieldRef *corev1.ObjectFieldSelector, pod *corev1.Pod) (string, error) {
	switch fieldRef.FieldPath {
	case "metadata.name":
		return pod.Name, nil
	case "metadata.namespace":
		return pod.Namespace, nil
	case "metadata.uid":
		return string(pod.UID), nil
	case "spec.nodeName":
		return pod.Spec.NodeName, nil
	case "status.hostIP":
		return pod.Status.HostIP, nil
	case "status.podIP":
		return pod.Status.PodIP, nil
	case "status.podIPs":
		if len(pod.Status.PodIPs) > 0 {
			return pod.Status.PodIPs[0].IP, nil
		}
		return pod.Status.PodIP, nil
	default:
		return i.resolveFieldRefFromEnv(fieldRef.FieldPath)
	}
}

// resolveFieldRefFromEnv attempts to resolve field ref from environment variables
func (i *Inspector) resolveFieldRefFromEnv(fieldPath string) (string, error) {
	if fieldPath == "metadata.name" {
		if val := os.Getenv("POD_NAME"); val != "" {
			return val, nil
		}
	}
	if fieldPath == "metadata.namespace" {
		if val := os.Getenv("POD_NAMESPACE"); val != "" {
			return val, nil
		}
	}
	return "", fmt.Errorf("unable to resolve field reference: %s", fieldPath)
}

// resolveSecretKeyRef retrieves a value from a Kubernetes secret
func (i *Inspector) resolveSecretKeyRef(secretKeyRef *corev1.SecretKeySelector) (string, error) {
	clientset := i.k8sClient.GetClientset()
	namespace := i.k8sClient.GetNamespace()
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretKeyRef.Name, err)
	}

	if val, ok := secret.Data[secretKeyRef.Key]; ok {
		return string(val), nil
	}

	return "", fmt.Errorf("key %s not found in secret %s", secretKeyRef.Key, secretKeyRef.Name)
}

// resolveConfigMapKeyRef retrieves a value from a Kubernetes configmap
func (i *Inspector) resolveConfigMapKeyRef(configMapKeyRef *corev1.ConfigMapKeySelector) (string, error) {
	clientset := i.k8sClient.GetClientset()
	namespace := i.k8sClient.GetNamespace()
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get configmap %s: %w", configMapKeyRef.Name, err)
	}

	if val, ok := configMap.Data[configMapKeyRef.Key]; ok {
		return val, nil
	}

	return "", fmt.Errorf("key %s not found in configmap %s", configMapKeyRef.Key, configMapKeyRef.Name)
}

// getEnvVarReference returns a string representation of an environment variable reference
func (i *Inspector) getEnvVarReference(valueFrom *corev1.EnvVarSource) string {
	switch {
	case valueFrom.SecretKeyRef != nil:
		return fmt.Sprintf("[Secret: %s/%s]", valueFrom.SecretKeyRef.Name, valueFrom.SecretKeyRef.Key)
	case valueFrom.ConfigMapKeyRef != nil:
		return fmt.Sprintf("[ConfigMap: %s/%s]", valueFrom.ConfigMapKeyRef.Name, valueFrom.ConfigMapKeyRef.Key)
	case valueFrom.FieldRef != nil:
		return fmt.Sprintf("[FieldRef: %s]", valueFrom.FieldRef.FieldPath)
	default:
		return "[Reference]"
	}
}
