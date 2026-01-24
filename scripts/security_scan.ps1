param(
    [string]$SarifOut = "Documentation\Security_Scan.sarif"
)

$ErrorActionPreference = "Stop"

function Test-Command {
    param([string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

Write-Host "=== ASGARD Security Scan ===" -ForegroundColor Cyan
Write-Host "Output SARIF: $SarifOut"

if (Test-Command "gosec") {
    Write-Host "Running gosec..."
    gosec -fmt sarif -out $SarifOut ./...
} else {
    Write-Warning "gosec not found. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
}

if (Test-Command "npm") {
    Write-Host "Running npm audit for Websites..."
    if (Test-Path "Websites\package.json") {
        Push-Location "Websites"
        npm audit --json | Out-File -Encoding utf8 "..\Documentation\Npm_Audit_Websites.json"
        Pop-Location
    }

    Write-Host "Running npm audit for Hubs..."
    if (Test-Path "Hubs\package.json") {
        Push-Location "Hubs"
        npm audit --json | Out-File -Encoding utf8 "..\Documentation\Npm_Audit_Hubs.json"
        Pop-Location
    }
} else {
    Write-Warning "npm not found. Install Node.js 20+ to run npm audit."
}

Write-Host "Security scan complete." -ForegroundColor Green
