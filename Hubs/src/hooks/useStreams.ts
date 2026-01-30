/**
 * ASGARD Hubs Streaming Hooks
 * React hooks for stream data and WebRTC connections
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import Hls from 'hls.js';
import { useQuery, useMutation } from '@tanstack/react-query';
import { hubsApi, hubsRealtime, WebRTCStreamClient, Stream, StreamSession, ChatMessage } from '@/lib/api';
import { useStreamsStore, usePlayerStore } from '@/stores/hubStore';

// ============================================================================
// Query Keys
// ============================================================================

export const streamQueryKeys = {
  streams: (params?: Record<string, unknown>) => ['streams', params] as const,
  stream: (id: string) => ['stream', id] as const,
  featured: ['streams', 'featured'] as const,
  recent: ['streams', 'recent'] as const,
  stats: ['streams', 'stats'] as const,
};

// ============================================================================
// Stream List Hooks
// ============================================================================

export function useStreams(params?: {
  type?: string;
  status?: string;
  limit?: number;
}) {
  const setStreams = useStreamsStore((state) => state.setStreams);
  const setLoading = useStreamsStore((state) => state.setLoading);
  const setError = useStreamsStore((state) => state.setError);

  return useQuery({
    queryKey: streamQueryKeys.streams(params),
    queryFn: async () => {
      setLoading(true);
      try {
        const result = await hubsApi.getStreams(params);
        setStreams(result.streams);
        return result;
      } catch (error) {
        setError(error instanceof Error ? error.message : 'Failed to load streams');
        throw error;
      }
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}

export function useFeaturedStreams() {
  const setFeaturedStreams = useStreamsStore((state) => state.setFeaturedStreams);

  return useQuery({
    queryKey: streamQueryKeys.featured,
    queryFn: async () => {
      const streams = await hubsApi.getFeaturedStreams();
      setFeaturedStreams(streams);
      return streams;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useRecentStreams(limit: number = 10) {
  return useQuery({
    queryKey: streamQueryKeys.recent,
    queryFn: () => hubsApi.getRecentStreams(limit),
    staleTime: 60 * 1000, // 1 minute
  });
}

export function useStreamStats() {
  const setStats = useStreamsStore((state) => state.setStats);

  return useQuery({
    queryKey: streamQueryKeys.stats,
    queryFn: async () => {
      const stats = await hubsApi.getStreamStats();
      setStats(stats);
      return stats;
    },
    refetchInterval: 10000, // Refresh every 10 seconds
  });
}

// ============================================================================
// Single Stream Hook
// ============================================================================

export function useStream(streamId: string) {
  const setCurrentStream = useStreamsStore((state) => state.setCurrentStream);

  return useQuery({
    queryKey: streamQueryKeys.stream(streamId),
    queryFn: async () => {
      const stream = await hubsApi.getStream(streamId);
      setCurrentStream(stream);
      return stream;
    },
    enabled: !!streamId,
  });
}

// ============================================================================
// WebRTC Streaming Hook
// ============================================================================

interface UseWebRTCStreamOptions {
  onConnected?: () => void;
  onDisconnected?: () => void;
  onError?: (error: Error) => void;
  mode?: 'subscriber' | 'publisher';
  localStream?: MediaStream | null;
}

export function useWebRTCStream(
  streamId: string | null,
  options: UseWebRTCStreamOptions = {}
) {
  const clientRef = useRef<WebRTCStreamClient | null>(null);
  const [mediaStream, setMediaStream] = useState<MediaStream | null>(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const setLatency = usePlayerStore((state) => state.setLatency);

  const connect = useCallback(async (session: StreamSession) => {
    setIsConnecting(true);
    setError(null);

    try {
      clientRef.current = new WebRTCStreamClient(session, {
        mode: options.mode,
        localStream: options.localStream ?? null,
      });
      
      clientRef.current.onTrack((stream) => {
        setMediaStream(stream);
        setIsConnecting(false);
        options.onConnected?.();
      });

      // Monitor stats for latency
      const statsInterval = setInterval(async () => {
        const stats = await clientRef.current?.getStats();
        if (stats) {
          stats.forEach((report) => {
            if (report.type === 'candidate-pair' && report.currentRoundTripTime) {
              setLatency(report.currentRoundTripTime * 1000);
            }
          });
        }
      }, 1000);

      return () => clearInterval(statsInterval);
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Connection failed');
      setError(error);
      setIsConnecting(false);
      options.onError?.(error);
    }
  }, [options, setLatency]);

  const disconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
      clientRef.current = null;
      setMediaStream(null);
      options.onDisconnected?.();
    }
  }, [options]);

  // Get session and connect
  const sessionMutation = useMutation({
    mutationFn: (id: string) => hubsApi.getStreamSession(id),
    onSuccess: (session) => {
      connect(session);
    },
    onError: (err) => {
      const error = err instanceof Error ? err : new Error('Failed to get session');
      setError(error);
      options.onError?.(error);
    },
  });

  useEffect(() => {
    if (streamId) {
      sessionMutation.mutate(streamId);
    }

    return () => {
      disconnect();
    };
  }, [streamId]);

  return {
    mediaStream,
    isConnecting: isConnecting || sessionMutation.isPending,
    error,
    disconnect,
    reconnect: () => streamId && sessionMutation.mutate(streamId),
  };
}

// ============================================================================
// Real-time Updates Hook
// ============================================================================

export function useStreamUpdates() {
  const updateStreamViewers = useStreamsStore((state) => state.updateStreamViewers);
  const updateStreamStatus = useStreamsStore((state) => state.updateStreamStatus);

  useEffect(() => {
    hubsRealtime.connect().catch((error) => {
      console.error('[Realtime] Connection failed:', error);
    });

    const unsubViewers = hubsRealtime.subscribe('stream.viewers', (data) => {
      const { streamId, viewers } = data as { streamId: string; viewers: number };
      updateStreamViewers(streamId, viewers);
    });

    const unsubStatus = hubsRealtime.subscribe('stream.status', (data) => {
      const { streamId, status } = data as { streamId: string; status: Stream['status'] };
      updateStreamStatus(streamId, status);
    });

    const unsubUpdate = hubsRealtime.subscribe('stream_update', (data) => {
      const payload = data as {
        streamId?: string;
        id?: string;
        viewers?: number;
        status?: Stream['status'];
        stream?: { id?: string; viewers?: number; status?: Stream['status'] };
      };
      const streamId = payload.streamId || payload.id || payload.stream?.id;
      if (!streamId) return;
      if (typeof payload.viewers === 'number') {
        updateStreamViewers(streamId, payload.viewers);
      } else if (typeof payload.stream?.viewers === 'number') {
        updateStreamViewers(streamId, payload.stream.viewers);
      }
      if (payload.status) {
        updateStreamStatus(streamId, payload.status);
      } else if (payload.stream?.status) {
        updateStreamStatus(streamId, payload.stream.status);
      }
    });

    return () => {
      unsubViewers();
      unsubStatus();
      unsubUpdate();
    };
  }, [updateStreamViewers, updateStreamStatus]);
}

// ============================================================================
// Stream Chat Hook (optional backend support)
// ============================================================================

export function useStreamChat(streamId: string | null) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const chatEnabled = import.meta.env.VITE_CHAT_ENABLED === 'true';

  useEffect(() => {
    if (!chatEnabled || !streamId) {
      setMessages([]);
      setIsConnected(false);
      setIsLoading(false);
      setError(null);
      return;
    }

    let unsubscribe: (() => void) | null = null;

    const setup = async () => {
      setIsLoading(true);
      setError(null);

      try {
        await hubsRealtime.connect();
        setIsConnected(true);
      } catch {
        setError('Failed to connect to chat service');
      }

      try {
        const history = await hubsApi.getStreamChat(streamId);
        setMessages(history);
      } catch {
        setError('Failed to load chat history');
      } finally {
        setIsLoading(false);
      }

      unsubscribe = hubsRealtime.subscribe('stream_chat', (data) => {
        const payload = data as Partial<ChatMessage> & { streamId?: string; message?: string };
        if (payload.streamId && payload.streamId !== streamId) {
          return;
        }
        if (!payload.id || !payload.message) {
          return;
        }
        setMessages((prev) => [...prev, payload as ChatMessage].slice(-200));
      });
    };

    setup();

    return () => {
      if (unsubscribe) {
        unsubscribe();
      }
    };
  }, [chatEnabled, streamId]);

  const sendMessage = useCallback(async (message: string) => {
    if (!chatEnabled || !streamId) {
      return;
    }
    await hubsApi.sendStreamChat(streamId, message);
  }, [chatEnabled, streamId]);

  return {
    enabled: chatEnabled,
    messages,
    isConnected,
    isLoading,
    error,
    sendMessage,
  };
}

// ============================================================================
// Video Element Hook
// ============================================================================

export function useVideoElement(mediaStream: MediaStream | null, playbackUrl?: string) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const isPlaying = usePlayerStore((state) => state.isPlaying);
  const isMuted = usePlayerStore((state) => state.isMuted);
  const volume = usePlayerStore((state) => state.volume);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    if (mediaStream) {
      if (hlsRef.current) {
        hlsRef.current.destroy();
        hlsRef.current = null;
      }
      video.src = '';
      video.srcObject = mediaStream;
      return;
    }

    if (playbackUrl) {
      video.srcObject = null;
      if (Hls.isSupported()) {
        const hls = new Hls({ enableWorker: true });
        hls.loadSource(playbackUrl);
        hls.attachMedia(video);
        hlsRef.current = hls;
        return () => {
          hls.destroy();
          hlsRef.current = null;
        };
      }
      video.src = playbackUrl;
      return;
    }

    video.srcObject = null;
    video.removeAttribute('src');
    video.load();
  }, [mediaStream, playbackUrl]);

  useEffect(() => {
    if (videoRef.current) {
      if (isPlaying) {
        videoRef.current.play().catch(console.error);
      } else {
        videoRef.current.pause();
      }
    }
  }, [isPlaying]);

  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.muted = isMuted;
    }
  }, [isMuted]);

  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.volume = volume / 100;
    }
  }, [volume]);

  return videoRef;
}

// ============================================================================
// Search Hook
// ============================================================================

export function useStreamSearch(query: string) {
  return useQuery({
    queryKey: ['streams', 'search', query],
    queryFn: () => hubsApi.searchStreams(query),
    enabled: query.length >= 2,
    staleTime: 30 * 1000, // 30 seconds
  });
}
