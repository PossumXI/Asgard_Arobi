# ASGARD Demo Video Generator
# This script runs the Playwright demo test and generates a video

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "   ASGARD Demo Video Generator" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Ensure we run from the e2e directory so Playwright
# uses the local config and dependencies.
Set-Location $PSScriptRoot

# Check if apps are running
$websitesRunning = $false
$hubsRunning = $false

try {
    $response = Invoke-WebRequest -Uri "http://localhost:3000" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
    if ($response.StatusCode -eq 200) { $websitesRunning = $true }
} catch {}

try {
    $response = Invoke-WebRequest -Uri "http://localhost:3001" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
    if ($response.StatusCode -eq 200) { $hubsRunning = $true }
} catch {}

Write-Host "Pre-flight Check:" -ForegroundColor Yellow
Write-Host "  - Websites App (3000): $(if ($websitesRunning) { 'RUNNING' } else { 'NOT RUNNING' })" -ForegroundColor $(if ($websitesRunning) { 'Green' } else { 'Red' })
Write-Host "  - Hubs App (3001): $(if ($hubsRunning) { 'RUNNING' } else { 'NOT RUNNING' })" -ForegroundColor $(if ($hubsRunning) { 'Green' } else { 'Red' })
Write-Host ""

if (-not $websitesRunning -or -not $hubsRunning) {
    Write-Host "WARNING: One or more apps are not running!" -ForegroundColor Yellow
    Write-Host "The demo will attempt to start them automatically via Playwright config." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "For best results, start the apps manually first:" -ForegroundColor Yellow
    Write-Host "  Terminal 1: cd Websites && npm run dev" -ForegroundColor Gray
    Write-Host "  Terminal 2: cd Hubs && npm run dev" -ForegroundColor Gray
    Write-Host ""
    
    $continue = Read-Host "Continue anyway? (y/n)"
    if ($continue -ne "y") {
        Write-Host "Aborted." -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "Starting Demo Recording..." -ForegroundColor Green
Write-Host ""

# Run the Playwright test
npx playwright test demo.spec.ts --config playwright.config.ts

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
        Write-Host ""
        
        # Open the video
        $open = Read-Host "Open video now? (y/n)"
        if ($open -eq "y") {
            Start-Process $latestVideo.FullName
        }
    }
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Green
