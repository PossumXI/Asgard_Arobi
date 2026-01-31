# Access Code System Change Log (2026-01-30)

This document records the access code automation, rotation policy, and portal updates.

## Summary

- Automated bootstrap of **Gaetano Comparcola** user profile on server start.
- Automated access code issuance with 24-hour rotation loop (restart-independent).
- Access codes required for government/admin sign-in when codes exist.
- Admin Hub access-code management (issue, rotate, revoke).
- Military Hub access-code validation.
- Electron admin portal entry point.
- Documentation and environment defaults updated.

## Key Files Updated

### Backend (Nysus)
- `internal/nysus/api/bootstrap_admin.go`
- `internal/nysus/api/bootstrap_access_code.go`
- `internal/nysus/api/server.go`
- `internal/nysus/api/handlers_auth.go`
- `internal/nysus/api/handlers_access_codes.go`
- `internal/nysus/api/handlers_admin.go`
- `internal/repositories/access_code.go`
- `internal/services/access_code.go`
- `internal/services/email.go`
- `internal/platform/db/models.go`
- `Data/migrations/postgres/000014_access_codes.*.sql`

### Websites (Portal)
- `Websites/src/pages/auth/SignIn.tsx`
- `Websites/src/pages/dashboard/AdminHub.tsx`
- `Websites/src/providers/AuthProvider.tsx`
- `Websites/src/hooks/useApi.ts`
- `Websites/src/lib/api.ts`
- `Websites/src/pages/gov/GovPortal.tsx`
- `Websites/src/pages/Contact.tsx`

### Hubs
- `Hubs/src/pages/MilitaryHub.tsx`
- `Hubs/src/lib/api.ts`

### Electron
- `Giru/Giru(jarvis)/main.js`
- `Giru/Giru(jarvis)/preload.js`
- `Giru/Giru(jarvis)/renderer/admin.html`

### Documentation
- `Documentation/Access_Code_System.md`
- `Documentation/README.md`
- `.env.example`

## Rotation Policy

- `ACCESS_CODE_ROTATION_HOURS=24`
- `ACCESS_CODE_ROTATION_CHECK_MINUTES=15`
- Rotation loop runs regardless of restarts.

## Bootstrap Output

- `Documentation/Bootstrap_Access.md` generated on server start.
