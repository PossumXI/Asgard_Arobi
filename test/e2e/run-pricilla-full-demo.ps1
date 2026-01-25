# PRICILLA Full Capabilities Demo Video Generator
# This script runs the comprehensive Playwright demo test and generates a video

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "   PRICILLA Full Capabilities Demo" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Ensure we run from the e2e directory
Set-Location $PSScriptRoot

# Check if Pricilla is running
$pricillaRunning = $false
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8089/health" -TimeoutSec 3 -UseBasicParsing -ErrorAction SilentlyContinue
    if ($response.StatusCode -eq 200) { $pricillaRunning = $true }
} catch {}

Write-Host "Pre-flight Check:" -ForegroundColor Yellow
Write-Host "  - Pricilla Service (8089): $(if ($pricillaRunning) { 'RUNNING' } else { 'NOT RUNNING' })" -ForegroundColor $(if ($pricillaRunning) { 'Green' } else { 'Red' })
Write-Host ""

if (-not $pricillaRunning) {
    Write-Host "WARNING: Pricilla service is not running!" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Please start Pricilla first:" -ForegroundColor Yellow
    Write-Host "  cd Pricilla && go run cmd/percila/main.go" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Or build and run:" -ForegroundColor Yellow
    Write-Host "  cd Pricilla && go build -o bin/pricilla.exe ./cmd/percila/main.go" -ForegroundColor Gray
    Write-Host "  ./Pricilla/bin/pricilla.exe" -ForegroundColor Gray
    Write-Host ""
    
    $startPricilla = Read-Host "Attempt to start Pricilla now? (y/n)"
    if ($startPricilla -eq "y") {
        Write-Host ""
        Write-Host "Starting Pricilla in background..." -ForegroundColor Green
        
        # Build Pricilla first
        Push-Location "..\..\Pricilla"
        Write-Host "Building Pricilla..." -ForegroundColor Gray
        go build -o bin/pricilla.exe ./cmd/percila/main.go
        
        # Start Pricilla in background
        Start-Process -FilePath ".\bin\pricilla.exe" -WindowStyle Minimized
        Pop-Location
        
        Write-Host "Waiting for Pricilla to start..." -ForegroundColor Gray
        Start-Sleep -Seconds 5
        
        # Check again
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8089/health" -TimeoutSec 3 -UseBasicParsing -ErrorAction SilentlyContinue
            if ($response.StatusCode -eq 200) { 
                $pricillaRunning = $true 
                Write-Host "Pricilla started successfully!" -ForegroundColor Green
            }
        } catch {}
        
        if (-not $pricillaRunning) {
            Write-Host "Failed to start Pricilla. Please start it manually." -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "Demo requires Pricilla to be running. Exiting." -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "Starting Full Capabilities Demo Recording..." -ForegroundColor Green
Write-Host "This will take approximately 3-4 minutes." -ForegroundColor Gray
Write-Host ""

# Run the Playwright test
npx playwright test pricilla-full-demo.spec.ts --config playwright.config.ts

# Check if video was created
$videoDir = ".\demo-videos"
if (Test-Path $videoDir) {
    $videos = Get-ChildItem -Path $videoDir -Filter "*.webm" -Recurse | Sort-Object LastWriteTime -Descending
    if ($videos.Count -gt 0) {
        $latestVideo = $videos[0]
        Write-Host ""
        Write-Host "============================================" -ForegroundColor Green
        Write-Host "   Demo Video Generated Successfully!" -ForegroundColor Green
        Write-Host "============================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "Video Location:" -ForegroundColor Cyan
        Write-Host "  $($latestVideo.FullName)" -ForegroundColor White
        Write-Host ""
        Write-Host "Video Size: $([math]::Round($latestVideo.Length / 1MB, 2)) MB" -ForegroundColor Gray
        Write-Host "Video Duration: ~3-4 minutes" -ForegroundColor Gray
        Write-Host ""
        
        # Check for metrics file
        $metricsDir = ".\metrics"
        if (Test-Path $metricsDir) {
            $metricsFiles = Get-ChildItem -Path $metricsDir -Filter "pricilla_full_demo_*.json" | Sort-Object LastWriteTime -Descending
            if ($metricsFiles.Count -gt 0) {
                Write-Host "Metrics saved to:" -ForegroundColor Cyan
                Write-Host "  $($metricsFiles[0].FullName)" -ForegroundColor White
                Write-Host ""
            }
        }
        
        # Open the video
        $open = Read-Host "Open video now? (y/n)"
        if ($open -eq "y") {
            Start-Process $latestVideo.FullName
        }
    } else {
        Write-Host "No video file found in output directory." -ForegroundColor Yellow
    }
} else {
    Write-Host "Video output directory not found." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Demo Features Showcased:" -ForegroundColor Cyan
Write-Host "  1. Mission Deployment" -ForegroundColor White
Write-Host "  2. Through-Wall WiFi Imaging" -ForegroundColor White
Write-Host "  3. Rapid Replanning (<100ms)" -ForegroundColor White
Write-Host "  4. Terminal Guidance Mode" -ForegroundColor White
Write-Host "  5. Hit Probability Estimation" -ForegroundColor White
Write-Host "  6. Weather Impact Modeling" -ForegroundColor White
Write-Host "  7. ECM/Jamming Detection" -ForegroundColor White
Write-Host "  8. Mission Abort/RTB" -ForegroundColor White
Write-Host ""
Write-Host "Done!" -ForegroundColor Green
