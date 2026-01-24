# Hunoid Mission Planning and Safety Runtime

This document describes the demo-ready mission planning, ethics, intervention,
and audit pipeline implemented in `cmd/hunoid/main.go`.

## Mission Planning and Execution

- **Mission plan**: A structured plan with name, objective, risk level, and steps.
- **Steps**: Each step includes a natural language command, criticality, consent
  requirements, and hazard level.
- **Execution pipeline**:
  1. Mission step ingested.
  2. VLA model infers an action with confidence.
  3. Ethical kernel evaluates action.
  4. Safety policy engine evaluates environmental and consent constraints.
  5. Intervention engine determines whether to proceed, hold, or abort.
  6. Action registry executes approved action.

## Ethics / Safety Policy Engine

- **Ethical kernel**: Rule-based evaluation for harm prevention, consent,
  proportionality, and transparency.
- **Safety policy engine**: Enforces battery limits, hazard oversight, and
  consent requirements.

## Intervention Decision Logic

Intervention decisions are derived from combined ethics and policy outcomes:

- `proceed`: Action is executed.
- `hold`: Operator approval required.
- `abort`: Action is blocked due to high risk or policy violations.

## Operator Control Interface

The operator console runs on stdin and accepts:

- `status` - show pause/abort state
- `pause` - pause mission execution
- `resume` - resume mission execution
- `abort` - abort the mission
- `approve <step-id>` - approve a held step
- `inject <command>` - add a new mission step

The UI-based operator console is served over HTTP and provides the same actions
with a minimal Apple-inspired interface.

- UI URL: `http://localhost:8090` (default)

## Logging, Auditability, Reports

- Audit events are written to `Documentation/Hunoid_Audit_Log.jsonl`.
- A mission summary report is written to
  `Documentation/Hunoid_Mission_Report.md`.
- Telemetry entries include pose, battery, and movement state.

## Usage

```powershell
go run .\cmd\hunoid\main.go -scenario medical_aid -operator-mode auto
```

## Flags

- `-scenario`: `medical_aid`, `perimeter_check`, `hazard_response`
- `-operator-mode`: `auto`, `manual`, `disabled`
- `-auto-approve-delay`: delay before auto approval
- `-operator-ui`: enable/disable the web operator console
- `-operator-ui-addr`: HTTP listen address for the UI
- `-audit-log`: audit log output path
- `-report`: report output path
- `-telemetry-interval`: telemetry cadence
