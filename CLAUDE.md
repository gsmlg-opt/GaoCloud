# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**GaoCloud** is a single-cloud management platform that provides Kubernetes cluster management, application deployment, and infrastructure orchestration capabilities. It's built as a Go-based REST API server with WebSocket support for real-time features like pod logs and terminal access.

## Project History - IMPORTANT RENAME NOTICE

This project was originally named **singlecloud** and has been successfully migrated to **GaoCloud** in August 2025.

### Migration Summary
- **Original Name**: singlecloud
- **New Name**: GaoCloud
- **GitHub Organization**: gsmlg-opt
- **Go Module**: Changed from `module gaocloud` to `module github.com/gsmlg-opt/GaoCloud`
- **Binary Name**: Changed from `singlecloud` to `gaocloud`
- **Configuration File**: Changed from `singlecloud.conf` to `gaocloud.conf`
- **Database File**: Changed from `singlecloud.db` to `gaocloud.db`

## Architecture

### Core Components
- **API Server**: Gin-based HTTP server with RESTful APIs and WebSocket endpoints
- **Cluster Management**: Manages multiple Kubernetes clusters via ZKE (Kubernetes Engine)
- **Resource Management**: Comprehensive Kubernetes resource management (Pods, Deployments, Services, etc.)
- **Authentication & Authorization**: CAS-based authentication with JWT tokens and role-based access control
- **Monitoring & Alerting**: Integrated monitoring, metrics collection, and alerting system
- **Application Store**: Helm chart-based application deployment and management
- **Storage Management**: Ceph, iSCSI, NFS, and LVM storage orchestration
- **Service Mesh**: Istio-based service mesh management
- **Workflow Engine**: Custom workflow orchestration for complex deployments

### Key Technologies
- **Backend**: Go 1.13 with Gin framework
- **Database**: BoltDB for configuration storage
- **Kubernetes Client**: Official k8s.io/client-go
- **Helm Integration**: Helm charts for application deployment
- **WebSocket**: Real-time features for logs, shells, and streaming
- **Service Discovery**: Global DNS management
- **Security**: TLS, RBAC, audit logging

## Development Commands

### Build Commands
```bash
# Build the gaocloud binary
make build

# Build Docker image
make build-image

# Push Docker image
make docker

# Clean build artifacts
make clean
make clean-image
```

### Development Setup
```bash
# Generate initial configuration file
./gaocloud -gen

# Run with custom config
./gaocloud -c gaocloud.conf

# Show version
./gaocloud -version
```

### Testing
```bash
# Run integration tests
go test ./test/gaocloud_test.go

# Run specific package tests
go test ./pkg/...

# Run unit tests for specific components
go test ./pkg/authentication/jwt/...
go test ./pkg/authorization/...
```

## Configuration

### Configuration File (gaocloud.conf)
```yaml
server:
  addr: ":80"                    # Server listen address
  tls_cert_file: ""             # TLS certificate file path
  tls_key_file: ""              # TLS private key file path
  dns_addr: ""                  # Global DNS server address
  cas_addr: ""                  # CAS authentication server address
  enable_debug: false           # Enable debug mode

db:
  path: ""                      # Database file path
  port: 6666                    # Database port
  role: master                  # Role: master or slave
  slave_db_addr: ""             # Slave database address for master role

chart:
  path: ""                      # Helm charts local path
  repo: ""                      # Helm charts repository URL

registry:
  ca_cert_path: ""              # Registry CA certificate path
  ca_key_path: ""               # Registry CA private key path
```

### Environment Variables
- `GOPROXY`: Go module proxy (used in Dockerfile)

## Project Structure

```
├── cmd/                    # Application entry points
│   ├── gaocloud/          # Main server binary
│   ├── getkubeconfig/     # Kubeconfig generation tool
│   ├── wstool/           # WebSocket testing tool
│   └── documentgenerate/ # Documentation generator
├── pkg/                   # Core packages
│   ├── handler/          # REST API handlers
│   ├── types/            # Data models and schemas
│   ├── authentication/   # Auth middleware (CAS, JWT)
│   ├── authorization/    # RBAC implementation
│   ├── clusteragent/     # Cluster agent management
│   ├── db/               # Database layer
│   ├── alarm/            # Alerting system
│   ├── auditlog/         # Audit logging
│   ├── globaldns/        # Global DNS management
│   ├── k8seventwatcher/  # Kubernetes event monitoring
│   ├── k8sshell/         # WebShell functionality
│   └── zke/              # ZKE cluster management
├── docs/                 # Documentation
│   ├── design/          # Design documents per feature
│   ├── resources/       # API resource schemas
│   └── websocket/       # WebSocket protocol docs
├── test/                # Integration tests
├── config/              # Configuration structures
└── server/              # HTTP server setup
```

## API Structure

### RESTful Endpoints
Base URL: `/apis/zcloud.cn/v1/`

Resource endpoints follow Kubernetes-style patterns:
- `/clusters/{cluster}/namespaces/{namespace}/{resource-type}`
- `/clusters/{cluster}/namespaces/{namespace}/{resource-type}/{resource-name}`

### WebSocket Endpoints
- `/apis/ws.zcloud.cn/v1/clusters/{cluster}/namespaces/{namespace}/pods/{pod}/containers/{container}/log`
- `/apis/ws.zcloud.cn/v1/clusters/{cluster}/namespaces/{namespace}/tap`
- `/apis/ws.zcloud.cn/v1/clusters/{cluster}/namespaces/{namespace}/workflows/{workflow}/workflowtasks/{task}/log`

## Key Development Patterns

### Adding New Resources
1. Define the type in `pkg/types/`
2. Create manager in `pkg/handler/` following existing patterns
3. Register in `pkg/handler/app.go:registerRestHandler()`
4. Add test cases in `test/` directory

### Authentication Flow
1. CAS server integration for SSO
2. JWT tokens for API authentication
3. RBAC authorization with fine-grained permissions
4. Audit logging for all operations

### Database Operations
- Master-slave replication support
- BoltDB for configuration storage
- In-memory caching for performance
- Event-driven updates via eventbus

## Common Development Tasks

### Adding a New Kubernetes Resource Type
```go
// 1. Define the type in pkg/types/
type NewResource struct {
    // ... fields
}

// 2. Create manager in pkg/handler/
func newNewResourceManager(cm *ClusterManager) *NewResourceManager {
    // ... implementation
}

// 3. Register in pkg/handler/app.go
schemas.MustImport(&Version, types.NewResource{}, newNewResourceManager(cm))
```

### Testing WebSocket Endpoints
```bash
# Use wstool for testing
./wstool -url ws://localhost:80/apis/ws.zcloud.cn/v1/clusters/test/namespaces/default/pods/nginx/containers/nginx/log
```

## Build and Deployment

### Docker Build
- Multi-stage build with Alpine base
- Static binary for minimal image size
- Build arguments for version and build time
- Registry: `gsmlg-opt/gaocloud:{branch}`

### Configuration Generation
```bash
# Generate initial config
./gaocloud -gen

# Edit gaocloud.conf as needed
vim gaocloud.conf
```

## Debugging

### Log Levels
- Debug: Detailed operation logs
- Info: General operation information
- Warn: Warning messages
- Error: Error conditions

### Common Debug Commands
```bash
# Run with debug logging
./gaocloud -c gaocloud.conf

# Check cluster status
curl -H "Authorization: Bearer $TOKEN" http://localhost:80/apis/zcloud.cn/v1/clusters

# Get pod logs via WebSocket
websocat ws://localhost:80/apis/ws.zcloud.cn/v1/clusters/test/namespaces/default/pods/nginx/containers/nginx/log
```