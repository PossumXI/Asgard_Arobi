# ASGARD Docker Log Monitor
# Automated monitoring, error detection, and documentation of Docker container logs
# Usage: .\docker_monitor.ps1 [-Continuous] [-IntervalMinutes 5] [-FixErrors]

param(
    [switch]$Continuous = $false,
    [int]$IntervalMinutes = 5,
    [switch]$FixErrors = $false
)

$ErrorActionPreference = "Continue"
$AsgardRoot = if ($env:ASGARD_ROOT) {
    $env:ASGARD_ROOT
} else {
    Resolve-Path (Join-Path $PSScriptRoot "..")
}
$DockerLogPath = "$AsgardRoot\Documentation\Docker_Logs.md"
$ComposeFile = "$AsgardRoot\Data\docker-compose.yml"

# Container names to monitor (from docker-compose.yml)
$AsgardContainers = @(
    "asgard_postgres",
    "asgard_mongodb", 
    "asgard_nats",
    "asgard_redis"
)

# Error patterns to detect
$ErrorPatterns = @{
    "CRITICAL" = @(
        "FATAL",
        "panic:",
        "SECURITY ATTACK",
        "Segmentation fault",
        "Out of memory",
        "killed"
    )
    "ERROR" = @(
        "ERROR",
        "error:",
        "failed",
        "FAIL",
        "exception",
        "Connection refused",
        "Permission denied"
    )
    "WARNING" = @(
        "WARNING",
        "WARN",
        "deprecated",
        "unhealthy",
        "timeout",
        "retry"
    )
}

# Known issues and their fixes
$KnownFixes = @{
    "SECURITY ATTACK detected.*Cross Protocol Scripting" = @{
        Description = "Redis exposed to external requests (potential CSRF/XSS attack)"
        Fix = "Bind Redis to localhost only or configure firewall rules"
        AutoFix = $false
    }
    "wget.*not found" = @{
        Description = "NATS healthcheck fails - wget not available in container"
        Fix = "Update docker-compose.yml to use nats healthcheck endpoint"
        AutoFix = $true
        FixAction = "Update-NatsHealthcheck"
    }
    "Connection not authenticating" = @{
        Description = "MongoDB connection without authentication (healthcheck behavior)"
        Fix = "This is normal for healthcheck probes using local connection"
        AutoFix = $false
        Severity = "INFO"
    }
}

function Write-DockerLog {
    param(
        [string]$Message,
        [string]$Level = "INFO",
        [switch]$ToFile
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $entry = "[$timestamp] [$Level] $Message"
    
    switch ($Level) {
        "INFO"     { Write-Host $entry -ForegroundColor Cyan }
        "SUCCESS"  { Write-Host $entry -ForegroundColor Green }
        "WARNING"  { Write-Host $entry -ForegroundColor Yellow }
        "ERROR"    { Write-Host $entry -ForegroundColor Red }
        "CRITICAL" { Write-Host $entry -ForegroundColor Magenta }
        default    { Write-Host $entry }
    }
    
    $writeToFile = $true
    if ($PSBoundParameters.ContainsKey("ToFile")) {
        $writeToFile = [bool]$ToFile
    }

    if ($writeToFile) {
        Add-Content -Path $DockerLogPath -Value $entry -ErrorAction SilentlyContinue
    }
}

function Initialize-DockerLog {
    if (-not (Test-Path $DockerLogPath)) {
        $header = @"
# ASGARD Docker Container Logs

This file is automatically maintained by the Docker log monitor script.
It tracks container status, errors, and warnings across all ASGARD services.

## Monitored Containers

| Container | Service | Port(s) |
|-----------|---------|---------|
| asgard_postgres | PostgreSQL/PostGIS | 55432 -> 5432 |
| asgard_mongodb | MongoDB | 27018 -> 27017 |
| asgard_nats | NATS JetStream | 4222, 8222, 6222 |
| asgard_redis | Redis | 6379 |

---

## Activity Log

"@
        Set-Content -Path $DockerLogPath -Value $header
    }
}

function Get-ContainerStatus {
    $status = @{}
    
    foreach ($container in $AsgardContainers) {
        try {
            $info = docker inspect $container 2>$null | ConvertFrom-Json
            if ($info) {
                $status[$container] = @{
                    Running = $info.State.Running
                    Status = $info.State.Status
                    Health = if ($info.State.Health) { $info.State.Health.Status } else { "none" }
                    StartedAt = $info.State.StartedAt
                    RestartCount = $info.RestartCount
                }
            }
        }
        catch {
            $status[$container] = @{
                Running = $false
                Status = "not_found"
                Health = "unknown"
                Error = $_.Exception.Message
            }
        }
    }
    
    return $status
}

function Get-ContainerLogs {
    param(
        [string]$ContainerName,
        [int]$TailLines = 100,
        [string]$Since = "5m"
    )
    
    try {
        $logs = docker logs $ContainerName --tail $TailLines --since $Since 2>&1
        return $logs -join "`n"
    }
    catch {
        return "ERROR: Could not retrieve logs - $_"
    }
}

function Find-LogErrors {
    param([string]$LogContent)
    
    $findings = @{
        CRITICAL = @()
        ERROR = @()
        WARNING = @()
    }
    
    $lines = $LogContent -split "`n"
    
    foreach ($line in $lines) {
        foreach ($severity in $ErrorPatterns.Keys) {
            foreach ($pattern in $ErrorPatterns[$severity]) {
                if ($line -match $pattern) {
                    $findings[$severity] += @{
                        Line = $line.Trim()
                        Pattern = $pattern
                    }
                    break
                }
            }
        }
    }
    
    return $findings
}

function Get-SuggestedFix {
    param([string]$ErrorLine)
    
    foreach ($pattern in $KnownFixes.Keys) {
        if ($ErrorLine -match $pattern) {
            return $KnownFixes[$pattern]
        }
    }
    
    return $null
}

function Update-NatsHealthcheck {
    Write-DockerLog "Applying fix: Updating NATS healthcheck in docker-compose.yml" "INFO"
    
    $composeContent = Get-Content $ComposeFile -Raw
    
    # Replace wget healthcheck with curl/nc alternative or NATS native
    if ($composeContent -match "wget.*8222/healthz") {
        # Actually need to modify the YAML properly
        Write-DockerLog "NATS healthcheck needs manual update - wget not available in NATS container" "WARNING"
        Write-DockerLog "Suggested: Use 'test: [\"CMD-SHELL\", \"nats-server --signal check\"]' or remove healthcheck" "INFO"
        return $false
    }
    
    return $true
}

function Write-StatusReport {
    param(
        $ContainerStatus,
        $Errors
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    
    $report = @"

---

## Status Report: $timestamp

### Container Health

| Container | Status | Health | Uptime |
|-----------|--------|--------|--------|
"@
    
    foreach ($container in $ContainerStatus.Keys) {
        $s = $ContainerStatus[$container]
        $uptime = if ($s.StartedAt) { 
            $start = [DateTime]::Parse($s.StartedAt)
            $span = (Get-Date) - $start
            "{0}h {1}m" -f [int]$span.TotalHours, $span.Minutes
        } else { "N/A" }
        
        $healthEmoji = switch ($s.Health) {
            "healthy" { "OK" }
            "unhealthy" { "UNHEALTHY" }
            "starting" { "STARTING" }
            default { "-" }
        }
        
        $report += "| $container | $($s.Status) | $healthEmoji | $uptime |`n"
    }
    
    # Add errors section if any
    $hasErrors = $false
    foreach ($container in $Errors.Keys) {
        $containerErrors = $Errors[$container]
        $totalErrors = $containerErrors.CRITICAL.Count + $containerErrors.ERROR.Count + $containerErrors.WARNING.Count
        if ($totalErrors -gt 0) {
            $hasErrors = $true
            break
        }
    }
    
    if ($hasErrors) {
        $report += "`n### Detected Issues`n`n"
        
        foreach ($container in $Errors.Keys) {
            $containerErrors = $Errors[$container]
            
            if ($containerErrors.CRITICAL.Count -gt 0) {
                $report += "#### CRITICAL - $container`n"
                foreach ($err in $containerErrors.CRITICAL) {
                    $report += "- ``$($err.Line.Substring(0, [Math]::Min(100, $err.Line.Length)))...```n"
                    $fix = Get-SuggestedFix -ErrorLine $err.Line
                    if ($fix) {
                        $report += "  - **Fix**: $($fix.Description) - $($fix.Fix)`n"
                    }
                }
            }
            
            if ($containerErrors.ERROR.Count -gt 0) {
                $report += "#### ERRORS - $container`n"
                foreach ($err in ($containerErrors.ERROR | Select-Object -First 5)) {
                    $report += "- ``$($err.Line.Substring(0, [Math]::Min(100, $err.Line.Length)))...```n"
                }
                if ($containerErrors.ERROR.Count -gt 5) {
                    $report += "- ... and $($containerErrors.ERROR.Count - 5) more errors`n"
                }
            }
            
            if ($containerErrors.WARNING.Count -gt 0) {
                $report += "#### WARNINGS - $container`n"
                foreach ($warn in ($containerErrors.WARNING | Select-Object -First 3)) {
                    $report += "- ``$($warn.Line.Substring(0, [Math]::Min(100, $warn.Line.Length)))...```n"
                }
                if ($containerErrors.WARNING.Count -gt 3) {
                    $report += "- ... and $($containerErrors.WARNING.Count - 3) more warnings`n"
                }
            }
        }
    }
    else {
        $report += "`n### Status: All Clear`n`nNo critical errors or warnings detected in this monitoring interval.`n"
    }
    
    Add-Content -Path $DockerLogPath -Value $report
    
    return $hasErrors
}

function Invoke-DockerMonitor {
    $checkTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    
    Write-DockerLog "Starting Docker log scan at $checkTime" "INFO"
    
    # Check if Docker is running
    try {
        $dockerInfo = docker info 2>$null
        if (-not $dockerInfo) {
            Write-DockerLog "Docker daemon is not running!" "CRITICAL"
            return @{ Success = $false; Error = "Docker not running" }
        }
    }
    catch {
        Write-DockerLog "Cannot connect to Docker: $_" "CRITICAL"
        return @{ Success = $false; Error = $_.Exception.Message }
    }
    
    # Get container status
    $containerStatus = Get-ContainerStatus
    
    $runningCount = ($containerStatus.Values | Where-Object { $_.Running }).Count
    $unhealthyCount = ($containerStatus.Values | Where-Object { $_.Health -eq "unhealthy" }).Count
    
    Write-DockerLog "Containers: $runningCount running, $unhealthyCount unhealthy" "INFO"
    
    # Collect and analyze logs from each container
    $allErrors = @{}
    $totalCritical = 0
    $totalErrors = 0
    $totalWarnings = 0
    
    foreach ($container in $AsgardContainers) {
        if ($containerStatus[$container].Running) {
            $logs = Get-ContainerLogs -ContainerName $container -TailLines 200 -Since "10m"
            $errors = Find-LogErrors -LogContent $logs
            
            $allErrors[$container] = $errors
            $totalCritical += $errors.CRITICAL.Count
            $totalErrors += $errors.ERROR.Count
            $totalWarnings += $errors.WARNING.Count
            
            if ($errors.CRITICAL.Count -gt 0) {
                Write-DockerLog "[$container] $($errors.CRITICAL.Count) CRITICAL issues detected!" "CRITICAL"
                
                foreach ($crit in $errors.CRITICAL) {
                    $fix = Get-SuggestedFix -ErrorLine $crit.Line
                    if ($fix) {
                        Write-DockerLog "  Suggested fix: $($fix.Fix)" "INFO"
                        
                        if ($FixErrors -and $fix.AutoFix -and $fix.FixAction) {
                            Write-DockerLog "  Attempting auto-fix..." "INFO"
                            & $fix.FixAction
                        }
                    }
                }
            }
        }
        else {
            Write-DockerLog "[$container] Container not running (Status: $($containerStatus[$container].Status))" "WARNING"
            $totalWarnings++
        }
    }
    
    # Write status report to documentation
    Write-StatusReport -ContainerStatus $containerStatus -Errors $allErrors
    
    # Summary
    if ($totalCritical -gt 0) {
        Write-DockerLog "SUMMARY: $totalCritical CRITICAL, $totalErrors errors, $totalWarnings warnings" "CRITICAL"
    }
    elseif ($totalErrors -gt 0) {
        Write-DockerLog "SUMMARY: $totalErrors errors, $totalWarnings warnings" "ERROR"
    }
    elseif ($totalWarnings -gt 0) {
        Write-DockerLog "SUMMARY: $totalWarnings warnings" "WARNING"
    }
    else {
        Write-DockerLog "SUMMARY: All containers healthy, no issues detected" "SUCCESS"
    }
    
    return @{
        Success = $true
        ContainerStatus = $containerStatus
        Errors = $allErrors
        Summary = @{
            Critical = $totalCritical
            Errors = $totalErrors
            Warnings = $totalWarnings
            Running = $runningCount
            Unhealthy = $unhealthyCount
        }
    }
}

function Show-RecentLogs {
    param(
        [string]$ContainerName = "",
        [int]$Lines = 50
    )
    
    if ($ContainerName) {
        $containers = @($ContainerName)
    }
    else {
        $containers = $AsgardContainers
    }
    
    foreach ($container in $containers) {
        Write-Host "`n========== $container ==========" -ForegroundColor Cyan
        docker logs $container --tail $Lines 2>&1
    }
}

# Main execution
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  ASGARD Docker Log Monitor" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Initialize-DockerLog

if ($Continuous) {
    Write-Host "Running in CONTINUOUS mode (Ctrl+C to stop)" -ForegroundColor Yellow
    Write-Host "Monitor interval: $IntervalMinutes minutes" -ForegroundColor Yellow
    if ($FixErrors) {
        Write-Host "Auto-fix mode: ENABLED" -ForegroundColor Green
    }
    Write-Host ""
    
    while ($true) {
        $result = Invoke-DockerMonitor
        
        if ($result.Summary.Critical -gt 0) {
            Write-Host ""
            Write-Host ">>> CRITICAL ISSUES DETECTED! <<<" -ForegroundColor Red
            Write-Host ">>> Review Docker_Logs.md for details <<<" -ForegroundColor Red
            Write-Host ""
            [Console]::Beep(1000, 500)
            [Console]::Beep(1000, 500)
        }
        elseif ($result.Summary.Unhealthy -gt 0) {
            Write-Host ""
            Write-Host ">>> Container health issues detected <<<" -ForegroundColor Yellow
            [Console]::Beep(800, 300)
        }
        
        Write-Host ""
        Write-Host "Next scan in $IntervalMinutes minutes... ($(Get-Date -Format 'HH:mm:ss'))" -ForegroundColor Gray
        Start-Sleep -Seconds ($IntervalMinutes * 60)
    }
}
else {
    # Single run mode
    $result = Invoke-DockerMonitor
    
    Write-Host ""
    if ($result.Summary.Critical -gt 0) {
        Write-Host "CRITICAL ISSUES FOUND! Review Docker_Logs.md" -ForegroundColor Red
    }
    elseif ($result.Summary.Errors -gt 0) {
        Write-Host "Errors detected. Review Docker_Logs.md" -ForegroundColor Yellow
    }
    else {
        Write-Host "All containers healthy." -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "Commands:" -ForegroundColor Gray
    Write-Host "  Continuous monitor:  .\docker_monitor.ps1 -Continuous" -ForegroundColor Gray
    Write-Host "  With auto-fix:       .\docker_monitor.ps1 -Continuous -FixErrors" -ForegroundColor Gray
    Write-Host "  Change interval:     .\docker_monitor.ps1 -Continuous -IntervalMinutes 10" -ForegroundColor Gray
}
