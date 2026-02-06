#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Stop all running ASGARD services.

.DESCRIPTION
    Kills all ASGARD backend Go services and frontend Node.js processes.
#>

$ErrorActionPreference = 'Continue'

Write-Host ""
Write-Host "  Stopping all ASGARD services..." -ForegroundColor Yellow
Write-Host ""

$services = @("valkyrie", "giru", "hunoid", "pricilla", "vault", "silenus", "nysus", "notifications")

foreach ($name in $services) {
    $procs = Get-Process -Name $name -ErrorAction SilentlyContinue
    if ($procs) {
        $procs | Stop-Process -Force -ErrorAction SilentlyContinue
        Write-Host "  [OK] Stopped $name" -ForegroundColor Green
    } else {
        Write-Host "  [--] $name not running" -ForegroundColor DarkGray
    }
}

Write-Host ""
Write-Host "  All ASGARD services stopped." -ForegroundColor Green
Write-Host ""
