import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, authApi, userApi, User } from '@/lib/api';

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (data: { email: string; password: string; fullName: string }) => Promise<void>;
  signOut: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const TOKEN_KEY = 'asgard-auth-token';

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const navigate = useNavigate();

  const refreshUser = useCallback(async () => {
    try {
      const userData = await userApi.getProfile();
      setUser(userData);
    } catch {
      setUser(null);
      localStorage.removeItem(TOKEN_KEY);
      api.setToken(null);
    }
  }, []);

  useEffect(() => {
    const initAuth = async () => {
      const token = localStorage.getItem(TOKEN_KEY);
      if (token) {
        api.setToken(token);
        await refreshUser();
      }
      setIsLoading(false);
    };

    initAuth();
  }, [refreshUser]);

  const signIn = async (email: string, password: string) => {
    const response = await authApi.signIn(email, password);
    localStorage.setItem(TOKEN_KEY, response.token);
    api.setToken(response.token);
    setUser(response.user);
    navigate('/dashboard');
  };

  const signUp = async (data: { email: string; password: string; fullName: string }) => {
    const response = await authApi.signUp(data);
    localStorage.setItem(TOKEN_KEY, response.token);
    api.setToken(response.token);
    setUser(response.user);
    navigate('/dashboard');
  };

  const signOut = async () => {
    try {
      await authApi.signOut();
    } finally {
      localStorage.removeItem(TOKEN_KEY);
      api.setToken(null);
      setUser(null);
      navigate('/');
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        signIn,
        signUp,
        signOut,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
