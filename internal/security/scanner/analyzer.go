package scanner

import (
	"context"
	"fmt"
	"math"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
)

// TrafficAnalyzer performs statistical analysis and anomaly detection on network traffic.
type TrafficAnalyzer struct {
	mu                sync.RWMutex
	baselines          map[string]*Baseline
	anomalyThresholds map[string]float64
	recentPackets      []PacketInfo
	maxRecentPackets   int
	threatPatterns     map[string]*ThreatPattern
}

// Baseline tracks statistical baselines for traffic patterns.
type Baseline struct {
	AvgPacketSize    float64
	AvgPacketsPerSec float64
	UniqueIPs        map[string]int
	PortDistribution map[int]int
	LastUpdated      time.Time
	SampleCount      int
}

// ThreatPattern defines detection rules for specific threats.
type ThreatPattern struct {
	Name        string
	Type        string
	Severity    ThreatLevel
	DetectFunc  func(PacketInfo, *Baseline) (bool, float64, string)
}

// NewTrafficAnalyzer creates a new traffic analyzer.
func NewTrafficAnalyzer() *TrafficAnalyzer {
	analyzer := &TrafficAnalyzer{
		baselines:          make(map[string]*Baseline),
		anomalyThresholds: make(map[string]float64),
		recentPackets:      make([]PacketInfo, 0, 1000),
		maxRecentPackets:   1000,
		threatPatterns:     make(map[string]*ThreatPattern),
	}

	// Initialize threat patterns
	analyzer.initThreatPatterns()

	return analyzer
}

// AnalyzePacket analyzes a packet and returns an anomaly if detected.
func (ta *TrafficAnalyzer) AnalyzePacket(ctx context.Context, packet PacketInfo) (*Anomaly, error) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	// Update recent packets buffer
	ta.recentPackets = append(ta.recentPackets, packet)
	if len(ta.recentPackets) > ta.maxRecentPackets {
		ta.recentPackets = ta.recentPackets[1:]
	}

	// Get or create baseline for this destination
	destKey := packet.DestIP.String()
	baseline, exists := ta.baselines[destKey]
	if !exists {
		baseline = &Baseline{
			UniqueIPs:        make(map[string]int),
			PortDistribution: make(map[int]int),
			LastUpdated:      time.Now(),
		}
		ta.baselines[destKey] = baseline
	}

	// Update baseline
	ta.updateBaseline(baseline, packet)

	// Check against threat patterns
	for _, pattern := range ta.threatPatterns {
		detected, confidence, description := pattern.DetectFunc(packet, baseline)
		if detected {
			observability.GetMetrics().SecurityPacketsScanned.Inc()
			return &Anomaly{
				Type:        pattern.Type,
				Severity:    pattern.Severity,
				SourceIP:    packet.SourceIP,
				Description: description,
				Timestamp:   packet.Timestamp,
				Confidence:  confidence,
			}, nil
		}
	}

	// Statistical anomaly detection
	anomaly := ta.detectStatisticalAnomaly(packet, baseline)
	if anomaly != nil {
		observability.GetMetrics().SecurityPacketsScanned.Inc()
		return anomaly, nil
	}

	observability.GetMetrics().SecurityPacketsScanned.Inc()
	return nil, nil
}

// updateBaseline updates statistical baselines.
func (ta *TrafficAnalyzer) updateBaseline(baseline *Baseline, packet PacketInfo) {
	now := time.Now()
	elapsed := now.Sub(baseline.LastUpdated).Seconds()
	if elapsed < 1.0 {
		elapsed = 1.0
	}

	// Update average packet size (exponential moving average)
	alpha := 0.1
	baseline.AvgPacketSize = alpha*float64(packet.Size) + (1-alpha)*baseline.AvgPacketSize

	// Update packets per second
	baseline.SampleCount++
	baseline.AvgPacketsPerSec = float64(baseline.SampleCount) / elapsed

	// Update unique IPs
	baseline.UniqueIPs[packet.SourceIP.String()]++

	// Update port distribution
	baseline.PortDistribution[packet.DestPort]++

	baseline.LastUpdated = now
}

// detectStatisticalAnomaly detects anomalies based on statistical deviations.
func (ta *TrafficAnalyzer) detectStatisticalAnomaly(packet PacketInfo, baseline *Baseline) *Anomaly {
	// Detect port scan (many unique IPs connecting to many ports)
	if len(baseline.UniqueIPs) > 50 && len(baseline.PortDistribution) > 20 {
		return &Anomaly{
			Type:        "port_scan",
			Severity:    ThreatLevelMedium,
			SourceIP:    packet.SourceIP,
			Description: fmt.Sprintf("Port scan detected: %d unique IPs, %d ports", len(baseline.UniqueIPs), len(baseline.PortDistribution)),
			Timestamp:   packet.Timestamp,
			Confidence:  0.75,
		}
	}

	// Detect unusually large packet
	if baseline.AvgPacketSize > 0 && float64(packet.Size) > baseline.AvgPacketSize*3 {
		return &Anomaly{
			Type:        "suspicious_payload",
			Severity:    ThreatLevelMedium,
			SourceIP:    packet.SourceIP,
			Description: fmt.Sprintf("Unusually large packet: %d bytes (avg: %.0f)", packet.Size, baseline.AvgPacketSize),
			Timestamp:   packet.Timestamp,
			Confidence:  0.65,
		}
	}

	// Detect high packet rate (potential DDoS)
	if baseline.AvgPacketsPerSec > 1000 {
		return &Anomaly{
			Type:        "ddos",
			Severity:    ThreatLevelCritical,
			SourceIP:    packet.SourceIP,
			Description: fmt.Sprintf("High packet rate detected: %.0f pps", baseline.AvgPacketsPerSec),
			Timestamp:   packet.Timestamp,
			Confidence:  0.85,
		}
	}

	return nil
}

// initThreatPatterns initializes threat detection patterns.
func (ta *TrafficAnalyzer) initThreatPatterns() {
	// SQL Injection pattern
	ta.threatPatterns["sql_injection"] = &ThreatPattern{
		Name:     "SQL Injection",
		Type:     "sql_injection",
		Severity: ThreatLevelHigh,
		DetectFunc: func(packet PacketInfo, baseline *Baseline) (bool, float64, string) {
			if len(packet.Payload) == 0 {
				return false, 0, ""
			}
			payload := strings.ToLower(string(packet.Payload))
			sqlPatterns := []string{
				"union.*select", "select.*from", "insert.*into", "delete.*from",
				"drop.*table", "exec.*xp_", "or.*1=1", "and.*1=1",
				"' or '1'='1", "'; drop table", "union all select",
			}
			for _, pattern := range sqlPatterns {
				matched, _ := regexp.MatchString(pattern, payload)
				if matched {
					return true, 0.9, fmt.Sprintf("SQL injection pattern detected: %s", pattern)
				}
			}
			return false, 0, ""
		},
	}

	// XSS pattern
	ta.threatPatterns["xss_attack"] = &ThreatPattern{
		Name:     "XSS Attack",
		Type:     "xss_attack",
		Severity: ThreatLevelHigh,
		DetectFunc: func(packet PacketInfo, baseline *Baseline) (bool, float64, string) {
			if len(packet.Payload) == 0 {
				return false, 0, ""
			}
			payload := strings.ToLower(string(packet.Payload))
			xssPatterns := []string{
				"<script", "javascript:", "onerror=", "onload=",
				"<iframe", "<img.*onerror", "eval\\(", "document\\.cookie",
			}
			for _, pattern := range xssPatterns {
				matched, _ := regexp.MatchString(pattern, payload)
				if matched {
					return true, 0.85, fmt.Sprintf("XSS pattern detected: %s", pattern)
				}
			}
			return false, 0, ""
		},
	}

	// Suspicious payload pattern (hex-encoded or base64-like)
	ta.threatPatterns["suspicious_payload"] = &ThreatPattern{
		Name:     "Suspicious Payload",
		Type:     "suspicious_payload",
		Severity: ThreatLevelMedium,
		DetectFunc: func(packet PacketInfo, baseline *Baseline) (bool, float64, string) {
			if len(packet.Payload) < 100 {
				return false, 0, ""
			}
			// Check for high entropy (encrypted/compressed data)
			entropy := calculateEntropy(packet.Payload)
			if entropy > 7.5 {
				return true, 0.7, fmt.Sprintf("High entropy payload detected (entropy: %.2f)", entropy)
			}
			return false, 0, ""
		},
	}
}

// calculateEntropy calculates Shannon entropy of data.
func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	entropy := 0.0
	length := float64(len(data))
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// GetBaseline returns the baseline for a destination.
func (ta *TrafficAnalyzer) GetBaseline(destIP net.IP) *Baseline {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	return ta.baselines[destIP.String()]
}
