# ASGARD Automated Audit Trigger Script
# Runs every 15 minutes to check for changes and trigger audit notifications
# Usage: .\audit_trigger.ps1 [-Continuous] [-IntervalMinutes 15]

param(
    [switch]$Continuous = $false,
    [int]$IntervalMinutes = 15
)

$ErrorActionPreference = "Continue"
$AsgardRoot = "C:\Users\hp\Desktop\Asgard"
$AuditLogPath = "$AsgardRoot\Documentation\Audit_Activity.md"
$StateFilePath = "$AsgardRoot\Documentation\.audit_state.json"

# File extensions to monitor
$MonitoredExtensions = @(
    "*.go", "*.ts", "*.tsx", "*.js", "*.jsx",
    "*.sql", "*.json", "*.yaml", "*.yml",
    "*.ps1", "*.sh", "*.md"
)

# Directories to exclude
$ExcludedDirs = @(
    "node_modules", ".git", "dist", "build", ".next"
)

function Get-FileHashes {
    param([string]$Path)
    
    $hashes = @{}
    
    foreach ($ext in $MonitoredExtensions) {
        Get-ChildItem -Path $Path -Filter $ext -Recurse -ErrorAction SilentlyContinue |
            Where-Object { 
                $excluded = $false
                foreach ($dir in $ExcludedDirs) {
                    if ($_.FullName -like "*\$dir\*") { $excluded = $true; break }
                }
                -not $excluded
            } |
            ForEach-Object {
                $relativePath = $_.FullName.Substring($Path.Length + 1)
                $hashes[$relativePath] = @{
                    Hash = (Get-FileHash $_.FullName -Algorithm MD5).Hash
                    Size = $_.Length
                    LastModified = $_.LastWriteTimeUtc.ToString("o")
                }
            }
    }
    
    return $hashes
}

function Import-PreviousState {
    if (Test-Path $StateFilePath) {
        $content = Get-Content $StateFilePath -Raw
        return $content | ConvertFrom-Json -AsHashtable
    }
    return @{}
}

function Save-CurrentState {
    param($State)
    $State | ConvertTo-Json -Depth 10 | Set-Content $StateFilePath
}

function Compare-States {
    param($Previous, $Current)
    
    $changes = @{
        Added = @()
        Modified = @()
        Deleted = @()
    }
    
    # Find added and modified files
    foreach ($file in $Current.Keys) {
        if (-not $Previous.ContainsKey($file)) {
            $changes.Added += $file
        }
        elseif ($Previous[$file].Hash -ne $Current[$file].Hash) {
            $changes.Modified += $file
        }
    }
    
    # Find deleted files
    foreach ($file in $Previous.Keys) {
        if (-not $Current.ContainsKey($file)) {
            $changes.Deleted += $file
        }
    }
    
    return $changes
}

function Write-AuditLog {
    param(
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $entry = "[$timestamp] [$Level] $Message"
    
    # Console output with color
    switch ($Level) {
        "INFO"    { Write-Host $entry -ForegroundColor Cyan }
        "CHANGE"  { Write-Host $entry -ForegroundColor Yellow }
        "WARNING" { Write-Host $entry -ForegroundColor Red }
        "SUCCESS" { Write-Host $entry -ForegroundColor Green }
        default   { Write-Host $entry }
    }
    
    # Append to audit log file
    Add-Content -Path $AuditLogPath -Value $entry
}

function Initialize-AuditLog {
    if (-not (Test-Path $AuditLogPath)) {
        $header = @"
# ASGARD Audit Activity Log

This file is automatically maintained by the audit trigger script.
It tracks file changes detected between audit intervals.

---

"@
        Set-Content -Path $AuditLogPath -Value $header
    }
}

function Invoke-AuditCheck {
    $checkTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    
    Write-AuditLog "Starting audit check at $checkTime" "INFO"
    
    # Get current file state
    $currentState = Get-FileHashes -Path $AsgardRoot
    $fileCount = $currentState.Count
    
    Write-AuditLog "Scanned $fileCount monitored files" "INFO"
    
    # Load previous state
    $previousState = Import-PreviousState
    
    if ($previousState.Count -eq 0) {
        Write-AuditLog "First run - establishing baseline with $fileCount files" "INFO"
        Save-CurrentState -State $currentState
        return @{ IsFirstRun = $true; Changes = $null }
    }
    
    # Compare states
    $changes = Compare-States -Previous $previousState -Current $currentState
    
    $totalChanges = $changes.Added.Count + $changes.Modified.Count + $changes.Deleted.Count
    
    if ($totalChanges -eq 0) {
        Write-AuditLog "No changes detected since last audit" "SUCCESS"
    }
    else {
        Write-AuditLog "CHANGES DETECTED: $totalChanges total changes" "CHANGE"
        
        # Log new files
        foreach ($file in $changes.Added) {
            Write-AuditLog "  [NEW] $file" "CHANGE"
        }
        
        # Log modified files
        foreach ($file in $changes.Modified) {
            Write-AuditLog "  [MOD] $file" "CHANGE"
        }
        
        # Log deleted files
        foreach ($file in $changes.Deleted) {
            Write-AuditLog "  [DEL] $file" "WARNING"
        }
        
        # Update audit activity log with summary
        $summaryEntry = @"

## Audit Check: $checkTime

**Changes Detected:**
- New files: $($changes.Added.Count)
- Modified files: $($changes.Modified.Count)
- Deleted files: $($changes.Deleted.Count)

"@
        
        if ($changes.Added.Count -gt 0) {
            $summaryEntry += "### New Files`n"
            foreach ($f in $changes.Added) {
                $summaryEntry += "- ``$f```n"
            }
            $summaryEntry += "`n"
        }
        
        if ($changes.Modified.Count -gt 0) {
            $summaryEntry += "### Modified Files`n"
            foreach ($f in $changes.Modified) {
                $summaryEntry += "- ``$f```n"
            }
            $summaryEntry += "`n"
        }
        
        Add-Content -Path $AuditLogPath -Value $summaryEntry
    }
    
    # Save current state for next comparison
    Save-CurrentState -State $currentState
    
    return @{ 
        IsFirstRun = $false
        Changes = $changes
        TotalChanges = $totalChanges
        FileCount = $fileCount
    }
}

function Get-ProjectStats {
    $stats = @{
        GoFiles = (Get-ChildItem -Path $AsgardRoot -Filter "*.go" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
        TsFiles = (Get-ChildItem -Path $AsgardRoot -Filter "*.ts" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
        TsxFiles = (Get-ChildItem -Path $AsgardRoot -Filter "*.tsx" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
        SqlFiles = (Get-ChildItem -Path $AsgardRoot -Filter "*.sql" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
        MdFiles = (Get-ChildItem -Path $AsgardRoot -Filter "*.md" -Recurse -ErrorAction SilentlyContinue | Measure-Object).Count
    }
    
    return $stats
}

# Main execution
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  ASGARD Automated Audit Trigger" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Initialize-AuditLog

$stats = Get-ProjectStats
Write-Host "Project Statistics:" -ForegroundColor Green
Write-Host "  Go files:   $($stats.GoFiles)"
Write-Host "  TS files:   $($stats.TsFiles)"
Write-Host "  TSX files:  $($stats.TsxFiles)"
Write-Host "  SQL files:  $($stats.SqlFiles)"
Write-Host "  MD files:   $($stats.MdFiles)"
Write-Host ""

if ($Continuous) {
    Write-Host "Running in CONTINUOUS mode (Ctrl+C to stop)" -ForegroundColor Yellow
    Write-Host "Audit interval: $IntervalMinutes minutes" -ForegroundColor Yellow
    Write-Host ""
    
    while ($true) {
        $result = Invoke-AuditCheck
        
        if ($result.TotalChanges -gt 0) {
            Write-Host ""
            Write-Host ">>> AUDIT NEEDED: $($result.TotalChanges) changes detected! <<<" -ForegroundColor Red
            Write-Host ">>> Review changes and run manual audit in Cursor <<<" -ForegroundColor Red
            Write-Host ""
            
            # Optional: Play a sound to alert
            [Console]::Beep(800, 500)
        }
        
        Write-Host ""
        Write-Host "Next audit in $IntervalMinutes minutes... ($(Get-Date -Format 'HH:mm:ss'))" -ForegroundColor Gray
        Start-Sleep -Seconds ($IntervalMinutes * 60)
    }
}
else {
    # Single run mode
    $result = Invoke-AuditCheck
    
    Write-Host ""
    if ($result.TotalChanges -gt 0) {
        Write-Host "ACTION REQUIRED: $($result.TotalChanges) changes detected!" -ForegroundColor Red
        Write-Host "Run manual audit in Cursor to verify agent work." -ForegroundColor Yellow
    }
    else {
        Write-Host "All clear - no changes since last audit." -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "To run continuously: .\audit_trigger.ps1 -Continuous" -ForegroundColor Gray
    Write-Host "To change interval:  .\audit_trigger.ps1 -Continuous -IntervalMinutes 10" -ForegroundColor Gray
}
