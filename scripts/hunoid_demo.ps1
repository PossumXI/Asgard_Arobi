Param(
    [string]$Scenario = "medical_aid",
    [ValidateSet("auto","manual","disabled")]
    [string]$OperatorMode = "auto"
)

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

Write-Host "Starting Hunoid demo scenario: $Scenario (operator: $OperatorMode)"
go run .\cmd\hunoid\main.go -scenario $Scenario -operator-mode $OperatorMode
