package scanner

import (
	"context"
	"sync"
	"time"
)

// RealtimeScanner implements the Scanner interface with real packet capture and analysis.
type RealtimeScanner struct {
	capture  *PacketCapture
	analyzer *TrafficAnalyzer
	stats    Statistics
	mu       sync.RWMutex
	running  bool
	onAnomaly func(*Anomaly)
}

// NewRealtimeScanner creates a new real-time packet scanner.
func NewRealtimeScanner(interfaceName string) (*RealtimeScanner, error) {
	capture, err := NewPacketCapture(interfaceName, 1600, true)
	if err != nil {
		return nil, err
	}

	return &RealtimeScanner{
		capture:  capture,
		analyzer: NewTrafficAnalyzer(),
		stats: Statistics{
			StartTime: time.Now(),
		},
	}, nil
}

// Start begins packet capture and analysis.
func (rs *RealtimeScanner) Start(ctx context.Context) error {
	rs.mu.Lock()
	if rs.running {
		rs.mu.Unlock()
		return nil
	}
	rs.running = true
	rs.mu.Unlock()

	if err := rs.capture.Start(ctx); err != nil {
		return err
	}

	// Start packet processing goroutine
	go rs.processPackets(ctx)

	return nil
}

// Stop stops packet capture.
func (rs *RealtimeScanner) Stop() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.running = false
	return rs.capture.Stop()
}

// ScanPacket scans a single packet (for compatibility with interface).
// In real-time mode, packets are processed automatically.
func (rs *RealtimeScanner) ScanPacket(ctx context.Context, packet PacketInfo) (*Anomaly, error) {
	rs.mu.Lock()
	rs.stats.PacketsScanned++
	rs.mu.Unlock()

	return rs.analyzer.AnalyzePacket(ctx, packet)
}

// GetStatistics returns scanner statistics.
func (rs *RealtimeScanner) GetStatistics() Statistics {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.stats
}

// processPackets processes captured packets in real-time.
func (rs *RealtimeScanner) processPackets(ctx context.Context) {
	packetChan := rs.capture.GetPacketChannel()

	for {
		select {
		case <-ctx.Done():
			return
		case packet, ok := <-packetChan:
			if !ok {
				return
			}

			// Convert gopacket to PacketInfo
			packetInfo := rs.capture.ConvertPacket(packet)

			// Analyze packet
			anomaly, err := rs.analyzer.AnalyzePacket(ctx, packetInfo)
			if err != nil {
				continue
			}

			rs.mu.Lock()
			rs.stats.PacketsScanned++
			if anomaly != nil {
				rs.stats.AnomaliesDetected++
				if anomaly.Severity == ThreatLevelCritical || anomaly.Severity == ThreatLevelHigh {
					rs.stats.ThreatsBlocked++
				}
			}
			rs.mu.Unlock()

			if anomaly != nil && rs.onAnomaly != nil {
				rs.onAnomaly(anomaly)
			}
		}
	}
}

// SetAnomalyCallback sets a callback function for when anomalies are detected.
func (rs *RealtimeScanner) SetAnomalyCallback(callback func(*Anomaly)) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.onAnomaly = callback
}
