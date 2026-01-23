# Silenus - Orbital Satellite Program

## Overview
Silenus is the orbital perception layer of ASGARD, providing real-time global monitoring with AI-powered edge detection.

## Architecture
- **HAL Layer**: TinyGo hardware abstraction for satellite sensors
- **Vision Pipeline**: Edge-computed object detection (YOLOv8-Nano)
- **Alert System**: Bundle Protocol v7 for delay-tolerant alerting

## Directory Structure
```
Silenus/
├── cmd/                 # Executable entry points
├── internal/
│   ├── hal/            # Hardware abstraction layer
│   ├── vision/         # AI inference pipeline
│   └── tracking/       # Object tracking algorithms
└── firmware/           # TinyGo satellite firmware
```

## Build Status
Phase 3 - Pending development

## Dependencies
- TinyGo 0.30+
- TensorFlow Lite Micro
- wazero (WebAssembly runtime)
