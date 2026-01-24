package integration_test

import (
	"context"
	"testing"

	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/pkg/bundle"
)

func TestInMemoryStorageBasic(t *testing.T) {
	ctx := context.Background()
	storage := dtn.NewInMemoryStorage(100)

	// Store a bundle
	b := bundle.NewBundle(
		"dtn://test/source",
		"dtn://test/dest",
		[]byte("test data"),
	)

	if err := storage.Store(ctx, b); err != nil {
		t.Fatalf("failed to store bundle: %v", err)
	}

	// Retrieve the bundle
	retrieved, err := storage.Retrieve(ctx, b.ID)
	if err != nil {
		t.Fatalf("failed to retrieve bundle: %v", err)
	}

	if retrieved.ID != b.ID {
		t.Errorf("ID mismatch: expected %s, got %s", b.ID, retrieved.ID)
	}

	if string(retrieved.Payload) != string(b.Payload) {
		t.Error("payload mismatch")
	}

	// Delete the bundle
	if err := storage.Delete(ctx, b.ID); err != nil {
		t.Fatalf("failed to delete bundle: %v", err)
	}

	// Verify deletion
	_, err = storage.Retrieve(ctx, b.ID)
	if err == nil {
		t.Error("expected error when retrieving deleted bundle")
	}
}

func TestStorageCount(t *testing.T) {
	ctx := context.Background()
	storage := dtn.NewInMemoryStorage(100)

	// Initially empty
	count, err := storage.Count(ctx)
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 bundles, got %d", count)
	}

	// Store some bundles
	for i := 0; i < 5; i++ {
		b := bundle.NewBundle(
			"dtn://test/source",
			"dtn://test/dest",
			[]byte("data"),
		)
		if err := storage.Store(ctx, b); err != nil {
			t.Fatalf("failed to store bundle %d: %v", i, err)
		}
	}

	count, err = storage.Count(ctx)
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 bundles, got %d", count)
	}
}

func TestStorageStatus(t *testing.T) {
	ctx := context.Background()
	storage := dtn.NewInMemoryStorage(100)

	b := bundle.NewBundle(
		"dtn://test/source",
		"dtn://test/dest",
		[]byte("data"),
	)

	if err := storage.Store(ctx, b); err != nil {
		t.Fatalf("failed to store bundle: %v", err)
	}

	// Initial status should be pending
	status, err := storage.GetStatus(ctx, b.ID)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if status != dtn.StatusPending {
		t.Errorf("expected pending status, got %s", status)
	}

	// Update status
	if err := storage.UpdateStatus(ctx, b.ID, dtn.StatusInTransit); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	status, err = storage.GetStatus(ctx, b.ID)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	if status != dtn.StatusInTransit {
		t.Errorf("expected in_transit status, got %s", status)
	}
}

func TestStorageList(t *testing.T) {
	ctx := context.Background()
	storage := dtn.NewInMemoryStorage(100)

	dest := "dtn://mars/relay"

	// Store bundles for specific destination
	for i := 0; i < 3; i++ {
		b := bundle.NewBundle(
			"dtn://earth/source",
			dest,
			[]byte("mars data"),
		)
		storage.Store(ctx, b)
	}

	// Also store bundles for different destination
	for i := 0; i < 2; i++ {
		b := bundle.NewBundle(
			"dtn://earth/source",
			"dtn://lunar/base",
			[]byte("lunar data"),
		)
		storage.Store(ctx, b)
	}

	// List by destination
	filter := dtn.BundleFilter{
		DestinationEID: dest,
	}
	bundles, err := storage.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list bundles: %v", err)
	}

	if len(bundles) != 3 {
		t.Errorf("expected 3 mars bundles, got %d", len(bundles))
	}

	// Verify all returned bundles have correct destination
	for _, b := range bundles {
		if b.DestinationEID != dest {
			t.Errorf("unexpected destination: %s", b.DestinationEID)
		}
	}
}

func TestStorageCapacityEviction(t *testing.T) {
	ctx := context.Background()
	storage := dtn.NewInMemoryStorage(3) // Small capacity

	// Store 5 bundles, should evict some
	for i := 0; i < 5; i++ {
		b := bundle.NewBundle(
			"dtn://test/source",
			"dtn://test/dest",
			[]byte("data"),
		)
		storage.Store(ctx, b)
	}

	count, _ := storage.Count(ctx)
	if count > 3 {
		t.Errorf("expected at most 3 bundles after eviction, got %d", count)
	}
}

func TestStoragePurgeExpired(t *testing.T) {
	ctx := context.Background()
	storage := dtn.NewInMemoryStorage(100)

	// Store a normal bundle
	normalBundle := bundle.NewBundle(
		"dtn://test/source",
		"dtn://test/dest",
		[]byte("normal data"),
	)
	storage.Store(ctx, normalBundle)

	// Initial count
	count, _ := storage.Count(ctx)
	if count != 1 {
		t.Errorf("expected 1 bundle, got %d", count)
	}

	// Purge expired (none should be expired)
	purged, err := storage.PurgeExpired(ctx)
	if err != nil {
		t.Fatalf("failed to purge expired: %v", err)
	}

	// No bundles should be purged since default lifetime is 24 hours
	t.Logf("purged %d expired bundles", purged)
}
