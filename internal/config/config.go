package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	AppTitle              string
	AppVersion            string
	Port                  int
	DashboardRefreshInterval int
	EnableMetrics         bool
	EnableLogs            bool
	LogTailLines          int
	ShowSecretValues      bool
	Theme                 string
	InCluster             bool
	PodName               string
	Namespace             string
}

// Load creates a new Config instance with values from environment variables or defaults
func Load() *Config {
	cfg := &Config{
		AppTitle:              getEnv("APP_TITLE", "k8s-inspector"),
		AppVersion:            getEnv("APP_VERSION", "1.0.0"),
		Port:                  getEnvAsInt("PORT", 8080),
		DashboardRefreshInterval: getEnvAsInt("DASHBOARD_REFRESH_INTERVAL", 5),
		EnableMetrics:         getEnvAsBool("ENABLE_METRICS", true),
		EnableLogs:            getEnvAsBool("ENABLE_LOGS", true),
		LogTailLines:          getEnvAsInt("LOG_TAIL_LINES", 100),
		ShowSecretValues:      getEnvAsBool("SHOW_SECRET_VALUES", false),
		Theme:                 getEnv("THEME", "light"),
		PodName:               getEnv("POD_NAME", ""),
		Namespace:             getEnv("POD_NAMESPACE", "default"),
	}

	// Auto-detect if running in cluster
	cfg.InCluster = detectInCluster()

	return cfg
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsBool retrieves an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// detectInCluster checks if the application is running inside a Kubernetes cluster
func detectInCluster() bool {
	// Check for service account token (in-cluster indicator)
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return true
	}
	// Check for service account namespace file
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return true
	}
	return false
}
