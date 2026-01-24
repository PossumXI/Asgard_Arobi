# ASGARD Complete System Verification Script
# This script verifies the full system is built and ready for deployment

Write-Host "=== ASGARD COMPLETE SYSTEM VERIFICATION ===" -ForegroundColor Cyan
Write-Host ""

$ErrorCount = 0

# Check for binaries
Write-Host "Checking binaries..." -ForegroundColor Yellow
$binaries = @("nysus.exe", "percila.exe", "hunoid.exe", "giru.exe", "silenus.exe", "satnet_router.exe")
foreach ($bin in $binaries) {
    $path = "bin\$bin"
    if (Test-Path $path) {
        $size = (Get-Item $path).Length / 1MB
        Write-Host "  [OK] $bin ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "  [MISSING] $bin" -ForegroundColor Red
        $ErrorCount++
    }
}
Write-Host ""

# Check Kubernetes manifests
Write-Host "Checking Kubernetes manifests..." -ForegroundColor Yellow
$manifests = @(
    "deployments\kubernetes\namespace.yaml",
    "deployments\kubernetes\configmap.yaml",
    "deployments\kubernetes\secrets.yaml",
    "deployments\kubernetes\postgres.yaml",
    "deployments\kubernetes\mongodb.yaml",
    "deployments\kubernetes\nysus.yaml",
    "deployments\kubernetes\giru.yaml",
    "deployments\kubernetes\percila.yaml",
    "deployments\kubernetes\kustomization.yaml"
)
foreach ($manifest in $manifests) {
    if (Test-Path $manifest) {
        Write-Host "  [OK] $manifest" -ForegroundColor Green
    } else {
        Write-Host "  [MISSING] $manifest" -ForegroundColor Red
        $ErrorCount++
    }
}
Write-Host ""

# Run Go tests
Write-Host "Running Go tests..." -ForegroundColor Yellow
$testResult = go test ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  [OK] All Go tests passed" -ForegroundColor Green
} else {
    Write-Host "  [FAILED] Some tests failed" -ForegroundColor Red
    $ErrorCount++
}
Write-Host ""

# Run Go vet
Write-Host "Running Go vet..." -ForegroundColor Yellow
$vetResult = go vet ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  [OK] No vet warnings" -ForegroundColor Green
} else {
    Write-Host "  [WARNING] Some vet warnings" -ForegroundColor Yellow
}
Write-Host ""

# Check Go build
Write-Host "Verifying Go build..." -ForegroundColor Yellow
$buildResult = go build ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  [OK] All packages build successfully" -ForegroundColor Green
} else {
    Write-Host "  [FAILED] Build errors" -ForegroundColor Red
    $ErrorCount++
}
Write-Host ""

# Summary
Write-Host "=== VERIFICATION SUMMARY ===" -ForegroundColor Cyan
if ($ErrorCount -eq 0) {
    Write-Host "All checks passed! System is ready for deployment." -ForegroundColor Green
    Write-Host ""
    Write-Host "System Components:" -ForegroundColor White
    Write-Host "  - Nysus: Central orchestration (port 8080)" -ForegroundColor White
    Write-Host "  - Percila: AI guidance system (port 8089)" -ForegroundColor White
    Write-Host "  - Giru: Security & defense (port 9090)" -ForegroundColor White
    Write-Host "  - Silenus: Satellite perception" -ForegroundColor White
    Write-Host "  - Hunoid: Robotics control" -ForegroundColor White
    Write-Host "  - Sat_Net: DTN networking" -ForegroundColor White
    Write-Host ""
    Write-Host "To deploy to Kubernetes:" -ForegroundColor Yellow
    Write-Host "  kubectl apply -k deployments/kubernetes/" -ForegroundColor White
} else {
    Write-Host "Found $ErrorCount error(s). Please fix before deployment." -ForegroundColor Red
}
Write-Host ""
Write-Host "=== SYSTEM STATUS: $(if ($ErrorCount -eq 0) { 'PRODUCTION READY' } else { 'NEEDS ATTENTION' }) ===" -ForegroundColor $(if ($ErrorCount -eq 0) { 'Green' } else { 'Red' })
