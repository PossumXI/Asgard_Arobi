# Silenus - Orbital Satellite Program

## Overview
Silenus is the orbital perception layer of ASGARD, providing real-time global monitoring with AI-powered edge detection. It functions as the "eyes in the sky" for the ASGARD platform.

## Architecture
- **HAL Layer**: Hardware abstraction for satellite sensors (camera, GPS, power)
- **Vision Pipeline**: Edge-computed object detection (Simple, TFLite, YOLO backends)
- **Alert System**: Bundle Protocol v7 for delay-tolerant alerting via Sat_Net
- **Telemetry**: Real-time satellite health and position reporting

## Directory Structure
```
Silenus/
├── README.md                    # This file
cmd/silenus/
├── main.go                      # Main entry point (~500 lines)
internal/orbital/
├── hal/                         # Hardware abstraction layer
│   ├── interfaces.go            # HAL interfaces
│   ├── camera.go                # Camera controller (MJPEG, RTSP, GigE, V4L2)
│   ├── gps_controller.go        # GPS/position tracking
│   ├── orbital_position.go      # Real orbital position via N2YO API
│   ├── power_controller.go      # Remote power management
│   └── power.go                 # Power system abstractions
├── tracking/
│   └── tracker.go               # Object tracking with alert generation
└── vision/
    ├── processor.go             # Vision processor interface
    ├── simple_processor.go      # Basic color-based detection
    ├── tflite_processor.go      # TensorFlow Lite inference
    ├── tflite_stub.go           # TFLite stub for builds without CGO
    └── yolo_processor.go        # YOLO model support
```

## Features Implemented

### Hardware Abstraction Layer
- **Camera Controller**: Supports MJPEG streams, RTSP, GigE Vision, V4L2/USB cameras
- **GPS Controller**: Real-time position tracking with N2YO satellite API integration
- **Power Controller**: Battery monitoring, solar panel power, eclipse detection

### Vision Pipeline
- **Simple Processor**: Fast color-based anomaly detection
- **TFLite Processor**: TensorFlow Lite Micro for edge inference
- **Alert Criteria**: Configurable confidence thresholds and target classes (fire, smoke, aircraft, ship)

### Networking
- **Sat_Net Integration**: Delay-tolerant networking via Bundle Protocol v7
- **Alert Forwarding**: Automatic alert bundle creation and transmission
- **Telemetry Loop**: 10-second interval health/position reporting

### Observability
- **Prometheus Metrics**: Exposed on configurable port
- **Health Endpoint**: `/healthz` for liveness probes
- **OpenTelemetry Tracing**: Distributed tracing support

## Build Status
**Phase: OPERATIONAL** (Core functionality complete)

## Usage

```powershell
# Run Silenus satellite node
$env:CAMERA_BACKEND = "mjpeg"
$env:CAMERA_ADDRESS = "192.168.1.100"
$env:SATNET_GATEWAY_ADDR = "192.168.1.1:4556"
$env:N2YO_API_KEY = "your-api-key"

go run ./cmd/silenus/main.go -id sat001 -vision-backend simple
```

### Command-Line Flags
| Flag | Default | Description |
|------|---------|-------------|
| `-id` | sat001 | Satellite identifier |
| `-model` | models/yolov8n.onnx | Vision model path |
| `-vision-backend` | simple | Vision backend (simple, tflite) |
| `-alert-min-confidence` | 0.85 | Alert confidence threshold |
| `-alert-eid` | dtn://earth/nysus/alerts | Alert destination EID |
| `-metrics-addr` | :9093 | Metrics server address |

### Environment Variables
| Variable | Description |
|----------|-------------|
| `CAMERA_BACKEND` | Camera type: mjpeg, rtsp, gige, v4l2, usb |
| `CAMERA_ADDRESS` | Camera network address |
| `CAMERA_PORT` | Camera port (default: 80) |
| `SATNET_GATEWAY_ADDR` | Sat_Net gateway address |
| `N2YO_API_KEY` | N2YO satellite tracking API key |
| `DTN_STORAGE_BACKEND` | Bundle storage: memory, postgres |

## Dependencies
- Go 1.24+
- TensorFlow Lite (optional, for tflite backend)
- N2YO API key (for real orbital position)
- Sat_Net gateway for bundle forwarding

## Integration Points
- **Nysus**: Receives alerts and telemetry via Sat_Net bundles
- **Pricilla**: Can use satellite imagery for targeting
- **Giru**: Provides threat zone data to satellites

## About Arobi

**Silenus** is part of the **ASGARD** platform, developed by **Arobi** - a cutting-edge technology company specializing in defense and civilian autonomous systems.

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
