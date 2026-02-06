# ASGARD Jira Project Access Controls

## Overview

This document defines the access control policy for the ASGARD Jira project. Access is tiered based on security clearance levels to protect sensitive government, military, and proprietary information.

## Access Tiers

### Tier 1: Public Contributors
**Role:** External contributors, open-source collaborators
**Access:**
- Issues labeled `public`
- Documentation tasks
- UI/UX improvements
- General bug reports (non-security)

**Restrictions:**
- Cannot view issues labeled `security`, `government`, `military`, `proprietary`
- Cannot access sprint boards
- Cannot view time tracking data
- Read-only access to wiki

### Tier 2: Developer
**Role:** Regular development team members
**Access:**
- All Tier 1 access
- Issues labeled `developer`, `frontend`, `backend`, `api`
- Sprint boards
- Time tracking
- Code review tasks
- Integration testing tasks

**Restrictions:**
- Cannot view issues labeled `security`, `government`, `military`, `admin`
- Cannot access security audit reports
- Cannot view proprietary algorithm details
- Cannot modify project configuration

### Tier 3: Senior Developer
**Role:** Trusted senior developers with NDA
**Access:**
- All Tier 2 access
- Issues labeled `security` (non-classified)
- Architecture decisions
- Performance optimization tasks
- Security testing (non-penetration)

**Restrictions:**
- Cannot view issues labeled `government`, `military`, `classified`
- Cannot access authentication configurations
- Cannot view encryption keys or secrets
- Must use 2FA for all access

### Tier 4: Security Team
**Role:** Security engineers and auditors
**Access:**
- All Tier 3 access
- Issues labeled `security-audit`, `penetration-testing`
- Vulnerability reports
- Compliance documentation
- Security incident responses

**Restrictions:**
- Cannot view issues labeled `government`, `military` (without clearance)
- Cannot access production credentials
- All actions logged and audited
- Must use FIDO2 authentication

### Tier 5: Admin
**Role:** Project administrators, technical leads
**Access:**
- All Tier 4 access
- Project configuration
- User management
- Integration settings
- Workflow customization

**Restrictions:**
- Cannot view issues labeled `government`, `military`
- Cannot access government contractor portals
- Cannot modify compliance-locked items
- Actions require approval for sensitive changes

### Tier 6: Government Contractor
**Role:** Cleared government contractors
**Access:**
- All Tier 5 access
- Issues labeled `government`
- DO-178C compliance items
- FAA certification tasks
- Government contract deliverables

**Restrictions:**
- Cannot view issues labeled `military`
- Must have active government clearance
- FIDO2 + biometric authentication required
- Access logged to government audit system

### Tier 7: Military Contractor
**Role:** Cleared military/defense contractors
**Access:**
- Full access to all issues
- Military-specific requirements
- Classified documentation
- Defense integration tasks

**Requirements:**
- Active military/defense clearance
- FIDO2 + biometric + hardware token
- Air-gapped access when required
- Full audit trail to DoD systems

## Label-Based Access Control

### Public Labels (Tier 1+)
- `public` - Open for anyone
- `documentation` - Doc updates
- `good-first-issue` - Beginner friendly

### Developer Labels (Tier 2+)
- `developer` - General dev tasks
- `frontend` - UI development
- `backend` - Server/API development
- `api` - API changes
- `testing` - Test development
- `ci-cd` - Pipeline tasks

### Senior Labels (Tier 3+)
- `architecture` - System design
- `performance` - Optimization
- `security` - Security improvements

### Security Labels (Tier 4+)
- `security-audit` - Audit findings
- `vulnerability` - Security issues
- `penetration-testing` - Pen test items

### Admin Labels (Tier 5+)
- `admin` - Administrative tasks
- `infrastructure` - Infra changes
- `compliance` - Compliance items

### Government Labels (Tier 6+)
- `government` - Gov contractor tasks
- `do-178c` - Aviation compliance
- `faa` - FAA certification

### Military Labels (Tier 7)
- `military` - Defense tasks
- `classified` - Classified items
- `restricted` - Export controlled

## Component Access Restrictions

| Component | Min. Tier | Notes |
|-----------|-----------|-------|
| Hubs (Web Frontend) | 2 | Public web app |
| Websites (Marketing) | 1 | Public website |
| Giru (Security) | 4 | Security monitoring |
| Valkyrie (Flight) | 6 | Flight control - gov only |
| Hunoid (Robotics) | 6 | Robotics - gov only |
| Pricilla (Trajectory) | 6 | Trajectory - gov only |
| Security Vault | 5 | Secrets management |
| Ethics Kernel | 6 | AI ethics - gov only |

## Sensitive Fields

These custom fields are restricted based on tier:

| Field | Visible To | Description |
|-------|------------|-------------|
| Security Impact | Tier 4+ | Security severity |
| Compliance Reference | Tier 5+ | DO-178C refs |
| Government Contract | Tier 6+ | Contract IDs |
| Clearance Required | Tier 6+ | Required clearance |
| Classification | Tier 7 | Security class |
| Export Control | Tier 7 | ITAR/EAR status |

## Workflow Transitions

### Standard Workflow (Tier 2+)
`To Do` → `In Progress` → `Code Review` → `Testing` → `Done`

### Security Workflow (Tier 4+)
`Reported` → `Triaged` → `In Progress` → `Security Review` → `Verified` → `Resolved`

### Compliance Workflow (Tier 6+)
`Draft` → `Internal Review` → `Compliance Review` → `Gov Review` → `Approved` → `Certified`

## API Access

### REST API Restrictions
- Tier 1-2: Read-only access to assigned issues
- Tier 3-4: Read/write to non-sensitive issues
- Tier 5+: Full API access with audit logging

### Webhooks
- Outbound webhooks require Tier 5+ approval
- No webhooks for issues labeled `government` or `military`
- All webhook payloads are sanitized

## Audit Requirements

| Action | Audit Level |
|--------|-------------|
| View issue | Tier 4+ logged |
| Edit issue | All tiers logged |
| Create issue | All tiers logged |
| Delete issue | Requires approval + logged |
| Export data | Tier 5+ only, logged |
| API access | All access logged |

## Enforcement

1. **Project Settings:** Configure Jira roles and permissions
2. **Issue Security Schemes:** Apply security schemes by label
3. **Permission Schemes:** Map roles to permissions
4. **Automation Rules:** Auto-label based on component
5. **Audit Logs:** Review weekly for compliance

## VS Code Extension Configuration

For developers using the Jira VS Code extension:

1. Configure your access tier in settings
2. Only issues matching your tier will appear
3. Custom JQL filters are pre-configured by tier
4. Sensitive fields are hidden based on permissions

## Contact

For access requests or questions:
- **Tier 1-3:** devops@arobi.com
- **Tier 4-5:** security@arobi.com
- **Tier 6-7:** contracts@arobi.com

---

*This document is CONFIDENTIAL and should not be shared outside the organization.*
*Last updated: 2026-02-05*
