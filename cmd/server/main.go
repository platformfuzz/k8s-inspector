package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/platformfuzz/k8s-inspector/internal/config"
	"github.com/platformfuzz/k8s-inspector/internal/k8s"
	"github.com/platformfuzz/k8s-inspector/internal/server"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create Kubernetes client
	k8sClient, err := k8s.NewClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to create Kubernetes client: %v", err)
		log.Println("Running in standalone mode")
	}

	// Create and start server
	srv, err := server.New(cfg, k8sClient)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down server...")

	// Graceful shutdown would go here if we had context support
	// For now, just exit
	os.Exit(0)
}
