package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/platformfuzz/k8s-inspector/internal/config"
	"github.com/platformfuzz/k8s-inspector/internal/dashboard"
	"github.com/platformfuzz/k8s-inspector/internal/inspector"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	config    *config.Config
	dashboard *dashboard.Dashboard
	inspector *inspector.Inspector
}

// NewHandlers creates a new Handlers instance
func NewHandlers(cfg *config.Config, dash *dashboard.Dashboard, insp *inspector.Inspector) *Handlers {
	return &Handlers{
		config:    cfg,
		dashboard: dash,
		inspector: insp,
	}
}

// DashboardHandler renders the dashboard page
func (h *Handlers) DashboardHandler(c *gin.Context) {
	data := h.dashboard.GetData()
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title":          h.config.AppTitle,
		"Version":        h.config.AppVersion,
		"Data":           data,
		"RefreshInterval": h.config.DashboardRefreshInterval,
		"Theme":          h.config.Theme,
		"Request":        c.Request,
	})
}

// InspectorHandler renders the inspector page
func (h *Handlers) InspectorHandler(c *gin.Context) {
	allData, err := h.inspector.GetAll()
	if err != nil || allData == nil {
		// Return empty struct instead of error to prevent template issues
		allData = &inspector.InspectorData{
			EnvVars: make(map[string]string),
			Secrets: make(map[string]string),
			Events:  []inspector.Event{},
		}
	}

	c.HTML(http.StatusOK, "inspector.html", gin.H{
		"Title":       h.config.AppTitle,
		"Version":     h.config.AppVersion,
		"Data":        allData,
		"ShowSecrets": h.config.ShowSecretValues,
		"Theme":      h.config.Theme,
		"Request":     c.Request,
	})
}

// HealthHandler returns health check status
func (h *Handlers) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": h.config.AppTitle,
		"version": h.config.AppVersion,
	})
}

// StatusHandler returns overall status
func (h *Handlers) StatusHandler(c *gin.Context) {
	data := h.dashboard.GetData()
	c.JSON(http.StatusOK, gin.H{
		"status":  data.Status.Overall,
		"message": data.Status.Message,
		"indicators": data.Status.Indicators,
		"inCluster": data.PodInfo.InCluster,
	})
}

// PodInfoHandler returns pod information as JSON
func (h *Handlers) PodInfoHandler(c *gin.Context) {
	podInfo, err := h.inspector.GetPodInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, podInfo)
}

// PodLogsHandler returns container logs
func (h *Handlers) PodLogsHandler(c *gin.Context) {
	containerName := c.Query("container")
	tailLinesStr := c.DefaultQuery("tail", "100")
	tailLines, err := strconv.Atoi(tailLinesStr)
	if err != nil {
		tailLines = 100
	}

	logs, err := h.inspector.GetLogs(containerName, tailLines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      logs,
		"container": containerName,
		"tailLines": tailLines,
	})
}

// PodMetricsHandler returns resource metrics
func (h *Handlers) PodMetricsHandler(c *gin.Context) {
	metrics, err := h.inspector.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// PodEnvHandler returns environment variables
func (h *Handlers) PodEnvHandler(c *gin.Context) {
	envVars, err := h.inspector.GetEnvVars()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"envVars": envVars,
	})
}

// PodSecretsHandler returns secret values
// SECURITY: Secrets are hidden by default. Only show if explicitly requested via query parameter AND config allows it.
func (h *Handlers) PodSecretsHandler(c *gin.Context) {
	secrets, err := h.inspector.GetSecrets()
	if err != nil {
		// Don't expose error details that might leak secret information
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve secrets",
		})
		return
	}

	// SECURITY: Only allow showing secrets if:
	// 1. Config explicitly allows it (SHOW_SECRET_VALUES=true), OR
	// 2. Query parameter explicitly requests it (requires user interaction in UI)
	showParam := c.Query("show")
	showSecrets := false

	// Only show secrets if BOTH conditions are met:
	// - Config allows it (server-side control)
	// - AND user explicitly requests it via query parameter (client-side control)
	if h.config.ShowSecretValues && showParam == "true" {
		showSecrets = true
	} else if showParam == "false" {
		showSecrets = false
	}

	// Always hide secret values by default for security
	if !showSecrets {
		hiddenSecrets := make(map[string]string)
		for key := range secrets {
			hiddenSecrets[key] = "***HIDDEN***"
		}
		c.JSON(http.StatusOK, gin.H{
			"secrets": hiddenSecrets,
			"hidden":  true,
			"warning": "Secret values are hidden by default for security. Enable SHOW_SECRET_VALUES config and use ?show=true to reveal.",
		})
		return
	}

	// SECURITY WARNING: Only reach here if explicitly enabled
	c.JSON(http.StatusOK, gin.H{
		"secrets": secrets,
		"hidden":  false,
		"warning": "⚠️ SECURITY WARNING: Secret values are visible. Do not share screenshots or logs containing these values.",
	})
}

// PodEventsHandler returns pod events
func (h *Handlers) PodEventsHandler(c *gin.Context) {
	events, err := h.inspector.GetEvents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// IndexHandler redirects to dashboard
func (h *Handlers) IndexHandler(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/dashboard")
}

// DiagnosticHandler returns diagnostic information for debugging
func (h *Handlers) DiagnosticHandler(c *gin.Context) {
	diagnostics := gin.H{
		"config": gin.H{
			"inCluster": h.config.InCluster,
			"podName":   h.config.PodName,
			"namespace": h.config.Namespace,
		},
		"k8sClient": gin.H{
			"available": h.inspector != nil && h.inspector.GetK8sClient() != nil && h.inspector.GetK8sClient().IsAvailable(),
			"inCluster": h.inspector != nil && h.inspector.GetK8sClient() != nil && h.inspector.GetK8sClient().IsInCluster(),
		},
	}

	// Try to get pod info to see what error we get
	if h.inspector != nil {
		podInfo, err := h.inspector.GetPodInfo()
		if err != nil {
			diagnostics["podInfoError"] = err.Error()
		} else {
			diagnostics["podInfo"] = gin.H{
				"name":      podInfo.Name,
				"namespace": podInfo.Namespace,
				"phase":     podInfo.Phase,
			}
		}
	}

	c.JSON(http.StatusOK, diagnostics)
}
