# ASGARD Enhanced Flight Simulation Demo

## Overview

The ASGARD Enhanced Flight Simulation Demo is a comprehensive 5-minute demonstration showcasing the integration of all ASGARD systems in a realistic mission scenario. This demo validates the platform's capabilities in autonomous flight control, precision targeting, robotics operations, security monitoring, and real-time decision-making.

## 🎯 Demo Features

### Multi-System Integration
- **Valkyrie**: Autonomous flight control with sensor fusion
- **Pricilla**: Precision trajectory planning and munitions guidance  
- **Hunoid**: Robotics control with 360° perception and ethics kernel
- **Giru**: Security monitoring and WiFi through-wall imaging
- **Silenus**: Satellite surveillance and orbital tracking
- **Nysus**: Central orchestration and real-time coordination
- **Sat_Net**: Delay-tolerant networking for space communications

### Advanced Capabilities
- **Real-time Physics Calculations**: Blast radius, trajectories, 360° perception
- **Defense Mission Simulation**: Target engagement with munitions
- **WiFi CSI Through-Wall Imaging**: Structural analysis and material detection
- **Ethics Kernel Validation**: Asimov's Three Laws compliance
- **DO-178C DAL-B Compliance**: Aerospace-grade software verification

## 🚀 Quick Start

### Prerequisites
- Node.js and npm installed
- Playwright test framework
- ASGARD project structure intact

### Installation
```bash
cd test/e2e
npm install
```

### Running the Demo

#### Option 1: Using PowerShell Script (Recommended)
```bash
# Run headless demo
./run-flight-simulation-demo.ps1

# Run with visible browser
./run-flight-simulation-demo.ps1 -Headed

# Show help
./run-flight-simulation-demo.ps1 -Help
```

#### Option 2: Using npm Scripts
```bash
# Run the demo
npm run flight:demo

# Run with visible browser
npm run flight:demo:headed
```

#### Option 3: Direct Playwright Command
```bash
npx playwright test flight-simulation-demo.spec.ts --config=playwright-flight-config.ts
```

## 📋 Demo Structure

### Phase 1: System Initialization (0-10%)
- Service health checks for all ASGARD systems
- Valkyrie, Pricilla, Hunoid, Giru, Silenus, Nysus, Sat_Net status verification
- Real-time metrics display

### Phase 2: Mission Planning (10-15%)
- Route calculation: New York → Los Angeles (3,940 km)
- Waypoint planning: NYC → CHI → DEN → PHX → LAX
- Cruise parameters: 10,000m altitude, 250 kts speed

### Phase 3: Flight Execution (15-70%)
- **Takeoff**: 1,000m altitude, 150 kts speed
- **Climb**: 5,000m altitude, 200 kts speed  
- **Cruise**: 10,000m altitude, 250 kts speed
- **Descent**: 2,000m altitude, 180 kts speed
- **Approach**: 500m altitude, 120 kts speed
- **Landing**: 0m altitude, 0 kts speed

### Phase 4: Weather Event (70-75%)
- Severe thunderstorm detection
- Visibility analysis and turbulence assessment
- Route replanning with northern diversion
- Real-time weather avoidance

### Phase 5: Defense Target Engagement (75-85%)
- Hostile target detection and classification
- **Physics Calculations**:
  - Target distance: 12,000m
  - Munitions speed: 850 m/s
  - Time to target: 14.12s
  - Blast radius: 25m
- Ethics kernel validation
- Precision engagement execution

### Phase 6: WiFi Through-Wall Imaging (85-90%)
- **CSI Frame Analysis**:
  - 1,000 frames processed in 45ms
  - Materials detected: Drywall, Concrete, Metal
  - Structural integrity: 94%
- Triangulation with multiple WiFi routers
- Safe entry point identification

### Phase 7: Hunoid Rescue Operations (90-95%)
- **360° Perception Analysis**:
  - Processing time: 68µs
  - Objects detected: 15
  - Risk assessment: Group A (85% survival), Group B (62% survival)
- Ethics kernel prioritization
- Rescue execution with safety validation

### Phase 8: Mission Complete (95-100%)
- Mission summary and statistics
- DO-178C compliance verification
- Ethics kernel validation results
- Performance metrics and safety scores

## 🎥 Video Output

The demo generates high-quality video recordings in the `demo-videos/` directory:
- **Resolution**: 1920x1080
- **Format**: WebM
- **Duration**: ~5 minutes
- **Content**: Complete mission simulation with real-time dashboard

## 📊 Performance Metrics

### Real-Time Processing
- **Sensor Fusion**: 0.574ms
- **360° Perception**: 68µs  
- **Ethics Evaluation**: <1ms
- **Decision Engine**: <10ms

### System Integration
- **AI Decisions**: 847 total
- **Route Replans**: 2
- **Ethics Violations**: 0
- **Safety Score**: 100%

### Compliance Verification
- **DO-178C DAL-B**: VERIFIED
- **Ethics Kernel**: ALL CONSTRAINTS SATISFIED
- **Asimov's Three Laws**: COMPLIANT

## 🔧 Configuration

### Test Configuration
The demo uses `playwright-flight-config.ts` for isolated test execution:
- Single test file matching
- Optimized video recording settings
- Automatic service startup
- 10-minute timeout for comprehensive testing

### Dashboard Features
- **Real-time System Status**: Live health monitoring
- **Flight Visualization**: Interactive map with waypoints
- **AI Decision Log**: Confidence scores and reasoning
- **Alert System**: Color-coded priority notifications
- **Performance Metrics**: Processing times and accuracy

## 🛠️ Troubleshooting

### Common Issues

#### Playwright Import Conflicts
```bash
# Error: Playwright Test did not expect test.describe() to be called here
# Solution: Use isolated configuration
npx playwright test flight-simulation-demo.spec.ts --config=playwright-flight-config.ts
```

#### Service Startup Failures
```bash
# Ensure Websites and Hubs applications are accessible
cd ../../Websites && npm run dev
cd ../../Hubs && npm run dev
```

#### Port Conflicts
```bash
# Check if ports 3000 and 3001 are available
netstat -an | grep :3000
netstat -an | grep :3001
```

#### Missing Dependencies
```bash
# Install Playwright and dependencies
npm install @playwright/test
npx playwright install
```

### Debug Mode
Run with `--headed` flag to see browser output:
```bash
./run-flight-simulation-demo.ps1 -Headed
```

## 📁 File Structure

```
test/e2e/
├── flight-simulation-demo.spec.ts     # Main test file
├── playwright-flight-config.ts        # Isolated configuration
├── run-flight-simulation-demo.ps1     # PowerShell runner script
├── package.json                        # NPM scripts
├── demo-videos/                        # Video output directory
│   └── flight-simulation-demo-*.webm
└── README.md                          # This documentation
```

## 🎯 Use Cases

### Development Testing
- Validate system integration
- Test real-time performance
- Verify physics calculations
- Check ethics kernel compliance

### Demonstration
- Investor presentations
- Technical showcases
- Compliance audits
- Training materials

### Performance Benchmarking
- Measure processing times
- Validate latency requirements
- Test system scalability
- Monitor resource usage

## 📞 Support

For issues or questions:
1. Check the troubleshooting section above
2. Run with `--headed` flag for visual debugging
3. Review generated video output for visual validation
4. Check console logs for detailed error information

## 📄 License

This demo is part of the ASGARD project and follows the same licensing terms.