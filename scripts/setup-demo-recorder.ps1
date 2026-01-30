# ASGARD Demo Recorder Setup Script
# Sets up the Playwright-based demo recorder environment

param(
    [switch]$SkipInstall,
    [switch]$RunDemo,
    [switch]$Headless,
    [switch]$Slow
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   ASGARD Demo Recorder Setup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check Node.js
Write-Host "[1/5] Checking Node.js installation..." -ForegroundColor Yellow
try {
    $nodeVersion = node --version
    Write-Host "      Node.js version: $nodeVersion" -ForegroundColor Green
    
    $majorVersion = [int]($nodeVersion -replace 'v(\d+)\..*', '$1')
    if ($majorVersion -lt 18) {
        Write-Host "      Warning: Node.js 18+ recommended" -ForegroundColor Yellow
    }
} catch {
    Write-Host "      ERROR: Node.js not found!" -ForegroundColor Red
    Write-Host "      Please install Node.js 18+ from https://nodejs.org/" -ForegroundColor Red
    exit 1
}

# Check npm
Write-Host "[2/5] Checking npm..." -ForegroundColor Yellow
try {
    $npmVersion = npm --version
    Write-Host "      npm version: $npmVersion" -ForegroundColor Green
} catch {
    Write-Host "      ERROR: npm not found!" -ForegroundColor Red
    exit 1
}

# Install dependencies
if (-not $SkipInstall) {
    Write-Host "[3/5] Installing dependencies..." -ForegroundColor Yellow
    Push-Location $ScriptDir
    try {
        npm install
        Write-Host "      Dependencies installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "      ERROR: Failed to install dependencies" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Pop-Location
} else {
    Write-Host "[3/5] Skipping dependency installation (--SkipInstall)" -ForegroundColor Gray
}

# Install Playwright browsers
if (-not $SkipInstall) {
    Write-Host "[4/5] Installing Playwright browsers..." -ForegroundColor Yellow
    Push-Location $ScriptDir
    try {
        npx playwright install chromium
        Write-Host "      Playwright Chromium installed successfully" -ForegroundColor Green
    } catch {
        Write-Host "      ERROR: Failed to install Playwright browsers" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Pop-Location
} else {
    Write-Host "[4/5] Skipping Playwright installation (--SkipInstall)" -ForegroundColor Gray
}

# Create output directories
Write-Host "[5/5] Creating output directories..." -ForegroundColor Yellow
$outputDir = Join-Path $ScriptDir "demo-output"
$screenshotsDir = Join-Path $outputDir "screenshots"
$videosDir = Join-Path $outputDir "videos"

@($outputDir, $screenshotsDir, $videosDir) | ForEach-Object {
    if (-not (Test-Path $_)) {
        New-Item -ItemType Directory -Path $_ -Force | Out-Null
        Write-Host "      Created: $_" -ForegroundColor Green
    } else {
        Write-Host "      Exists: $_" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "   Setup Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

# Check service availability
Write-Host "Checking ASGARD service availability:" -ForegroundColor Yellow
Write-Host ""

$services = @(
    @{ Name = "Website"; Url = "http://localhost:3000"; Port = 3000 },
    @{ Name = "Valkyrie"; Url = "http://localhost:8093/health"; Port = 8093 },
    @{ Name = "Pricilla"; Url = "http://localhost:8092/health"; Port = 8092 },
    @{ Name = "Giru JARVIS"; Url = "http://localhost:5000"; Port = 5000 }
)

$availableServices = 0
foreach ($service in $services) {
    try {
        $response = Invoke-WebRequest -Uri $service.Url -TimeoutSec 3 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($response.StatusCode -lt 500) {
            Write-Host "   [OK] $($service.Name) (port $($service.Port))" -ForegroundColor Green
            $availableServices++
        } else {
            Write-Host "   [--] $($service.Name) (port $($service.Port)) - Not healthy" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "   [--] $($service.Name) (port $($service.Port)) - Not running" -ForegroundColor Gray
    }
}

Write-Host ""
if ($availableServices -eq 0) {
    Write-Host "Warning: No ASGARD services detected!" -ForegroundColor Yellow
    Write-Host "Please start the services before running the demo recorder." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To start services:" -ForegroundColor Cyan
    Write-Host "  Website:  cd Websites && npm run dev" -ForegroundColor White
    Write-Host "  Backend:  .\scripts\run_backend.ps1" -ForegroundColor White
    Write-Host "  Giru:     cd Giru/Giru(jarvis) && python app.py" -ForegroundColor White
} else {
    Write-Host "$availableServices of $($services.Count) services available" -ForegroundColor Cyan
}

Write-Host ""

# Run demo if requested
if ($RunDemo) {
    Write-Host "Running demo recorder..." -ForegroundColor Yellow
    Write-Host ""
    
    Push-Location $ScriptDir
    
    $args = @()
    if ($Headless) { $args += "--headless" }
    if ($Slow) { $args += "--slow" }
    
    if ($args.Count -gt 0) {
        npx ts-node demo-recorder.ts @args
    } else {
        npx ts-node demo-recorder.ts
    }
    
    Pop-Location
} else {
    Write-Host "Usage:" -ForegroundColor Cyan
    Write-Host "  npm run demo            # Record all available systems" -ForegroundColor White
    Write-Host "  npm run demo:website    # Record website only" -ForegroundColor White
    Write-Host "  npm run demo:valkyrie   # Record Valkyrie only" -ForegroundColor White
    Write-Host "  npm run demo:pricilla   # Record Pricilla only" -ForegroundColor White
    Write-Host "  npm run demo:giru       # Record Giru JARVIS only" -ForegroundColor White
    Write-Host ""
    Write-Host "Or run this script with -RunDemo to start recording:" -ForegroundColor Cyan
    Write-Host "  .\setup-demo-recorder.ps1 -RunDemo" -ForegroundColor White
    Write-Host "  .\setup-demo-recorder.ps1 -RunDemo -Slow" -ForegroundColor White
    Write-Host "  .\setup-demo-recorder.ps1 -RunDemo -Headless" -ForegroundColor White
}

Write-Host ""
