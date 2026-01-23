# Control_net - Infrastructure Management

## Overview
Control_net is the Kubernetes management plane for ASGARD, providing unified control over all deployed services.

## Architecture
- **Operators**: Custom Kubernetes operators for each subsystem
- **Helm Charts**: Declarative deployment configurations
- **Controllable Interface**: Standardized Start/Stop/Status API

## Directory Structure
```
Control_net/
├── cmd/                 # Control operator
├── charts/             # Helm charts
│   ├── nysus/
│   ├── silenus/
│   ├── giru/
│   └── data/
├── operators/          # Kubernetes operators
└── manifests/          # Raw K8s manifests
```

## Build Status
Phase 2 - In progress (operator framework)

## Dependencies
- Kubernetes 1.28+
- Helm 3.x
- operator-sdk
