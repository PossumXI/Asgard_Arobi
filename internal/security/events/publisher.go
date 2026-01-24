// Package events provides NATS publishing for security events.
package events

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher publishes security events to NATS.
type Publisher struct {
	nc     *nats.Conn
	mu     sync.RWMutex
	stats  PublisherStats
}

// PublisherStats tracks publishing statistics.
type PublisherStats struct {
	AlertsPublished    int64
	FindingsPublished  int64
	ResponsesPublished int64
	Errors             int64
	LastPublished      time.Time
}

// PublisherConfig holds publisher configuration.
type PublisherConfig struct {
	NATSURL        string
	ReconnectWait  time.Duration
	MaxReconnects  int
}

// DefaultPublisherConfig returns default configuration.
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		NATSURL:       "nats://localhost:4222",
		ReconnectWait: 2 * time.Second,
		MaxReconnects: 60,
	}
}

// NewPublisher creates a new security event publisher.
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	opts := []nats.Option{
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[Giru Publisher] Reconnected to NATS: %s", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("[Giru Publisher] Disconnected from NATS: %v", err)
			}
		}),
	}

	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		return nil, err
	}

	return &Publisher{
		nc: nc,
	}, nil
}

// PublishAlert publishes a security alert.
func (p *Publisher) PublishAlert(alert AlertEvent) error {
	subject := GetSubjectForEvent(EventTypeAlert, alert.Severity)
	data, err := json.Marshal(alert.ToMap())
	if err != nil {
		p.recordError()
		return err
	}

	if err := p.nc.Publish(subject, data); err != nil {
		p.recordError()
		return err
	}

	p.mu.Lock()
	p.stats.AlertsPublished++
	p.stats.LastPublished = time.Now()
	p.mu.Unlock()

	log.Printf("[Giru Publisher] Published alert to %s: %s (%s)", subject, alert.ThreatType, alert.Severity)
	return nil
}

// PublishFinding publishes a security finding.
func (p *Publisher) PublishFinding(finding FindingEvent) error {
	subject := GetSubjectForEvent(EventTypeFinding, finding.Severity)
	data, err := json.Marshal(finding)
	if err != nil {
		p.recordError()
		return err
	}

	if err := p.nc.Publish(subject, data); err != nil {
		p.recordError()
		return err
	}

	p.mu.Lock()
	p.stats.FindingsPublished++
	p.stats.LastPublished = time.Now()
	p.mu.Unlock()

	log.Printf("[Giru Publisher] Published finding to %s: %s", subject, finding.Category)
	return nil
}

// PublishResponse publishes a response action event.
func (p *Publisher) PublishResponse(response ResponseEvent) error {
	subject := GetSubjectForEvent(EventTypeResponse, response.Severity)
	data, err := json.Marshal(response.ToMap())
	if err != nil {
		p.recordError()
		return err
	}

	if err := p.nc.Publish(subject, data); err != nil {
		p.recordError()
		return err
	}

	p.mu.Lock()
	p.stats.ResponsesPublished++
	p.stats.LastPublished = time.Now()
	p.mu.Unlock()

	log.Printf("[Giru Publisher] Published response: %s (%s)", response.ActionType, 
		map[bool]string{true: "success", false: "failed"}[response.Success])
	return nil
}

// PublishRaw publishes raw data to a subject.
func (p *Publisher) PublishRaw(subject string, data map[string]interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		p.recordError()
		return err
	}

	if err := p.nc.Publish(subject, payload); err != nil {
		p.recordError()
		return err
	}

	p.mu.Lock()
	p.stats.LastPublished = time.Now()
	p.mu.Unlock()

	return nil
}

func (p *Publisher) recordError() {
	p.mu.Lock()
	p.stats.Errors++
	p.mu.Unlock()
}

// Stats returns publishing statistics.
func (p *Publisher) Stats() PublisherStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// IsConnected returns whether the publisher is connected to NATS.
func (p *Publisher) IsConnected() bool {
	return p.nc != nil && p.nc.IsConnected()
}

// Close closes the NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

// ThreatSeverityToEventSeverity converts threat level to event severity.
func ThreatSeverityToEventSeverity(level string) Severity {
	switch level {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	default:
		return SeverityInfo
	}
}
