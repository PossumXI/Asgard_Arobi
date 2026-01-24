# ASGARD Database Initialization Script
$ErrorActionPreference = "Stop"

Write-Host "Starting ASGARD database stack..." -ForegroundColor Green

function Resolve-CommandPath([string]$commandName, [string[]]$candidatePaths) {
    $cmd = Get-Command $commandName -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }
    foreach ($path in $candidatePaths) {
        if ($path -and (Test-Path $path)) {
            return $path
        }
    }
    throw "Missing required command: $commandName"
}

function Resolve-ComposeCommand {
    if (Get-Command "docker-compose" -ErrorAction SilentlyContinue) {
        return @{ Command = "docker-compose"; Args = @() }
    }
    if (Get-Command "docker" -ErrorAction SilentlyContinue) {
        & docker compose version 2>$null
        if ($LASTEXITCODE -eq 0) {
            return @{ Command = "docker"; Args = @("compose") }
        }
    }
    throw "Missing required command: docker-compose or docker compose"
}

$dockerCmd = Resolve-CommandPath "docker" @()
$gopath = & go env GOPATH
$migrateCandidates = @()
if ($gopath) {
    $migrateCandidates += (Join-Path $gopath "bin\\migrate.exe")
}
$migrateCmd = Resolve-CommandPath "migrate" $migrateCandidates

$mongoshCandidates = @(
    $env:MONGOSH_PATH,
    "C:\\Program Files\\MongoDB\\mongosh\\bin\\mongosh.exe",
    "C:\\Program Files\\MongoDB\\Server\\8.0\\bin\\mongosh.exe",
    "C:\\Program Files\\MongoDB\\Server\\7.0\\bin\\mongosh.exe",
    "C:\\Program Files\\MongoDB\\Server\\6.0\\bin\\mongosh.exe"
)
$mongoshCmd = $null
try {
    $mongoshCmd = Resolve-CommandPath "mongosh" $mongoshCandidates
} catch {
    $mongoshCmd = $null
}

try {
    $compose = Resolve-ComposeCommand

    # Start Docker Compose
    Set-Location "C:\Users\hp\Desktop\Asgard\Data"
    & $compose.Command @($compose.Args + @("up", "-d"))

    # Ensure Docker Engine is responsive
    & $dockerCmd info > $null

    # Wait for databases to be healthy
    Write-Host "Waiting for databases to be ready..." -ForegroundColor Yellow
    Start-Sleep -Seconds 15

    # Run PostgreSQL migrations
    Write-Host "Running PostgreSQL migrations..." -ForegroundColor Green
    $env:POSTGRES_PORT = "55432"
    $env:DATABASE_URL = "postgres://postgres:asgard_secure_2026@127.0.0.1:55432/asgard?sslmode=disable"
    try {
        & $migrateCmd -path ./migrations/postgres -database $env:DATABASE_URL up
    } catch {
        if ($_.Exception.Message -match "already exists" -or $_.Exception.Message -match "Dirty database") {
            Write-Host "Schema already present or dirty state detected, forcing migration version..." -ForegroundColor Yellow
            & $migrateCmd -path ./migrations/postgres -database $env:DATABASE_URL force 1
        } else {
            throw
        }
    }

    # Initialize MongoDB collections
    Write-Host "Initializing MongoDB collections..." -ForegroundColor Green
    if ($mongoshCmd) {
        & $mongoshCmd "mongodb://admin:asgard_mongo_2026@localhost:27017" --file ./migrations/mongo/001_create_collections.js
    } else {
        Write-Host "Local mongosh not found, using container shell..." -ForegroundColor Yellow
        & $dockerCmd exec asgard_mongodb mongosh "mongodb://admin:asgard_mongo_2026@localhost:27017" --file /docker-entrypoint-initdb.d/001_create_collections.js
    }

    Write-Host "Database stack initialized successfully!" -ForegroundColor Green
    Write-Host "PostgreSQL: localhost:55432 (user: postgres, db: asgard)" -ForegroundColor Cyan
    Write-Host "MongoDB: localhost:27017 (user: admin)" -ForegroundColor Cyan
    Write-Host "NATS: localhost:4222" -ForegroundColor Cyan
    Write-Host "Redis: localhost:6379" -ForegroundColor Cyan
} catch {
    Write-Host "Database initialization failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
