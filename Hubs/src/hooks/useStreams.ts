/**
 * ASGARD Hubs Streaming Hooks
 * React hooks for stream data and WebRTC connections
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { hubsApi, hubsRealtime, WebRTCStreamClient, Stream, StreamSession } from '@/lib/api';
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
      clientRef.current = new WebRTCStreamClient(session);
      
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
    const unsubViewers = hubsRealtime.subscribe('stream.viewers', (data) => {
      const { streamId, viewers } = data as { streamId: string; viewers: number };
      updateStreamViewers(streamId, viewers);
    });

    const unsubStatus = hubsRealtime.subscribe('stream.status', (data) => {
      const { streamId, status } = data as { streamId: string; status: Stream['status'] };
      updateStreamStatus(streamId, status);
    });

    return () => {
      unsubViewers();
      unsubStatus();
    };
  }, [updateStreamViewers, updateStreamStatus]);
}

// ============================================================================
// Video Element Hook
// ============================================================================

export function useVideoElement(mediaStream: MediaStream | null) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const isPlaying = usePlayerStore((state) => state.isPlaying);
  const isMuted = usePlayerStore((state) => state.isMuted);
  const volume = usePlayerStore((state) => state.volume);

  useEffect(() => {
    if (videoRef.current && mediaStream) {
      videoRef.current.srcObject = mediaStream;
    }
  }, [mediaStream]);

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
