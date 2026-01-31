// Package metrics provides Prometheus metrics for PRICILLA missile guidance system.
package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all PRICILLA Prometheus metrics.
type Metrics struct {
	// Mission metrics
	MissionsTotal     *prometheus.CounterVec
	MissionsActive    prometheus.Gauge
	MissionsCompleted *prometheus.CounterVec
	MissionsFailed    *prometheus.CounterVec
	MissionDuration   *prometheus.HistogramVec

	// Trajectory metrics
	TrajectoriesPlanned    prometheus.Counter
	TrajectoriesOptimized  prometheus.Counter
	TrajectoriesRecomputed *prometheus.CounterVec
	TrajectoryPlanDuration prometheus.Histogram
	TrajectoryDeviations   *prometheus.HistogramVec

	// Stealth metrics
	DetectionEvents       *prometheus.CounterVec
	EvasionManeuvers      *prometheus.CounterVec
	StealthScoreCurrent   *prometheus.GaugeVec
	RadarExposureTime     *prometheus.HistogramVec
	CountermeasuresActive *prometheus.GaugeVec

	// Payload metrics
	PayloadsRegistered   prometheus.Gauge
	PayloadsActive       prometheus.Gauge
	PayloadsByType       *prometheus.GaugeVec
	PayloadDeployments   *prometheus.CounterVec
	PayloadStatusChanges *prometheus.CounterVec

	// Integration metrics (ASGARD services)
	ServiceLatency          *prometheus.HistogramVec
	ServiceRequestsTotal    *prometheus.CounterVec
	ServiceErrors           *prometheus.CounterVec
	ServiceConnectionStatus *prometheus.GaugeVec

	// Prediction metrics
	ConfidenceDistribution *prometheus.HistogramVec
	InterceptCalculations  *prometheus.CounterVec
	PredictionAccuracy     *prometheus.GaugeVec
	TimeToImpact           *prometheus.GaugeVec
	ThreatAssessments      *prometheus.CounterVec

	// Navigation metrics
	PositionUpdates   prometheus.Counter
	NavigationFixes   *prometheus.CounterVec
	GPSSignalStrength *prometheus.GaugeVec
	InertialDrift     *prometheus.GaugeVec

	// AI Guidance metrics
	AIInferenceDuration prometheus.Histogram
	AIDecisionsTotal    *prometheus.CounterVec
	AIModelConfidence   *prometheus.GaugeVec
	AIOverridesManual   *prometheus.CounterVec
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// GetMetrics returns the global PRICILLA metrics instance.
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = initializeMetrics()
	})
	return globalMetrics
}

// initializeMetrics creates all PRICILLA Prometheus metrics.
func initializeMetrics() *Metrics {
	m := &Metrics{}

	// Mission metrics
	m.MissionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "missions_total",
			Help:      "Total number of missions initiated",
		},
		[]string{"type", "priority"},
	)

	m.MissionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "missions_active",
			Help:      "Number of currently active missions",
		},
	)

	m.MissionsCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "missions_completed_total",
			Help:      "Total number of missions completed successfully",
		},
		[]string{"type", "outcome"},
	)

	m.MissionsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "missions_failed_total",
			Help:      "Total number of missions that failed",
		},
		[]string{"type", "reason"},
	)

	m.MissionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "mission_duration_seconds",
			Help:      "Mission duration from launch to completion",
			Buckets:   []float64{10, 30, 60, 120, 300, 600, 1200, 1800, 3600},
		},
		[]string{"type"},
	)

	// Trajectory metrics
	m.TrajectoriesPlanned = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "trajectories_planned_total",
			Help:      "Total number of trajectories planned",
		},
	)

	m.TrajectoriesOptimized = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "trajectories_optimized_total",
			Help:      "Total number of trajectories optimized",
		},
	)

	m.TrajectoriesRecomputed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "trajectories_recomputed_total",
			Help:      "Total number of trajectory recomputations",
		},
		[]string{"reason", "phase"},
	)

	m.TrajectoryPlanDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "trajectory_plan_duration_seconds",
			Help:      "Time to compute a trajectory plan",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2},
		},
	)

	m.TrajectoryDeviations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "trajectory_deviation_meters",
			Help:      "Deviation from planned trajectory in meters",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 25, 50, 100, 250},
		},
		[]string{"mission_id", "axis"},
	)

	// Stealth metrics
	m.DetectionEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "detection_events_total",
			Help:      "Total radar/sensor detection events",
		},
		[]string{"source_type", "threat_level"},
	)

	m.EvasionManeuvers = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "evasion_maneuvers_total",
			Help:      "Total evasion maneuvers executed",
		},
		[]string{"maneuver_type", "success"},
	)

	m.StealthScoreCurrent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "stealth_score_current",
			Help:      "Current stealth effectiveness score (0-1)",
		},
		[]string{"mission_id"},
	)

	m.RadarExposureTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "radar_exposure_seconds",
			Help:      "Duration of radar exposure events",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"radar_type"},
	)

	m.CountermeasuresActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "countermeasures_active",
			Help:      "Number of active countermeasures by type",
		},
		[]string{"countermeasure_type"},
	)

	// Payload metrics
	m.PayloadsRegistered = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "payloads_registered",
			Help:      "Total number of registered payloads",
		},
	)

	m.PayloadsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "payloads_active",
			Help:      "Number of currently active payloads",
		},
	)

	m.PayloadsByType = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "payloads_by_type",
			Help:      "Number of payloads by type",
		},
		[]string{"payload_type", "status"},
	)

	m.PayloadDeployments = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "payload_deployments_total",
			Help:      "Total payload deployments",
		},
		[]string{"payload_type", "success"},
	)

	m.PayloadStatusChanges = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "payload_status_changes_total",
			Help:      "Total payload status transitions",
		},
		[]string{"payload_type", "from_status", "to_status"},
	)

	// Integration metrics (ASGARD services)
	m.ServiceLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "service_latency_seconds",
			Help:      "Latency of ASGARD service calls",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"service", "operation"},
	)

	m.ServiceRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "service_requests_total",
			Help:      "Total requests to ASGARD services",
		},
		[]string{"service", "operation", "status"},
	)

	m.ServiceErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "service_errors_total",
			Help:      "Total errors from ASGARD service calls",
		},
		[]string{"service", "operation", "error_type"},
	)

	m.ServiceConnectionStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "service_connection_status",
			Help:      "Connection status to ASGARD services (1=connected, 0=disconnected)",
		},
		[]string{"service"},
	)

	// Prediction metrics
	m.ConfidenceDistribution = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "prediction_confidence",
			Help:      "Distribution of prediction confidence scores",
			Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99},
		},
		[]string{"prediction_type"},
	)

	m.InterceptCalculations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "intercept_calculations_total",
			Help:      "Total intercept point calculations performed",
		},
		[]string{"target_type", "result"},
	)

	m.PredictionAccuracy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "prediction_accuracy",
			Help:      "Rolling accuracy of predictions (0-1)",
		},
		[]string{"prediction_type", "time_horizon"},
	)

	m.TimeToImpact = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "time_to_impact_seconds",
			Help:      "Estimated time to impact/intercept",
		},
		[]string{"mission_id"},
	)

	m.ThreatAssessments = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "threat_assessments_total",
			Help:      "Total threat assessments performed",
		},
		[]string{"threat_level", "response_action"},
	)

	// Navigation metrics
	m.PositionUpdates = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "position_updates_total",
			Help:      "Total position updates received",
		},
	)

	m.NavigationFixes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "navigation_fixes_total",
			Help:      "Total navigation fixes by source",
		},
		[]string{"source", "quality"},
	)

	m.GPSSignalStrength = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "gps_signal_strength_dbm",
			Help:      "GPS signal strength in dBm",
		},
		[]string{"satellite_prn"},
	)

	m.InertialDrift = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "inertial_drift_meters",
			Help:      "Accumulated inertial navigation drift",
		},
		[]string{"mission_id", "axis"},
	)

	// AI Guidance metrics
	m.AIInferenceDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "ai_inference_duration_seconds",
			Help:      "AI guidance inference duration",
			Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25},
		},
	)

	m.AIDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "ai_decisions_total",
			Help:      "Total AI guidance decisions made",
		},
		[]string{"decision_type", "confidence_level"},
	)

	m.AIModelConfidence = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "ai_model_confidence",
			Help:      "Current AI model confidence level",
		},
		[]string{"model_name"},
	)

	m.AIOverridesManual = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "asgard",
			Subsystem: "pricilla",
			Name:      "ai_overrides_manual_total",
			Help:      "Total manual overrides of AI decisions",
		},
		[]string{"override_reason"},
	)

	return m
}

// ========================================
// Mission Helper Functions
// ========================================

// RecordMissionStarted records a new mission being initiated.
func RecordMissionStarted(missionType, priority string) {
	m := GetMetrics()
	m.MissionsTotal.WithLabelValues(missionType, priority).Inc()
	m.MissionsActive.Inc()
}

// RecordMissionCompleted records a mission completion.
func RecordMissionCompleted(missionType, outcome string, duration time.Duration) {
	m := GetMetrics()
	m.MissionsCompleted.WithLabelValues(missionType, outcome).Inc()
	m.MissionsActive.Dec()
	m.MissionDuration.WithLabelValues(missionType).Observe(duration.Seconds())
}

// RecordMissionFailed records a mission failure.
func RecordMissionFailed(missionType, reason string) {
	m := GetMetrics()
	m.MissionsFailed.WithLabelValues(missionType, reason).Inc()
	m.MissionsActive.Dec()
}

// UpdateActiveMissions sets the active mission count.
func UpdateActiveMissions(count int) {
	GetMetrics().MissionsActive.Set(float64(count))
}

// ========================================
// Trajectory Helper Functions
// ========================================

// RecordTrajectoryPlanned records a new trajectory being planned.
func RecordTrajectoryPlanned(planDuration time.Duration) {
	m := GetMetrics()
	m.TrajectoriesPlanned.Inc()
	m.TrajectoryPlanDuration.Observe(planDuration.Seconds())
}

// RecordTrajectoryOptimized records a trajectory optimization.
func RecordTrajectoryOptimized() {
	GetMetrics().TrajectoriesOptimized.Inc()
}

// RecordTrajectoryRecomputed records a trajectory recomputation.
func RecordTrajectoryRecomputed(reason, phase string) {
	GetMetrics().TrajectoriesRecomputed.WithLabelValues(reason, phase).Inc()
}

// RecordTrajectoryDeviation records deviation from planned trajectory.
func RecordTrajectoryDeviation(missionID, axis string, deviationMeters float64) {
	GetMetrics().TrajectoryDeviations.WithLabelValues(missionID, axis).Observe(deviationMeters)
}

// ========================================
// Stealth Helper Functions
// ========================================

// RecordDetectionEvent records a radar/sensor detection event.
func RecordDetectionEvent(sourceType, threatLevel string) {
	GetMetrics().DetectionEvents.WithLabelValues(sourceType, threatLevel).Inc()
}

// RecordEvasionManeuver records an evasion maneuver execution.
func RecordEvasionManeuver(maneuverType string, success bool) {
	successStr := "true"
	if !success {
		successStr = "false"
	}
	GetMetrics().EvasionManeuvers.WithLabelValues(maneuverType, successStr).Inc()
}

// UpdateStealthScore updates the current stealth effectiveness score.
func UpdateStealthScore(missionID string, score float64) {
	GetMetrics().StealthScoreCurrent.WithLabelValues(missionID).Set(score)
}

// RecordRadarExposure records radar exposure duration.
func RecordRadarExposure(radarType string, duration time.Duration) {
	GetMetrics().RadarExposureTime.WithLabelValues(radarType).Observe(duration.Seconds())
}

// UpdateCountermeasures updates active countermeasure count.
func UpdateCountermeasures(countermeasureType string, count int) {
	GetMetrics().CountermeasuresActive.WithLabelValues(countermeasureType).Set(float64(count))
}

// ========================================
// Payload Helper Functions
// ========================================

// UpdatePayloadCounts updates payload registration and active counts.
func UpdatePayloadCounts(registered, active int) {
	m := GetMetrics()
	m.PayloadsRegistered.Set(float64(registered))
	m.PayloadsActive.Set(float64(active))
}

// UpdatePayloadsByType updates payload count for a specific type.
func UpdatePayloadsByType(payloadType, status string, count int) {
	GetMetrics().PayloadsByType.WithLabelValues(payloadType, status).Set(float64(count))
}

// RecordPayloadDeployment records a payload deployment.
func RecordPayloadDeployment(payloadType string, success bool) {
	successStr := "true"
	if !success {
		successStr = "false"
	}
	GetMetrics().PayloadDeployments.WithLabelValues(payloadType, successStr).Inc()
}

// RecordPayloadStatusChange records a payload status transition.
func RecordPayloadStatusChange(payloadType, fromStatus, toStatus string) {
	GetMetrics().PayloadStatusChanges.WithLabelValues(payloadType, fromStatus, toStatus).Inc()
}

// ========================================
// Integration Helper Functions
// ========================================

// RecordServiceCall records an ASGARD service call with latency.
func RecordServiceCall(service, operation string, duration time.Duration, err error) {
	m := GetMetrics()
	status := "success"
	if err != nil {
		status = "error"
	}
	m.ServiceLatency.WithLabelValues(service, operation).Observe(duration.Seconds())
	m.ServiceRequestsTotal.WithLabelValues(service, operation, status).Inc()
}

// RecordServiceError records an ASGARD service error.
func RecordServiceError(service, operation, errorType string) {
	GetMetrics().ServiceErrors.WithLabelValues(service, operation, errorType).Inc()
}

// UpdateServiceConnectionStatus updates service connection status.
func UpdateServiceConnectionStatus(service string, connected bool) {
	status := float64(0)
	if connected {
		status = 1
	}
	GetMetrics().ServiceConnectionStatus.WithLabelValues(service).Set(status)
}

// ========================================
// Prediction Helper Functions
// ========================================

// RecordPredictionConfidence records a prediction confidence score.
func RecordPredictionConfidence(predictionType string, confidence float64) {
	GetMetrics().ConfidenceDistribution.WithLabelValues(predictionType).Observe(confidence)
}

// RecordInterceptCalculation records an intercept calculation.
func RecordInterceptCalculation(targetType, result string) {
	GetMetrics().InterceptCalculations.WithLabelValues(targetType, result).Inc()
}

// UpdatePredictionAccuracy updates the rolling prediction accuracy.
func UpdatePredictionAccuracy(predictionType, timeHorizon string, accuracy float64) {
	GetMetrics().PredictionAccuracy.WithLabelValues(predictionType, timeHorizon).Set(accuracy)
}

// UpdateTimeToImpact updates the estimated time to impact.
func UpdateTimeToImpact(missionID string, seconds float64) {
	GetMetrics().TimeToImpact.WithLabelValues(missionID).Set(seconds)
}

// RecordThreatAssessment records a threat assessment.
func RecordThreatAssessment(threatLevel, responseAction string) {
	GetMetrics().ThreatAssessments.WithLabelValues(threatLevel, responseAction).Inc()
}

// ========================================
// Navigation Helper Functions
// ========================================

// RecordPositionUpdate records a position update.
func RecordPositionUpdate() {
	GetMetrics().PositionUpdates.Inc()
}

// RecordNavigationFix records a navigation fix.
func RecordNavigationFix(source, quality string) {
	GetMetrics().NavigationFixes.WithLabelValues(source, quality).Inc()
}

// UpdateGPSSignalStrength updates GPS signal strength.
func UpdateGPSSignalStrength(satellitePRN string, strengthDBm float64) {
	GetMetrics().GPSSignalStrength.WithLabelValues(satellitePRN).Set(strengthDBm)
}

// UpdateInertialDrift updates the inertial navigation drift.
func UpdateInertialDrift(missionID, axis string, driftMeters float64) {
	GetMetrics().InertialDrift.WithLabelValues(missionID, axis).Set(driftMeters)
}

// ========================================
// AI Guidance Helper Functions
// ========================================

// RecordAIInference records an AI inference duration.
func RecordAIInference(duration time.Duration) {
	GetMetrics().AIInferenceDuration.Observe(duration.Seconds())
}

// RecordAIDecision records an AI guidance decision.
func RecordAIDecision(decisionType, confidenceLevel string) {
	GetMetrics().AIDecisionsTotal.WithLabelValues(decisionType, confidenceLevel).Inc()
}

// UpdateAIModelConfidence updates the AI model confidence.
func UpdateAIModelConfidence(modelName string, confidence float64) {
	GetMetrics().AIModelConfidence.WithLabelValues(modelName).Set(confidence)
}

// RecordAIManualOverride records a manual override of an AI decision.
func RecordAIManualOverride(reason string) {
	GetMetrics().AIOverridesManual.WithLabelValues(reason).Inc()
}
