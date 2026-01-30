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
- Go 1.24+
- dtn7-go (base DTN library)
- ONNX Runtime (for RL inference)

## About Arobi

**Sat_Net** is part of the **ASGARD** platform, developed by **Arobi** - a cutting-edge technology company specializing in defense and civilian autonomous systems.

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
