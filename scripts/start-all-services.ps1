#!/usr/bin/env pwsh
<#
.SYNOPSIS
    ASGARD Local Services Launcher - Start all backend services with live persistence.

.DESCRIPTION
    Starts all ASGARD backend Go services (Valkyrie, Giru, Hunoid, Pricilla, Vault,
    Silenus, Nysus, Notifications) and optionally the frontend dev servers (Websites, Hubs).
    Services run with hardware bypass flags for local development.

    Services persist until you press Ctrl+C or run the stop script.

.PARAMETER SkipFrontends
    Skip starting Websites and Hubs frontend dev servers.

.PARAMETER SkipBuild
    Skip rebuilding Go binaries (use existing bin/ executables).

.PARAMETER Headed
    After starting services, also run the flight simulation demo in headed mode.

.EXAMPLE
    .\scripts\start-all-services.ps1
    .\scripts\start-all-services.ps1 -SkipFrontends
    .\scripts\start-all-services.ps1 -SkipBuild
#>

param(
    [switch]$SkipFrontends,
    [switch]$SkipBuild,
    [switch]$Headed,
    [switch]$Help
)

$ErrorActionPreference = 'Continue'
$ASGARD_ROOT = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if (-not $ASGARD_ROOT -or -not (Test-Path "$ASGARD_ROOT\go.mod")) {
    $ASGARD_ROOT = Split-Path -Parent $PSScriptRoot
}
if (-not (Test-Path "$ASGARD_ROOT\go.mod")) {
    $ASGARD_ROOT = $PSScriptRoot
}
# Final fallback: use the directory containing this script's parent
$BIN_DIR = Join-Path $ASGARD_ROOT "bin"

# Track started processes for cleanup
$script:StartedProcesses = @()

function Write-Banner {
    Write-Host ""
    Write-Host "  ============================================================" -ForegroundColor Cyan
    Write-Host "        ASGARD - Autonomous Space Guardian & Robotic Defense" -ForegroundColor Cyan
    Write-Host "                    Local Services Launcher" -ForegroundColor DarkCyan
    Write-Host "  ============================================================" -ForegroundColor Cyan
    Write-Host ""
}

function Write-ServiceStatus {
    param([string]$Name, [string]$Status, [int]$Port, [string]$Color = "White")
    $icon = if ($Status -eq "ONLINE") { "[OK]" } elseif ($Status -eq "STARTING") { "[..]" } else { "[!!]" }
    $fg = if ($Status -eq "ONLINE") { "Green" } elseif ($Status -eq "STARTING") { "Yellow" } else { "Red" }
    Write-Host "  $icon " -ForegroundColor $fg -NoNewline
    Write-Host "$Name" -ForegroundColor $Color -NoNewline
    Write-Host " :$Port " -ForegroundColor DarkGray -NoNewline
    Write-Host "[$Status]" -ForegroundColor $fg
}

function Test-ServiceHealth {
    param([int]$Port, [string]$Endpoint = "/health")
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port$Endpoint" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        return $response.StatusCode -eq 200
    } catch {
        return $false
    }
}

function Wait-ForService {
    param([int]$Port, [string]$Endpoint, [int]$TimeoutSeconds = 15, [string]$Name)
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        if (Test-ServiceHealth -Port $Port -Endpoint $Endpoint) {
            return $true
        }
        Start-Sleep -Milliseconds 500
        $elapsed += 0.5
    }
    return $false
}

function Start-GoService {
    param(
        [string]$Name,
        [string]$Executable,
        [string[]]$Arguments,
        [int]$Port,
        [string]$HealthEndpoint,
        [hashtable]$EnvVars = @{}
    )

    # Check if already running
    if (Test-ServiceHealth -Port $Port -Endpoint $HealthEndpoint) {
        Write-ServiceStatus -Name $Name -Status "ONLINE" -Port $Port -Color "White"
        return $true
    }

    $exePath = Join-Path $BIN_DIR $Executable
    if (-not (Test-Path $exePath)) {
        Write-Host "  [!!] Binary not found: $exePath" -ForegroundColor Red
        return $false
    }

    Write-ServiceStatus -Name $Name -Status "STARTING" -Port $Port

    # Set environment variables for this process
    $envBlock = @{}
    foreach ($key in [System.Environment]::GetEnvironmentVariables().Keys) {
        $envBlock[$key] = [System.Environment]::GetEnvironmentVariable($key)
    }
    foreach ($key in $EnvVars.Keys) {
        $envBlock[$key] = $EnvVars[$key]
    }

    # Start the process
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $exePath
    $psi.Arguments = $Arguments -join " "
    $psi.WorkingDirectory = $ASGARD_ROOT
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true

    foreach ($key in $EnvVars.Keys) {
        $psi.EnvironmentVariables[$key] = $EnvVars[$key]
    }

    try {
        $proc = [System.Diagnostics.Process]::Start($psi)
        $script:StartedProcesses += $proc

        # Wait for health
        $healthy = Wait-ForService -Port $Port -Endpoint $HealthEndpoint -TimeoutSeconds 15 -Name $Name
        if ($healthy) {
            Write-ServiceStatus -Name $Name -Status "ONLINE" -Port $Port -Color "Green"
            return $true
        } else {
            Write-ServiceStatus -Name $Name -Status "FAILED" -Port $Port -Color "Red"
            return $false
        }
    } catch {
        Write-Host "  [!!] Failed to start $Name`: $_" -ForegroundColor Red
        return $false
    }
}

function Start-FrontendService {
    param([string]$Name, [string]$Directory, [int]$Port)

    # Check if already running
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        if ($response.StatusCode -eq 200) {
            Write-ServiceStatus -Name $Name -Status "ONLINE" -Port $Port -Color "White"
            return $true
        }
    } catch { }

    $dir = Join-Path $ASGARD_ROOT $Directory
    if (-not (Test-Path (Join-Path $dir "package.json"))) {
        Write-Host "  [!!] $Name package.json not found at $dir" -ForegroundColor Red
        return $false
    }

    Write-ServiceStatus -Name $Name -Status "STARTING" -Port $Port

    try {
        $proc = Start-Process -FilePath "npm" -ArgumentList "run dev" -WorkingDirectory $dir -WindowStyle Hidden -PassThru
        $script:StartedProcesses += $proc
        Start-Sleep -Seconds 5

        $healthy = Test-ServiceHealth -Port $Port -Endpoint "/"
        if ($healthy) {
            Write-ServiceStatus -Name $Name -Status "ONLINE" -Port $Port -Color "Green"
        } else {
            Write-ServiceStatus -Name $Name -Status "STARTING" -Port $Port -Color "Yellow"
            Write-Host "         (may still be compiling)" -ForegroundColor DarkGray
        }
        return $true
    } catch {
        Write-Host "  [!!] Failed to start $Name`: $_" -ForegroundColor Red
        return $false
    }
}

function Stop-AllServices {
    Write-Host ""
    Write-Host "  Shutting down ASGARD services..." -ForegroundColor Yellow

    foreach ($proc in $script:StartedProcesses) {
        if ($proc -and -not $proc.HasExited) {
            try {
                $proc.Kill()
                $proc.WaitForExit(3000)
            } catch { }
        }
    }

    # Also kill any remaining service processes by name
    $serviceNames = @("valkyrie", "giru", "hunoid", "pricilla", "vault", "silenus", "nysus", "notifications")
    foreach ($name in $serviceNames) {
        Get-Process -Name $name -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    }

    Write-Host "  All services stopped." -ForegroundColor Green
    Write-Host ""
}

function Build-Services {
    Write-Host "  Building Go services..." -ForegroundColor Yellow

    $services = @(
        @{ Name = "valkyrie"; Path = "./Valkyrie/cmd/valkyrie" },
        @{ Name = "giru"; Path = "./cmd/giru" },
        @{ Name = "hunoid"; Path = "./cmd/hunoid" },
        @{ Name = "pricilla"; Path = "./Pricilla/cmd/pricilla" },
        @{ Name = "vault"; Path = "./cmd/vault" },
        @{ Name = "silenus"; Path = "./cmd/silenus" },
        @{ Name = "nysus"; Path = "./cmd/nysus" },
        @{ Name = "notifications"; Path = "./cmd/notifications" }
    )

    foreach ($svc in $services) {
        $output = Join-Path $BIN_DIR "$($svc.Name).exe"
        Write-Host "    Building $($svc.Name)..." -ForegroundColor DarkGray -NoNewline
        try {
            $result = & go build -o $output $svc.Path 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host " OK" -ForegroundColor Green
            } else {
                Write-Host " FAILED" -ForegroundColor Red
                Write-Host "    $result" -ForegroundColor DarkRed
            }
        } catch {
            Write-Host " ERROR: $_" -ForegroundColor Red
        }
    }
    Write-Host ""
}

function Show-Dashboard {
    Write-Host ""
    Write-Host "  ============================================================" -ForegroundColor Cyan
    Write-Host "                    Service Health Dashboard" -ForegroundColor Cyan
    Write-Host "  ============================================================" -ForegroundColor Cyan
    Write-Host ""

    $services = @(
        @{ Name = "Valkyrie (Flight)"; Port = 8093; Endpoint = "/health" },
        @{ Name = "GIRU (Security)"; Port = 9090; Endpoint = "/health" },
        @{ Name = "Hunoid (Robotics)"; Port = 8090; Endpoint = "/api/status" },
        @{ Name = "Pricilla (Guidance)"; Port = 8089; Endpoint = "/health" },
        @{ Name = "Vault (Secrets)"; Port = 8094; Endpoint = "/vault/health" },
        @{ Name = "Silenus (Satellite)"; Port = 9094; Endpoint = "/healthz" },
        @{ Name = "Nysus (Orchestration)"; Port = 8080; Endpoint = "/health" },
        @{ Name = "Notifications"; Port = 8095; Endpoint = "/api/notifications/status" }
    )

    $onlineCount = 0
    foreach ($svc in $services) {
        $healthy = Test-ServiceHealth -Port $svc.Port -Endpoint $svc.Endpoint
        $status = if ($healthy) { "ONLINE"; $onlineCount++ } else { "OFFLINE" }
        Write-ServiceStatus -Name $svc.Name -Status $status -Port $svc.Port
    }

    Write-Host ""
    Write-Host "  Backend: $onlineCount/$($services.Count) services online" -ForegroundColor $(if ($onlineCount -eq $services.Count) { "Green" } else { "Yellow" })

    if (-not $SkipFrontends) {
        Write-Host ""
        Write-Host "  Frontends:" -ForegroundColor DarkCyan
        $webOk = Test-ServiceHealth -Port 3000 -Endpoint "/"
        $hubOk = Test-ServiceHealth -Port 3001 -Endpoint "/"
        Write-ServiceStatus -Name "Websites" -Status $(if ($webOk) { "ONLINE" } else { "OFFLINE" }) -Port 3000
        Write-ServiceStatus -Name "Hubs" -Status $(if ($hubOk) { "ONLINE" } else { "OFFLINE" }) -Port 3001
    }

    Write-Host ""
}

# ============================================================
# MAIN EXECUTION
# ============================================================

if ($Help) {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    exit 0
}

Write-Banner

# Resolve ASGARD_ROOT
Set-Location $ASGARD_ROOT
Write-Host "  Project root: $ASGARD_ROOT" -ForegroundColor DarkGray
Write-Host ""

# Build if needed
if (-not $SkipBuild) {
    Build-Services
}

# Register cleanup handler
$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Stop-AllServices }
trap { Stop-AllServices; break }

# ---- Start Backend Services ----
Write-Host "  Starting backend services..." -ForegroundColor Cyan
Write-Host "  ------------------------------------------------------------" -ForegroundColor DarkGray

Start-GoService -Name "Valkyrie" -Executable "valkyrie.exe" `
    -Arguments @("-sim", "-http-port", "8093", "-metrics-port", "9193") `
    -Port 8093 -HealthEndpoint "/health"

Start-GoService -Name "GIRU" -Executable "giru.exe" `
    -Arguments @("-api-only", "-api-addr", ":9090", "-metrics-addr", ":9091") `
    -Port 9090 -HealthEndpoint "/health" `
    -EnvVars @{ ASGARD_ENV = "development" }

Start-GoService -Name "Hunoid" -Executable "hunoid.exe" `
    -Arguments @("-operator-ui-addr", ":8090", "-metrics-addr", ":9092", "-operator-mode", "auto", "-stay-alive") `
    -Port 8090 -HealthEndpoint "/api/status" `
    -EnvVars @{ HUNOID_BYPASS_HARDWARE = "1" }

Start-GoService -Name "Pricilla" -Executable "pricilla.exe" `
    -Arguments @("-http-port", "8089", "-metrics-port", "9089", "-enable-nats=false") `
    -Port 8089 -HealthEndpoint "/health"

Start-GoService -Name "Vault" -Executable "vault.exe" `
    -Arguments @("-http", ":8094", "-auto-unseal") `
    -Port 8094 -HealthEndpoint "/vault/health" `
    -EnvVars @{ VAULT_MASTER_PASSWORD = "asgard-dev-vault-2026" }

Start-GoService -Name "Silenus" -Executable "silenus.exe" `
    -Arguments @("-metrics-addr", ":9094", "-vision-backend", "simple") `
    -Port 9094 -HealthEndpoint "/healthz" `
    -EnvVars @{ SILENUS_BYPASS_HARDWARE = "1" }

Start-GoService -Name "Nysus" -Executable "nysus.exe" `
    -Arguments @() `
    -Port 8080 -HealthEndpoint "/health" `
    -EnvVars @{ ASGARD_ALLOW_NO_DB = "true" }

Start-GoService -Name "Notifications" -Executable "notifications.exe" `
    -Arguments @("-http", ":8095") `
    -Port 8095 -HealthEndpoint "/api/notifications/status"

Write-Host ""

# ---- Start Frontend Services ----
if (-not $SkipFrontends) {
    Write-Host "  Starting frontend services..." -ForegroundColor Cyan
    Write-Host "  ------------------------------------------------------------" -ForegroundColor DarkGray
    Start-FrontendService -Name "Websites" -Directory "Websites" -Port 3000
    Start-FrontendService -Name "Hubs" -Directory "Hubs" -Port 3001
    Write-Host ""
}

# ---- Show Dashboard ----
Show-Dashboard

Write-Host "  ============================================================" -ForegroundColor Green
Write-Host "    ASGARD is OPERATIONAL - Press Ctrl+C to stop all services" -ForegroundColor Green
Write-Host "  ============================================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Quick Links:" -ForegroundColor DarkCyan
Write-Host "    Websites:      http://localhost:3000" -ForegroundColor Gray
Write-Host "    Hubs:          http://localhost:3001" -ForegroundColor Gray
Write-Host "    Nysus API:     http://localhost:8080/health" -ForegroundColor Gray
Write-Host "    Valkyrie:      http://localhost:8093/health" -ForegroundColor Gray
Write-Host "    GIRU:          http://localhost:9090/health" -ForegroundColor Gray
Write-Host "    Hunoid:        http://localhost:8090/api/status" -ForegroundColor Gray
Write-Host "    Pricilla:      http://localhost:8089/health" -ForegroundColor Gray
Write-Host "    Vault:         http://localhost:8094/vault/health" -ForegroundColor Gray
Write-Host "    Silenus:       http://localhost:9094/healthz" -ForegroundColor Gray
Write-Host "    Notifications: http://localhost:8095/api/notifications/status" -ForegroundColor Gray
Write-Host ""

# Keep running until Ctrl+C
try {
    while ($true) {
        Start-Sleep -Seconds 30
        # Periodic health check
        $unhealthy = @()
        $checks = @(
            @{ Name = "Valkyrie"; Port = 8093; Endpoint = "/health" },
            @{ Name = "GIRU"; Port = 9090; Endpoint = "/health" },
            @{ Name = "Hunoid"; Port = 8090; Endpoint = "/api/status" },
            @{ Name = "Pricilla"; Port = 8089; Endpoint = "/health" },
            @{ Name = "Nysus"; Port = 8080; Endpoint = "/health" }
        )
        foreach ($c in $checks) {
            if (-not (Test-ServiceHealth -Port $c.Port -Endpoint $c.Endpoint)) {
                $unhealthy += $c.Name
            }
        }
        if ($unhealthy.Count -gt 0) {
            Write-Host "  [WARN] Unhealthy: $($unhealthy -join ', ')" -ForegroundColor Yellow
        }
    }
} finally {
    Stop-AllServices
}
