# k8s-inspector

A Kubernetes inspection and debugging tool built in Go. Provides real-time pod information, metrics visualization, log viewing capabilities, and works both standalone (Docker) and in Kubernetes clusters.

## Features

- **Interactive Dashboard**: Real-time metrics display with WebSocket support for live updates
- **Pod Inspector**: Comprehensive pod inspection with:
  - Complete pod metadata and status
  - Container logs with real-time streaming, filtering, and search
  - Environment variables browser
  - Secrets viewer (with show/hide toggle)
  - Network information (IPs, ports, service endpoints)
  - Resource metrics (CPU, memory usage)
  - Recent Kubernetes events
- **REST API**: Full API for programmatic access
- **WebSocket Support**: Real-time updates for dashboard
- **Dual Mode**: Works both standalone (Docker) and in-cluster (Kubernetes)
- **Auto-Detection**: Automatically detects in-cluster vs standalone mode

## Quick Start

### Using Docker

```bash
docker run -p 8080:8080 ghcr.io/platformfuzz/k8s-inspector:latest
```

Then open <http://localhost:8080/dashboard> in your browser.

### Using Go

```bash
# Clone the repository
git clone https://github.com/platformfuzz/k8s-inspector.git
cd k8s-inspector

# Full workflow: deps, test, build, and run
make all

# Or run directly
go run ./cmd/server
```

### In Kubernetes

**Basic Deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-inspector
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: k8s-inspector
  template:
    metadata:
      labels:
        app: k8s-inspector
    spec:
      containers:
      - name: k8s-inspector
        image: ghcr.io/platformfuzz/k8s-inspector:latest
        ports:
        - containerPort: 8080
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
---
apiVersion: v1
kind: Service
metadata:
  name: k8s-inspector
  namespace: default
spec:
  selector:
    app: k8s-inspector
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

**With RBAC:**

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-inspector
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-inspector
  namespace: default
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "pods/status", "events", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: k8s-inspector
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: k8s-inspector
subjects:
- kind: ServiceAccount
  name: k8s-inspector
  namespace: default
```

Then add `serviceAccountName: k8s-inspector` to the Deployment spec.

## Usage

### Dashboard

Navigate to `http://localhost:8080/dashboard` for a real-time overview of your pod's status and metrics.

**Features:**

- **Real-time Updates**: WebSocket connection provides live updates every 5 seconds (configurable)
- **Status Indicators**: Color-coded status indicators for pod and container health
  - Green: Healthy
  - Yellow: Warning
  - Red: Error
- **Pod Information**: Name, namespace, phase, IP, node name
- **Container Status**: Status of all containers in the pod
- **Metrics**: CPU, memory, and pod count (if metrics-server is available)

The status badge shows overall pod health:

- **healthy**: All containers are running and ready
- **warning**: Some containers are not ready or pod is pending
- **error**: Pod has failed or containers have errors

### Inspector

Navigate to `http://localhost:8080/inspector` for detailed pod inspection.

**Tabs:**

- **Pod Info**: Pod metadata, labels, annotations, network information, container details, pod conditions
- **Logs**: Container logs with container selection, tail lines configuration, real-time refresh, and syntax highlighting
- **Environment**: Browse all environment variables (direct, ConfigMap, and Secret references)
- **Secrets**: View secret values with show/hide toggle (hidden by default for security)
- **Events**: Recent Kubernetes events with type, reason, message, count, and timestamps
- **Metrics**: Resource usage metrics (CPU, memory, pod count)

### API Endpoints

All functionality is available via REST API. Base URL: `/api/v1`

**Endpoints:**

- `GET /api/v1/health` - Health check
- `GET /api/v1/status` - Overall pod status and indicators
- `GET /api/v1/pod/info` - Comprehensive pod information
- `GET /api/v1/pod/logs?container=<name>&tail=<lines>` - Container logs (container and tail are optional)
- `GET /api/v1/pod/metrics` - Resource usage metrics
- `GET /api/v1/pod/env` - Environment variables
- `GET /api/v1/pod/secrets` - Secret values (hidden by default)
- `GET /api/v1/pod/events` - Recent Kubernetes events
- `GET /api/v1/diagnostic` - Diagnostic information for troubleshooting

**Examples:**

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Get pod information
curl http://localhost:8080/api/v1/pod/info | jq

# Get logs (last 50 lines)
curl "http://localhost:8080/api/v1/pod/logs?tail=50" | jq

# Get environment variables
curl http://localhost:8080/api/v1/pod/env | jq

# Get secrets (hidden by default)
curl http://localhost:8080/api/v1/pod/secrets | jq

# Get events
curl http://localhost:8080/api/v1/pod/events | jq

# Overall status
curl http://localhost:8080/api/v1/status | jq

# Diagnostic information
curl http://localhost:8080/api/v1/diagnostic | jq
```

**Response Format:**

All endpoints return JSON. Error responses include an `error` field:

```json
{
  "error": "Error message describing what went wrong"
}
```

**Status Codes:**

- `200 OK`: Success
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

### WebSocket

The dashboard uses WebSocket for real-time updates. Connect to `ws://localhost:8080/ws` to receive JSON updates.

**Message Format:**

```json
{
  "podInfo": {
    "name": "my-pod",
    "namespace": "default",
    "phase": "Running"
  },
  "metrics": {
    "cpu": "100m",
    "memory": "128Mi",
    "pods": 5
  },
  "status": {
    "overall": "healthy",
    "message": "Pod is running normally",
    "indicators": {
      "pod": "healthy",
      "container-1": "healthy"
    }
  },
  "refreshInterval": 5
}
```

## Configuration

All configuration is done via environment variables:

**Server Configuration:**

- `APP_TITLE` - Application title (default: "k8s-inspector")
- `APP_VERSION` - Version string (default: "1.0.0")
- `PORT` - Server port (default: 8080)

**Dashboard Configuration:**

- `DASHBOARD_REFRESH_INTERVAL` - Refresh interval in seconds (default: 5)
- `THEME` - UI theme: "light" or "dark" (default: "light")

**Feature Flags:**

- `ENABLE_METRICS` - Enable metrics collection (default: true)
- `ENABLE_LOGS` - Enable log streaming (default: true)
- `LOG_TAIL_LINES` - Default log tail lines (default: 100)

**Security:**

- `SHOW_SECRET_VALUES` - Show secret values by default (default: false)

**Example:**

```bash
export PORT=8080
export DASHBOARD_REFRESH_INTERVAL=10
export THEME="dark"
export SHOW_SECRET_VALUES=false
```

### Kubernetes Integration

When running in a Kubernetes cluster:

- The application automatically detects it's in-cluster
- Uses the service account for API access
- Reads pod information from the downward API
- Requires appropriate RBAC permissions

### Standalone Mode

When running outside Kubernetes:

- Uses hostname as pod name
- Shows basic system information
- Limited Kubernetes features
- Still provides inspection interface

## Troubleshooting

**Dashboard not updating:**

- Check WebSocket connection status (shown at bottom of dashboard)
- Verify `DASHBOARD_REFRESH_INTERVAL` is set correctly
- Check browser console for WebSocket errors

**Logs not loading:**

- Verify container name is correct
- Check that the container is running
- Ensure RBAC permissions allow log access

**Secrets not showing:**

- Secrets are hidden by default for security
- Toggle "Show Secret Values" checkbox in Inspector
- Or set `SHOW_SECRET_VALUES=true` environment variable

**Metrics showing "N/A":**

- Metrics require metrics-server to be installed in the cluster
- In standalone mode, metrics are not available
- This is expected behavior if metrics-server is not available

## Project Structure

```plaintext
k8s-inspector/
├── cmd/server/          # Application entry point
├── internal/            # Internal packages
│   ├── config/         # Configuration management
│   ├── dashboard/      # Dashboard data provider
│   ├── inspector/      # Inspection utilities
│   ├── k8s/            # Kubernetes client wrapper
│   └── server/         # HTTP server and handlers
├── web/                 # Web assets
│   ├── static/         # CSS and JavaScript
│   └── templates/      # HTML templates
└── tests/              # Test files
```

## Technology Stack

- **Go 1.23+**: Programming language
- **Gin**: Lightweight HTTP framework
- **k8s.io/client-go**: Official Kubernetes client library
- **gorilla/websocket**: WebSocket support
- **Alpine Linux**: Minimal container base image

## Development

### Using Make

```bash
# Full workflow: deps, test, build, and run
make all

# Fast workflow: deps, build, and run (skip tests)
make all-fast

# Individual tasks
make test           # Run tests
make build          # Build the application
make run            # Run locally
make docker-build   # Build Docker image
make docker-run     # Run Docker container
make clean          # Clean build artifacts
make help           # Show all targets
```

## Security Considerations

- **Secrets**: Hidden by default. Only show when necessary.
- **RBAC**: Ensure proper RBAC permissions are configured
- **Network**: Consider network policies to restrict access
- **TLS**: Use TLS/HTTPS in production environments

## Documentation

For advanced deployment scenarios (Ingress, sidecar, Helm charts, Docker Compose), see the repository examples.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
