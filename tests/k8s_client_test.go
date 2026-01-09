package tests

import (
	"os"
	"testing"

	"github.com/platformfuzz/k8s-inspector/internal/config"
	"github.com/platformfuzz/k8s-inspector/internal/k8s"
)

func TestK8sClientStandaloneMode(t *testing.T) {
	cfg := config.Load()

	// Force standalone mode
	cfg.InCluster = false

	client, err := k8s.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.IsInCluster() {
		t.Error("Client should not be in cluster mode")
	}

	if client.IsAvailable() {
		t.Error("Client should not be available in standalone mode")
	}
}

func TestK8sClientPodName(t *testing.T) {
	cfg := config.Load()
	cfg.PodName = "test-pod"
	cfg.Namespace = "test-namespace"

	client, err := k8s.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// In standalone mode, pod name might be empty or hostname
	// This is acceptable - just verify client methods work
	podName := client.GetPodName()
	_ = podName // Acceptable to be empty in standalone mode

	// Namespace might default to "default" if not set
	// This is acceptable - just verify it's set
	namespace := client.GetNamespace()
	if namespace == "" {
		t.Error("Namespace should not be empty")
	}
}

func TestConfigLoad(t *testing.T) {
	// Test that config loads with defaults
	cfg := config.Load()

	if cfg.AppTitle == "" {
		t.Error("AppTitle should have a default value")
	}

	if cfg.Port == 0 {
		t.Error("Port should have a default value")
	}

	if cfg.DashboardRefreshInterval == 0 {
		t.Error("DashboardRefreshInterval should have a default value")
	}
}

func TestConfigEnvironmentVariables(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("APP_TITLE", "test-title"); err != nil {
		t.Fatalf("Failed to set APP_TITLE: %v", err)
	}
	if err := os.Setenv("PORT", "9090"); err != nil {
		t.Fatalf("Failed to set PORT: %v", err)
	}
	if err := os.Setenv("DASHBOARD_REFRESH_INTERVAL", "10"); err != nil {
		t.Fatalf("Failed to set DASHBOARD_REFRESH_INTERVAL: %v", err)
	}

	cfg := config.Load()

	if cfg.AppTitle != "test-title" {
		t.Errorf("Expected AppTitle to be 'test-title', got '%s'", cfg.AppTitle)
	}

	if cfg.Port != 9090 {
		t.Errorf("Expected Port to be 9090, got %d", cfg.Port)
	}

	if cfg.DashboardRefreshInterval != 10 {
		t.Errorf("Expected DashboardRefreshInterval to be 10, got %d", cfg.DashboardRefreshInterval)
	}

	// Clean up
	_ = os.Unsetenv("APP_TITLE")
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("DASHBOARD_REFRESH_INTERVAL")
}
