# Hunoid - Autonomous Humanoid Unit

## Overview
Hunoid is the physical effector of ASGARD, providing autonomous humanitarian aid and emergency response capabilities.

## Architecture
- **Reflexive Layer**: Local Go control loops (1kHz) for balance and obstacle avoidance
- **Cognitive Layer**: VLA model integration via Nysus for high-level planning
- **Ethical Kernel**: Runtime action validation against ethical constraints

## Directory Structure
```
Hunoid/
├── cmd/                 # Executable entry points
├── internal/
│   ├── control/        # ROS2 Go bridge (rclgo)
│   ├── vla/            # Vision-Language-Action integration
│   └── ethics/         # Ethical pre-processor
└── models/             # Quantized VLA models for edge
```

## Build Status
Phase 4 - Pending development

## Dependencies
- ROS2 Humble
- rclgo (ROS2 Go bindings)
- OpenVLA / RT-2 models
