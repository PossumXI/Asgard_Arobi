# ASGARD Backend Run Script
# Runs the Nysus API server

param(
    [string]$Port = "8080",
    [string]$Config = ""
)

Write-Host "=== ASGARD Nysus Server ===" -ForegroundColor Green

# Set working directory
Set-Location "$PSScriptRoot\.."

# Check if binary exists
$binary = "bin\nysus.exe"
if (-not (Test-Path $binary)) {
    Write-Host "Binary not found. Building..." -ForegroundColor Yellow
    & "$PSScriptRoot\build_backend.ps1"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Build failed" -ForegroundColor Red
        exit 1
    }
}

# Check if databases are running
Write-Host "`nChecking database connections..." -ForegroundColor Yellow
$pgRunning = Test-NetConnection -ComputerName localhost -Port 55432 -InformationLevel Quiet -WarningAction SilentlyContinue
$mongoRunning = Test-NetConnection -ComputerName localhost -Port 27018 -InformationLevel Quiet -WarningAction SilentlyContinue

if (-not $pgRunning) {
    Write-Host "Warning: PostgreSQL not running on localhost:55432" -ForegroundColor Yellow
    Write-Host "Start databases with: cd Data && docker compose up -d" -ForegroundColor Cyan
}

if (-not $mongoRunning) {
    Write-Host "Warning: MongoDB not running on localhost:27018" -ForegroundColor Yellow
    Write-Host "Start databases with: cd Data && docker compose up -d" -ForegroundColor Cyan
}

if (-not $env:ASGARD_ENV) {
    $env:ASGARD_ENV = "development"
}
if (-not $env:MONGO_PORT) {
    $env:MONGO_PORT = "27018"
}

# Build command
$cmd = "$binary -port $Port"
if ($Config -ne "") {
    $cmd += " -config $Config"
}

Write-Host "`nStarting server on port $Port..." -ForegroundColor Yellow
Write-Host "API: http://localhost:$Port/api" -ForegroundColor Cyan
Write-Host "Health: http://localhost:$Port/api/health" -ForegroundColor Cyan
Write-Host "`nPress Ctrl+C to stop`n" -ForegroundColor Gray

# Run the server
Invoke-Expression $cmd
