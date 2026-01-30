# ASGARD WiFi Imaging & Vision Systems Guide

<p align="center">
  <strong>Through-Wall Perception • Object Detection • Sensor Fusion</strong><br>
  <em>Advanced Sensing Capabilities for ASGARD Platform</em>
</p>

---

## Table of Contents

1. [Overview](#overview)
2. [WiFi CSI Imaging](#wifi-csi-imaging)
   - [How It Works](#how-it-works)
   - [Router Setup](#router-setup)
   - [Material Detection](#material-detection)
   - [Triangulation](#triangulation)
3. [Vision Systems](#vision-systems)
   - [Camera Backends](#camera-backends)
   - [Detection Models](#detection-models)
   - [Object Classes](#object-classes)
4. [Sensor Fusion](#sensor-fusion)
   - [Extended Kalman Filter](#extended-kalman-filter)
   - [Supported Sensors](#supported-sensors)
5. [Integration Guide](#integration-guide)
   - [API Endpoints](#api-endpoints)
   - [Voice Commands](#voice-commands)
6. [Use Cases](#use-cases)
7. [Configuration](#configuration)

---

## Overview

ASGARD's sensing capabilities combine:

| System | Purpose | Integration |
|--------|---------|-------------|
| **WiFi Imaging** | Through-wall perception | Pricilla, Valkyrie |
| **Vision** | Object detection | Silenus, Valkyrie |
| **Sensor Fusion** | Multi-sensor positioning | Pricilla, Valkyrie |

---

## WiFi CSI Imaging

### How It Works

WiFi Channel State Information (CSI) imaging uses WiFi signals to perceive objects and materials through walls:

```
                    WiFi Router
                        │
                        ▼ (WiFi Signal)
    ┌───────────────────────────────────────┐
    │            Obstruction               │
    │    ┌─────────────────────────────┐   │
    │    │    Wall (Material Loss)     │   │
    │    │    - Drywall: 3 dB          │   │
    │    │    - Brick: 8 dB            │   │
    │    │    - Concrete: 12 dB        │   │
    │    └─────────────────────────────┘   │
    └───────────────────────────────────────┘
                        │
                        ▼ (Attenuated Signal)
                    Receiver
                        │
                        ▼
              CSI Analysis
              - Path Loss
              - Multipath Spread
              - Phase/Magnitude
```

**Key Measurements:**
- **Path Loss (dB)** - Signal attenuation indicates material and distance
- **Multipath Spread** - Reflection patterns reveal obstacles
- **CSI Magnitudes** - Signal strength across subcarriers
- **CSI Phases** - Phase shifts indicate target movement

### Router Setup

Register WiFi routers to enable triangulation:

```http
POST /api/v1/wifi/routers
Content-Type: application/json

{
  "id": "router-alpha",
  "position": {"x": 0.0, "y": 0.0, "z": 2.5},
  "frequencyGhz": 2.4,
  "txPowerDbm": 20.0
}
```

**Requirements:**
- Minimum 2 routers for 2D triangulation
- 3+ routers recommended for accuracy
- Known positions required
- 2.4 GHz or 5 GHz frequency

### Material Detection

The system automatically classifies wall materials based on excess signal loss:

| Material | Loss (dB) | Signal Penetration | Detection |
|----------|-----------|-------------------|-----------|
| Glass | 2.0 | Excellent | Very High |
| Drywall | 3.0 | Excellent | High |
| Wood | 4.0 | Good | High |
| Composite | 6.0 | Good | Medium |
| Brick | 8.0 | Limited | Medium |
| Concrete | 12.0 | Limited | Low |

**Material Classification Algorithm:**
1. Calculate free-space path loss: `FSPL = 32.4 + 20*log10(freq_GHz)`
2. Compute excess loss: `excess = measured_loss - FSPL`
3. Match to closest material in database

### Triangulation

Multi-router triangulation uses iterative weighted least squares:

```
Algorithm: triangulate2D(samples)

1. Initialize position at weighted centroid of routers
2. For 6 iterations:
   a. Build Jacobian matrix from range residuals
   b. Solve for position update via least squares
   c. Apply update, break if converged (< 0.01m)
3. Return estimated position with confidence

Confidence = 0.6×avgConfidence + 0.2×routerFactor + 0.2×fitScore
```

**Accuracy Factors:**
- Number of routers (more = better)
- Router geometry (spread out is better)
- Signal quality (CSI magnitude/phase stability)
- Material complexity (multiple walls reduce accuracy)

---

## Vision Systems

### Camera Backends

ASGARD supports multiple camera interfaces:

| Backend | Protocol | Use Case |
|---------|----------|----------|
| **RTSP** | H.264/H.265 streaming | IP cameras, drones |
| **MJPEG** | Motion JPEG | Web cameras |
| **GigE Vision** | Industrial standard | High-speed cameras |
| **V4L2/USB** | Direct capture | Local cameras |

### Detection Models

Three vision processors are available:

#### 1. Simple Processor (No ML)
```
- Fast, deterministic
- Color-based fire/smoke detection
- RGB heuristics
- Classes: fire, smoke
```

#### 2. TensorFlow Lite Processor
```
- Edge inference
- SSD MobileNet architecture
- Quantized models (uint8/float32)
- 80 COCO classes
- Model: coco_ssd_mobilenet_v1_1.0_quant
```

#### 3. YOLO Processor
```
- YOLOv8 support
- Remote inference (HTTP/Triton)
- ONNX Runtime / TensorRT
- Non-maximum suppression
- Satellite-specific classes
```

### Object Classes

**COCO Classes (80):**
- person, bicycle, car, motorcycle, airplane
- bus, train, truck, boat, traffic light
- fire hydrant, stop sign, bench, bird, cat
- dog, horse, sheep, cow, elephant
- ... and 60+ more

**Satellite-Specific Classes:**
- fire, smoke, flood
- solar panel, oil tank
- container, crane
- wind turbine, bridge
- aircraft, ship, vehicle

---

## Sensor Fusion

### Extended Kalman Filter

The EKF fuses multiple sensors for robust positioning:

```
State Vector (15 dimensions):
┌─────────────────────────────────────┐
│ Position: x, y, z                   │
│ Velocity: vx, vy, vz                │
│ Attitude: roll, pitch, yaw          │
│ Gyro bias: bx, by, bz               │
│ Accel bias: ax, ay, az              │
└─────────────────────────────────────┘

Update Rate: 100 Hz
```

**Sensor Weights:**

| Sensor | Priority | Weight | Use |
|--------|----------|--------|-----|
| GPS | 1 | 0.95 | Position reference |
| INS | 2 | 0.85 | Attitude/velocity |
| RADAR | 3 | 0.80 | Range/obstacles |
| LIDAR | 4 | 0.80 | 3D mapping |
| WiFi CSI | 4 | 0.75 | Through-wall |
| Visual | 5 | 0.70 | Features |
| IR | 6 | 0.65 | Thermal |

### Supported Sensors

| Sensor Type | Data | Integration |
|-------------|------|-------------|
| **GPS** | Lat/Lon/Alt, satellites, fix | Valkyrie, Pricilla |
| **INS** | Accel, gyro, attitude | Valkyrie |
| **RADAR** | Range, velocity | Valkyrie |
| **LIDAR** | Point cloud, obstacles | Valkyrie |
| **Barometer** | Pressure, altitude | Valkyrie |
| **Pitot** | Airspeed | Valkyrie |
| **WiFi CSI** | Path loss, multipath | Pricilla, Valkyrie |
| **Visual** | Features, optical flow | Silenus, Valkyrie |
| **Infrared** | Thermal signatures | Silenus |

---

## Integration Guide

### API Endpoints

#### WiFi Imaging (Pricilla)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/wifi/routers` | GET | List registered routers |
| `/api/v1/wifi/routers` | POST | Register new router |
| `/api/v1/wifi/imaging` | POST | Process CSI frame |
| `/api/v1/wifi/observations` | GET | Get through-wall observations |
| `/api/v1/wifi/triangulation` | GET | Get triangulation result |

#### Vision (Silenus)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/cameras` | GET | List cameras |
| `/api/v1/vision/detections` | GET | Current detections |
| `/api/v1/vision/alerts` | GET | Visual threat alerts |
| `/api/v1/vision/detect` | POST | Analyze frame |

#### Sensor Fusion (Valkyrie)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/sensors` | GET | Sensor status |
| `/api/v1/state` | GET | Fused state estimate |

### Voice Commands

Use Giru JARVIS for hands-free control:

```
WiFi Imaging:
- "WiFi scan" - Run imaging scan
- "Through wall scan" - Get observations
- "Material analysis" - Analyze walls
- "Triangulation" - Get position

Vision:
- "What do you see?" - Get detections
- "Detect person" - Search for people
- "Visual threats" - Check for fire/smoke
- "Camera status" - System status

Fusion:
- "Sensor fusion status" - EKF status
- "Sensors active" - List active sensors
```

---

## Use Cases

### 1. Search & Rescue
```
Scenario: Building collapse, survivors trapped

WiFi Imaging:
- Deploy routers around perimeter
- Scan for human presence through rubble
- Estimate position for extraction team

Vision:
- Thermal detection of body heat
- Smoke/fire detection for safety

Voice Command: "Giru, scan walls for survivors"
```

### 2. Security Surveillance
```
Scenario: Perimeter monitoring

WiFi Imaging:
- Detect movement through walls
- Track intruders in building
- Material-aware path planning

Vision:
- Person detection at entry points
- Vehicle tracking
- Fire/smoke early warning

Voice Command: "Giru, any intruders detected?"
```

### 3. Autonomous Navigation
```
Scenario: Valkyrie drone mission

Sensor Fusion:
- GPS for outdoor navigation
- WiFi CSI for indoor positioning
- LIDAR for obstacle avoidance
- Vision for target identification

Voice Command: "Giru, Valkyrie position estimate"
```

### 4. Structural Analysis
```
Scenario: Building inspection

WiFi Imaging:
- Map wall materials
- Identify structural elements
- Detect voids and anomalies

Voice Command: "Giru, material analysis of this wall"
```

### 5. Counter-Surveillance
```
Scenario: Detecting hidden threats

WiFi Imaging:
- Detect electronic devices through walls
- Identify concealed rooms
- Track movement patterns

Vision:
- Identify suspicious objects
- Detect concealed cameras

Voice Command: "Giru, what's behind that wall?"
```

---

## Configuration

### WiFi Imaging (Pricilla)

```yaml
# configs/pricilla.yaml
wifi_imaging:
  enabled: true
  min_routers: 2
  max_iterations: 6
  confidence_threshold: 0.5
  material_database:
    drywall: 3.0
    brick: 8.0
    concrete: 12.0
    glass: 2.0
    wood: 4.0
```

### Vision (Silenus)

```yaml
# configs/silenus.yaml
vision:
  processor: tflite  # simple, tflite, yolo
  model_path: models/coco_ssd_mobilenet_v1_1.0_quant/detect.tflite
  confidence_threshold: 0.5
  nms_threshold: 0.4
  frame_rate: 30
```

### Sensor Fusion (Valkyrie)

```yaml
# configs/valkyrie.yaml
fusion:
  update_rate: 100.0  # Hz
  outlier_threshold: 3.0  # Mahalanobis distance
  sensor_weights:
    gps: 0.95
    ins: 0.85
    radar: 0.80
    lidar: 0.80
    wifi: 0.75
    visual: 0.70
```

### Environment Variables

```bash
# Pricilla
PRICILLA_WIFI_ENABLED=true
PRICILLA_URL=http://localhost:8089

# Silenus
SILENUS_VISION_MODEL=tflite
SILENUS_URL=http://localhost:9093

# Valkyrie
VALKYRIE_FUSION_RATE=100
VALKYRIE_URL=http://localhost:8093
```

---

## Performance Specifications

| Capability | Specification |
|------------|---------------|
| WiFi triangulation accuracy | ~1-3 meters |
| WiFi material classification | 6 materials |
| WiFi update rate | Real-time (frame-based) |
| Vision frame rate | Up to 30 FPS |
| Vision detection latency | ~50-200ms |
| Vision classes | 80+ (COCO + custom) |
| EKF update rate | 100 Hz |
| EKF state dimensions | 15 |
| Sensor types supported | 8+ |

---

## About

**WiFi Imaging & Vision Systems** are part of the **ASGARD** platform by **Arobi**.

- **Website**: [aura-genesis.org](https://aura-genesis.org)
- **Contact**: Gaetano@aura-genesis.org

© 2026 Arobi. All Rights Reserved.
