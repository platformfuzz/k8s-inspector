package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/platformfuzz/k8s-inspector/internal/config"
	"github.com/platformfuzz/k8s-inspector/internal/dashboard"
	"github.com/platformfuzz/k8s-inspector/internal/inspector"
	"github.com/platformfuzz/k8s-inspector/internal/k8s"
)

// Server represents the HTTP server
type Server struct {
	config    *config.Config
	router    *gin.Engine
	handlers  *Handlers
	dashboard *dashboard.Dashboard
	inspector *inspector.Inspector
	httpServer *http.Server
}

// New creates a new Server instance
func New(cfg *config.Config, k8sClient *k8s.Client) (*Server, error) {
	// Create inspector and dashboard
	insp := inspector.NewInspector(k8sClient)
	dash := dashboard.NewDashboard(insp)

	// Create handlers
	handlers := NewHandlers(cfg, dash, insp)

	// Set Gin mode
	if cfg.AppVersion == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	server := &Server{
		config:    cfg,
		router:    router,
		handlers:  handlers,
		dashboard: dash,
		inspector: insp,
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Load HTML templates (only if they exist)
	templatePattern := "web/templates/*"
	if matches, err := filepath.Glob(templatePattern); err == nil && len(matches) > 0 {
		// Sort to ensure consistent loading order (base.html first, then others)
		sort.Strings(matches)
		log.Printf("Loading templates: %v", matches)
		// Load templates - Gin uses base filename as template name
		s.router.LoadHTMLGlob(templatePattern)
		log.Printf("Templates loaded successfully")
	} else {
		log.Printf("No templates found or error: %v", err)
	}

	// Serve static files (only if directory exists)
	if _, err := os.Stat("./web/static"); err == nil {
		s.router.Static("/static", "./web/static")
	}

	// Favicon
	s.router.GET("/favicon.svg", func(c *gin.Context) {
		c.File("./web/static/favicon.svg")
	})
	s.router.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./web/static/favicon.svg")
	})

	// Index redirect
	s.router.GET("/", s.handlers.IndexHandler)

	// Dashboard page
	s.router.GET("/dashboard", s.handlers.DashboardHandler)

	// Inspector page
	s.router.GET("/inspector", s.handlers.InspectorHandler)

	// API routes
	api := s.router.Group("/api/v1")
	{
		api.GET("/health", s.handlers.HealthHandler)
		api.GET("/status", s.handlers.StatusHandler)
		api.GET("/pod/info", s.handlers.PodInfoHandler)
		api.GET("/pod/logs", s.handlers.PodLogsHandler)
		api.GET("/pod/metrics", s.handlers.PodMetricsHandler)
		api.GET("/pod/env", s.handlers.PodEnvHandler)
		api.GET("/pod/secrets", s.handlers.PodSecretsHandler)
		api.GET("/pod/events", s.handlers.PodEventsHandler)
	}

	// WebSocket endpoint
	s.router.GET("/ws", WebSocketHandler(s.dashboard, s.config.DashboardRefreshInterval))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting %s v%s on %s", s.config.AppTitle, s.config.AppVersion, addr)
	log.Printf("Dashboard: http://localhost%s/dashboard", addr)
	log.Printf("Inspector: http://localhost%s/inspector", addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// GetRouter returns the router for testing purposes
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
