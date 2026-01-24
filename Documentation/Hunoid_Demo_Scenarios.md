# Hunoid Demo Scenarios

This file lists the built-in scenarios for live demonstrations.

## medical_aid

Objective: Deliver a medical kit and assess hazards.

Steps:
- Navigate to supply depot
- Pick up medical kit
- Move to injured person
- Put down medical kit gently (consent required)
- Inspect area for hazards

## perimeter_check

Objective: Validate safety perimeter with low risk actions.

Steps:
- Navigate to checkpoint alpha
- Inspect area for hazards
- Navigate to checkpoint beta
- Inspect area for hazards

## hazard_response

Objective: Investigate a high-risk hazard zone with operator oversight.

Steps:
- Navigate to hazard zone
- Inspect hazards
- Move to containment unit

## Demo Commands

```powershell
go run .\cmd\hunoid\main.go -scenario medical_aid -operator-mode auto
go run .\cmd\hunoid\main.go -scenario perimeter_check -operator-mode auto
go run .\cmd\hunoid\main.go -scenario hazard_response -operator-mode manual
```

Open the operator UI at `http://localhost:8090` while a scenario runs.
