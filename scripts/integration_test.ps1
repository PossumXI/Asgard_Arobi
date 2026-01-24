# ASGARD Integration Test Script
# Tests all components working together

Write-Host "=== ASGARD Integration Testing ===" -ForegroundColor Cyan

$testResults = @()
$allPassed = $true

# Test 1: Database connectivity
Write-Host "`n[TEST 1] Database Connectivity..." -ForegroundColor Yellow
try {
    $result = & ".\bin\db_migrate.exe" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "PASS: Database connectivity test" -ForegroundColor Green
        $testResults += @{Test="Database Connectivity"; Status="PASS"}
    } else {
        Write-Host "FAIL: Database connectivity test" -ForegroundColor Red
        $testResults += @{Test="Database Connectivity"; Status="FAIL"}
        $allPassed = $false
    }
} catch {
    Write-Host "FAIL: Database connectivity test - $($_.Exception.Message)" -ForegroundColor Red
    $testResults += @{Test="Database Connectivity"; Status="FAIL"}
    $allPassed = $false
}

# Test 2: Nysus API health check
Write-Host "`n[TEST 2] Nysus API Health..." -ForegroundColor Yellow
try {
    Start-Process -FilePath ".\bin\nysus.exe" -ArgumentList "-addr", ":8080" -NoNewWindow -PassThru | Out-Null
    Start-Sleep -Seconds 5
    
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
        Write-Host "PASS: Nysus API health check" -ForegroundColor Green
        $testResults += @{Test="Nysus API Health"; Status="PASS"}
    } else {
        Write-Host "FAIL: Nysus API health check" -ForegroundColor Red
        $testResults += @{Test="Nysus API Health"; Status="FAIL"}
        $allPassed = $false
    }
    
    Stop-Process -Name "nysus" -Force -ErrorAction SilentlyContinue
} catch {
    Write-Host "FAIL: Nysus API health check - $($_.Exception.Message)" -ForegroundColor Red
    $testResults += @{Test="Nysus API Health"; Status="FAIL"}
    $allPassed = $false
    Stop-Process -Name "nysus" -Force -ErrorAction SilentlyContinue
}

# Test 3: Silenus service startup
Write-Host "`n[TEST 3] Silenus Service..." -ForegroundColor Yellow
try {
    $process = Start-Process -FilePath ".\bin\silenus.exe" -ArgumentList "-id", "test-sat" -NoNewWindow -PassThru
    Start-Sleep -Seconds 3
    
    if (-not $process.HasExited) {
        Write-Host "PASS: Silenus service startup" -ForegroundColor Green
        $testResults += @{Test="Silenus Service"; Status="PASS"}
        Stop-Process -Id $process.Id -Force
    } else {
        Write-Host "FAIL: Silenus service exited unexpectedly" -ForegroundColor Red
        $testResults += @{Test="Silenus Service"; Status="FAIL"}
        $allPassed = $false
    }
} catch {
    Write-Host "FAIL: Silenus service test - $($_.Exception.Message)" -ForegroundColor Red
    $testResults += @{Test="Silenus Service"; Status="FAIL"}
    $allPassed = $false
}

# Test 4: Hunoid service startup
Write-Host "`n[TEST 4] Hunoid Service..." -ForegroundColor Yellow
try {
    $process = Start-Process -FilePath ".\bin\hunoid.exe" -ArgumentList "-id", "test-hunoid" -NoNewWindow -PassThru
    Start-Sleep -Seconds 3
    
    if (-not $process.HasExited) {
        Write-Host "PASS: Hunoid service startup" -ForegroundColor Green
        $testResults += @{Test="Hunoid Service"; Status="PASS"}
        Stop-Process -Id $process.Id -Force
    } else {
        Write-Host "FAIL: Hunoid service exited unexpectedly" -ForegroundColor Red
        $testResults += @{Test="Hunoid Service"; Status="FAIL"}
        $allPassed = $false
    }
} catch {
    Write-Host "FAIL: Hunoid service test - $($_.Exception.Message)" -ForegroundColor Red
    $testResults += @{Test="Hunoid Service"; Status="FAIL"}
    $allPassed = $false
}

# Test 5: Giru service startup
Write-Host "`n[TEST 5] Giru Service..." -ForegroundColor Yellow
try {
    $process = Start-Process -FilePath ".\bin\giru.exe" -NoNewWindow -PassThru
    Start-Sleep -Seconds 3
    
    if (-not $process.HasExited) {
        Write-Host "PASS: Giru service startup" -ForegroundColor Green
        $testResults += @{Test="Giru Service"; Status="PASS"}
        Stop-Process -Id $process.Id -Force
    } else {
        Write-Host "FAIL: Giru service exited unexpectedly" -ForegroundColor Red
        $testResults += @{Test="Giru Service"; Status="FAIL"}
        $allPassed = $false
    }
} catch {
    Write-Host "FAIL: Giru service test - $($_.Exception.Message)" -ForegroundColor Red
    $testResults += @{Test="Giru Service"; Status="FAIL"}
    $allPassed = $false
}

# Test 6: Binary compilation check
Write-Host "`n[TEST 6] Binary Compilation..." -ForegroundColor Yellow
$binaries = @("nysus.exe", "silenus.exe", "hunoid.exe", "giru.exe", "db_migrate.exe")
$allBinariesExist = $true

foreach ($bin in $binaries) {
    if (Test-Path ".\bin\$bin") {
        Write-Host "  ✓ $bin exists" -ForegroundColor Green
    } else {
        Write-Host "  ✗ $bin missing" -ForegroundColor Red
        $allBinariesExist = $false
    }
}

if ($allBinariesExist) {
    Write-Host "PASS: All binaries compiled" -ForegroundColor Green
    $testResults += @{Test="Binary Compilation"; Status="PASS"}
} else {
    Write-Host "FAIL: Some binaries missing" -ForegroundColor Red
    $testResults += @{Test="Binary Compilation"; Status="FAIL"}
    $allPassed = $false
}

# Test 7: Go integration tests
Write-Host "`n[TEST 7] Go Integration Tests..." -ForegroundColor Yellow
try {
    $result = & go test ./test/integration/... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "PASS: Go integration tests" -ForegroundColor Green
        $testResults += @{Test="Go Integration Tests"; Status="PASS"}
    } else {
        Write-Host "FAIL: Go integration tests" -ForegroundColor Red
        $testResults += @{Test="Go Integration Tests"; Status="FAIL"}
        $allPassed = $false
    }
} catch {
    Write-Host "FAIL: Go integration tests - $($_.Exception.Message)" -ForegroundColor Red
    $testResults += @{Test="Go Integration Tests"; Status="FAIL"}
    $allPassed = $false
}

# Test 8: Optional load tests (set ASGARD_RUN_LOAD_TESTS=1)
Write-Host "`n[TEST 8] Load Tests (optional)..." -ForegroundColor Yellow
if ($env:ASGARD_RUN_LOAD_TESTS -eq "1") {
    try {
        & ".\scripts\load_test_signaling.ps1"
        & ".\scripts\load_test_realtime.ps1"
        if ($LASTEXITCODE -eq 0) {
            Write-Host "PASS: Load tests executed" -ForegroundColor Green
            $testResults += @{Test="Load Tests"; Status="PASS"}
        } else {
            Write-Host "FAIL: Load tests reported errors" -ForegroundColor Red
            $testResults += @{Test="Load Tests"; Status="FAIL"}
            $allPassed = $false
        }
    } catch {
        Write-Host "FAIL: Load tests - $($_.Exception.Message)" -ForegroundColor Red
        $testResults += @{Test="Load Tests"; Status="FAIL"}
        $allPassed = $false
    }
} else {
    Write-Host "SKIP: Set ASGARD_RUN_LOAD_TESTS=1 to run load tests" -ForegroundColor DarkYellow
    $testResults += @{Test="Load Tests"; Status="SKIP"}
}

# Summary
Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
foreach ($result in $testResults) {
    $color = if ($result.Status -eq "PASS") { "Green" } else { "Red" }
    Write-Host "$($result.Test): $($result.Status)" -ForegroundColor $color
}

if ($allPassed) {
    Write-Host "`n=== ALL TESTS PASSED ===" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n=== SOME TESTS FAILED ===" -ForegroundColor Red
    exit 1
}
