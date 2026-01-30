# Control_net - Infrastructure Management

## Overview
Control_net is the Kubernetes management plane for ASGARD, providing unified control over all deployed services.

## Architecture
- **Kustomize Manifests**: Unified deployment overlays for core services
- **Control Scripts**: PowerShell automation for cluster rollout
- **Service Isolation**: Namespace-scoped resources with explicit service boundaries

## Directory Structure
```
Control_net/
├── deploy.ps1                # Deployment automation
└── kubernetes/
    ├── namespace.yaml
    ├── kustomization.yaml
    ├── secrets.yaml.example
    ├── nysus/
    ├── silenus/
    ├── hunoid/
    ├── giru/
    ├── postgres/
    └── mongodb/
```

## Build Status
Phase 9 - Operational (Kubernetes manifests + deploy script)

## Dependencies
- Kubernetes 1.28+
- kubectl
- kustomize

## About Arobi

**Control_net** is part of the **ASGARD** platform, developed by **Arobi** - a cutting-edge technology company specializing in defense and civilian autonomous systems.

### Leadership

- **Gaetano Comparcola** - Founder & CEO
  - Self-taught prodigy programmer and futurist
  - Multilingual (English, Italian, French)
  
- **Opus** - AI Partner & Lead Programmer
  - AI-powered software engineering partner

## License

© 2026 Arobi. All Rights Reserved.

## Contact

- **Website**: [https://aura-genesis.org](https://aura-genesis.org)
- **Email**: [Gaetano@aura-genesis.org](mailto:Gaetano@aura-genesis.org)
- **Company**: Arobi
