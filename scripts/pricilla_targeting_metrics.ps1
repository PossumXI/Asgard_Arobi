# PRICILLA Targeting Metrics Demo
# Simulates target acquisition, movement, and rapid replanning

param(
    [string]$BaseURL = "http://localhost:8092",
    [int]$Steps = 6
)

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "   PRICILLA Targeting Metrics Demo" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

function Invoke-JsonPost($url, $payload) {
    return Invoke-RestMethod -Uri $url -Method Post -Body ($payload | ConvertTo-Json -Depth 6) -ContentType "application/json"
}

function Invoke-JsonPut($url, $payload) {
    return Invoke-RestMethod -Uri $url -Method Put -Body ($payload | ConvertTo-Json -Depth 6) -ContentType "application/json"
}

$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$metricsDir = Join-Path $PSScriptRoot "..\test\e2e\metrics"
if (-not (Test-Path $metricsDir)) {
    New-Item -ItemType Directory -Path $metricsDir | Out-Null
}

$reportPath = Join-Path $metricsDir "pricilla_targeting_metrics_$timestamp.json"

Write-Host "Checking Pricilla service..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$BaseURL/health" -Method Get -TimeoutSec 5
    Write-Host "  OK Pricilla healthy: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "  ERROR Pricilla not reachable at $BaseURL" -ForegroundColor Red
    exit 1
}

$missionId = [guid]::NewGuid().ToString()
$payloadId = "payload-demo-001"

$start = @{ x = 0; y = 0; z = 300 }
$target = @{ x = 1500; y = 1000; z = 50 }

Write-Host ""
Write-Host "1) Target picked + mission created..." -ForegroundColor Yellow
$mission = @{
    id = $missionId
    type = "strike"
    payloadId = $payloadId
    payloadType = "uav"
    startPosition = $start
    targetPosition = $target
    priority = 1
    stealthRequired = $true
}
$missionResponse = Invoke-JsonPost "$BaseURL/api/v1/missions" $mission
Write-Host "  OK Mission created: $($missionResponse.id)" -ForegroundColor Green

Write-Host ""
Write-Host "2) Registering payload state..." -ForegroundColor Yellow
$payloadState = @{
    id = $payloadId
    type = "uav"
    position = $start
    velocity = @{ x = 80; y = 40; z = 0 }
    heading = 0
    fuel = 95
    battery = 98
    health = 99
    status = "navigating"
}
Invoke-JsonPost "$BaseURL/api/v1/payloads" $payloadState | Out-Null
Write-Host "  OK Payload registered" -ForegroundColor Green

Write-Host ""
Write-Host "3) Bootstrapping WiFi imaging..." -ForegroundColor Yellow
$router = @{
    id = "router-metrics-01"
    position = @{ x = 400; y = 200; z = 5 }
    frequencyGhz = 5.8
    txPowerDbm = 18
}
Invoke-JsonPost "$BaseURL/api/v1/wifi/routers" $router | Out-Null

$frame = @{
    routerId = "router-metrics-01"
    receiverId = $payloadId
    pathLossDb = 65
    multipathSpread = 5
    confidence = 0.9
}
Invoke-JsonPost "$BaseURL/api/v1/wifi/imaging" $frame | Out-Null
Write-Host "  OK WiFi imaging ingested" -ForegroundColor Green

Write-Host ""
Write-Host "4) Simulating moving target + rapid replans..." -ForegroundColor Yellow
$samples = @()

for ($i = 1; $i -le $Steps; $i++) {
    $target = @{
        x = 1500 + ($i * 120)
        y = 1000 + ($i * 80)
        z = 50
    }

    $payloadState.position = @{
        x = $start.x + ($i * 200)
        y = $start.y + ($i * 130)
        z = 280
    }

    Invoke-JsonPut "$BaseURL/api/v1/payloads/$payloadId" $payloadState | Out-Null
    Invoke-JsonPost "$BaseURL/api/v1/missions/target/$missionId" $target | Out-Null

    $metrics = Invoke-RestMethod -Uri "$BaseURL/api/v1/metrics/targeting" -Method Get
    $missionStatus = Invoke-RestMethod -Uri "$BaseURL/api/v1/missions/$missionId" -Method Get

    $samples += [pscustomobject]@{
        step = $i
        target = $target
        payload = $payloadState.position
        replanCount = $metrics.replanCount
        lastReplanReason = $metrics.lastReplanReason
        lastTrajectoryId = $metrics.lastTrajectoryId
        missionStatus = $missionStatus.status
        timestamp = (Get-Date).ToString("o")
    }

    Write-Host "  OK Step ${i}: target moved, replan count=$($metrics.replanCount)" -ForegroundColor Green
    Start-Sleep -Milliseconds 400
}

Write-Host ""
Write-Host "5) Final intercept + completion check..." -ForegroundColor Yellow
$payloadState.position = $target
Invoke-JsonPut "$BaseURL/api/v1/payloads/$payloadId" $payloadState | Out-Null

$finalMetrics = Invoke-RestMethod -Uri "$BaseURL/api/v1/metrics/targeting" -Method Get
$finalMission = Invoke-RestMethod -Uri "$BaseURL/api/v1/missions/$missionId" -Method Get

Write-Host "  OK Mission status: $($finalMission.status)" -ForegroundColor Green
Write-Host "  OK Completion distance: $([math]::Round($finalMetrics.completionDistance,2))m" -ForegroundColor Green

$report = [pscustomobject]@{
    missionId = $missionId
    payloadId = $payloadId
    startedAt = $timestamp
    samples = $samples
    finalMetrics = $finalMetrics
    finalMission = $finalMission
}

$report | ConvertTo-Json -Depth 8 | Set-Content -Path $reportPath

Write-Host ""
Write-Host "Report saved to:" -ForegroundColor Cyan
Write-Host "  $reportPath" -ForegroundColor White
Write-Host ""
Write-Host "Done!" -ForegroundColor Green
