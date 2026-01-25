# PRICILLA Complete Capabilities Demo Video Generator
# This script runs the comprehensive Playwright demo test and generates a video
# showcasing ALL Pricilla features including:
# - All 6 payload types (UAV, Missile, Hunoid, Spacecraft, Drone, Rocket)
# - WiFi CSI Through-Wall Imaging
# - Terminal Guidance Mode
# - Hit Probability & CEP Estimation
# - Weather Impact Modeling
# - ECM/Jamming Detection
# - Rapid Replanning
# - Stealth Optimization
# - Mission Abort/RTB
# - Accuracy & Benchmark Reports

param(
    [switch]$SkipPreflightCheck,
    [switch]$AutoStartPricilla,
    [switch]$KeepRunning
)

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "   PRICILLA Complete Capabilities Demo" -ForegroundColor Cyan
Write-Host "   ALL Features & Payload Types" -ForegroundColor Yellow
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Ensure we run from the e2e directory
Set-Location $PSScriptRoot

# Check if Pricilla is running
$pricillaRunning = $false
if (-not $SkipPreflightCheck) {
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
        
        if ($AutoStartPricilla) {
            $startPricilla = "y"
        } else {
            $startPricilla = Read-Host "Attempt to start Pricilla now? (y/n)"
        }
        
        if ($startPricilla -eq "y") {
            Write-Host ""
            Write-Host "Starting Pricilla in background..." -ForegroundColor Green
            
            # Build Pricilla first
            Push-Location "..\..\Pricilla"
            Write-Host "Building Pricilla..." -ForegroundColor Gray
            go build -o bin/pricilla.exe ./cmd/percila/main.go
            
            if ($LASTEXITCODE -ne 0) {
                Write-Host "Failed to build Pricilla. Please check for errors." -ForegroundColor Red
                Pop-Location
                exit 1
            }
            
            # Start Pricilla in background
            Start-Process -FilePath ".\bin\pricilla.exe" -WindowStyle Minimized
            Pop-Location
            
            Write-Host "Waiting for Pricilla to start..." -ForegroundColor Gray
            Start-Sleep -Seconds 5
            
            # Check again
            for ($i = 0; $i -lt 5; $i++) {
                try {
                    $response = Invoke-WebRequest -Uri "http://localhost:8089/health" -TimeoutSec 3 -UseBasicParsing -ErrorAction SilentlyContinue
                    if ($response.StatusCode -eq 200) { 
                        $pricillaRunning = $true 
                        break
                    }
                } catch {}
                Start-Sleep -Seconds 2
            }
            
            if ($pricillaRunning) {
                Write-Host "Pricilla started successfully!" -ForegroundColor Green
            } else {
                Write-Host "Failed to start Pricilla. Please start it manually." -ForegroundColor Red
                exit 1
            }
        } else {
            Write-Host "Demo requires Pricilla to be running. Exiting." -ForegroundColor Red
            exit 1
        }
    }
}

Write-Host ""
Write-Host "Starting Complete Capabilities Demo Recording..." -ForegroundColor Green
Write-Host "This will take approximately 6-7 minutes." -ForegroundColor Gray
Write-Host ""

# Run the Playwright test
Write-Host "Running Playwright test: pricilla-complete-demo.spec.ts" -ForegroundColor Cyan
npx playwright test pricilla-complete-demo.spec.ts --config playwright.config.ts

$testResult = $LASTEXITCODE

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
        Write-Host "Video Duration: ~6-7 minutes" -ForegroundColor Gray
        Write-Host ""
        
        # Check for metrics file
        $metricsDir = ".\metrics"
        if (Test-Path $metricsDir) {
            $metricsFiles = Get-ChildItem -Path $metricsDir -Filter "pricilla_complete_demo_*.json" | Sort-Object LastWriteTime -Descending
            if ($metricsFiles.Count -gt 0) {
                Write-Host "Metrics & Benchmarks saved to:" -ForegroundColor Cyan
                Write-Host "  $($metricsFiles[0].FullName)" -ForegroundColor White
                Write-Host ""
                
                # Display summary from metrics file
                $metricsContent = Get-Content $metricsFiles[0].FullName | ConvertFrom-Json
                if ($metricsContent.summary) {
                    Write-Host "Performance Summary:" -ForegroundColor Yellow
                    Write-Host "  - Payload Types Tested: $($metricsContent.summary.totalPayloadTypes)" -ForegroundColor White
                    Write-Host "  - Average Accuracy: $($metricsContent.summary.avgAccuracy)%" -ForegroundColor White
                    Write-Host "  - Average Hit Probability: $([math]::Round($metricsContent.summary.avgHitProbability * 100, 1))%" -ForegroundColor White
                    Write-Host "  - Average CEP: $($metricsContent.summary.avgCEP)m" -ForegroundColor White
                    Write-Host "  - Average Stealth Score: $($metricsContent.summary.avgStealthScore)" -ForegroundColor White
                    Write-Host ""
                }
            }
        }
        
        # Open the video
        if (-not $KeepRunning) {
            $open = Read-Host "Open video now? (y/n)"
            if ($open -eq "y") {
                Start-Process $latestVideo.FullName
            }
        }
    } else {
        Write-Host "No video file found in output directory." -ForegroundColor Yellow
    }
} else {
    Write-Host "Video output directory not found." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Demo Features Showcased:" -ForegroundColor Cyan
Write-Host "  1.  Multi-Payload Support (6 types)" -ForegroundColor White
Write-Host "      - UAV Drone" -ForegroundColor Gray
Write-Host "      - Cruise Missile" -ForegroundColor Gray
Write-Host "      - Hunoid Robot" -ForegroundColor Gray
Write-Host "      - Orbital Spacecraft" -ForegroundColor Gray
Write-Host "      - Recon Drone" -ForegroundColor Gray
Write-Host "      - Ballistic Rocket" -ForegroundColor Gray
Write-Host "  2.  Through-Wall WiFi CSI Imaging" -ForegroundColor White
Write-Host "  3.  Terminal Guidance Mode (100Hz)" -ForegroundColor White
Write-Host "  4.  Hit Probability & CEP Estimation" -ForegroundColor White
Write-Host "  5.  Weather Impact Modeling" -ForegroundColor White
Write-Host "  6.  ECM/Jamming Detection & Adaptation" -ForegroundColor White
Write-Host "  7.  Rapid Replanning (<100ms)" -ForegroundColor White
Write-Host "  8.  Stealth Optimization (RCS/Thermal)" -ForegroundColor White
Write-Host "  9.  Mission Abort / RTB" -ForegroundColor White
Write-Host "  10. Accuracy & Benchmark Reports" -ForegroundColor White
Write-Host ""

if ($testResult -eq 0) {
    Write-Host "Demo completed successfully!" -ForegroundColor Green
} else {
    Write-Host "Demo completed with some issues (exit code: $testResult)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Green
