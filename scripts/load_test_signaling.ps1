param(
    [int]$Connections = 5,
    [string]$Url = "ws://localhost:8080/ws/signaling"
)

$ErrorActionPreference = "Stop"

Write-Host "Opening $Connections signaling connections to $Url" -ForegroundColor Cyan

$clients = @()
for ($i = 0; $i -lt $Connections; $i++) {
    $client = New-Object System.Net.WebSockets.ClientWebSocket
    $uri = [System.Uri]::new($Url)
    $client.ConnectAsync($uri, [Threading.CancellationToken]::None).Wait()

    $sessionId = "session-$i"
    $streamId = "stream-$i"
    $payload = [System.Text.Encoding]::UTF8.GetBytes("{ ""type"": ""join"", ""sessionId"": ""$sessionId"", ""streamId"": ""$streamId"" }")
    $segment = [System.ArraySegment[byte]]::new($payload)
    $client.SendAsync($segment, [System.Net.WebSockets.WebSocketMessageType]::Text, $true, [Threading.CancellationToken]::None).Wait()

    $clients += $client
}

Write-Host "Holding connections for 5 seconds..."
Start-Sleep -Seconds 5

foreach ($client in $clients) {
    $client.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "done", [Threading.CancellationToken]::None).Wait()
    $client.Dispose()
}

Write-Host "Signaling load test complete." -ForegroundColor Green
