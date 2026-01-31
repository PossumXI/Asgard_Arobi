// Package dtn implements Delay Tolerant Networking infrastructure for ASGARD.
// This enables communication between Earth, orbital satellites, and
// interplanetary assets with support for intermittent connectivity.
package dtn

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/asgard/pandora/pkg/bundle"
	"github.com/google/uuid"
)

// BundleStatus represents the lifecycle state of a bundle.
type BundleStatus string

const (
	StatusPending   BundleStatus = "pending"    // Awaiting transmission
	StatusInTransit BundleStatus = "in_transit" // Currently being forwarded
	StatusDelivered BundleStatus = "delivered"  // Successfully delivered
	StatusFailed    BundleStatus = "failed"     // Delivery failed
	StatusExpired   BundleStatus = "expired"    // TTL exceeded
)

// BundleStorage defines the interface for bundle persistence.
type BundleStorage interface {
	Store(ctx context.Context, b *bundle.Bundle) error
	Retrieve(ctx context.Context, id uuid.UUID) (*bundle.Bundle, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter BundleFilter) ([]*bundle.Bundle, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status BundleStatus) error
	GetStatus(ctx context.Context, id uuid.UUID) (BundleStatus, error)
	Count(ctx context.Context) (int, error)
	PurgeExpired(ctx context.Context) (int, error)
}

// BundleFilter specifies criteria for querying bundles.
type BundleFilter struct {
	DestinationEID string
	SourceEID      string
	Status         BundleStatus
	MinPriority    uint8
	MaxAge         time.Duration
	Limit          int
	OrderBy        string // "priority", "age", "size"
}

// storedBundle wraps a bundle with metadata.
type storedBundle struct {
	bundle   *bundle.Bundle
	status   BundleStatus
	storedAt time.Time
}

// InMemoryStorage provides an in-memory bundle store.
// Suitable for development and testing, or nodes with limited persistence needs.
type InMemoryStorage struct {
	mu      sync.RWMutex
	bundles map[uuid.UUID]*storedBundle
	maxSize int
}

// NewInMemoryStorage creates a new in-memory storage with optional max capacity.
func NewInMemoryStorage(maxSize int) *InMemoryStorage {
	if maxSize <= 0 {
		maxSize = 10000 // Default 10k bundles
	}
	return &InMemoryStorage{
		bundles: make(map[uuid.UUID]*storedBundle),
		maxSize: maxSize,
	}
}

// Store persists a bundle to storage.
func (s *InMemoryStorage) Store(ctx context.Context, b *bundle.Bundle) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid bundle: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check capacity
	if len(s.bundles) >= s.maxSize {
		// Evict expired bundles first
		s.evictExpiredLocked()

		// If still at capacity, evict lowest priority
		if len(s.bundles) >= s.maxSize {
			s.evictLowestPriorityLocked()
		}
	}

	s.bundles[b.ID] = &storedBundle{
		bundle:   b.Clone(),
		status:   StatusPending,
		storedAt: time.Now().UTC(),
	}

	return nil
}

// Retrieve fetches a bundle by ID.
func (s *InMemoryStorage) Retrieve(ctx context.Context, id uuid.UUID) (*bundle.Bundle, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.bundles[id]
	if !exists {
		return nil, fmt.Errorf("bundle not found: %s", id)
	}

	return stored.bundle.Clone(), nil
}

// Delete removes a bundle from storage.
func (s *InMemoryStorage) Delete(ctx context.Context, id uuid.UUID) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.bundles[id]; !exists {
		return fmt.Errorf("bundle not found: %s", id)
	}

	delete(s.bundles, id)
	return nil
}

// List returns bundles matching the filter criteria.
func (s *InMemoryStorage) List(ctx context.Context, filter BundleFilter) ([]*bundle.Bundle, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*storedBundle

	for _, stored := range s.bundles {
		if s.matchesFilter(stored, filter) {
			results = append(results, stored)
		}
	}

	// Sort results
	switch filter.OrderBy {
	case "priority":
		sort.Slice(results, func(i, j int) bool {
			return results[i].bundle.Priority > results[j].bundle.Priority
		})
	case "age":
		sort.Slice(results, func(i, j int) bool {
			return results[i].storedAt.Before(results[j].storedAt)
		})
	case "size":
		sort.Slice(results, func(i, j int) bool {
			return results[i].bundle.Size() < results[j].bundle.Size()
		})
	default:
		// Default: priority then age
		sort.Slice(results, func(i, j int) bool {
			if results[i].bundle.Priority != results[j].bundle.Priority {
				return results[i].bundle.Priority > results[j].bundle.Priority
			}
			return results[i].storedAt.Before(results[j].storedAt)
		})
	}

	// Apply limit
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	// Convert to bundle slice
	bundles := make([]*bundle.Bundle, len(results))
	for i, stored := range results {
		bundles[i] = stored.bundle.Clone()
	}

	return bundles, nil
}

// UpdateStatus changes the status of a bundle.
func (s *InMemoryStorage) UpdateStatus(ctx context.Context, id uuid.UUID, status BundleStatus) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stored, exists := s.bundles[id]
	if !exists {
		return fmt.Errorf("bundle not found: %s", id)
	}

	stored.status = status
	return nil
}

// GetStatus returns the current status of a bundle.
func (s *InMemoryStorage) GetStatus(ctx context.Context, id uuid.UUID) (BundleStatus, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.bundles[id]
	if !exists {
		return "", fmt.Errorf("bundle not found: %s", id)
	}

	return stored.status, nil
}

// Count returns the total number of bundles in storage.
func (s *InMemoryStorage) Count(ctx context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.bundles), nil
}

// PurgeExpired removes all expired bundles.
func (s *InMemoryStorage) PurgeExpired(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.evictExpiredLocked(), nil
}

// matchesFilter checks if a stored bundle matches the filter criteria.
func (s *InMemoryStorage) matchesFilter(stored *storedBundle, filter BundleFilter) bool {
	if filter.DestinationEID != "" && stored.bundle.DestinationEID != filter.DestinationEID {
		return false
	}
	if filter.SourceEID != "" && stored.bundle.SourceEID != filter.SourceEID {
		return false
	}
	if filter.Status != "" && stored.status != filter.Status {
		return false
	}
	if stored.bundle.Priority < filter.MinPriority {
		return false
	}
	if filter.MaxAge > 0 && time.Since(stored.storedAt) > filter.MaxAge {
		return false
	}
	return true
}

// evictExpiredLocked removes expired bundles (caller must hold lock).
func (s *InMemoryStorage) evictExpiredLocked() int {
	count := 0
	for id, stored := range s.bundles {
		if stored.bundle.IsExpired() {
			delete(s.bundles, id)
			count++
		}
	}
	return count
}

// evictLowestPriorityLocked removes the lowest priority bundle (caller must hold lock).
func (s *InMemoryStorage) evictLowestPriorityLocked() {
	var lowestID uuid.UUID
	var lowestPriority uint8 = 255
	var oldestTime time.Time

	for id, stored := range s.bundles {
		if stored.bundle.Priority < lowestPriority ||
			(stored.bundle.Priority == lowestPriority && stored.storedAt.Before(oldestTime)) {
			lowestID = id
			lowestPriority = stored.bundle.Priority
			oldestTime = stored.storedAt
		}
	}

	if lowestID != uuid.Nil {
		delete(s.bundles, lowestID)
	}
}
