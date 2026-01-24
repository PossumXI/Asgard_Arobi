/**
 * ASGARD API Client
 * Production-grade API client with error handling, retries, and type safety.
 */

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

const buildQueryString = (
  params?: Record<string, string | number | boolean | null | undefined>
): string => {
  if (!params) return '';
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return;
    search.append(key, String(value));
  });
  const query = search.toString();
  return query ? `?${query}` : '';
};

export interface ApiError {
  message: string;
  code: string;
  status: number;
}

export interface ApiResponse<T> {
  data: T;
  success: boolean;
  message?: string;
}

class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  setToken(token: string | null): void {
    this.token = token;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(this.token && { Authorization: `Bearer ${this.token}` }),
      ...options.headers,
    };

    const config: RequestInit = {
      ...options,
      headers,
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        const error = await response.json().catch(() => ({
          message: 'An unexpected error occurred',
          code: 'UNKNOWN_ERROR',
        }));
        
        throw {
          message: error.message || 'Request failed',
          code: error.code || 'REQUEST_FAILED',
          status: response.status,
        } as ApiError;
      }

      return response.json();
    } catch (error) {
      if ((error as ApiError).status) {
        throw error;
      }
      
      throw {
        message: 'Network error. Please check your connection.',
        code: 'NETWORK_ERROR',
        status: 0,
      } as ApiError;
    }
  }

  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  async post<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }
}

export const api = new ApiClient(API_BASE_URL);

// Auth API
export interface WebAuthnCredentialPayload {
  id: string;
  rawId: string;
  type: string;
  response: Record<string, unknown>;
}

export const authApi = {
  signIn: (email: string, password: string) =>
    api.post<{ user: User; token: string }>('/auth/signin', { email, password }),
  
  signUp: (data: SignUpData) =>
    api.post<{ user: User; token: string }>('/auth/signup', data),
  
  signOut: () => api.post<void>('/auth/signout'),
  
  refreshToken: () => api.post<{ token: string }>('/auth/refresh'),
  
  requestPasswordReset: (email: string) =>
    api.post<void>('/auth/password-reset/request', { email }),
  
  resetPassword: (token: string, password: string) =>
    api.post<void>('/auth/password-reset/confirm', { token, password }),
  
  verifyEmail: (token: string) =>
    api.post<void>('/auth/verify-email', { token }),
  
  // FIDO2/WebAuthn for Government Portal
  startFido2Registration: () =>
    api.post<PublicKeyCredentialCreationOptions>('/auth/fido2/register/start'),
  
  completeFido2Registration: (credential: WebAuthnCredentialPayload) =>
    api.post<void>('/auth/fido2/register/complete', credential),
  
  startFido2Auth: (email: string) =>
    api.post<PublicKeyCredentialRequestOptions>('/auth/fido2/auth/start', { email }),
  
  completeFido2Auth: (email: string, credential: WebAuthnCredentialPayload) =>
    api.post<{ user: User; token: string }>('/auth/fido2/auth/complete', { email, credential }),
};

// User API
export const userApi = {
  getProfile: () => api.get<User>('/user/profile'),
  
  updateProfile: (data: Partial<User>) =>
    api.patch<User>('/user/profile', data),
  
  getSubscription: () => api.get<Subscription>('/user/subscription'),
  
  updateNotificationSettings: (settings: NotificationSettings) =>
    api.patch<NotificationSettings>('/user/notifications', settings),
};

// Subscription API
export const subscriptionApi = {
  getPlans: () => api.get<SubscriptionPlan[]>('/subscriptions/plans'),
  
  createCheckoutSession: (planId: string) =>
    api.post<{ sessionUrl: string }>('/subscriptions/checkout', { planId }),
  
  createPortalSession: () =>
    api.post<{ portalUrl: string }>('/subscriptions/portal'),
  
  cancelSubscription: () =>
    api.post<void>('/subscriptions/cancel'),
  
  reactivateSubscription: () =>
    api.post<void>('/subscriptions/reactivate'),
};

// Types
export interface User {
  id: string;
  email: string;
  fullName: string | null;
  avatarUrl?: string;
  subscriptionTier: 'free' | 'observer' | 'supporter' | 'commander';
  isGovernment: boolean;
  emailVerified?: boolean;
  createdAt: string;
  updatedAt: string;
  lastLogin: string | null;
}

export interface SignUpData {
  email: string;
  password: string;
  fullName: string;
  organizationType?: 'individual' | 'organization' | 'government';
}

export interface Subscription {
  id: string;
  userId: string;
  tier: 'free' | 'observer' | 'supporter' | 'commander';
  status: 'active' | 'canceled' | 'past_due' | 'trialing';
  currentPeriodStart: string;
  currentPeriodEnd: string;
  cancelAtPeriodEnd: boolean;
}

export interface SubscriptionPlan {
  id: string;
  name: string;
  tier: 'observer' | 'supporter' | 'commander';
  price: number;
  interval: 'month' | 'year';
  features: string[];
  highlighted?: boolean;
}

export interface AdminUser {
  id: string;
  email: string;
  fullName: string;
  subscriptionTier: 'observer' | 'supporter' | 'commander';
  isGovernment: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ControlCommandPayload {
  targetDomain: string;
  targetSystem?: string;
  commandType: string;
  parameters?: Record<string, unknown>;
  priority?: number;
}

export interface ControlPlaneStatusResponse {
  health: Record<string, unknown> | null;
  systems: Record<string, unknown>;
  metrics: Record<string, unknown> | null;
  coordinator?: Record<string, unknown> | null;
  timestamp: string;
}

export interface ControlPlaneMetricsResponse {
  controlplane: Record<string, unknown> | null;
  coordinator?: Record<string, unknown> | null;
  timestamp: string;
}

export interface ControlPlaneEvent {
  id: string;
  domain?: string;
  type?: string;
  severity?: string;
  source?: string;
  payload?: Record<string, unknown>;
  timestamp?: string;
}

export interface ControlPlaneEventsResponse {
  events: ControlPlaneEvent[];
  total_count: number;
  limit: number;
  offset: number;
}

export interface ControlPlaneCommandResponse {
  command_id: string;
  success: boolean;
  error?: string;
  result?: Record<string, unknown>;
  executed_at: string;
  duration_ms: number;
}

export interface ControlPlanePolicyAction {
  target_domain: string;
  command_type: string;
  parameters?: Record<string, unknown>;
  async: boolean;
}

export interface ControlPlanePolicy {
  id: string;
  name: string;
  description: string;
  priority: number;
  enabled: boolean;
  trigger_type?: string;
  actions: ControlPlanePolicyAction[];
  cooldown_ms: number;
  last_triggered?: string | null;
}

export interface AuditLog {
  ID: number;
  Component: string;
  Action: string;
  UserID?: string | null;
  Metadata?: Record<string, unknown>;
  CreatedAt: string;
}

export interface AuditLogFilters {
  [key: string]: string | number | boolean | null | undefined;
  component?: string;
  action?: string;
  user_id?: string;
  since?: string;
  until?: string;
  limit?: number;
  offset?: number;
}

export interface AuditLogsResponse {
  logs: AuditLog[];
  count: number;
  limit: number;
  offset: number;
}

export interface AuditStats {
  by_component: Record<string, number>;
  total: number;
  since: string;
}

export interface EthicalDecision {
  ID: string;
  HunoidID: string;
  ProposedAction: string;
  EthicalAssessment: Record<string, unknown>;
  Decision: string;
  Reasoning?: string | null;
  HumanOverride: boolean;
  CreatedAt: string;
}

export interface EthicsStats {
  by_decision: Record<string, number>;
  total: number;
  approval_rate: number;
}

export interface EthicalDecisionFilters {
  [key: string]: string | number | boolean | null | undefined;
  hunoid_id?: string;
  mission_id?: string;
  decision_type?: string;
  limit?: number;
  offset?: number;
}

export const adminApi = {
  listUsers: () => api.get<AdminUser[]>('/admin/users'),
  updateUser: (userId: string, data: Partial<Pick<AdminUser, 'fullName' | 'subscriptionTier' | 'isGovernment'>>) =>
    api.patch<AdminUser>(`/admin/users/${userId}`, data),
};

export const controlPlaneApi = {
  getStatus: () => api.get<ControlPlaneStatusResponse>('/controlplane/status'),
  getHealth: () => api.get<Record<string, unknown>>('/controlplane/health'),
  getMetrics: () => api.get<ControlPlaneMetricsResponse>('/controlplane/metrics'),
  getEvents: (params?: {
    limit?: number;
    offset?: number;
    domain?: string;
    severity?: string;
    type?: string;
  }) => api.get<ControlPlaneEventsResponse>(`/controlplane/events${buildQueryString(params)}`),
  getEvent: (id: string) => api.get<ControlPlaneEvent>(`/controlplane/events/${id}`),
  sendCommand: (payload: ControlCommandPayload) => api.post<ControlPlaneCommandResponse>('/controlplane/command', payload),
  getSystems: () => api.get<Record<string, unknown>>('/controlplane/systems'),
  getSystem: (id: string) => api.get<Record<string, unknown>>(`/controlplane/systems/${id}`),
  getPolicies: () => api.get<ControlPlanePolicy[]>('/controlplane/policies'),
  patchPolicy: (id: string, data: { enabled?: boolean }) =>
    api.patch<Record<string, unknown>>(`/controlplane/policies/${id}`, data),
  getResponses: () => api.get<Record<string, unknown>[]>('/controlplane/responses'),
};

export const auditApi = {
  getLogs: (filters?: AuditLogFilters) =>
    api.get<AuditLogsResponse>(`/audit/logs${buildQueryString(filters)}`),
  getLog: (id: number) => api.get<AuditLog>(`/audit/logs/${id}`),
  getLogsByComponent: (component: string, since?: string) =>
    api.get<{ logs: AuditLog[]; count: number; component: string; since: string }>(
      `/audit/logs/component/${component}${buildQueryString({ since })}`
    ),
  getLogsByUser: (userId: string, limit?: number) =>
    api.get<{ logs: AuditLog[]; count: number; user_id: string }>(
      `/audit/logs/user/${userId}${buildQueryString({ limit })}`
    ),
  getStats: (since?: string) => api.get<AuditStats>(`/audit/stats${buildQueryString({ since })}`),
};

export const ethicsApi = {
  getDecisions: (filters?: EthicalDecisionFilters) =>
    api.get<{ decisions: EthicalDecision[]; limit: number }>(
      `/ethics/decisions${buildQueryString(filters)}`
    ),
  getDecision: (id: string) => api.get<EthicalDecision>(`/ethics/decisions/${id}`),
  getDecisionsByHunoid: (hunoidId: string, limit?: number) =>
    api.get<{ decisions: EthicalDecision[]; count: number; hunoid_id: string }>(
      `/ethics/decisions/hunoid/${hunoidId}${buildQueryString({ limit })}`
    ),
  getDecisionsByMission: (missionId: string) =>
    api.get<{ decisions: EthicalDecision[]; count: number; mission_id: string }>(
      `/ethics/decisions/mission/${missionId}`
    ),
  getStats: () => api.get<EthicsStats>('/ethics/stats'),
};

export interface NotificationSettings {
  emailAlerts: boolean;
  pushNotifications: boolean;
  weeklyDigest: boolean;
  securityAlerts: boolean;
}
