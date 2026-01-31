# ASGARD Full Platform Demo Runner
# Launches all services and runs the comprehensive Playwright demo

$ErrorActionPreference = "Stop"
$AsgardRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

Write-Host "`n=== ASGARD Platform Demo ===" -ForegroundColor Cyan
Write-Host "Building and launching all services...`n" -ForegroundColor Gray

# Kill any existing services
Write-Host "Stopping existing services..." -ForegroundColor Yellow
Get-Process -Name nysus,giru,pricilla,valkyrie -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# Build all services
Write-Host "Building services..." -ForegroundColor Yellow
Push-Location $AsgardRoot
try {
    go build -o bin/pricilla.exe ./Pricilla/cmd/pricilla
    go build -o bin/giru.exe ./cmd/giru
    go build -o bin/nysus.exe ./cmd/nysus
    go build -o bin/hunoid.exe ./cmd/hunoid
    go build -o bin/silenus.exe ./cmd/silenus
    Push-Location "$AsgardRoot\Valkyrie"
    go build -o "$AsgardRoot\bin\valkyrie.exe" .\cmd\valkyrie\main.go
    Pop-Location
    Write-Host "  All services built successfully" -ForegroundColor Green
} catch {
    Write-Host "  Build failed: $_" -ForegroundColor Red
    exit 1
}

# Start Giru (API-only mode)
Write-Host "Starting GIRU (Security)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\giru.exe" -ArgumentList "-api-only" -WindowStyle Hidden
Start-Sleep -Seconds 1

# Start GIRU JARVIS Electron app (desktop UI)
Write-Host "Starting GIRU JARVIS (Electron)..." -ForegroundColor Yellow
$jarvisDir = "$AsgardRoot\Giru\Giru(jarvis)"
$env:GIRU_TTS_SELFTEST = "1"
Start-Process -FilePath "npm.cmd" -ArgumentList "run","dev:win" -WorkingDirectory $jarvisDir -WindowStyle Normal
Start-Sleep -Seconds 2

# Start Nysus (development mode)
Write-Host "Starting NYSUS (Orchestration)..." -ForegroundColor Yellow
$env:ASGARD_ENV = "development"
Start-Process -FilePath "$AsgardRoot\bin\nysus.exe" -ArgumentList "-addr",":8080" -WindowStyle Hidden
Start-Sleep -Seconds 1

# Start Pricilla
Write-Host "Starting PRICILLA (Guidance)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\pricilla.exe" -ArgumentList "-http-port","8092","-giru","http://localhost:9090","-nysus","http://localhost:8080" -WindowStyle Hidden
Start-Sleep -Seconds 2

# Enable hardware bypass for Hunoid/Silenus in dev unless explicitly set
if (-not $env:HUNOID_BYPASS_HARDWARE) {
    $env:HUNOID_BYPASS_HARDWARE = "1"
}
if (-not $env:SILENUS_BYPASS_HARDWARE) {
    $env:SILENUS_BYPASS_HARDWARE = "1"
}

# Start Hunoid (simulated hardware)
# Note: Removed -stay-alive flag since demo auto-shuts down via Stop-Process.
# Use -stay-alive only for interactive sessions where graceful SIGINT shutdown is needed.
Write-Host "Starting HUNOID (Robotics)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\hunoid.exe" -ArgumentList "-id","hunoid-demo-001","-operator-mode","auto","-operator-ui-addr",":8090" -WindowStyle Hidden
Start-Sleep -Seconds 2

# Start Silenus (simulated hardware)
Write-Host "Starting SILENUS (Satellite)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\silenus.exe" -ArgumentList "-id","sat-demo-001" -WindowStyle Hidden
Start-Sleep -Seconds 2

# Start Valkyrie
Write-Host "Starting VALKYRIE (Flight System)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\valkyrie.exe" -ArgumentList "-sim","-ai","-security","-failsafe","-livefeed","-http-port","8093","-metrics-port","9095" -WindowStyle Hidden
Start-Sleep -Seconds 2

# Verify services are running
Write-Host "`nVerifying services..." -ForegroundColor Yellow
$services = @(
    @{ Name = "PRICILLA"; Port = 8092 },
    @{ Name = "GIRU"; Port = 9090 },
    @{ Name = "NYSUS"; Port = 8080 },
    @{ Name = "HUNOID"; Port = 8090 },
    @{ Name = "SILENUS"; Port = 9093 },
    @{ Name = "VALKYRIE"; Port = 8093 }
)

$allHealthy = $true
foreach ($svc in $services) {
    try {
        $null = Invoke-RestMethod -Uri "http://localhost:$($svc.Port)/health" -TimeoutSec 5 -ErrorAction Stop
        Write-Host "  $($svc.Name): " -NoNewline
        Write-Host "HEALTHY" -ForegroundColor Green
    } catch {
        Write-Host "  $($svc.Name): " -NoNewline
        # Check if process is at least running
        $proc = Get-Process -Name $svc.Name.ToLower() -ErrorAction SilentlyContinue
        if ($proc) {
            Write-Host "RUNNING" -ForegroundColor Yellow
        } else {
            Write-Host "OFFLINE" -ForegroundColor Red
            $allHealthy = $false
        }
    }
}

if (-not $allHealthy) {
    Write-Host "`nSome services failed to start. Check logs." -ForegroundColor Red
}

Pop-Location

# Run Playwright demo
Write-Host "`n=== Running ASGARD Platform Demo ===" -ForegroundColor Cyan
Write-Host "Recording video with overlay...`n" -ForegroundColor Gray

Push-Location "$AsgardRoot\test\e2e"

# Ensure playwright is installed
if (-not (Test-Path "node_modules")) {
    Write-Host "Installing dependencies..." -ForegroundColor Yellow
    npm install
}

# Run the demo
$env:ASGARD_VALKYRIE_METRICS = "http://localhost:9095/metrics"
npx playwright test asgard-complete-demo.spec.ts --project=demo-chromium --reporter=list

Pop-Location

# Show results
$demoDir = "$AsgardRoot\test\e2e\demo-videos"
if (Test-Path $demoDir) {
    $latestVideo = Get-ChildItem -Path $demoDir -Recurse -Filter "*.webm" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($latestVideo) {
        Write-Host "`n=== Demo Complete ===" -ForegroundColor Green
        Write-Host "Video: $($latestVideo.FullName)" -ForegroundColor Cyan
    }
}

# Clean shutdown - demo complete, terminate all services gracefully
Write-Host "`nShutting down demo services..." -ForegroundColor Yellow

# Give services a moment to complete any pending operations
Start-Sleep -Seconds 1

# Stop all demo services
$demoProcesses = @("nysus", "giru", "pricilla", "hunoid", "silenus", "valkyrie")
foreach ($procName in $demoProcesses) {
    $proc = Get-Process -Name $procName -ErrorAction SilentlyContinue
    if ($proc) {
        Write-Host "  Stopping $procName..." -ForegroundColor Gray
        Stop-Process -Name $procName -Force -ErrorAction SilentlyContinue
    }
}

# Stop Electron (GIRU JARVIS)
Get-Process -Name electron -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

Write-Host "All demo services stopped." -ForegroundColor Green
Write-Host "`nTo run services manually: .\test\e2e\run-asgard-demo.ps1" -ForegroundColor Gray
