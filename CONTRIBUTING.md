# Contributing to ASGARD

Thank you for your interest in contributing to ASGARD! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Node.js 18+ and npm
- Git
- VS Code (recommended) with Jira extension

### Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/asgard/pandora.git
   cd pandora
   ```

2. Install Go dependencies:
   ```bash
   go mod download
   ```

3. Install Node.js dependencies:
   ```bash
   npm install
   ```

4. Build all services:
   ```bash
   go build -o bin/giru.exe ./cmd/giru
   go build -o bin/valkyrie.exe ./Valkyrie/cmd/valkyrie
   go build -o bin/hunoid.exe ./cmd/hunoid
   go build -o bin/vault.exe ./cmd/vault
   ```

5. Configure Jira VS Code extension:
   - Open VS Code settings
   - Configure Jira credentials
   - Project settings are in `.jira/settings.json`

## Project Structure

```
ASGARD/
├── cmd/                    # Service entry points
├── internal/               # Internal packages
│   ├── robotics/          # Robotics (Tier 6 restricted)
│   ├── security/          # Security systems
│   └── ...
├── Hubs/                   # Web frontend (Tier 2+)
├── Websites/               # Marketing site (Tier 1+)
├── Valkyrie/              # Flight control (Tier 6 restricted)
├── Documentation/          # Project docs
├── test/                   # Test suites
└── configs/                # Configuration files
```

## Access Tiers

Contributors have different access levels based on their role:

| Tier | Role | What You Can Work On |
|------|------|---------------------|
| 1 | Public | Documentation, UI improvements |
| 2 | Developer | Frontend, backend, APIs |
| 3 | Senior Dev | Architecture, performance |
| 4 | Security | Security features |
| 5 | Admin | Infrastructure, configs |
| 6+ | Cleared | Government/Military components |

See `.jira/access-controls.md` for complete details.

## Making Contributions

### 1. Find an Issue

- Browse issues matching your access tier
- Look for `good-first-issue` label if you're new
- Check that the issue isn't already assigned

### 2. Create a Branch

```bash
git checkout -b feature/ASGARD-123-description
```

Use the Jira issue key in your branch name.

### 3. Make Your Changes

- Follow the code style guides
- Add tests for new functionality
- Update documentation as needed
- Keep commits focused and atomic

### 4. Run Tests

```bash
# Go tests
go test ./...

# Frontend tests
cd Hubs && npm test

# E2E tests (requires services running)
npm run test:e2e
```

### 5. Submit a Pull Request

- Reference the Jira issue in your PR
- Provide a clear description of changes
- Ensure all CI checks pass
- Request review from appropriate team members

## Code Style

### Go

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions focused and small

### TypeScript/React

- Use TypeScript strict mode
- Follow React best practices
- Use functional components with hooks
- Add PropTypes or interfaces

### Commit Messages

```
type(scope): description

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(giru): add shadow stack monitoring

Implements parallel execution monitoring for zero-day detection.
Includes behavioral analysis and anomaly reporting.

Closes ASGARD-456
```

## Restricted Areas

The following areas require special clearance:

### Tier 6+ Required
- `internal/robotics/decision/` - Ethics kernel
- `internal/robotics/perception/` - 360° perception
- `Valkyrie/` - Flight control system
- `cmd/hunoid/` - Humanoid robotics

### Security Team (Tier 4+) Required
- `internal/security/` - All security components
- `internal/security/vault/` - Secrets management
- Penetration testing tasks

### Do Not Modify Without Approval
- `configs/government.yaml`
- `internal/security/vault/fido2.go`
- Any file marked with `PROPRIETARY` header
- DO-178C compliance documentation

## Testing Requirements

### Unit Tests
- Minimum 80% coverage for new code
- Test edge cases and error conditions
- Use table-driven tests in Go

### Integration Tests
- Test service interactions
- Use the service manager for E2E tests
- Video recording for demo scenarios

### Security Tests
- No hardcoded credentials
- Validate all inputs
- Check for OWASP Top 10

## Documentation

- Update README when adding features
- Add JSDoc/GoDoc comments
- Keep API documentation current
- Document breaking changes

## Questions?

- **General:** Create an issue with `question` label
- **Security:** Email security@arobi.com
- **Access Requests:** Contact your team lead

## Code of Conduct

Be respectful, professional, and collaborative. We're building safety-critical systems that protect lives.

---

*By contributing, you agree to the project's license terms and acknowledge the security requirements.*
