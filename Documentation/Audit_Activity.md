# ASGARD Audit Activity Log

This file is automatically maintained by the audit trigger script.
It tracks file changes detected between audit intervals.

---

## Initial Setup: 2026-01-21

Audit trigger script installed at `Documentation/audit_trigger.ps1`

### Usage

**Single Check:**
```powershell
cd C:\Users\hp\Desktop\Asgard\Documentation
.\audit_trigger.ps1
```

**Continuous Monitoring (every 15 minutes):**
```powershell
.\audit_trigger.ps1 -Continuous
```

**Custom Interval (every 10 minutes):**
```powershell
.\audit_trigger.ps1 -Continuous -IntervalMinutes 10
```

### What Gets Monitored

- Go files (*.go)
- TypeScript files (*.ts, *.tsx)
- JavaScript files (*.js, *.jsx)
- SQL files (*.sql)
- Config files (*.json, *.yaml, *.yml)
- Scripts (*.ps1, *.sh)
- Documentation (*.md)

### Excluded Directories

- node_modules/
- .git/
- dist/
- build/
- .next/

---

## Audit Log Entries

[2026-01-20 22:24:34] [INFO] Starting audit check at 2026-01-20 22:24:34
[2026-01-20 22:24:35] [INFO] Scanned 86 monitored files
[2026-01-20 22:24:35] [INFO] First run - establishing baseline with 86 files
