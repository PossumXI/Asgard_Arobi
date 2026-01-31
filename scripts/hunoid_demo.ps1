Param(
    [string]$Scenario = "medical_aid",
    [ValidateSet("auto","manual","disabled")]
    [string]$OperatorMode = "auto"
)

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

Write-Host "Starting Hunoid demo scenario: $Scenario (operator: $OperatorMode)"
if (-not $env:HUNOID_BYPASS_HARDWARE) {
    $env:HUNOID_BYPASS_HARDWARE = "1"
}
go run .\cmd\hunoid\main.go -scenario $Scenario -operator-mode $OperatorMode
