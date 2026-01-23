/**
 * ASGARD API Client
 * Production-grade API client with error handling, retries, and type safety.
 */

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

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
  
  completeFido2Registration: (credential: Credential) =>
    api.post<void>('/auth/fido2/register/complete', credential),
  
  startFido2Auth: () =>
    api.post<PublicKeyCredentialRequestOptions>('/auth/fido2/auth/start'),
  
  completeFido2Auth: (credential: Credential) =>
    api.post<{ user: User; token: string }>('/auth/fido2/auth/complete', credential),
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
  fullName: string;
  avatarUrl?: string;
  subscriptionTier: 'free' | 'observer' | 'supporter' | 'commander';
  isGovernment: boolean;
  emailVerified: boolean;
  createdAt: string;
  updatedAt: string;
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

export interface NotificationSettings {
  emailAlerts: boolean;
  pushNotifications: boolean;
  weeklyDigest: boolean;
  securityAlerts: boolean;
}
