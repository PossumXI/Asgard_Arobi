// Package observability provides metrics, tracing, and logging infrastructure.
package observability

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all ASGARD Prometheus metrics.
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// WebSocket metrics
	WebSocketConnections prometheus.Gauge
	WebSocketMessages    *prometheus.CounterVec

	// NATS metrics
	NATSMessagesReceived *prometheus.CounterVec
	NATSMessagesPublished *prometheus.CounterVec
	NATSConnectionStatus prometheus.Gauge

	// Event bus metrics
	EventsProcessed *prometheus.CounterVec
	EventsQueued    prometheus.Gauge
	EventLatency    *prometheus.HistogramVec

	// Database metrics
	DBQueryDuration *prometheus.HistogramVec
	DBConnectionPool *prometheus.GaugeVec
	DBErrors        *prometheus.CounterVec

	// Satellite metrics (Silenus)
	SatelliteFramesProcessed prometheus.Counter
	SatelliteAlertsGenerated *prometheus.CounterVec
	SatelliteBatteryLevel    *prometheus.GaugeVec
	SatelliteLatency         prometheus.Histogram

	// Hunoid metrics
	HunoidActionsExecuted *prometheus.CounterVec
	HunoidEthicsRejected  *prometheus.CounterVec
	HunoidJointPositions  *prometheus.GaugeVec

	// Security metrics (Giru)
	SecurityThreatsDetected *prometheus.CounterVec
	SecurityPacketsScanned  prometheus.Counter
	SecurityMitigations     *prometheus.CounterVec

	// DTN metrics (Sat_Net)
	DTNBundlesTransmitted *prometheus.CounterVec
	DTNBundlesReceived    *prometheus.CounterVec
	DTNQueueDepth         *prometheus.GaugeVec
	DTNRoutingDecisions   *prometheus.CounterVec
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = initializeMetrics()
	})
	return globalMetrics
}

// initializeMetrics creates all Prometheus metrics.
func initializeMetrics() *Metrics {
	m := &Metrics{}

	// HTTP metrics
	m.HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	m.HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	m.HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 2, 12),
		},
		[]string{"endpoint"},
	)

	// WebSocket metrics
	m.WebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "websocket",
			Name:      "connections_active",
			Help:      "Number of active WebSocket connections",
		},
	)

	m.WebSocketMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "websocket",
			Name:      "messages_total",
			Help:      "Total WebSocket messages",
		},
		[]string{"direction", "type"},
	)

	// NATS metrics
	m.NATSMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "nats",
			Name:      "messages_received_total",
			Help:      "Total NATS messages received",
		},
		[]string{"subject"},
	)

	m.NATSMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "nats",
			Name:      "messages_published_total",
			Help:      "Total NATS messages published",
		},
		[]string{"subject"},
	)

	m.NATSConnectionStatus = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "nats",
			Name:      "connection_status",
			Help:      "NATS connection status (1 = connected, 0 = disconnected)",
		},
	)

	// Event bus metrics
	m.EventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "events",
			Name:      "processed_total",
			Help:      "Total events processed",
		},
		[]string{"type", "source"},
	)

	m.EventsQueued = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "events",
			Name:      "queued",
			Help:      "Number of events currently queued",
		},
	)

	m.EventLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "events",
			Name:      "latency_seconds",
			Help:      "Event processing latency in seconds",
			Buckets:   []float64{.0001, .0005, .001, .005, .01, .05, .1, .5, 1},
		},
		[]string{"type"},
	)

	// Database metrics
	m.DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "database",
			Name:      "query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"database", "operation"},
	)

	m.DBConnectionPool = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "database",
			Name:      "connections",
			Help:      "Number of database connections",
		},
		[]string{"database", "state"},
	)

	m.DBErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "database",
			Name:      "errors_total",
			Help:      "Total database errors",
		},
		[]string{"database", "operation"},
	)

	// Satellite metrics (Silenus)
	m.SatelliteFramesProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "silenus",
			Name:      "frames_processed_total",
			Help:      "Total video frames processed by vision system",
		},
	)

	m.SatelliteAlertsGenerated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "silenus",
			Name:      "alerts_generated_total",
			Help:      "Total alerts generated by satellite vision",
		},
		[]string{"satellite_id", "alert_type"},
	)

	m.SatelliteBatteryLevel = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "silenus",
			Name:      "battery_level_percent",
			Help:      "Satellite battery level percentage",
		},
		[]string{"satellite_id"},
	)

	m.SatelliteLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "silenus",
			Name:      "processing_latency_seconds",
			Help:      "Vision processing latency in seconds",
			Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
	)

	// Hunoid metrics
	m.HunoidActionsExecuted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "hunoid",
			Name:      "actions_executed_total",
			Help:      "Total actions executed by hunoids",
		},
		[]string{"hunoid_id", "action_type"},
	)

	m.HunoidEthicsRejected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "hunoid",
			Name:      "ethics_rejections_total",
			Help:      "Total actions rejected by ethical kernel",
		},
		[]string{"hunoid_id", "rule"},
	)

	m.HunoidJointPositions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "hunoid",
			Name:      "joint_position_degrees",
			Help:      "Current joint positions in degrees",
		},
		[]string{"hunoid_id", "joint"},
	)

	// Security metrics (Giru)
	m.SecurityThreatsDetected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "giru",
			Name:      "threats_detected_total",
			Help:      "Total security threats detected",
		},
		[]string{"type", "severity"},
	)

	m.SecurityPacketsScanned = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "giru",
			Name:      "packets_scanned_total",
			Help:      "Total network packets scanned",
		},
	)

	m.SecurityMitigations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "giru",
			Name:      "mitigations_total",
			Help:      "Total mitigation actions taken",
		},
		[]string{"action", "success"},
	)

	// DTN metrics (Sat_Net)
	m.DTNBundlesTransmitted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "satnet",
			Name:      "bundles_transmitted_total",
			Help:      "Total DTN bundles transmitted",
		},
		[]string{"priority", "destination"},
	)

	m.DTNBundlesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "satnet",
			Name:      "bundles_received_total",
			Help:      "Total DTN bundles received",
		},
		[]string{"priority", "source"},
	)

	m.DTNQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "satnet",
			Name:      "queue_depth",
			Help:      "Current DTN bundle queue depth",
		},
		[]string{"node_id", "priority"},
	)

	m.DTNRoutingDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "satnet",
			Name:      "routing_decisions_total",
			Help:      "Total routing decisions made",
		},
		[]string{"router_type", "result"},
	)

	return m
}

// Handler returns the Prometheus HTTP handler for /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// HTTPMiddleware wraps an HTTP handler with metrics collection.
func HTTPMiddleware(next http.Handler) http.Handler {
	m := GetMetrics()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status and size
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		endpoint := normalizeEndpoint(r.URL.Path)

		m.HTTPRequestsTotal.WithLabelValues(r.Method, endpoint, statusToStr(wrapped.status)).Inc()
		m.HTTPRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
		m.HTTPResponseSize.WithLabelValues(endpoint).Observe(float64(wrapped.size))
	})
}

// responseWriter wraps http.ResponseWriter to capture status and size.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijacker not supported")
	}
	return hijacker.Hijack()
}

func (w *responseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// normalizeEndpoint normalizes URL paths to prevent cardinality explosion.
func normalizeEndpoint(path string) string {
	// Group common patterns
	switch {
	case len(path) > 13 && path[:13] == "/api/streams/":
		return "/api/streams/:id"
	case len(path) > 12 && path[:12] == "/api/alerts/":
		return "/api/alerts/:id"
	case len(path) > 12 && path[:12] == "/api/hunoids/":
		return "/api/hunoids/:id"
	case len(path) > 16 && path[:16] == "/api/satellites/":
		return "/api/satellites/:id"
	case len(path) > 14 && path[:14] == "/api/missions/":
		return "/api/missions/:id"
	default:
		return path
	}
}

func statusToStr(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "other"
	}
}

// RecordEventProcessed records an event being processed.
func RecordEventProcessed(eventType, source string) {
	GetMetrics().EventsProcessed.WithLabelValues(eventType, source).Inc()
}

// RecordEventLatency records event processing latency.
func RecordEventLatency(eventType string, duration time.Duration) {
	GetMetrics().EventLatency.WithLabelValues(eventType).Observe(duration.Seconds())
}

// RecordDBQuery records a database query duration.
func RecordDBQuery(database, operation string, duration time.Duration) {
	GetMetrics().DBQueryDuration.WithLabelValues(database, operation).Observe(duration.Seconds())
}

// RecordDBError records a database error.
func RecordDBError(database, operation string) {
	GetMetrics().DBErrors.WithLabelValues(database, operation).Inc()
}

// RecordThreatDetected records a security threat detection.
func RecordThreatDetected(threatType, severity string) {
	GetMetrics().SecurityThreatsDetected.WithLabelValues(threatType, severity).Inc()
}

// RecordMitigation records a mitigation action.
func RecordMitigation(action string, success bool) {
	successStr := "true"
	if !success {
		successStr = "false"
	}
	GetMetrics().SecurityMitigations.WithLabelValues(action, successStr).Inc()
}

// RecordDTNBundle records a DTN bundle transmission or reception.
func RecordDTNBundle(direction, priority, endpoint string) {
	m := GetMetrics()
	if direction == "tx" {
		m.DTNBundlesTransmitted.WithLabelValues(priority, endpoint).Inc()
	} else {
		m.DTNBundlesReceived.WithLabelValues(priority, endpoint).Inc()
	}
}

// UpdateWebSocketConnections updates the active WebSocket connection gauge.
func UpdateWebSocketConnections(count int) {
	GetMetrics().WebSocketConnections.Set(float64(count))
}

// UpdateNATSConnectionStatus updates the NATS connection status.
func UpdateNATSConnectionStatus(connected bool) {
	if connected {
		GetMetrics().NATSConnectionStatus.Set(1)
	} else {
		GetMetrics().NATSConnectionStatus.Set(0)
	}
}
