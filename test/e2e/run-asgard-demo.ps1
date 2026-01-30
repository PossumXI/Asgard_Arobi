# ASGARD Full Platform Demo Runner
# Launches all services and runs the comprehensive Playwright demo

$ErrorActionPreference = "Stop"
$AsgardRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

Write-Host "`n=== ASGARD Platform Demo ===" -ForegroundColor Cyan
Write-Host "Building and launching all services...`n" -ForegroundColor Gray

# Kill any existing services
Write-Host "Stopping existing services..." -ForegroundColor Yellow
Get-Process -Name nysus,giru,pricilla -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# Build all services
Write-Host "Building services..." -ForegroundColor Yellow
Push-Location $AsgardRoot
try {
    go build -o bin/pricilla.exe ./Pricilla/cmd/pricilla
    go build -o bin/giru.exe ./cmd/giru
    go build -o bin/nysus.exe ./cmd/nysus
    Write-Host "  All services built successfully" -ForegroundColor Green
} catch {
    Write-Host "  Build failed: $_" -ForegroundColor Red
    exit 1
}

# Start Giru (API-only mode)
Write-Host "Starting GIRU (Security)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\giru.exe" -ArgumentList "-api-only" -WindowStyle Hidden
Start-Sleep -Seconds 1

# Start Nysus (development mode)
Write-Host "Starting NYSUS (Orchestration)..." -ForegroundColor Yellow
$env:ASGARD_ENV = "development"
Start-Process -FilePath "$AsgardRoot\bin\nysus.exe" -ArgumentList "-addr",":8080" -WindowStyle Hidden
Start-Sleep -Seconds 1

# Start Pricilla
Write-Host "Starting PRICILLA (Guidance)..." -ForegroundColor Yellow
Start-Process -FilePath "$AsgardRoot\bin\pricilla.exe" -ArgumentList "-http-port","8092","-giru","http://localhost:9090","-nysus","http://localhost:8080" -WindowStyle Hidden
Start-Sleep -Seconds 2

# Verify services are running
Write-Host "`nVerifying services..." -ForegroundColor Yellow
$services = @(
    @{ Name = "PRICILLA"; Port = 8092 },
    @{ Name = "GIRU"; Port = 9090 },
    @{ Name = "NYSUS"; Port = 8080 }
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
npx playwright test asgard-full-demo.spec.ts --project=demo-chromium --reporter=list

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

Write-Host "`nServices still running. To stop: Get-Process -Name nysus,giru,pricilla | Stop-Process" -ForegroundColor Gray
