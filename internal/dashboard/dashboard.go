package dashboard

import (
	"log"

	"github.com/platformfuzz/k8s-inspector/internal/inspector"
)

// Status constants
const (
	statusHealthy = "healthy"
	statusWarning = "warning"
	statusError   = "error"
	statusUnknown = "unknown"
)

// Dashboard provides dashboard data aggregation
type Dashboard struct {
	inspector *inspector.Inspector
}

// DashboardData contains all data needed for the dashboard
type DashboardData struct {
	PodInfo      *inspector.PodInfo   `json:"podInfo"`
	Metrics      *inspector.Metrics    `json:"metrics"`
	Status       Status                `json:"status"`
	RefreshInterval int                `json:"refreshInterval"`
}

// Status represents the overall status of the pod
type Status struct {
	Overall    string            `json:"overall"`    // "healthy", "warning", "error"
	Message    string            `json:"message"`
	Indicators map[string]string `json:"indicators"` // Component statuses
}

// NewDashboard creates a new Dashboard instance
func NewDashboard(insp *inspector.Inspector) *Dashboard {
	return &Dashboard{
		inspector: insp,
	}
}

// GetData retrieves and aggregates all dashboard data
func (d *Dashboard) GetData() *DashboardData {
	podInfo, err := d.inspector.GetPodInfo()
	if err != nil {
		log.Printf("Warning: Failed to get pod info: %v", err)
		// GetPodInfo() should always return a valid PodInfo (handles standalone mode),
		// but if it's nil, create a minimal fallback
		if podInfo == nil {
			podInfo = &inspector.PodInfo{
				Name:      "unknown",
				Namespace: "unknown",
				Phase:     "Unknown",
				InCluster: false,
			}
		}
	}

	metrics, _ := d.inspector.GetMetrics()

	status := d.calculateStatus(podInfo)

	return &DashboardData{
		PodInfo:      podInfo,
		Metrics:      metrics,
		Status:       status,
		RefreshInterval: 5, // Default refresh interval in seconds
	}
}

// calculateStatus determines the overall pod status and component indicators
func (d *Dashboard) calculateStatus(podInfo *inspector.PodInfo) Status {
	status := Status{
		Overall:    statusHealthy,
		Message:    "Pod is running normally",
		Indicators: make(map[string]string),
	}

	if podInfo == nil {
		status.Overall = statusError
		status.Message = "Unable to retrieve pod information"
		return status
	}

	// Check pod phase
	d.checkPodPhase(podInfo, &status)

	// Check container statuses
	allReady, hasErrors := d.checkContainerStatuses(podInfo, &status)

	// Update overall status based on containers
	d.updateOverallStatus(allReady, hasErrors, &status)

	// Check pod conditions
	d.checkPodConditions(podInfo, &status)

	return status
}

// checkPodPhase checks the pod phase and updates status
func (d *Dashboard) checkPodPhase(podInfo *inspector.PodInfo, status *Status) {
	switch podInfo.Phase {
	case "Running":
		status.Indicators["pod"] = statusHealthy
	case "Pending":
		status.Overall = statusWarning
		status.Message = "Pod is pending"
		status.Indicators["pod"] = statusWarning
	case "Failed":
		status.Overall = statusError
		status.Message = "Pod has failed"
		status.Indicators["pod"] = statusError
	case "Succeeded":
		status.Overall = statusWarning
		status.Message = "Pod has completed"
		status.Indicators["pod"] = statusWarning
	case "Unknown":
		status.Overall = statusError
		status.Message = "Pod status is unknown"
		status.Indicators["pod"] = statusError
	default:
		status.Indicators["pod"] = statusUnknown
	}
}

// checkContainerStatuses checks container statuses and returns readiness info
func (d *Dashboard) checkContainerStatuses(podInfo *inspector.PodInfo, status *Status) (allReady bool, hasErrors bool) {
	allReady = true
	hasErrors = false

	for _, container := range podInfo.Containers {
		if !container.Ready {
			allReady = false
		}

		containerStatus := d.getContainerStatus(container)
		status.Indicators[container.Name] = containerStatus

		if containerStatus == statusError {
			hasErrors = true
		}
		if containerStatus == statusWarning && container.Ready {
			allReady = false
		}
	}

	return allReady, hasErrors
}

// getContainerStatus determines the status of a single container
func (d *Dashboard) getContainerStatus(container inspector.ContainerInfo) string {
	switch {
	case container.State.Waiting != nil:
		if container.State.Waiting.Reason == "Error" || container.State.Waiting.Reason == "CrashLoopBackOff" {
			return statusError
		}
		return statusWarning
	case container.State.Terminated != nil:
		if container.State.Terminated.ExitCode != 0 {
			return statusError
		}
		return statusWarning
	case container.State.Running != nil:
		if container.Ready {
			return statusHealthy
		}
		return statusWarning
	default:
		return statusUnknown
	}
}

// updateOverallStatus updates the overall status based on container state
func (d *Dashboard) updateOverallStatus(allReady, hasErrors bool, status *Status) {
	if hasErrors {
		status.Overall = statusError
		status.Message = "One or more containers have errors"
	} else if !allReady && status.Overall == statusHealthy {
		status.Overall = statusWarning
		status.Message = "Not all containers are ready"
	}
}

// checkPodConditions checks pod conditions and updates status
func (d *Dashboard) checkPodConditions(podInfo *inspector.PodInfo, status *Status) {
	for _, condition := range podInfo.Conditions {
		switch condition.Type {
		case "Ready":
			if condition.Status != "True" {
				status.Overall = statusWarning
				status.Message = "Pod is not ready"
				status.Indicators["readiness"] = statusWarning
			} else {
				status.Indicators["readiness"] = statusHealthy
			}
		case "PodScheduled":
			if condition.Status != "True" {
				status.Overall = statusError
				status.Message = "Pod scheduling failed"
				status.Indicators["scheduling"] = statusError
			} else {
				status.Indicators["scheduling"] = statusHealthy
			}
		}
	}
}

// GetStatusColor returns a color code for a status
func GetStatusColor(status string) string {
	switch status {
	case statusHealthy:
		return "#28a745" // Green
	case statusWarning:
		return "#ffc107" // Yellow/Orange
	case statusError:
		return "#dc3545" // Red
	default:
		return "#6c757d" // Gray
	}
}
