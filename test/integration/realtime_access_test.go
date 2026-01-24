package integration_test

import (
	"testing"

	"github.com/asgard/pandora/internal/platform/realtime"
)

func TestAccessRules(t *testing.T) {
	rules := realtime.NewAccessRules()

	if !rules.CanAccess(realtime.AccessLevelCivilian, realtime.EventTypeTelemetry) {
		t.Fatalf("expected civilian access to telemetry")
	}

	if rules.CanAccess(realtime.AccessLevelPublic, realtime.EventTypeThreat) {
		t.Fatalf("public access should not include threats")
	}

	if !rules.CanAccess(realtime.AccessLevelGovernment, realtime.EventTypeSecurityFinding) {
		t.Fatalf("government access should include security findings")
	}
}

func TestSubjectChannels(t *testing.T) {
	channels := realtime.NewSubjectChannels()

	level := channels.GetRequiredLevel("asgard.telemetry.sat-001")
	if level != realtime.AccessLevelCivilian {
		t.Fatalf("expected telemetry to require civilian access")
	}

	level = channels.GetRequiredLevel("asgard.gov.threats")
	if level != realtime.AccessLevelGovernment {
		t.Fatalf("expected government threats to require government access")
	}
}
