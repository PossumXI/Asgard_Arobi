#!/usr/bin/env pwsh
<#
.SYNOPSIS
    ASGARD Flight Simulation Demo Runner

.DESCRIPTION
    Starts all ASGARD backend services, then runs the Playwright flight simulation
    demo test which generates a demo video.

.PARAMETER Headed
    Run in headed mode (show browser window).

.PARAMETER SkipServices
    Skip starting backend services (assume they are already running).

.EXAMPLE
    .\run-flight-simulation-demo.ps1
    .\run-flight-simulation-demo.ps1 -Headed
    .\run-flight-simulation-demo.ps1 -SkipServices
#>

param(
    [switch]$Headed,
    [switch]$SkipServices,
    [switch]$Help
)

$ErrorActionPreference = 'Stop'
$SCRIPT_DIR = $PSScriptRoot
$ASGARD_ROOT = Resolve-Path (Join-Path $SCRIPT_DIR "../../")

function Write-Header {
    Write-Host ""
    Write-Host "  ASGARD Flight Simulation Demo Runner" -ForegroundColor Cyan
    Write-Host "  ============================================================" -ForegroundColor DarkCyan
    Write-Host ""
}

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    exit 0
}

Write-Header

# Check prerequisites
Write-Host "  Checking prerequisites..." -ForegroundColor Yellow

if (-not (Get-Command "npx" -ErrorAction SilentlyContinue)) {
    Write-Host "  [!!] npx not found. Install Node.js and npm." -ForegroundColor Red
    exit 1
}

$testFile = Join-Path $SCRIPT_DIR "flight-simulation-demo.spec.ts"
$configFile = Join-Path $SCRIPT_DIR "playwright-flight-config.ts"

if (-not (Test-Path $testFile)) {
    Write-Host "  [!!] Test file not found: $testFile" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $configFile)) {
    Write-Host "  [!!] Config file not found: $configFile" -ForegroundColor Red
    exit 1
}

# Install playwright if needed
Set-Location $SCRIPT_DIR
if (-not (Test-Path "node_modules")) {
    Write-Host "  Installing Playwright..." -ForegroundColor Yellow
    npm install
    npx playwright install chromium
}

Write-Host "  [OK] Prerequisites met" -ForegroundColor Green
Write-Host ""

# Start backend services if not skipped
if (-not $SkipServices) {
    Write-Host "  Starting ASGARD backend services..." -ForegroundColor Yellow
    $startScript = Join-Path $ASGARD_ROOT "scripts/start-all-services.ps1"
    if (Test-Path $startScript) {
        # Start services in background
        $serviceJob = Start-Job -ScriptBlock {
            param($script, $root)
            Set-Location $root
            & $script -SkipFrontends -SkipBuild
        } -ArgumentList $startScript, $ASGARD_ROOT

        # Wait for services to come up
        Write-Host "  Waiting for services to initialize..." -ForegroundColor Yellow
        Start-Sleep -Seconds 20
    } else {
        Write-Host "  [WARN] start-all-services.ps1 not found, services must be started manually" -ForegroundColor Yellow
    }
}

# Run the demo
Write-Host "  Running Flight Simulation Demo..." -ForegroundColor Cyan
Write-Host ""

$command = "npx playwright test flight-simulation-demo.spec.ts --config=playwright-flight-config.ts"
if ($Headed) {
    $command += " --headed"
}

Write-Host "  Command: $command" -ForegroundColor DarkGray
Write-Host ""

try {
    Set-Location $SCRIPT_DIR
    Invoke-Expression $command

    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "  [OK] Demo completed successfully!" -ForegroundColor Green
        Write-Host ""
        if (Test-Path "./demo-videos") {
            Write-Host "  Video files:" -ForegroundColor Cyan
            Get-ChildItem "./demo-videos" -Filter "*.webm" -Recurse | ForEach-Object {
                Write-Host "    $($_.FullName)" -ForegroundColor Gray
            }
        }
    } else {
        Write-Host ""
        Write-Host "  [!!] Demo failed with exit code $LASTEXITCODE" -ForegroundColor Red
        Write-Host ""
        Write-Host "  Troubleshooting:" -ForegroundColor Yellow
        Write-Host "    1. Run .\scripts\start-all-services.ps1 first" -ForegroundColor Gray
        Write-Host "    2. Check service health endpoints" -ForegroundColor Gray
        Write-Host "    3. Run with -Headed to see browser output" -ForegroundColor Gray
        exit $LASTEXITCODE
    }
} catch {
    Write-Host "  [!!] Demo execution failed: $_" -ForegroundColor Red
    exit 1
}
