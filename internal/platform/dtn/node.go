package dtn

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/pkg/bundle"
)

// Node represents a DTN network node (satellite, ground station, Hunoid, etc.).
type Node struct {
	ID          string
	EID         string // Endpoint Identifier (e.g., "dtn://earth/nysus")
	storage     BundleStorage
	router      Router
	transport   TransportAdapter
	neighbors   map[string]*Neighbor
	neighborsMu sync.RWMutex
	ingressChan chan *bundle.Bundle
	egressChan  chan *bundle.Bundle
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	metrics     *NodeMetrics
	metricsMu   sync.RWMutex
}

// Neighbor represents a connected DTN node with link quality information.
type Neighbor struct {
	ID           string
	EID          string
	Address      string
	LinkQuality  float64 // 0.0 to 1.0
	LastContact  time.Time
	IsActive     bool
	Latency      time.Duration
	Bandwidth    int64 // bytes per second
	ContactStart time.Time
	ContactEnd   time.Time
}

// NodeMetrics tracks node performance statistics.
type NodeMetrics struct {
	BundlesReceived   int64
	BundlesSent       int64
	BundlesDropped    int64
	BundlesExpired    int64
	BytesReceived     int64
	BytesSent         int64
	AverageLatency    time.Duration
	ActiveConnections int
}

// Router interface for selecting the next hop for bundle forwarding.
type Router interface {
	SelectNextHop(ctx context.Context, b *bundle.Bundle, neighbors map[string]*Neighbor) (string, error)
	UpdateContactGraph(nodeID string, neighbor *Neighbor)
}

// NodeConfig contains configuration options for a DTN node.
type NodeConfig struct {
	BufferSize     int           // Channel buffer size
	ProcessTimeout time.Duration // Max time to process a bundle
	MaxRetries     int           // Retry attempts for failed transmissions
	PurgeInterval  time.Duration // How often to purge expired bundles
}

// DefaultNodeConfig returns sensible defaults.
func DefaultNodeConfig() NodeConfig {
	return NodeConfig{
		BufferSize:     1000,
		ProcessTimeout: 30 * time.Second,
		MaxRetries:     3,
		PurgeInterval:  5 * time.Minute,
	}
}

// NewNode creates a new DTN node.
func NewNode(id, eid string, storage BundleStorage, router Router, config NodeConfig) *Node {
	ctx, cancel := context.WithCancel(context.Background())

	return &Node{
		ID:          id,
		EID:         eid,
		storage:     storage,
		router:      router,
		transport:   nil, // Transport is optional, set via SetTransport
		neighbors:   make(map[string]*Neighbor),
		ingressChan: make(chan *bundle.Bundle, config.BufferSize),
		egressChan:  make(chan *bundle.Bundle, config.BufferSize),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &NodeMetrics{},
	}
}

// NewNodeWithTransport creates a new DTN node with a transport adapter.
func NewNodeWithTransport(id, eid string, storage BundleStorage, router Router, transport TransportAdapter, config NodeConfig) *Node {
	node := NewNode(id, eid, storage, router, config)
	node.transport = transport
	return node
}

// SetTransport sets the transport adapter for the node.
func (n *Node) SetTransport(transport TransportAdapter) {
	n.transport = transport
}

// Start begins node operations.
func (n *Node) Start() error {
	log.Printf("[DTN Node %s] Starting at EID: %s", n.ID, n.EID)

	// Start transport if configured
	if n.transport != nil {
		if err := n.transport.Start(n.ctx); err != nil {
			return fmt.Errorf("failed to start transport: %w", err)
		}

		// Start transport receiver goroutine
		n.wg.Add(1)
		go n.processTransportReceive()
	}

	// Start ingress processor
	n.wg.Add(1)
	go n.processIngress()

	// Start egress processor
	n.wg.Add(1)
	go n.processEgress()

	// Start maintenance goroutine
	n.wg.Add(1)
	go n.runMaintenance()

	return nil
}

// Stop gracefully shuts down the node.
func (n *Node) Stop() error {
	log.Printf("[DTN Node %s] Shutting down", n.ID)
	n.cancel()

	// Stop transport if configured
	if n.transport != nil {
		if err := n.transport.Stop(); err != nil {
			log.Printf("[DTN Node %s] Error stopping transport: %v", n.ID, err)
		}
	}

	n.wg.Wait()
	close(n.ingressChan)
	close(n.egressChan)
	return nil
}

// Receive accepts an incoming bundle for processing.
func (n *Node) Receive(ctx context.Context, b *bundle.Bundle) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case n.ingressChan <- b:
		n.recordMetric(func(m *NodeMetrics) {
			m.BundlesReceived++
			m.BytesReceived += int64(b.Size())
		})
		return nil
	default:
		n.recordMetric(func(m *NodeMetrics) { m.BundlesDropped++ })
		return fmt.Errorf("ingress queue full")
	}
}

// Send queues a bundle for outgoing transmission.
func (n *Node) Send(ctx context.Context, b *bundle.Bundle) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case n.egressChan <- b:
		return nil
	default:
		return fmt.Errorf("egress queue full")
	}
}

// RegisterNeighbor adds or updates a neighbor node.
func (n *Node) RegisterNeighbor(neighbor *Neighbor) {
	n.neighborsMu.Lock()
	defer n.neighborsMu.Unlock()

	neighbor.LastContact = time.Now().UTC()
	n.neighbors[neighbor.ID] = neighbor

	// Update router's contact graph
	if n.router != nil {
		n.router.UpdateContactGraph(n.ID, neighbor)
	}

	n.recordMetric(func(m *NodeMetrics) {
		m.ActiveConnections = len(n.neighbors)
	})
}

// UnregisterNeighbor removes a neighbor node.
func (n *Node) UnregisterNeighbor(neighborID string) {
	n.neighborsMu.Lock()
	defer n.neighborsMu.Unlock()

	delete(n.neighbors, neighborID)

	n.recordMetric(func(m *NodeMetrics) {
		m.ActiveConnections = len(n.neighbors)
	})
}

// GetNeighbors returns a copy of the current neighbors.
func (n *Node) GetNeighbors() map[string]*Neighbor {
	n.neighborsMu.RLock()
	defer n.neighborsMu.RUnlock()

	result := make(map[string]*Neighbor, len(n.neighbors))
	for k, v := range n.neighbors {
		result[k] = &Neighbor{
			ID:           v.ID,
			EID:          v.EID,
			Address:      v.Address,
			LinkQuality:  v.LinkQuality,
			LastContact:  v.LastContact,
			IsActive:     v.IsActive,
			Latency:      v.Latency,
			Bandwidth:    v.Bandwidth,
			ContactStart: v.ContactStart,
			ContactEnd:   v.ContactEnd,
		}
	}
	return result
}

// GetMetrics returns current node metrics.
func (n *Node) GetMetrics() NodeMetrics {
	n.metricsMu.RLock()
	defer n.metricsMu.RUnlock()
	return *n.metrics
}

// processIngress handles incoming bundles.
func (n *Node) processIngress() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		case b := <-n.ingressChan:
			if b == nil {
				continue
			}
			n.handleIncomingBundle(b)
		}
	}
}

// handleIncomingBundle processes a single incoming bundle.
func (n *Node) handleIncomingBundle(b *bundle.Bundle) {
	// Validate bundle
	if err := b.Validate(); err != nil {
		log.Printf("[DTN Node %s] Invalid bundle: %v", n.ID, err)
		n.recordMetric(func(m *NodeMetrics) { m.BundlesDropped++ })
		return
	}

	// Check if we are the destination
	if b.DestinationEID == n.EID {
		n.deliverLocally(b)
		return
	}

	// Store for forwarding
	if err := n.storage.Store(n.ctx, b); err != nil {
		log.Printf("[DTN Node %s] Failed to store bundle: %v", n.ID, err)
		return
	}

	// Increment hop count
	if err := b.IncrementHop(n.ID); err != nil {
		log.Printf("[DTN Node %s] Bundle exceeded hop limit: %v", n.ID, err)
		n.recordMetric(func(m *NodeMetrics) { m.BundlesDropped++ })
		return
	}

	// Queue for forwarding
	if err := n.Send(n.ctx, b); err != nil {
		log.Printf("[DTN Node %s] Failed to queue for egress: %v", n.ID, err)
	}
}

// deliverLocally handles bundles destined for this node.
func (n *Node) deliverLocally(b *bundle.Bundle) {
	log.Printf("[DTN Node %s] Delivered bundle %s from %s", n.ID, b.ID, b.SourceEID)

	// Store as delivered
	if err := n.storage.Store(n.ctx, b); err == nil {
		n.storage.UpdateStatus(n.ctx, b.ID, StatusDelivered)
	}

	// In a full implementation, this would dispatch to local handlers
}

// processEgress handles outgoing bundle forwarding.
func (n *Node) processEgress() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		case b := <-n.egressChan:
			if b == nil {
				continue
			}
			n.forwardBundle(b)
		}
	}
}

// forwardBundle attempts to forward a bundle to the next hop.
func (n *Node) forwardBundle(b *bundle.Bundle) {
	neighbors := n.GetNeighbors()

	// Select next hop using router
	nextHop, err := n.router.SelectNextHop(n.ctx, b, neighbors)
	if err != nil {
		log.Printf("[DTN Node %s] No route to %s: %v", n.ID, b.DestinationEID, err)
		// Keep in storage for later retry
		return
	}

	neighbor, exists := neighbors[nextHop]
	if !exists || !neighbor.IsActive {
		log.Printf("[DTN Node %s] Next hop %s not available", n.ID, nextHop)
		return
	}

	if n.transport == nil {
		log.Printf("[DTN Node %s] Transport not configured; cannot send bundle %s to %s",
			n.ID, b.ID.String()[:8], nextHop)
		n.storage.UpdateStatus(n.ctx, b.ID, StatusFailed)
		return
	}

	if !n.transport.IsConnected(nextHop) {
		log.Printf("[DTN Node %s] Transport not connected to %s (EID: %s)", n.ID, nextHop, neighbor.EID)
		n.storage.UpdateStatus(n.ctx, b.ID, StatusFailed)
		return
	}

	if err := n.transport.Send(n.ctx, nextHop, b); err != nil {
		log.Printf("[DTN Node %s] Failed to send bundle %s to %s: %v",
			n.ID, b.ID.String()[:8], nextHop, err)
		n.storage.UpdateStatus(n.ctx, b.ID, StatusFailed)
		return
	}

	log.Printf("[DTN Node %s] Successfully sent bundle %s to %s via transport",
		n.ID, b.ID.String()[:8], nextHop)

	n.recordMetric(func(m *NodeMetrics) {
		m.BundlesSent++
		m.BytesSent += int64(b.Size())
	})

	n.storage.UpdateStatus(n.ctx, b.ID, StatusInTransit)
}

// runMaintenance performs periodic cleanup tasks.
func (n *Node) runMaintenance() {
	defer n.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			// Purge expired bundles
			purged, err := n.storage.PurgeExpired(n.ctx)
			if err != nil {
				log.Printf("[DTN Node %s] Purge error: %v", n.ID, err)
			} else if purged > 0 {
				log.Printf("[DTN Node %s] Purged %d expired bundles", n.ID, purged)
				n.recordMetric(func(m *NodeMetrics) { m.BundlesExpired += int64(purged) })
			}

			// Check neighbor health
			n.checkNeighborHealth()
		}
	}
}

// processTransportReceive handles incoming bundles from the transport layer.
func (n *Node) processTransportReceive() {
	defer n.wg.Done()

	if n.transport == nil {
		return
	}

	receiveChan := n.transport.Receive()

	for {
		select {
		case <-n.ctx.Done():
			return
		case b, ok := <-receiveChan:
			if !ok {
				// Channel closed, transport shut down
				log.Printf("[DTN Node %s] Transport receive channel closed", n.ID)
				return
			}
			if b == nil {
				continue
			}

			log.Printf("[DTN Node %s] Received bundle %s from transport (src: %s, dst: %s)",
				n.ID, b.ID.String()[:8], b.SourceEID, b.DestinationEID)

			// Process the received bundle through normal ingress
			if err := n.Receive(n.ctx, b); err != nil {
				log.Printf("[DTN Node %s] Failed to process received bundle %s: %v",
					n.ID, b.ID.String()[:8], err)
			}
		}
	}
}

// checkNeighborHealth marks stale neighbors as inactive.
func (n *Node) checkNeighborHealth() {
	n.neighborsMu.Lock()
	defer n.neighborsMu.Unlock()

	threshold := time.Now().Add(-10 * time.Minute)

	for _, neighbor := range n.neighbors {
		if neighbor.LastContact.Before(threshold) {
			neighbor.IsActive = false
		}
	}
}

// recordMetric safely updates node metrics.
func (n *Node) recordMetric(update func(*NodeMetrics)) {
	n.metricsMu.Lock()
	defer n.metricsMu.Unlock()
	update(n.metrics)
}

// CreateBundle is a convenience method to create and queue a bundle.
func (n *Node) CreateBundle(destination string, payload []byte, priority uint8) error {
	b, err := bundle.NewPriorityBundle(n.EID, destination, payload, priority)
	if err != nil {
		return err
	}
	return n.Send(n.ctx, b)
}

// ConnectNeighbor establishes a transport connection to a neighbor.
// This should be called after RegisterNeighbor to enable actual data transmission.
func (n *Node) ConnectNeighbor(ctx context.Context, neighborID string, address string) error {
	if n.transport == nil {
		return fmt.Errorf("no transport configured")
	}

	if err := n.transport.Connect(ctx, neighborID, address); err != nil {
		return fmt.Errorf("failed to connect to neighbor %s: %w", neighborID, err)
	}

	// Update neighbor's active status
	n.neighborsMu.Lock()
	if neighbor, exists := n.neighbors[neighborID]; exists {
		neighbor.IsActive = true
		neighbor.LastContact = time.Now().UTC()
	}
	n.neighborsMu.Unlock()

	return nil
}

// DisconnectNeighbor closes the transport connection to a neighbor.
func (n *Node) DisconnectNeighbor(neighborID string) error {
	if n.transport == nil {
		return fmt.Errorf("no transport configured")
	}

	if err := n.transport.Disconnect(neighborID); err != nil {
		return fmt.Errorf("failed to disconnect from neighbor %s: %w", neighborID, err)
	}

	// Update neighbor's active status
	n.neighborsMu.Lock()
	if neighbor, exists := n.neighbors[neighborID]; exists {
		neighbor.IsActive = false
	}
	n.neighborsMu.Unlock()

	return nil
}

// IsNeighborConnected returns true if the transport is connected to a neighbor.
func (n *Node) IsNeighborConnected(neighborID string) bool {
	if n.transport == nil {
		return false
	}
	return n.transport.IsConnected(neighborID)
}

// GetTransport returns the node's transport adapter.
func (n *Node) GetTransport() TransportAdapter {
	return n.transport
}
