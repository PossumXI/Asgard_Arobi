# ASGARD Founder Access Setup

## Quick Start

### Step 1: Get Free Email API Key

1. Go to [https://resend.com](https://resend.com)
2. Sign up for free (3,000 emails/month)
3. Go to API Keys section
4. Create a new API key
5. Copy the key (starts with `re_`)

### Step 2: Set Environment Variable

**Windows (Command Prompt):**
```cmd
set RESEND_API_KEY=re_your_api_key_here
```

**Windows (PowerShell):**
```powershell
$env:RESEND_API_KEY = "re_your_api_key_here"
```

**Linux/Mac:**
```bash
export RESEND_API_KEY=re_your_api_key_here
```

### Step 3: Generate & Send Access Keys

```bash
go run scripts/send-founder-access.go
```

This will:
- Generate 3 secure access keys
- Email them to Gaetano@aura-genesis.org
- Save a backup file locally

### Step 4: Access the Portal

1. Download the Electron app from the admin portal
2. Enter your access key when prompted
3. Complete FIDO2 security setup
4. Access all ASGARD systems

---

## Access Keys Generated

| Key Type | Access Level | Use Case |
|----------|--------------|----------|
| FOUNDER_MASTER | Maximum | Full system override |
| ADMIN_ACCESS | Admin | Admin portal, user management |
| GOVERNMENT_ACCESS | Government | DO-178C, FAA certification |

## Security Notes

- Keys expire in 24 hours
- One-time use only
- After first use, FIDO2 becomes primary auth
- Store securely, delete email after saving

---

## Alternative: Manual API Call

If the script doesn't work, you can call the API directly:

```bash
curl -X POST http://localhost:8095/api/access-keys/founder \
  -H "Authorization: Bearer admin" \
  -H "Content-Type: application/json"
```

This requires the notification service to be running.

---

**Contact:** security@arobi.com
**Last Updated:** 2026-02-05
