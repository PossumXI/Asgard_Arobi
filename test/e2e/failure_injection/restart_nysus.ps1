param(
    [int]$WaitSeconds = 10
)

$ErrorActionPreference = "Stop"

Write-Host "Restarting Nysus service..." -ForegroundColor Yellow

Get-Process -Name "nysus" -ErrorAction SilentlyContinue | Stop-Process -Force

if (-Not (Test-Path "bin\nysus.exe")) {
    throw "bin\nysus.exe not found. Build the binary first."
}

Start-Process -FilePath "bin\nysus.exe" -ArgumentList "-addr :8080" -NoNewWindow
Start-Sleep -Seconds $WaitSeconds

Write-Host "Nysus restart injection complete." -ForegroundColor Green
