/**
 * ASGARD Frontend Type Definitions
 * Aligned with backend Go models (internal/platform/db/models.go)
 */

// ============================================================================
// Core Entity Types (matching backend exactly)
// ============================================================================

export interface User {
  id: string;
  email: string;
  fullName: string | null;
  subscriptionTier: SubscriptionTier;
  isGovernment: boolean;
  createdAt: string;
  updatedAt: string;
  lastLogin: string | null;
  avatarUrl?: string;
  emailVerified?: boolean;
}

export type SubscriptionTier = 'free' | 'observer' | 'supporter' | 'commander';

export interface Satellite {
  id: string;
  noradId: number | null;
  name: string;
  orbitalElements: OrbitalElements;
  hardwareConfig: SatelliteHardwareConfig;
  currentBatteryPercent: number | null;
  status: SatelliteStatus;
  lastTelemetry: string | null;
  firmwareVersion: string | null;
  createdAt: string;
  updatedAt: string;
}

export type SatelliteStatus = 'operational' | 'eclipse' | 'maintenance' | 'decommissioned';

export interface OrbitalElements {
  tle1: string;
  tle2: string;
  epoch: string;
  inclination: number;
  eccentricity: number;
  meanMotion: number;
}

export interface SatelliteHardwareConfig {
  cameraResolution: string;
  sensorTypes: string[];
  computeModule: string;
  storageGb: number;
}

export interface Hunoid {
  id: string;
  serialNumber: string;
  currentLocation: GeoLocation | null;
  currentMissionId: string | null;
  hardwareConfig: HunoidHardwareConfig;
  batteryPercent: number | null;
  status: HunoidStatus;
  vlaModelVersion: string | null;
  ethicalScore: number;
  lastTelemetry: string | null;
  createdAt: string;
  updatedAt: string;
}

export type HunoidStatus = 'idle' | 'active' | 'charging' | 'maintenance' | 'emergency';

export interface GeoLocation {
  latitude: number;
  longitude: number;
  altitude?: number;
}

export interface HunoidHardwareConfig {
  actuatorCount: number;
  sensorSuite: string[];
  computeUnit: string;
  maxPayloadKg: number;
}

export interface Mission {
  id: string;
  missionType: MissionType;
  priority: number;
  status: MissionStatus;
  assignedHunoidIds: string[];
  targetLocation: GeoLocation | null;
  description: string | null;
  createdBy: string | null;
  createdAt: string;
  startedAt: string | null;
  completedAt: string | null;
}

export type MissionType = 'search_rescue' | 'aid_delivery' | 'reconnaissance' | 'disaster_response' | 'medical';
export type MissionStatus = 'pending' | 'active' | 'completed' | 'aborted';

export interface Alert {
  id: string;
  satelliteId: string | null;
  alertType: AlertType;
  confidenceScore: number;
  detectionLocation: GeoLocation | null;
  videoSegmentUrl: string | null;
  metadata: AlertMetadata;
  status: AlertStatus;
  createdAt: string;
}

export type AlertType = 'tsunami' | 'fire' | 'troop_movement' | 'missile_launch' | 'maritime_distress' | 'earthquake';
export type AlertStatus = 'new' | 'acknowledged' | 'dispatched' | 'resolved';

export interface AlertMetadata {
  boundingBox?: { x: number; y: number; width: number; height: number };
  detectionClass: string;
  frameNumber?: number;
  additionalData?: Record<string, unknown>;
}

export interface Threat {
  id: string;
  threatType: ThreatType;
  severity: ThreatSeverity;
  sourceIp: string | null;
  targetComponent: string | null;
  attackVector: string | null;
  mitigationAction: string | null;
  status: ThreatStatus;
  detectedAt: string;
  resolvedAt: string | null;
}

export type ThreatType = 'ddos' | 'intrusion' | 'malware' | 'phishing' | 'reconnaissance';
export type ThreatSeverity = 'low' | 'medium' | 'high' | 'critical';
export type ThreatStatus = 'detected' | 'mitigated' | 'resolved';

export interface Subscription {
  id: string;
  userId: string;
  stripeSubscriptionId: string | null;
  stripeCustomerId: string | null;
  tier: SubscriptionTier | null;
  status: SubscriptionStatus;
  currentPeriodStart: string | null;
  currentPeriodEnd: string | null;
  createdAt: string;
  updatedAt: string;
  cancelAtPeriodEnd?: boolean;
}

export type SubscriptionStatus = 'active' | 'cancelled' | 'expired' | 'past_due' | 'trialing';

export interface AuditLog {
  id: number;
  component: string;
  action: string;
  userId: string | null;
  metadata: Record<string, unknown>;
  createdAt: string;
}

export interface EthicalDecision {
  id: string;
  hunoidId: string;
  proposedAction: string;
  ethicalAssessment: EthicalAssessmentData;
  decision: EthicalDecisionResult;
  reasoning: string | null;
  humanOverride: boolean;
  createdAt: string;
}

export type EthicalDecisionResult = 'approved' | 'rejected' | 'escalated';

export interface EthicalAssessmentData {
  rulesChecked: string[];
  score: number;
  violations: string[];
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface AuthResponse {
  user: User;
  token: string;
  refreshToken?: string;
  expiresAt: string;
}

export interface SignUpRequest {
  email: string;
  password: string;
  fullName: string;
  organizationType?: 'individual' | 'organization' | 'government';
}

export interface SignInRequest {
  email: string;
  password: string;
  rememberMe?: boolean;
}

export interface SubscriptionPlan {
  id: string;
  name: string;
  tier: SubscriptionTier;
  price: number;
  interval: 'month' | 'year';
  features: string[];
  highlighted?: boolean;
  stripePriceId: string;
}

export interface CheckoutSessionResponse {
  sessionId: string;
  sessionUrl: string;
}

export interface BillingPortalResponse {
  portalUrl: string;
}

// ============================================================================
// Dashboard & Statistics Types
// ============================================================================

export interface DashboardStats {
  activeSatellites: number;
  activeHunoids: number;
  pendingAlerts: number;
  activeMissions: number;
  threatsToday: number;
  systemHealth: number;
}

export interface SystemHealthMetrics {
  nysus: ServiceHealth;
  satNet: ServiceHealth;
  giru: ServiceHealth;
  data: ServiceHealth;
  controlNet: ServiceHealth;
}

export interface ServiceHealth {
  status: 'healthy' | 'degraded' | 'down';
  latency: number;
  uptime: number;
  lastCheck: string;
}

// ============================================================================
// Real-time Event Types (for WebSocket/NATS)
// ============================================================================

export type RealtimeEventType = 
  | 'alert.new'
  | 'alert.updated'
  | 'mission.started'
  | 'mission.completed'
  | 'hunoid.status'
  | 'satellite.telemetry'
  | 'threat.detected'
  | 'threat.mitigated';

export interface RealtimeEvent<T = unknown> {
  type: RealtimeEventType;
  timestamp: string;
  payload: T;
}

export interface TelemetryData {
  entityId: string;
  entityType: 'satellite' | 'hunoid';
  timestamp: string;
  batteryPercent: number;
  status: string;
  location?: GeoLocation;
  additionalMetrics: Record<string, number>;
}

// ============================================================================
// Hub Streaming Types
// ============================================================================

export interface Stream {
  id: string;
  title: string;
  source: string;
  sourceType: 'satellite' | 'hunoid' | 'ground_station';
  sourceId: string;
  location: string;
  geoLocation?: GeoLocation;
  type: StreamCategory;
  status: StreamStatus;
  viewers: number;
  latency: number;
  thumbnail?: string;
  description?: string;
  resolution: string;
  bitrate: number;
  startedAt: string;
  metadata?: StreamMetadata;
}

export type StreamCategory = 'civilian' | 'military' | 'interstellar';
export type StreamStatus = 'live' | 'delayed' | 'offline' | 'buffering';

export interface StreamMetadata {
  missionId?: string;
  alertId?: string;
  classification?: string;
  encryptionEnabled: boolean;
}

export interface StreamSession {
  streamId: string;
  sessionId: string;
  iceServers: RTCIceServer[];
  signallingUrl: string;
  authToken: string;
  expiresAt: string;
}

// ============================================================================
// Government Portal Types
// ============================================================================

export interface GovAccessRequest {
  firstName: string;
  lastName: string;
  email: string;
  agency: string;
  position: string;
  useCase: string;
  securityClearanceLevel?: string;
}

export interface GovUser extends User {
  clearanceLevel: ClearanceLevel;
  agency: string;
  fido2Registered: boolean;
  lastFido2Auth: string | null;
}

export type ClearanceLevel = 'unclassified' | 'confidential' | 'secret' | 'top_secret';

export interface MissionRequest {
  missionType: MissionType;
  priority: number;
  description: string;
  targetLocation: GeoLocation;
  requestedHunoidCount: number;
  estimatedDuration: string;
  justification: string;
}

// ============================================================================
// Notification Types
// ============================================================================

export interface NotificationSettings {
  emailAlerts: boolean;
  pushNotifications: boolean;
  weeklyDigest: boolean;
  securityAlerts: boolean;
  missionUpdates: boolean;
  systemStatus: boolean;
}

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  read: boolean;
  actionUrl?: string;
  createdAt: string;
}

export type NotificationType = 'alert' | 'mission' | 'system' | 'security' | 'subscription';
