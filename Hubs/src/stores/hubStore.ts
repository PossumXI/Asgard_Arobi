/**
 * ASGARD Hubs Store
 * State management for streaming interface
 */

import { create } from 'zustand';
import type { Stream, StreamStats } from '@/lib/api';

// ============================================================================
// Streams Store
// ============================================================================

interface StreamsState {
  streams: Stream[];
  featuredStreams: Stream[];
  currentStream: Stream | null;
  stats: StreamStats | null;
  isLoading: boolean;
  error: string | null;
  filters: StreamFilters;
  
  setStreams: (streams: Stream[]) => void;
  setFeaturedStreams: (streams: Stream[]) => void;
  setCurrentStream: (stream: Stream | null) => void;
  setStats: (stats: StreamStats) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setFilters: (filters: Partial<StreamFilters>) => void;
  updateStreamViewers: (streamId: string, viewers: number) => void;
  updateStreamStatus: (streamId: string, status: Stream['status']) => void;
}

interface StreamFilters {
  type: 'all' | 'civilian' | 'military' | 'interstellar';
  status: 'all' | 'live' | 'delayed' | 'offline';
  search: string;
  sortBy: 'viewers' | 'recent' | 'name';
}

const defaultFilters: StreamFilters = {
  type: 'all',
  status: 'all',
  search: '',
  sortBy: 'viewers',
};

export const useStreamsStore = create<StreamsState>((set) => ({
  streams: [],
  featuredStreams: [],
  currentStream: null,
  stats: null,
  isLoading: false,
  error: null,
  filters: defaultFilters,
  
  setStreams: (streams) => set({ streams, isLoading: false, error: null }),
  
  setFeaturedStreams: (featuredStreams) => set({ featuredStreams }),
  
  setCurrentStream: (currentStream) => set({ currentStream }),
  
  setStats: (stats) => set({ stats }),
  
  setLoading: (isLoading) => set({ isLoading }),
  
  setError: (error) => set({ error, isLoading: false }),
  
  setFilters: (newFilters) =>
    set((state) => ({ filters: { ...state.filters, ...newFilters } })),
  
  updateStreamViewers: (streamId, viewers) =>
    set((state) => ({
      streams: state.streams.map((s) =>
        s.id === streamId ? { ...s, viewers } : s
      ),
      currentStream:
        state.currentStream?.id === streamId
          ? { ...state.currentStream, viewers }
          : state.currentStream,
    })),
  
  updateStreamStatus: (streamId, status) =>
    set((state) => ({
      streams: state.streams.map((s) =>
        s.id === streamId ? { ...s, status } : s
      ),
      currentStream:
        state.currentStream?.id === streamId
          ? { ...state.currentStream, status }
          : state.currentStream,
    })),
}));

// ============================================================================
// Player Store
// ============================================================================

interface PlayerState {
  isPlaying: boolean;
  isMuted: boolean;
  volume: number;
  isFullscreen: boolean;
  showStats: boolean;
  quality: 'auto' | '1080p' | '720p' | '480p' | '360p';
  latency: number;
  bufferHealth: number;
  
  setPlaying: (playing: boolean) => void;
  togglePlay: () => void;
  setMuted: (muted: boolean) => void;
  toggleMute: () => void;
  setVolume: (volume: number) => void;
  setFullscreen: (fullscreen: boolean) => void;
  toggleFullscreen: () => void;
  setShowStats: (show: boolean) => void;
  setQuality: (quality: PlayerState['quality']) => void;
  setLatency: (latency: number) => void;
  setBufferHealth: (health: number) => void;
}

export const usePlayerStore = create<PlayerState>((set) => ({
  isPlaying: true,
  isMuted: false,
  volume: 100,
  isFullscreen: false,
  showStats: false,
  quality: 'auto',
  latency: 0,
  bufferHealth: 100,
  
  setPlaying: (isPlaying) => set({ isPlaying }),
  togglePlay: () => set((state) => ({ isPlaying: !state.isPlaying })),
  
  setMuted: (isMuted) => set({ isMuted }),
  toggleMute: () => set((state) => ({ isMuted: !state.isMuted })),
  
  setVolume: (volume) => set({ volume, isMuted: volume === 0 }),
  
  setFullscreen: (isFullscreen) => set({ isFullscreen }),
  toggleFullscreen: () => set((state) => ({ isFullscreen: !state.isFullscreen })),
  
  setShowStats: (showStats) => set({ showStats }),
  setQuality: (quality) => set({ quality }),
  setLatency: (latency) => set({ latency }),
  setBufferHealth: (bufferHealth) => set({ bufferHealth }),
}));

// ============================================================================
// Chat Store (for stream comments)
// ============================================================================

interface ChatMessage {
  id: string;
  userId: string;
  username: string;
  message: string;
  timestamp: string;
  isHighlighted?: boolean;
}

interface ChatState {
  messages: ChatMessage[];
  isConnected: boolean;
  isLoading: boolean;
  
  addMessage: (message: ChatMessage) => void;
  clearMessages: () => void;
  setConnected: (connected: boolean) => void;
  setLoading: (loading: boolean) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  messages: [],
  isConnected: false,
  isLoading: false,
  
  addMessage: (message) =>
    set((state) => ({
      messages: [...state.messages.slice(-199), message], // Keep last 200 messages
    })),
  
  clearMessages: () => set({ messages: [] }),
  
  setConnected: (isConnected) => set({ isConnected }),
  
  setLoading: (isLoading) => set({ isLoading }),
}));
