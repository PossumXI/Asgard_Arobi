package scanner

import (
	"context"
	"net"
	"time"
)

// PacketInfo represents network packet metadata
type PacketInfo struct {
	SourceIP      net.IP
	DestIP        net.IP
	SourcePort    int
	DestPort      int
	Protocol      string
	Size          int
	Timestamp     time.Time
	Payload       []byte
	Flags         string
}

// ThreatLevel represents threat severity
type ThreatLevel string

const (
	ThreatLevelLow      ThreatLevel = "low"
	ThreatLevelMedium   ThreatLevel = "medium"
	ThreatLevelHigh     ThreatLevel = "high"
	ThreatLevelCritical ThreatLevel = "critical"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        string
	Severity    ThreatLevel
	SourceIP    net.IP
	Description string
	Timestamp   time.Time
	Confidence  float64
}

// Scanner defines the interface for network traffic analysis
type Scanner interface {
	Start(ctx context.Context) error
	Stop() error
	ScanPacket(ctx context.Context, packet PacketInfo) (*Anomaly, error)
	GetStatistics() Statistics
}

// Statistics contains scanner metrics
type Statistics struct {
	PacketsScanned   uint64
	AnomaliesDetected uint64
	ThreatsBlocked    uint64
	StartTime         time.Time
}
