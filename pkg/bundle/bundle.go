// Package bundle implements Bundle Protocol v7 (RFC 9171) for ASGARD's
// Delay Tolerant Networking layer. This enables communication across
// interplanetary distances with potentially hours of light-speed delay.
package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Version constants
const (
	BPv7Version uint8  = 7
	MaxHopCount uint32 = 255
)

// Priority levels for bundle transmission
const (
	PriorityBulk      uint8 = 0 // Best effort, lowest priority
	PriorityNormal    uint8 = 1 // Standard delivery
	PriorityExpedited uint8 = 2 // Highest priority, critical data
)

// Bundle represents a BPv7 bundle for delay-tolerant networking.
// This is the core data structure for all inter-node communication in ASGARD.
type Bundle struct {
	ID                uuid.UUID     `json:"id"`
	Version           uint8         `json:"version"`
	BundleFlags       uint64        `json:"bundleFlags"`
	DestinationEID    string        `json:"destinationEid"` // e.g., "dtn://earth/nysus"
	SourceEID         string        `json:"sourceEid"`      // e.g., "dtn://mars/sat001"
	ReportTo          string        `json:"reportTo"`       // Status report destination
	CreationTimestamp time.Time     `json:"creationTimestamp"`
	Lifetime          time.Duration `json:"lifetime"` // Bundle validity period
	Payload           []byte        `json:"payload"`
	CRCType           uint8         `json:"crcType"`      // 0=none, 1=CRC16, 2=CRC32
	PreviousNode      string        `json:"previousNode"` // Last node that forwarded
	HopCount          uint32        `json:"hopCount"`
	Priority          uint8         `json:"priority"`
	FragmentOffset    uint64        `json:"fragmentOffset,omitempty"`
	TotalADULength    uint64        `json:"totalAduLength,omitempty"`
	IsFragment        bool          `json:"isFragment"`
}

// NewBundle creates a new bundle with sensible defaults for ASGARD operations.
func NewBundle(source, destination string, payload []byte) *Bundle {
	return &Bundle{
		ID:                uuid.New(),
		Version:           BPv7Version,
		BundleFlags:       0,
		DestinationEID:    destination,
		SourceEID:         source,
		ReportTo:          source,
		CreationTimestamp: time.Now().UTC(),
		Lifetime:          24 * time.Hour, // Default 24-hour lifetime
		Payload:           payload,
		CRCType:           1, // CRC16 by default
		HopCount:          0,
		Priority:          PriorityNormal,
		IsFragment:        false,
	}
}

// NewPriorityBundle creates a bundle with specified priority level.
func NewPriorityBundle(source, destination string, payload []byte, priority uint8) (*Bundle, error) {
	if priority > PriorityExpedited {
		return nil, fmt.Errorf("invalid priority: %d (must be 0-2)", priority)
	}
	b := NewBundle(source, destination, payload)
	b.Priority = priority
	return b, nil
}

// Hash returns the SHA256 hash of the bundle for integrity verification.
func (b *Bundle) Hash() string {
	h := sha256.New()
	h.Write([]byte(b.ID.String()))
	h.Write([]byte(b.SourceEID))
	h.Write([]byte(b.DestinationEID))
	h.Write(b.Payload)
	return hex.EncodeToString(h.Sum(nil))
}

// IsExpired checks if the bundle has exceeded its lifetime.
func (b *Bundle) IsExpired() bool {
	expiryTime := b.CreationTimestamp.Add(b.Lifetime)
	return time.Now().UTC().After(expiryTime)
}

// ExpiresAt returns the expiration timestamp.
func (b *Bundle) ExpiresAt() time.Time {
	return b.CreationTimestamp.Add(b.Lifetime)
}

// RemainingLifetime returns the time until expiration.
func (b *Bundle) RemainingLifetime() time.Duration {
	remaining := b.ExpiresAt().Sub(time.Now().UTC())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IncrementHop updates hop tracking when forwarded through a node.
func (b *Bundle) IncrementHop(nodeID string) error {
	if b.HopCount >= MaxHopCount {
		return fmt.Errorf("max hop count exceeded (%d)", MaxHopCount)
	}
	b.HopCount++
	b.PreviousNode = nodeID
	return nil
}

// Validate performs comprehensive bundle validation.
func (b *Bundle) Validate() error {
	if b.Version != BPv7Version {
		return fmt.Errorf("invalid bundle version: %d (expected %d)", b.Version, BPv7Version)
	}
	if b.DestinationEID == "" {
		return fmt.Errorf("destination EID cannot be empty")
	}
	if b.SourceEID == "" {
		return fmt.Errorf("source EID cannot be empty")
	}
	if b.IsExpired() {
		return fmt.Errorf("bundle has expired at %s", b.ExpiresAt().Format(time.RFC3339))
	}
	if b.HopCount > MaxHopCount {
		return fmt.Errorf("hop count exceeded maximum (%d)", MaxHopCount)
	}
	if b.Priority > PriorityExpedited {
		return fmt.Errorf("invalid priority: %d", b.Priority)
	}
	return nil
}

// SetPriority safely sets the bundle priority.
func (b *Bundle) SetPriority(priority uint8) error {
	if priority > PriorityExpedited {
		return fmt.Errorf("invalid priority: %d (must be 0-2)", priority)
	}
	b.Priority = priority
	return nil
}

// SetLifetime updates the bundle lifetime.
func (b *Bundle) SetLifetime(d time.Duration) {
	b.Lifetime = d
}

// Clone creates a deep copy of the bundle.
func (b *Bundle) Clone() *Bundle {
	payloadCopy := make([]byte, len(b.Payload))
	copy(payloadCopy, b.Payload)

	return &Bundle{
		ID:                b.ID,
		Version:           b.Version,
		BundleFlags:       b.BundleFlags,
		DestinationEID:    b.DestinationEID,
		SourceEID:         b.SourceEID,
		ReportTo:          b.ReportTo,
		CreationTimestamp: b.CreationTimestamp,
		Lifetime:          b.Lifetime,
		Payload:           payloadCopy,
		CRCType:           b.CRCType,
		PreviousNode:      b.PreviousNode,
		HopCount:          b.HopCount,
		Priority:          b.Priority,
		FragmentOffset:    b.FragmentOffset,
		TotalADULength:    b.TotalADULength,
		IsFragment:        b.IsFragment,
	}
}

// Size returns the approximate size of the bundle in bytes.
func (b *Bundle) Size() int {
	// Header size + payload
	headerSize := 64 + len(b.SourceEID) + len(b.DestinationEID) + len(b.ReportTo) + len(b.PreviousNode)
	return headerSize + len(b.Payload)
}

// String returns a human-readable representation of the bundle.
func (b *Bundle) String() string {
	return fmt.Sprintf("Bundle[id=%s, src=%s, dst=%s, priority=%d, hops=%d, size=%d]",
		b.ID.String()[:8], b.SourceEID, b.DestinationEID, b.Priority, b.HopCount, b.Size())
}
