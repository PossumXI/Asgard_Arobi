/**
 * ASGARD API Hooks
 * React Query hooks for data fetching with type safety
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api, authApi, userApi, subscriptionApi } from '@/lib/api';
import type {
  DashboardStats,
  Alert,
  Mission,
  Satellite,
  Hunoid,
  NotificationSettings,
  SignUpRequest,
  SignInRequest,
} from '@/lib/types';
import type { User } from '@/lib/api';

// Auth response from our API
interface ApiAuthResponse {
  user: User;
  token: string;
}
import { useAuthStore } from '@/stores/appStore';

// ============================================================================
// Query Keys
// ============================================================================

export const queryKeys = {
  user: ['user'] as const,
  subscription: ['subscription'] as const,
  subscriptionPlans: ['subscription-plans'] as const,
  dashboardStats: ['dashboard-stats'] as const,
  alerts: (params?: Record<string, unknown>) => ['alerts', params] as const,
  missions: (params?: Record<string, unknown>) => ['missions', params] as const,
  satellites: (params?: Record<string, unknown>) => ['satellites', params] as const,
  hunoids: (params?: Record<string, unknown>) => ['hunoids', params] as const,
  notificationSettings: ['notification-settings'] as const,
};

// ============================================================================
// Auth Hooks
// ============================================================================

export function useSignIn() {
  const setAuth = useAuthStore((state) => state.setAuth);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SignInRequest) => authApi.signIn(data.email, data.password, data.accessCode),
    onSuccess: (response: ApiAuthResponse) => {
      api.setToken(response.token);
      setAuth(response.user, response.token);
      queryClient.invalidateQueries({ queryKey: queryKeys.user });
    },
  });
}

export function useSignUp() {
  const setAuth = useAuthStore((state) => state.setAuth);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SignUpRequest) => authApi.signUp(data),
    onSuccess: (response: ApiAuthResponse) => {
      api.setToken(response.token);
      setAuth(response.user, response.token);
      queryClient.invalidateQueries({ queryKey: queryKeys.user });
    },
  });
}

export function useSignOut() {
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => authApi.signOut(),
    onSettled: () => {
      api.setToken(null);
      clearAuth();
      queryClient.clear();
    },
  });
}

// ============================================================================
// User Hooks
// ============================================================================

export function useUser() {
  const token = useAuthStore((state) => state.token);
  const setAuth = useAuthStore((state) => state.setAuth);

  return useQuery({
    queryKey: queryKeys.user,
    queryFn: async () => {
      const user = await userApi.getProfile();
      if (token) {
        setAuth(user, token);
      }
      return user;
    },
    enabled: !!token,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  const updateUser = useAuthStore((state) => state.updateUser);

  return useMutation({
    mutationFn: (data: Partial<User>) => userApi.updateProfile(data),
    onSuccess: (user) => {
      updateUser(user);
      queryClient.setQueryData(queryKeys.user, user);
    },
  });
}

// ============================================================================
// Subscription Hooks
// ============================================================================

export function useSubscription() {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.subscription,
    queryFn: () => userApi.getSubscription(),
    enabled: !!token,
    staleTime: 5 * 60 * 1000,
  });
}

export function useSubscriptionPlans() {
  return useQuery({
    queryKey: queryKeys.subscriptionPlans,
    queryFn: () => subscriptionApi.getPlans(),
    staleTime: 30 * 60 * 1000, // 30 minutes
  });
}

export function useCreateCheckoutSession() {
  return useMutation({
    mutationFn: (planId: string) => subscriptionApi.createCheckoutSession(planId),
    onSuccess: (response) => {
      // Redirect to Stripe Checkout
      window.location.href = response.sessionUrl;
    },
  });
}

export function useCreateBillingPortal() {
  return useMutation({
    mutationFn: () => subscriptionApi.createPortalSession(),
    onSuccess: (response) => {
      window.location.href = response.portalUrl;
    },
  });
}

export function useCancelSubscription() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => subscriptionApi.cancelSubscription(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subscription });
    },
  });
}

// ============================================================================
// Dashboard Hooks
// ============================================================================

export function useDashboardStats() {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.dashboardStats,
    queryFn: async (): Promise<DashboardStats> => {
      const response = await api.get<DashboardStats>('/dashboard/stats');
      return response;
    },
    enabled: !!token,
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}

// ============================================================================
// Alerts Hooks
// ============================================================================

export function useAlerts(params?: { status?: string; type?: string; limit?: number }) {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.alerts(params),
    queryFn: async () => {
      const query = new URLSearchParams(params as Record<string, string>).toString();
      return api.get<{ alerts: Alert[]; total: number }>(`/alerts${query ? `?${query}` : ''}`);
    },
    enabled: !!token,
  });
}

// ============================================================================
// Missions Hooks
// ============================================================================

export function useMissions(params?: { status?: string; limit?: number }) {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.missions(params),
    queryFn: async () => {
      const query = new URLSearchParams(params as Record<string, string>).toString();
      return api.get<{ missions: Mission[]; total: number }>(`/missions${query ? `?${query}` : ''}`);
    },
    enabled: !!token,
  });
}

// ============================================================================
// Satellite Hooks
// ============================================================================

export function useSatellites(params?: { status?: string; limit?: number }) {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.satellites(params),
    queryFn: async () => {
      const query = new URLSearchParams(params as Record<string, string>).toString();
      return api.get<{ satellites: Satellite[]; total: number }>(`/satellites${query ? `?${query}` : ''}`);
    },
    enabled: !!token,
  });
}

// ============================================================================
// Hunoid Hooks
// ============================================================================

export function useHunoids(params?: { status?: string; limit?: number }) {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.hunoids(params),
    queryFn: async () => {
      const query = new URLSearchParams(params as Record<string, string>).toString();
      return api.get<{ hunoids: Hunoid[]; total: number }>(`/hunoids${query ? `?${query}` : ''}`);
    },
    enabled: !!token,
  });
}

// ============================================================================
// Notification Settings Hooks
// ============================================================================

export function useNotificationSettings() {
  const token = useAuthStore((state) => state.token);

  return useQuery({
    queryKey: queryKeys.notificationSettings,
    queryFn: async () => {
      return api.get<NotificationSettings>('/user/notifications');
    },
    enabled: !!token,
  });
}

export function useUpdateNotificationSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settings: NotificationSettings) =>
      userApi.updateNotificationSettings(settings),
    onSuccess: (settings) => {
      queryClient.setQueryData(queryKeys.notificationSettings, settings);
    },
  });
}
