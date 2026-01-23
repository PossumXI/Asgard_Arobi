# Sat_Net - Interstellar Network Layer

## Overview
Sat_Net provides delay-tolerant networking using Bundle Protocol v7, enabling communication from LEO to Mars.

## Architecture
- **DTN Core**: Bundle Protocol v7 implementation (based on dtn7-go)
- **RL Router**: Deep Q-Network agent for adaptive path selection
- **Energy Manager**: Battery-aware routing to preserve node lifetime

## Directory Structure
```
Sat_Net/
├── cmd/                 # Router and node executables
├── internal/
│   ├── dtn/            # Bundle Protocol implementation
│   ├── router/         # RL-based routing engine
│   └── convergence/    # Physical layer adapters
└── models/             # Trained RL models (ONNX)
```

## Build Status
Phase 3 - Pending development

## Dependencies
- Go 1.21+
- dtn7-go (base DTN library)
- ONNX Runtime (for RL inference)
