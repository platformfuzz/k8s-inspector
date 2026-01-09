package tests

import (
	"testing"

	"github.com/platformfuzz/k8s-inspector/internal/dashboard"
	"github.com/platformfuzz/k8s-inspector/internal/inspector"
	"github.com/platformfuzz/k8s-inspector/internal/k8s"
)

func TestDashboardGetData(t *testing.T) {
	// Create a mock k8s client (standalone mode)
	k8sClient := &k8s.Client{
		// In standalone mode, clientset is nil
	}

	insp := inspector.NewInspector(k8sClient)
	dash := dashboard.NewDashboard(insp)

	data := dash.GetData()

	if data == nil {
		t.Fatal("Dashboard data should not be nil")
	}

	if data.PodInfo == nil {
		t.Fatal("PodInfo should not be nil")
	}

	if data.Status.Overall == "" {
		t.Fatal("Status.Overall should not be empty")
	}
}

func TestDashboardStatusCalculation(t *testing.T) {
	// Test status color function
	colors := map[string]string{
		"healthy": "#28a745",
		"warning": "#ffc107",
		"error":   "#dc3545",
		"unknown": "#6c757d",
	}

	for status, expectedColor := range colors {
		color := dashboard.GetStatusColor(status)
		if color != expectedColor {
			t.Errorf("Expected color %s for status %s, got %s", expectedColor, status, color)
		}
	}
}
