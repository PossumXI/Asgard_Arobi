package integration_test

import (
	"testing"

	"github.com/asgard/pandora/pkg/bundle"
)

func TestBundleCreation(t *testing.T) {
	b := bundle.NewBundle(
		"dtn://source/test",
		"dtn://dest/nysus",
		[]byte("test payload data"),
	)

	if b.ID.String() == "" {
		t.Fatal("bundle ID should not be empty")
	}

	if b.SourceEID != "dtn://source/test" {
		t.Errorf("unexpected source: %s", b.SourceEID)
	}

	if b.DestinationEID != "dtn://dest/nysus" {
		t.Errorf("unexpected destination: %s", b.DestinationEID)
	}

	if string(b.Payload) != "test payload data" {
		t.Error("payload mismatch")
	}

	if b.Priority != bundle.PriorityNormal {
		t.Errorf("expected priority %d, got %d", bundle.PriorityNormal, b.Priority)
	}
}

func TestBundleValidation(t *testing.T) {
	// Valid bundle
	b := bundle.NewBundle(
		"dtn://source/valid",
		"dtn://dest/valid",
		[]byte("data"),
	)

	if err := b.Validate(); err != nil {
		t.Fatalf("valid bundle failed validation: %v", err)
	}

	// Invalid source EID
	invalidBundle := bundle.NewBundle(
		"",
		"dtn://dest/valid",
		[]byte("data"),
	)
	if err := invalidBundle.Validate(); err == nil {
		t.Fatal("bundle with empty source should fail validation")
	}

	// Invalid destination EID
	invalidBundle = bundle.NewBundle(
		"dtn://source/valid",
		"",
		[]byte("data"),
	)
	if err := invalidBundle.Validate(); err == nil {
		t.Fatal("bundle with empty destination should fail validation")
	}
}

func TestBundlePriority(t *testing.T) {
	b := bundle.NewBundle(
		"dtn://src",
		"dtn://dst",
		[]byte("data"),
	)

	// Test setting valid priority
	if err := b.SetPriority(bundle.PriorityExpedited); err != nil {
		t.Errorf("failed to set expedited priority: %v", err)
	}

	if b.Priority != bundle.PriorityExpedited {
		t.Errorf("expected priority %d, got %d", bundle.PriorityExpedited, b.Priority)
	}

	// Test setting invalid priority
	if err := b.SetPriority(255); err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestNewPriorityBundle(t *testing.T) {
	// Valid priorities
	for _, priority := range []uint8{bundle.PriorityBulk, bundle.PriorityNormal, bundle.PriorityExpedited} {
		b, err := bundle.NewPriorityBundle(
			"dtn://src",
			"dtn://dst",
			[]byte("data"),
			priority,
		)
		if err != nil {
			t.Errorf("failed to create bundle with priority %d: %v", priority, err)
		}
		if b.Priority != priority {
			t.Errorf("expected priority %d, got %d", priority, b.Priority)
		}
	}

	// Invalid priority
	_, err := bundle.NewPriorityBundle(
		"dtn://src",
		"dtn://dst",
		[]byte("data"),
		100, // Invalid
	)
	if err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestBundleClone(t *testing.T) {
	original := bundle.NewBundle(
		"dtn://original",
		"dtn://destination",
		[]byte("original payload"),
	)
	original.Priority = bundle.PriorityExpedited

	cloned := original.Clone()

	// Verify clone has same data
	if cloned.ID != original.ID {
		t.Error("cloned bundle should have same ID")
	}

	if cloned.SourceEID != original.SourceEID {
		t.Error("source should match")
	}

	if cloned.DestinationEID != original.DestinationEID {
		t.Error("destination should match")
	}

	if string(cloned.Payload) != string(original.Payload) {
		t.Error("payload should match")
	}

	// Verify modifying clone doesn't affect original
	cloned.Payload = []byte("modified")
	if string(original.Payload) == "modified" {
		t.Error("modifying clone should not affect original")
	}
}

func TestBundleHash(t *testing.T) {
	b := bundle.NewBundle(
		"dtn://source",
		"dtn://dest",
		[]byte("payload for hashing"),
	)

	hash := b.Hash()
	if len(hash) == 0 {
		t.Fatal("hash should not be empty")
	}

	// Same bundle should produce same hash
	hash2 := b.Hash()
	if hash != hash2 {
		t.Error("same bundle should produce consistent hash")
	}

	// Different payload should produce different hash
	b2 := bundle.NewBundle(
		"dtn://source",
		"dtn://dest",
		[]byte("different payload"),
	)
	if b.Hash() == b2.Hash() {
		t.Error("different payloads should produce different hashes")
	}
}

func TestBundleExpiration(t *testing.T) {
	b := bundle.NewBundle(
		"dtn://src",
		"dtn://dst",
		[]byte("data"),
	)

	// Fresh bundle should not be expired
	if b.IsExpired() {
		t.Error("fresh bundle should not be expired")
	}

	// Check remaining lifetime is positive
	remaining := b.RemainingLifetime()
	if remaining <= 0 {
		t.Errorf("remaining lifetime should be positive, got %v", remaining)
	}

	// Expiry time should be in the future
	expiresAt := b.ExpiresAt()
	if !expiresAt.After(b.CreationTimestamp) {
		t.Error("expiry should be after creation")
	}
}

func TestBundleHopCount(t *testing.T) {
	b := bundle.NewBundle(
		"dtn://src",
		"dtn://dst",
		[]byte("data"),
	)

	// Initial hop count should be 0
	if b.HopCount != 0 {
		t.Errorf("initial hop count should be 0, got %d", b.HopCount)
	}

	// Increment hop
	if err := b.IncrementHop("node-1"); err != nil {
		t.Errorf("failed to increment hop: %v", err)
	}

	if b.HopCount != 1 {
		t.Errorf("hop count should be 1, got %d", b.HopCount)
	}

	if b.PreviousNode != "node-1" {
		t.Errorf("previous node should be node-1, got %s", b.PreviousNode)
	}
}

func TestBundleSize(t *testing.T) {
	payload := []byte("test payload data")
	b := bundle.NewBundle(
		"dtn://src",
		"dtn://dst",
		payload,
	)

	size := b.Size()
	if size < len(payload) {
		t.Errorf("bundle size %d should be at least payload size %d", size, len(payload))
	}
}

func TestBundleString(t *testing.T) {
	b := bundle.NewBundle(
		"dtn://source/test",
		"dtn://dest/test",
		[]byte("data"),
	)

	str := b.String()
	if str == "" {
		t.Error("string representation should not be empty")
	}
}
