# ASGARD Access Code System

This document records the access-code based clearance system, the admin bootstrap flow,
and the 24-hour rotation policy for restricted portals (Website, Hubs, Electron).

## Overview

- Access codes are tied to user profiles and stored hashed.
- Codes are scoped (`portal`, `api`, `hubs`, `electron`, `all`) and carry a clearance level.
- Rotation runs on a timer every 24 hours, independent of server restarts.
- Admins can issue, rotate, and revoke codes via the Admin Hub or API.

## Rotation Policy

- **Default rotation interval:** 24 hours (`ACCESS_CODE_ROTATION_HOURS=24`)
- **Check interval:** 15 minutes (`ACCESS_CODE_ROTATION_CHECK_MINUTES=15`)
- Rotation runs in a background loop and **does not depend on restarts**.

## Bootstrap Flow (Auto-create Gaetano)

On server start, the system:
- Ensures a user exists for **Gaetano Comparcola** (`Gaetano@aura-genesis.org`)
- Issues or rotates a government-level access code
- Writes the result to `Documentation/Bootstrap_Access.md`

### Bootstrap environment variables

```
ASGARD_BOOTSTRAP_ADMIN_EMAIL=
ASGARD_BOOTSTRAP_ADMIN_PASSWORD=
ASGARD_BOOTSTRAP_ADMIN_FULL_NAME=
ASGARD_BOOTSTRAP_ROTATE_ON_START=false
ASGARD_BOOTSTRAP_OUTPUT_PATH=Documentation/Bootstrap_Access.md
```

Notes:
- If password is omitted, a temporary one is generated.
- `ASGARD_BOOTSTRAP_ROTATE_ON_START` only forces a new code on each boot. Rotation still happens every 24h.

## Admin API Endpoints

- `GET /api/admin/access-codes` — list access codes
- `POST /api/admin/access-codes` — issue new code
- `POST /api/admin/access-codes/rotate` — rotate user code or all codes
- `DELETE /api/admin/access-codes/{id}` — revoke code
- `POST /api/access-codes/validate` — validate a code

## UI Entry Points

- **Admin Hub:** `Websites/src/pages/dashboard/AdminHub.tsx`
- **Military Hub validation:** `Hubs/src/pages/MilitaryHub.tsx`
- **Sign In access code field:** `Websites/src/pages/auth/SignIn.tsx`
- **Electron admin portal:** `Giru/Giru(jarvis)/renderer/admin.html`

## Data Storage

- Table: `access_codes`
- Migration: `Data/migrations/postgres/000014_access_codes.*.sql`
