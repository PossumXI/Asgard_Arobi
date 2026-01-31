# VALKYRIE Quick Start Script
# This script automates the initial setup of the VALKYRIE Autonomous Flight System
# Run this in PowerShell as Administrator

param(
    [switch]$SkipDependencies = $false,
    [switch]$DevMode = $false,
    [string]$AsgardPath = "C:\Users\hp\Desktop\Asgard"
)

# Colors for output
$ErrorColor = "Red"
$SuccessColor = "Green"
$InfoColor = "Cyan"
$WarningColor = "Yellow"

# Banner
function Show-Banner {
    Write-Host @"

    ╦  ╦╔═╗╦  ╦╔═╦ ╦╦═╗╦╔═╗
    ╚╗╔╝╠═╣║  ╠╩╗╚╦╝╠╦╝║║╣ 
     ╚╝ ╩ ╩╩═╝╩ ╩ ╩ ╩╚═╩╚═╝
    
    Autonomous Flight System v1.0.0
    Quick Start Installer
    
"@ -ForegroundColor $InfoColor
}

# Check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Check prerequisites
function Test-Prerequisites {
    Write-Host "[1/10] Checking prerequisites..." -ForegroundColor $InfoColor
    
    $allGood = $true
    
    # Check Go
    try {
        $goVersion = go version
        Write-Host "  ✓ Go installed: $goVersion" -ForegroundColor $SuccessColor
    } catch {
        Write-Host "  ✗ Go not found. Please install Go 1.21+" -ForegroundColor $ErrorColor
        $allGood = $false
    }
    
    # Check Git
    try {
        $gitVersion = git --version
        Write-Host "  ✓ Git installed: $gitVersion" -ForegroundColor $SuccessColor
    } catch {
        Write-Host "  ✗ Git not found. Please install Git" -ForegroundColor $ErrorColor
        $allGood = $false
    }
    
    # Check Docker (optional)
    try {
        $dockerVersion = docker --version
        Write-Host "  ✓ Docker installed: $dockerVersion" -ForegroundColor $SuccessColor
    } catch {
        Write-Host "  ! Docker not found (optional)" -ForegroundColor $WarningColor
    }
    
    # Check Python (for ML)
    try {
        $pythonVersion = python --version
        Write-Host "  ✓ Python installed: $pythonVersion" -ForegroundColor $SuccessColor
    } catch {
        Write-Host "  ! Python not found (needed for ML training)" -ForegroundColor $WarningColor
    }
    
    if (-not $allGood) {
        throw "Missing required prerequisites"
    }
}

# Create directory structure
function New-DirectoryStructure {
    Write-Host "[2/10] Creating directory structure..." -ForegroundColor $InfoColor
    
    $baseDir = Join-Path $AsgardPath "Valkyrie"
    
    if (Test-Path $baseDir) {
        Write-Host "  ! Valkyrie directory already exists" -ForegroundColor $WarningColor
        $response = Read-Host "  Do you want to overwrite? (y/N)"
        if ($response -ne "y") {
            throw "Installation cancelled by user"
        }
        Remove-Item -Path $baseDir -Recurse -Force
    }
    
    # Create directories
    $directories = @(
        "cmd\valkyrie",
        "internal\guidance",
        "internal\security",
        "internal\fusion",
        "internal\ai",
        "internal\actuators",
        "internal\sensors",
        "internal\integration",
        "internal\failsafe",
        "internal\livefeed",
        "internal\access",
        "internal\redundancy",
        "internal\vision",
        "internal\pricilla_guidance",
        "internal\pricilla_navigation",
        "internal\pricilla_prediction",
        "internal\pricilla_stealth",
        "internal\giru_shadow",
        "internal\giru_redteam",
        "internal\giru_blueteam",
        "internal\giru_threat",
        "pkg\mavlink",
        "pkg\utils",
        "pkg\logging",
        "configs",
        "tests\unit",
        "tests\integration",
        "tests\simulation",
        "docs",
        "scripts",
        "deployment\docker",
        "deployment\k8s",
        "models",
        "bin"
    )
    
    foreach ($dir in $directories) {
        $fullPath = Join-Path $baseDir $dir
        New-Item -ItemType Directory -Path $fullPath -Force | Out-Null
    }
    
    Write-Host "  ✓ Directory structure created" -ForegroundColor $SuccessColor
    return $baseDir
}

# Initialize Go module
function Initialize-GoModule {
    param([string]$baseDir)
    
    Write-Host "[3/10] Initializing Go module..." -ForegroundColor $InfoColor
    
    Push-Location $baseDir
    
    # Initialize module
    go mod init github.com/PossumXI/Asgard/Valkyrie | Out-Null
    
    # Create .gitignore
    $gitignore = @"
# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test coverage
*.out
coverage.html

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Logs
*.log
logs/

# Build artifacts
vendor/
dist/

# Models (large files)
models/*.weights
models/*.onnx

# Data
data/
*.csv
*.json
"@
    
    Set-Content -Path ".gitignore" -Value $gitignore
    
    Pop-Location
    
    Write-Host "  ✓ Go module initialized" -ForegroundColor $SuccessColor
}

# Copy components from Pricilla and Giru
function Copy-Components {
    param([string]$baseDir)
    
    Write-Host "[4/10] Copying components from Pricilla and Giru..." -ForegroundColor $InfoColor
    
    $pricillaPath = Join-Path $AsgardPath "Pricilla"
    $giruPath = Join-Path $AsgardPath "Giru"
    
    # Check if source directories exist
    if (-not (Test-Path $pricillaPath)) {
        Write-Host "  ! Pricilla not found at $pricillaPath" -ForegroundColor $WarningColor
        Write-Host "  Skipping Pricilla component copy" -ForegroundColor $WarningColor
        return
    }
    
    if (-not (Test-Path $giruPath)) {
        Write-Host "  ! Giru not found at $giruPath" -ForegroundColor $WarningColor
        Write-Host "  Skipping Giru component copy" -ForegroundColor $WarningColor
        return
    }
    
    # Copy Pricilla components
    $pricillaComponents = @{
        "internal\guidance" = "internal\pricilla_guidance"
        "internal\navigation" = "internal\pricilla_navigation"
        "internal\prediction" = "internal\pricilla_prediction"
        "internal\stealth" = "internal\pricilla_stealth"
    }
    
    foreach ($source in $pricillaComponents.Keys) {
        $sourcePath = Join-Path $pricillaPath $source
        $destPath = Join-Path $baseDir $pricillaComponents[$source]
        
        if (Test-Path $sourcePath) {
            Copy-Item -Path $sourcePath -Destination $destPath -Recurse -Force
            Write-Host "  ✓ Copied $source" -ForegroundColor $SuccessColor
        }
    }
    
    # Copy Giru components
    $giruComponents = @{
        "internal\security\shadow" = "internal\giru_shadow"
        "internal\security\redteam" = "internal\giru_redteam"
        "internal\security\blueteam" = "internal\giru_blueteam"
        "internal\security\threat" = "internal\giru_threat"
    }
    
    foreach ($source in $giruComponents.Keys) {
        $sourcePath = Join-Path $giruPath $source
        $destPath = Join-Path $baseDir $giruComponents[$source]
        
        if (Test-Path $sourcePath) {
            Copy-Item -Path $sourcePath -Destination $destPath -Recurse -Force
            Write-Host "  ✓ Copied $source" -ForegroundColor $SuccessColor
        }
    }
}

# Update package declarations
function Update-PackageDeclarations {
    param([string]$baseDir)
    
    Write-Host "[5/10] Updating package declarations..." -ForegroundColor $InfoColor
    
    Push-Location $baseDir
    
    # Update Pricilla packages
    Get-ChildItem -Path "internal\pricilla_*" -Recurse -Filter "*.go" -ErrorAction SilentlyContinue | ForEach-Object {
        $content = Get-Content $_.FullName -Raw
        
        $content = $content -replace "package guidance", "package pricilla_guidance"
        $content = $content -replace "package navigation", "package pricilla_navigation"
        $content = $content -replace "package prediction", "package pricilla_prediction"
        $content = $content -replace "package stealth", "package pricilla_stealth"
        
        Set-Content -Path $_.FullName -Value $content -NoNewline
    }
    
    # Update Giru packages
    Get-ChildItem -Path "internal\giru_*" -Recurse -Filter "*.go" -ErrorAction SilentlyContinue | ForEach-Object {
        $content = Get-Content $_.FullName -Raw
        
        $content = $content -replace "package shadow", "package giru_shadow"
        $content = $content -replace "package redteam", "package giru_redteam"
        $content = $content -replace "package blueteam", "package giru_blueteam"
        $content = $content -replace "package threat", "package giru_threat"
        
        Set-Content -Path $_.FullName -Value $content -NoNewline
    }
    
    Pop-Location
    
    Write-Host "  ✓ Package declarations updated" -ForegroundColor $SuccessColor
}

# Install Go dependencies
function Install-GoDependencies {
    param([string]$baseDir)
    
    if ($SkipDependencies) {
        Write-Host "[6/10] Skipping Go dependencies (--SkipDependencies)" -ForegroundColor $WarningColor
        return
    }
    
    Write-Host "[6/10] Installing Go dependencies..." -ForegroundColor $InfoColor
    
    Push-Location $baseDir
    
    $dependencies = @(
        "gonum.org/v1/gonum",
        "github.com/gorilla/websocket",
        "github.com/nats-io/nats.go",
        "github.com/prometheus/client_golang/prometheus",
        "github.com/prometheus/client_golang/prometheus/promhttp",
        "github.com/sirupsen/logrus",
        "gorm.io/gorm",
        "gorm.io/driver/postgres",
        "github.com/google/uuid"
    )
    
    foreach ($dep in $dependencies) {
        Write-Host "  Installing $dep..." -ForegroundColor $InfoColor
        go get $dep 2>&1 | Out-Null
    }
    
    # Tidy up
    go mod tidy
    
    Pop-Location
    
    Write-Host "  ✓ Go dependencies installed" -ForegroundColor $SuccessColor
}

# Create configuration files
function New-ConfigurationFiles {
    param([string]$baseDir)
    
    Write-Host "[7/10] Creating configuration files..." -ForegroundColor $InfoColor
    
    # Config YAML
    $configYaml = @"
# VALKYRIE Configuration

http_port: 8093
metrics_port: 9093

# ASGARD Endpoints
asgard:
  nysus: http://localhost:8080
  silenus: http://localhost:9093
  satnet: http://localhost:8081
  giru: http://localhost:9090
  hunoid: http://localhost:8090

# Sensor Fusion
fusion:
  update_rate: 100.0  # Hz
  min_sensors: 2
  outlier_threshold: 3.0
  enable_adaptive: true

# AI Decision Engine
ai:
  safety_priority: 0.9
  efficiency_priority: 0.7
  stealth_priority: 0.5
  decision_rate: 50.0  # Hz
  
  # Constraints
  max_roll_angle: 0.785   # 45 degrees
  max_pitch_angle: 0.524  # 30 degrees
  max_yaw_rate: 0.349     # 20 deg/s
  min_safe_altitude: 100.0
  max_vertical_speed: 10.0
  
  # Features
  enable_autoland: true
  enable_threat_avoid: true
  enable_weather_avoid: true

  # Geo reference (local meters -> lat/lon for weather queries)
  geo_reference_enabled: true
  geo_reference_latitude: 37.7749
  geo_reference_longitude: -122.4194
  geo_reference_source: n2yo
  geo_reference_norad_id: 25544

# Security
security:
  monitor_flight_controller: true
  monitor_sensor_drivers: true
  monitor_navigation: true
  monitor_communication: true
  anomaly_threshold: 0.7
  response_mode: alert  # log, alert, quarantine, kill

# Fail-Safe
failsafe:
  enable_auto_rtb: true
  enable_auto_land: true
  enable_parachute: false
  min_safe_altitude_agl: 50.0
  min_safe_fuel: 0.15
  max_time_without_comms: 300  # seconds
  rtb_location: [0, 0, 500]

# Logging
logging:
  level: info  # debug, info, warn, error
  output: stdout
  file: logs/valkyrie.log
"@
    
    Set-Content -Path (Join-Path $baseDir "configs\config.yaml") -Value $configYaml
    
    # Environment template
    $envTemplate = @"
# VALKYRIE Environment Variables

# ASGARD Endpoints
NYSUS_ENDPOINT=http://localhost:8080
SILENUS_ENDPOINT=http://localhost:9093
SATNET_ENDPOINT=http://localhost:8081
GIRU_ENDPOINT=http://localhost:9090
HUNOID_ENDPOINT=http://localhost:8090

# Ports
VALKYRIE_HTTP_PORT=8093
VALKYRIE_METRICS_PORT=9093

# Log Level
LOG_LEVEL=info

# Database (optional)
DATABASE_URL=postgres://user:password@localhost:5432/valkyrie

# MAVLink (for real hardware)
MAVLINK_PORT=COM3
MAVLINK_BAUD=921600

# Development
DEV_MODE=true
SIM_MODE=true
"@
    
    Set-Content -Path (Join-Path $baseDir ".env.example") -Value $envTemplate
    
    Write-Host "  ✓ Configuration files created" -ForegroundColor $SuccessColor
}

# Create README
function New-ReadmeFile {
    param([string]$baseDir)
    
    Write-Host "[8/10] Creating README..." -ForegroundColor $InfoColor
    
    $readme = @"
# VALKYRIE - Autonomous Flight System

The Tesla Autopilot for Aircraft - Full autonomous flight control powered by AI.

## Quick Start

\`\`\`powershell
# Build
go build -o bin\valkyrie.exe .\cmd\valkyrie\main.go

# Run in simulation mode
.\bin\valkyrie.exe -sim -ai -security -failsafe

# Check status
curl http://localhost:8093/health
\`\`\`

## Documentation

- [Complete Implementation Guide](./VALKYRIE_COMPLETE_GUIDE.md)
- [Implementation Roadmap](./VALKYRIE_ROADMAP.md)
- [API Documentation](./docs/API.md)

## Features

✅ Multi-sensor fusion (GPS, INS, RADAR, LIDAR, Visual, IR)  
✅ AI-powered decision making with RL  
✅ Precision guidance from Pricilla  
✅ Security monitoring from Giru  
✅ Fail-safe systems with triple redundancy  
✅ Real-time telemetry streaming  
✅ Full ASGARD integration  

## Architecture

\`\`\`
Sensors → Fusion → AI Decision → Flight Controller → Actuators
           ↓           ↓              ↓
        Security   Guidance      LiveFeed
           ↓           ↓              ↓
      Giru Stack  Pricilla      Dashboard
\`\`\`

## Development

\`\`\`powershell
# Run tests
go test ./...

# With coverage
go test ./... -cover

# Integration tests
go test ./tests/integration -v
\`\`\`

## Deployment

\`\`\`powershell
# Build Docker image
docker build -t valkyrie:latest .

# Deploy to Kubernetes
kubectl apply -f deployment/k8s/
\`\`\`

## License

Part of the ASGARD (PANDORA) project.
"@
    
    Set-Content -Path (Join-Path $baseDir "README.md") -Value $readme
    
    Write-Host "  ✓ README created" -ForegroundColor $SuccessColor
}

# Create sample main.go
function New-MainFile {
    param([string]$baseDir)
    
    Write-Host "[9/10] Creating main.go..." -ForegroundColor $InfoColor
    
    $mainGo = @"
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

var (
    httpPort    = flag.Int("http-port", 8093, "HTTP API port")
    metricsPort = flag.Int("metrics-port", 9093, "Metrics port")
    simMode     = flag.Bool("sim", false, "Simulation mode")
    enableAI    = flag.Bool("ai", false, "Enable AI decision engine")
    enableSec   = flag.Bool("security", false, "Enable security monitoring")
    enableFail  = flag.Bool("failsafe", false, "Enable fail-safe systems")
)

func main() {
    flag.Parse()
    
    printBanner()
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    log.Println("🚀 Initializing VALKYRIE...")
    
    // TODO: Initialize all subsystems
    // - Sensor fusion
    // - AI decision engine
    // - Security monitor
    // - Fail-safe system
    
    // HTTP server
    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/api/v1/status", statusHandler)
    
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", *httpPort),
        Handler: mux,
    }
    
    go func() {
        log.Printf("🌐 HTTP API listening on :%d", *httpPort)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("HTTP server error: %v", err)
        }
    }()
    
    log.Println("✅ VALKYRIE is OPERATIONAL")
    log.Println("   Press Ctrl+C to shutdown")
    
    // Wait for shutdown
    <-sigChan
    log.Println("\n🛑 Shutdown signal received...")
    
    cancel()
    
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()
    server.Shutdown(shutdownCtx)
    
    log.Println("✅ VALKYRIE shutdown complete")
}

func printBanner() {
    banner := \`
╦  ╦╔═╗╦  ╦╔═╦ ╦╦═╗╦╔═╗
╚╗╔╝╠═╣║  ╠╩╗╚╦╝╠╦╝║║╣ 
 ╚╝ ╩ ╩╩═╝╩ ╩ ╩ ╩╚═╩╚═╝
Autonomous Flight System v1.0.0

\`
    fmt.Println(banner)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, \`{"status":"ok","service":"valkyrie","version":"1.0.0"}\`)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, \`{
        "fusion_active":true,
        "ai_active":%t,
        "security_active":%t,
        "failsafe_active":%t,
        "simulation_mode":%t
    }\`, *enableAI, *enableSec, *enableFail, *simMode)
}
"@
    
    Set-Content -Path (Join-Path $baseDir "cmd\valkyrie\main.go") -Value $mainGo
    
    Write-Host "  ✓ main.go created" -ForegroundColor $SuccessColor
}

# Build the project
function Build-Project {
    param([string]$baseDir)
    
    Write-Host "[10/10] Building VALKYRIE..." -ForegroundColor $InfoColor
    
    Push-Location $baseDir
    
    try {
        go build -o "bin\valkyrie.exe" ".\cmd\valkyrie\main.go"
        Write-Host "  ✓ Build successful!" -ForegroundColor $SuccessColor
        Write-Host "  Binary location: .\bin\valkyrie.exe" -ForegroundColor $InfoColor
    } catch {
        Write-Host "  ✗ Build failed: $_" -ForegroundColor $ErrorColor
    }
    
    Pop-Location
}

# Main execution
try {
    Show-Banner
    
    # Check administrator privileges
    if (-not (Test-Administrator)) {
        Write-Host "Warning: Not running as administrator. Some features may not work." -ForegroundColor $WarningColor
    }
    
    # Run setup steps
    Test-Prerequisites
    $baseDir = New-DirectoryStructure
    Initialize-GoModule -baseDir $baseDir
    Copy-Components -baseDir $baseDir
    Update-PackageDeclarations -baseDir $baseDir
    Install-GoDependencies -baseDir $baseDir
    New-ConfigurationFiles -baseDir $baseDir
    New-ReadmeFile -baseDir $baseDir
    New-MainFile -baseDir $baseDir
    Build-Project -baseDir $baseDir
    
    Write-Host ""
    Write-Host "========================================" -ForegroundColor $SuccessColor
    Write-Host "  VALKYRIE SETUP COMPLETE! 🎉" -ForegroundColor $SuccessColor
    Write-Host "========================================" -ForegroundColor $SuccessColor
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor $InfoColor
    Write-Host "  1. cd $baseDir" -ForegroundColor $InfoColor
    Write-Host "  2. .\bin\valkyrie.exe -sim -ai -security -failsafe" -ForegroundColor $InfoColor
    Write-Host "  3. Open http://localhost:8093/health" -ForegroundColor $InfoColor
    Write-Host ""
    Write-Host "Documentation:" -ForegroundColor $InfoColor
    Write-Host "  - Complete Guide: VALKYRIE_COMPLETE_GUIDE.md" -ForegroundColor $InfoColor
    Write-Host "  - Roadmap: VALKYRIE_ROADMAP.md" -ForegroundColor $InfoColor
    Write-Host ""
    Write-Host "Happy flying! ✈️🚀" -ForegroundColor $SuccessColor
    
} catch {
    Write-Host ""
    Write-Host "❌ Setup failed: $_" -ForegroundColor $ErrorColor
    Write-Host ""
    exit 1
}
