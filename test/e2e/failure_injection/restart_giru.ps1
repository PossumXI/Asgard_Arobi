param(
    [int]$WaitSeconds = 10
)

$ErrorActionPreference = "Stop"

Write-Host "Restarting Giru service..." -ForegroundColor Yellow

Get-Process -Name "giru" -ErrorAction SilentlyContinue | Stop-Process -Force

if (-Not (Test-Path "bin\giru.exe")) {
    throw "bin\giru.exe not found. Build the binary first."
}

Start-Process -FilePath "bin\giru.exe" -ArgumentList "-metrics-addr :9091" -NoNewWindow
Start-Sleep -Seconds $WaitSeconds

Write-Host "Giru restart injection complete." -ForegroundColor Green
