package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/platformfuzz/k8s-inspector/internal/config"
	"github.com/platformfuzz/k8s-inspector/internal/k8s"
	"github.com/platformfuzz/k8s-inspector/internal/server"
)

func setupTestServer() *server.Server {
	cfg := config.Load()
	k8sClient, _ := k8s.NewClient(cfg)
	srv, _ := server.New(cfg, k8sClient)
	return srv
}

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := setupTestServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)

	srv.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check response body contains "status"
	if w.Body.String() == "" {
		t.Error("Response body should not be empty")
	}
}

func TestStatusHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := setupTestServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/status", nil)

	srv.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestIndexHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := setupTestServer()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	srv.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Expected status code %d, got %d", http.StatusMovedPermanently, w.Code)
	}
}
