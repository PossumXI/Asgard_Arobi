/**
 * ASGARD Hubs API Client
 * Handles stream data, WebRTC signaling, and real-time updates
 */

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';
const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws';

// ============================================================================
// Types
// ============================================================================

export interface Stream {
  id: string;
  title: string;
  source: string;
  sourceType: 'satellite' | 'hunoid' | 'ground_station';
  sourceId: string;
  location: string;
  geoLocation?: { latitude: number; longitude: number };
  type: 'civilian' | 'military' | 'interstellar';
  status: 'live' | 'delayed' | 'offline' | 'buffering';
  viewers: number;
  latency: number;
  thumbnail?: string;
  description?: string;
  resolution: string;
  bitrate: number;
  startedAt: string;
}

export interface StreamSession {
  streamId: string;
  sessionId: string;
  iceServers: RTCIceServer[];
  signallingUrl: string;
  authToken: string;
  expiresAt: string;
}

export interface StreamStats {
  totalStreams: number;
  liveStreams: number;
  totalViewers: number;
  byCategory: {
    civilian: number;
    military: number;
    interstellar: number;
  };
}

export interface TelemetryData {
  entityId: string;
  entityType: 'satellite' | 'hunoid';
  timestamp: string;
  batteryPercent: number;
  status: string;
  location?: { latitude: number; longitude: number };
  metrics: Record<string, number>;
}

// ============================================================================
// API Client
// ============================================================================

class HubsApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  setToken(token: string | null): void {
    this.token = token;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(this.token && { Authorization: `Bearer ${this.token}` }),
      ...options.headers,
    };

    const response = await fetch(url, { ...options, headers });
    
    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Request failed' }));
      throw new Error(error.message);
    }

    return response.json();
  }

  // Stream endpoints
  async getStreams(params?: {
    type?: string;
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ streams: Stream[]; total: number }> {
    const query = new URLSearchParams(params as Record<string, string>).toString();
    return this.request(`/streams${query ? `?${query}` : ''}`);
  }

  async getStream(id: string): Promise<Stream> {
    return this.request(`/streams/${id}`);
  }

  async getStreamSession(streamId: string): Promise<StreamSession> {
    return this.request(`/streams/${streamId}/session`, { method: 'POST' });
  }

  async getStreamStats(): Promise<StreamStats> {
    return this.request('/streams/stats');
  }

  async getFeaturedStreams(): Promise<Stream[]> {
    return this.request('/streams/featured');
  }

  async getRecentStreams(limit: number = 10): Promise<Stream[]> {
    return this.request(`/streams/recent?limit=${limit}`);
  }

  // Telemetry endpoints
  async getSatelliteTelemetry(satelliteId: string): Promise<TelemetryData> {
    return this.request(`/telemetry/satellite/${satelliteId}`);
  }

  async getHunoidTelemetry(hunoidId: string): Promise<TelemetryData> {
    return this.request(`/telemetry/hunoid/${hunoidId}`);
  }

  // Search
  async searchStreams(query: string): Promise<Stream[]> {
    return this.request(`/streams/search?q=${encodeURIComponent(query)}`);
  }
}

export const hubsApi = new HubsApiClient(API_BASE_URL);

// ============================================================================
// WebRTC Streaming Client
// ============================================================================

export class WebRTCStreamClient {
  private peerConnection: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;
  private ws: WebSocket | null = null;
  private streamId: string;
  private sessionId: string;
  private onTrackCallback: ((stream: MediaStream) => void) | null = null;
  private onStatsCallback: ((stats: RTCStatsReport) => void) | null = null;

  constructor(session: StreamSession) {
    this.streamId = session.streamId;
    this.sessionId = session.sessionId;
    
    this.peerConnection = new RTCPeerConnection({
      iceServers: session.iceServers,
    });

    this.setupPeerConnection();
    this.connectSignaling(session.signallingUrl, session.authToken);
  }

  private setupPeerConnection(): void {
    if (!this.peerConnection) return;

    this.peerConnection.ontrack = (event) => {
      console.log('[WebRTC] Track received:', event.track.kind);
      if (this.onTrackCallback && event.streams[0]) {
        this.onTrackCallback(event.streams[0]);
      }
    };

    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate && this.ws) {
        this.ws.send(JSON.stringify({
          type: 'ice-candidate',
          candidate: event.candidate,
        }));
      }
    };

    this.peerConnection.oniceconnectionstatechange = () => {
      console.log('[WebRTC] ICE state:', this.peerConnection?.iceConnectionState);
    };

    this.peerConnection.onconnectionstatechange = () => {
      console.log('[WebRTC] Connection state:', this.peerConnection?.connectionState);
    };
  }

  private connectSignaling(url: string, token: string): void {
    const wsUrl = new URL(url);
    wsUrl.searchParams.set('token', token);
    wsUrl.searchParams.set('session', this.sessionId);

    this.ws = new WebSocket(wsUrl.toString());

    this.ws.onopen = () => {
      console.log('[WebRTC] Signaling connected');
      this.ws?.send(JSON.stringify({
        type: 'join',
        streamId: this.streamId,
        sessionId: this.sessionId,
      }));
    };

    this.ws.onmessage = async (event) => {
      const message = JSON.parse(event.data);
      await this.handleSignalingMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('[WebRTC] Signaling error:', error);
    };

    this.ws.onclose = () => {
      console.log('[WebRTC] Signaling disconnected');
    };
  }

  private async handleSignalingMessage(message: { type: string; [key: string]: unknown }): Promise<void> {
    if (!this.peerConnection) return;

    switch (message.type) {
      case 'offer':
        await this.peerConnection.setRemoteDescription(message.sdp as RTCSessionDescriptionInit);
        const answer = await this.peerConnection.createAnswer();
        await this.peerConnection.setLocalDescription(answer);
        this.ws?.send(JSON.stringify({
          type: 'answer',
          sdp: answer,
        }));
        break;

      case 'ice-candidate':
        await this.peerConnection.addIceCandidate(message.candidate as RTCIceCandidateInit);
        break;

      case 'error':
        console.error('[WebRTC] Server error:', message.message);
        break;
    }
  }

  onTrack(callback: (stream: MediaStream) => void): void {
    this.onTrackCallback = callback;
  }

  onStats(callback: (stats: RTCStatsReport) => void): void {
    this.onStatsCallback = callback;
  }

  async getStats(): Promise<RTCStatsReport | null> {
    if (!this.peerConnection) return null;
    return this.peerConnection.getStats();
  }

  disconnect(): void {
    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }

    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// ============================================================================
// Real-time Updates Client
// ============================================================================

type EventCallback = (data: unknown) => void;

export class HubsRealtimeClient {
  private ws: WebSocket | null = null;
  private handlers: Map<string, Set<EventCallback>> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  constructor(private baseUrl: string = WS_BASE_URL) {}

  connect(token?: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = new URL(`${this.baseUrl}/realtime`);
      if (token) {
        url.searchParams.set('token', token);
      }

      this.ws = new WebSocket(url.toString());

      this.ws.onopen = () => {
        console.log('[Realtime] Connected');
        this.reconnectAttempts = 0;
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.dispatchEvent(message.type, message.payload);
        } catch (error) {
          console.error('[Realtime] Parse error:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('[Realtime] Error:', error);
        reject(error);
      };

      this.ws.onclose = () => {
        console.log('[Realtime] Disconnected');
        this.attemptReconnect(token);
      };
    });
  }

  subscribe(eventType: string, callback: EventCallback): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers.get(eventType)!.add(callback);

    // Return unsubscribe function
    return () => {
      const handlers = this.handlers.get(eventType);
      if (handlers) {
        handlers.delete(callback);
      }
    };
  }

  private dispatchEvent(type: string, payload: unknown): void {
    const handlers = this.handlers.get(type);
    if (handlers) {
      handlers.forEach((callback) => callback(payload));
    }

    // Also dispatch to wildcard handlers
    const wildcardHandlers = this.handlers.get('*');
    if (wildcardHandlers) {
      wildcardHandlers.forEach((callback) => callback({ type, payload }));
    }
  }

  private attemptReconnect(token?: string): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[Realtime] Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = 1000 * Math.pow(2, this.reconnectAttempts);

    setTimeout(() => {
      console.log(`[Realtime] Reconnecting (attempt ${this.reconnectAttempts})`);
      this.connect(token).catch(console.error);
    }, delay);
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

export const hubsRealtime = new HubsRealtimeClient();
