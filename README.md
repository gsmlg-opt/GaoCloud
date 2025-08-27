# GaoCloud

A comprehensive single-cloud management platform for Kubernetes cluster orchestration, application deployment, and infrastructure management.

## 🔄 Project History

This project was originally named **singlecloud** and has been migrated to **GaoCloud** as part of a broader rebranding initiative.

### Migration Timeline
- **Original Name**: singlecloud
- **New Name**: GaoCloud
- **GitHub Organization**: gsmlg-opt
- **Migration Date**: August 2025

### Key Changes During Migration
- **Module Path**: Changed from `module gaocloud` to `module github.com/gsmlg-opt/GaoCloud`
- **Import Paths**: All `github.com/zdnscloud/*` packages moved to `github.com/gsmlg-opt/GaoCloud/*`
- **Binary Name**: Changed from `singlecloud` to `gaocloud`
- **Configuration File**: Changed from `singlecloud.conf` to `gaocloud.conf`
- **Docker Images**: Updated from `zdnscloud/gaocloud` to `gsmlg-opt/GaoCloud`

## 🏗️ Architecture

GaoCloud provides a unified platform for:
- **Kubernetes Cluster Management**: Multi-cluster orchestration via ZKE
- **Application Deployment**: Helm chart-based application lifecycle management
- **Resource Management**: Comprehensive Kubernetes resource control
- **Security**: CAS-based authentication with RBAC authorization
- **Monitoring**: Integrated alerting, metrics, and observability
- **Storage**: Ceph, iSCSI, NFS, and LVM storage orchestration
- **Service Mesh**: Istio-based service mesh management
- **Workflow Engine**: Custom deployment orchestration workflows

## 🚀 Quick Start

### Prerequisites
- Go 1.13+
- Docker (optional)
- Kubernetes cluster access

### Build and Run

```bash
# Build the binary
make build

# Generate configuration
cp gaocloud.conf.example gaocloud.conf

# Run the server
./gaocloud -c gaocloud.conf

# Or with Docker
docker run -p 80:80 gsmlg-opt/GaoCloud:latest
```

### Configuration
```yaml
# gaocloud.conf
server:
  addr: ":80"
  tls_cert_file: ""
  tls_key_file: ""
  dns_addr: ""
  cas_addr: ""
  enable_debug: false

db:
  path: ""
  port: 6666
  role: master
  slave_db_addr: ""

chart:
  path: ""
  repo: ""

registry:
  ca_cert_path: ""
  ca_key_path: ""
```

## 📁 Project Structure

```
GaoCloud/
├── cmd/gaocloud/           # Main server binary
├── application-operator/   # Kubernetes application operator
├── cement/                 # Common utilities and libraries
├── g53/                    # DNS library
├── gok8s/                  # Kubernetes client library
├── goproxy/                # Proxy functionality
├── gorest/                 # REST API framework
├── immense/                # Storage management
├── iniconfig/              # Configuration management
├── kvzoo/                  # Key-value database
├── servicemesh/            # Service mesh management
├── vanguard/               # DNS server
├── zke/                    # Kubernetes engine
├── pkg/                    # Core application packages
├── server/                 # HTTP server setup
├── config/                 # Configuration structures
├── docs/                   # Documentation
└── test/                   # Integration tests
```

## 🛠️ Development

### Build Commands
```bash
# Build binary
make build

# Build Docker image
make build-image

# Push to registry
make docker

# Clean build artifacts
make clean
```

### Testing
```bash
# Run tests
go test ./...

# Run specific package tests
go test ./pkg/...
```

## 📊 Features

- **Multi-cluster Management**: Manage multiple Kubernetes clusters
- **Application Store**: Deploy applications via Helm charts
- **Resource Management**: Full Kubernetes API coverage
- **Security**: CAS authentication, RBAC authorization, audit logging
- **Monitoring**: Real-time metrics, alerts, and observability
- **WebSocket Support**: Real-time logs, shells, and streaming
- **Storage Orchestration**: Ceph, iSCSI, NFS, LVM support
- **Service Mesh**: Istio integration
- **Workflow Engine**: Complex deployment orchestration

## 🔗 API Documentation

- RESTful API: `/apis/zcloud.cn/v1/`
- WebSocket endpoints for real-time features
- Kubernetes-style resource patterns

## 🐳 Docker Support

```bash
# Build image
docker build -t gsmlg-opt/GaoCloud:latest .

# Run container
docker run -d -p 80:80 gsmlg-opt/GaoCloud:latest
```

## 📝 License

This project is part of the GaoCloud ecosystem under the gsmlg-opt organization.