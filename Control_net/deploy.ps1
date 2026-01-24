# ASGARD Kubernetes Deployment Script
# Deploys all components to Kubernetes cluster

Write-Host "=== ASGARD Kubernetes Deployment ===" -ForegroundColor Cyan

# Check if kubectl is available
if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: kubectl not found. Please install Kubernetes CLI." -ForegroundColor Red
    exit 1
}

# Check if cluster is accessible
Write-Host "`nChecking cluster connection..." -ForegroundColor Yellow
kubectl cluster-info | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Cannot connect to Kubernetes cluster." -ForegroundColor Red
    exit 1
}

Write-Host "Cluster connected successfully" -ForegroundColor Green

# Deploy namespace
Write-Host "`nDeploying namespace..." -ForegroundColor Yellow
kubectl apply -f kubernetes/namespace.yaml
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to deploy namespace" -ForegroundColor Red
    exit 1
}

# Deploy secrets
Write-Host "Deploying secrets..." -ForegroundColor Yellow
kubectl apply -f kubernetes/secrets.yaml
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to deploy secrets" -ForegroundColor Red
    exit 1
}

# Deploy databases
Write-Host "`nDeploying databases..." -ForegroundColor Yellow
kubectl apply -f kubernetes/postgres/
kubectl apply -f kubernetes/mongodb/
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to deploy databases" -ForegroundColor Red
    exit 1
}

# Wait for databases to be ready
Write-Host "Waiting for databases to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=ready pod -l app=postgres -n asgard --timeout=300s
kubectl wait --for=condition=ready pod -l app=mongodb -n asgard --timeout=300s

# Deploy services
Write-Host "`nDeploying services..." -ForegroundColor Yellow
kubectl apply -f kubernetes/nysus/
kubectl apply -f kubernetes/silenus/
kubectl apply -f kubernetes/hunoid/
kubectl apply -f kubernetes/giru/
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to deploy services" -ForegroundColor Red
    exit 1
}

# Wait for services to be ready
Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=available deployment/nysus -n asgard --timeout=300s
kubectl wait --for=condition=available deployment/silenus -n asgard --timeout=300s
kubectl wait --for=condition=available deployment/hunoid -n asgard --timeout=300s
kubectl wait --for=condition=available deployment/giru -n asgard --timeout=300s

Write-Host "`n=== Deployment Complete ===" -ForegroundColor Green
Write-Host "`nDeployment Status:" -ForegroundColor Cyan
kubectl get pods -n asgard
Write-Host "`nServices:" -ForegroundColor Cyan
kubectl get svc -n asgard
