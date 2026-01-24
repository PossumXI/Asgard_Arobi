# ASGARD Backend Build Script
# Builds the Nysus API server

Write-Host "=== ASGARD Backend Build ===" -ForegroundColor Green

# Check if Go is installed
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

Write-Host "Go version: $(go version)" -ForegroundColor Cyan

# Set working directory
Set-Location "$PSScriptRoot\.."

# Download dependencies
Write-Host "`nDownloading dependencies..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to download dependencies" -ForegroundColor Red
    exit 1
}

# Tidy dependencies
Write-Host "Tidying dependencies..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to tidy dependencies" -ForegroundColor Red
    exit 1
}

# Build the server
Write-Host "`nBuilding Nysus server..." -ForegroundColor Yellow
$buildOutput = "bin\nysus.exe"
if ($IsWindows -or $env:OS -like "*Windows*") {
    $buildOutput = "bin\nysus.exe"
} else {
    $buildOutput = "bin/nysus"
}

go build -o $buildOutput ./cmd/nysus
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Build failed" -ForegroundColor Red
    exit 1
}

Write-Host "`n✓ Build successful!" -ForegroundColor Green
Write-Host "Binary: $buildOutput" -ForegroundColor Cyan

# Run tests if requested
if ($args -contains "-test") {
    Write-Host "`nRunning tests..." -ForegroundColor Yellow
    go test ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Warning: Some tests failed" -ForegroundColor Yellow
    }
}

Write-Host "`n=== Build Complete ===" -ForegroundColor Green
