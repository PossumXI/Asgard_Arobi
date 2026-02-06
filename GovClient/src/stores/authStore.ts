/**
 * ASGARD Government Client - Authentication Store
 * Secure state management for authentication
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { create } from 'zustand';

interface User {
  id: string;
  email: string;
  name: string;
  role: 'operator' | 'commander' | 'admin' | 'super_admin';
  clearanceLevel: number;
  department: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  hasAccessCode: boolean;
  accessCodeExpiry: number | null;

  // Actions
  setAccessCode: (expiry: number) => void;
  setAuth: (user: User, token: string) => void;
  logout: () => void;
  checkStoredAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isAuthenticated: false,
  hasAccessCode: false,
  accessCodeExpiry: null,

  setAccessCode: (expiry: number) => {
    set({ hasAccessCode: true, accessCodeExpiry: expiry });
    // Store securely in electron store
    if (window.asgardAPI) {
      window.asgardAPI.store.set('accessCodeExpiry', expiry);
    }
  },

  setAuth: (user: User, token: string) => {
    set({ user, token, isAuthenticated: true });
    // Store securely in electron store
    if (window.asgardAPI) {
      window.asgardAPI.store.set('authToken', token);
      window.asgardAPI.store.set('user', user);
    }
  },

  logout: () => {
    set({
      user: null,
      token: null,
      isAuthenticated: false,
    });
    // Clear from electron store
    if (window.asgardAPI) {
      window.asgardAPI.store.delete('authToken');
      window.asgardAPI.store.delete('user');
    }
  },

  checkStoredAuth: async () => {
    if (!window.asgardAPI) {
      return;
    }

    // Check access code
    const accessCodeExpiry = await window.asgardAPI.store.get('accessCodeExpiry') as number | null;
    if (accessCodeExpiry && accessCodeExpiry > Date.now()) {
      set({ hasAccessCode: true, accessCodeExpiry });
    }

    // Check auth token
    const token = await window.asgardAPI.store.get('authToken') as string | null;
    const user = await window.asgardAPI.store.get('user') as User | null;

    if (token && user) {
      // Validate token with backend
      try {
        const response = await fetch('/api/auth/validate', {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (response.ok) {
          set({ user, token, isAuthenticated: true });
        }
      } catch {
        // Token invalid, clear auth
        window.asgardAPI.store.delete('authToken');
        window.asgardAPI.store.delete('user');
      }
    }
  },
}));
