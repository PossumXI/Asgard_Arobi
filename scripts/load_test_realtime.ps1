param(
    [int]$Connections = 10,
    [string]$Url = "ws://localhost:8080/ws/realtime?access=public"
)

$ErrorActionPreference = "Stop"

Write-Host "Opening $Connections WebSocket connections to $Url" -ForegroundColor Cyan

$clients = @()
for ($i = 0; $i -lt $Connections; $i++) {
    $client = New-Object System.Net.WebSockets.ClientWebSocket
    $uri = [System.Uri]::new($Url)
    $client.ConnectAsync($uri, [Threading.CancellationToken]::None).Wait()
    $clients += $client
}

foreach ($client in $clients) {
    $payload = [System.Text.Encoding]::UTF8.GetBytes('{ "type": "ping" }')
    $segment = [System.ArraySegment[byte]]::new($payload)
    $client.SendAsync($segment, [System.Net.WebSockets.WebSocketMessageType]::Text, $true, [Threading.CancellationToken]::None).Wait()
}

Write-Host "Holding connections for 5 seconds..."
Start-Sleep -Seconds 5

foreach ($client in $clients) {
    $client.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "done", [Threading.CancellationToken]::None).Wait()
    $client.Dispose()
}

Write-Host "Realtime load test complete." -ForegroundColor Green
