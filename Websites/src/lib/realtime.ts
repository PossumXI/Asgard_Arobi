/**
 * ASGARD Real-time Connection Manager
 * Handles WebSocket connections for live data updates
 */

import { RealtimeEvent, RealtimeEventType } from './types';

type EventHandler<T = unknown> = (event: RealtimeEvent<T>) => void;

interface RealtimeConfig {
  url: string;
  authToken?: string;
  reconnectAttempts?: number;
  reconnectDelay?: number;
  heartbeatInterval?: number;
}

const DEFAULT_CONFIG: Partial<RealtimeConfig> = {
  reconnectAttempts: 5,
  reconnectDelay: 1000,
  heartbeatInterval: 30000,
};

class RealtimeConnection {
  private ws: WebSocket | null = null;
  private config: RealtimeConfig;
  private handlers: Map<RealtimeEventType | '*', Set<EventHandler>> = new Map();
  private reconnectCount = 0;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private isIntentionallyClosed = false;

  constructor(config: RealtimeConfig) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.isIntentionallyClosed = false;
      
      const url = new URL(this.config.url);
      if (this.config.authToken) {
        url.searchParams.set('token', this.config.authToken);
      }

      this.ws = new WebSocket(url.toString());

      this.ws.onopen = () => {
        console.log('[Realtime] Connected');
        this.reconnectCount = 0;
        this.startHeartbeat();
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as RealtimeEvent;
          this.dispatchEvent(data);
        } catch (error) {
          console.error('[Realtime] Failed to parse message:', error);
        }
      };

      this.ws.onclose = (event) => {
        console.log('[Realtime] Disconnected:', event.code, event.reason);
        this.stopHeartbeat();
        
        if (!this.isIntentionallyClosed) {
          this.attemptReconnect();
        }
      };

      this.ws.onerror = (error) => {
        console.error('[Realtime] WebSocket error:', error);
        reject(error);
      };
    });
  }

  disconnect(): void {
    this.isIntentionallyClosed = true;
    this.stopHeartbeat();
    
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
  }

  subscribe<T = unknown>(
    eventType: RealtimeEventType | '*',
    handler: EventHandler<T>
  ): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    
    this.handlers.get(eventType)!.add(handler as EventHandler);

    // Send subscription message to server
    this.send({
      type: 'subscribe',
      channel: eventType,
    });

    // Return unsubscribe function
    return () => {
      const handlers = this.handlers.get(eventType);
      if (handlers) {
        handlers.delete(handler as EventHandler);
        if (handlers.size === 0) {
          this.handlers.delete(eventType);
          this.send({
            type: 'unsubscribe',
            channel: eventType,
          });
        }
      }
    };
  }

  private dispatchEvent(event: RealtimeEvent): void {
    // Dispatch to specific handlers
    const typeHandlers = this.handlers.get(event.type);
    if (typeHandlers) {
      typeHandlers.forEach((handler) => handler(event));
    }

    // Dispatch to wildcard handlers
    const wildcardHandlers = this.handlers.get('*');
    if (wildcardHandlers) {
      wildcardHandlers.forEach((handler) => handler(event));
    }
  }

  private send(data: unknown): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  private startHeartbeat(): void {
    if (this.config.heartbeatInterval) {
      this.heartbeatTimer = setInterval(() => {
        this.send({ type: 'ping', timestamp: new Date().toISOString() });
      }, this.config.heartbeatInterval);
    }
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectCount >= (this.config.reconnectAttempts || 5)) {
      console.error('[Realtime] Max reconnection attempts reached');
      return;
    }

    this.reconnectCount++;
    const delay = (this.config.reconnectDelay || 1000) * Math.pow(2, this.reconnectCount - 1);
    
    console.log(`[Realtime] Reconnecting in ${delay}ms (attempt ${this.reconnectCount})`);
    
    setTimeout(() => {
      this.connect().catch((error) => {
        console.error('[Realtime] Reconnection failed:', error);
      });
    }, delay);
  }

  get isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

// Singleton instance
let realtimeInstance: RealtimeConnection | null = null;

export function initRealtime(config: RealtimeConfig): RealtimeConnection {
  if (realtimeInstance) {
    realtimeInstance.disconnect();
  }
  
  realtimeInstance = new RealtimeConnection(config);
  return realtimeInstance;
}

export function getRealtime(): RealtimeConnection | null {
  return realtimeInstance;
}

export function useRealtime(): RealtimeConnection {
  if (!realtimeInstance) {
    throw new Error('Realtime not initialized. Call initRealtime() first.');
  }
  return realtimeInstance;
}

// React hook for subscribing to events
import { useEffect, useState } from 'react';

export function useRealtimeEvent<T = unknown>(
  eventType: RealtimeEventType | '*'
): T | null {
  const [data, setData] = useState<T | null>(null);

  useEffect(() => {
    const realtime = getRealtime();
    if (!realtime) return;

    const unsubscribe = realtime.subscribe<T>(eventType, (event) => {
      setData(event.payload as T);
    });

    return unsubscribe;
  }, [eventType]);

  return data;
}

export function useRealtimeEvents<T = unknown>(
  eventTypes: RealtimeEventType[]
): Map<RealtimeEventType, T> {
  const [events, setEvents] = useState<Map<RealtimeEventType, T>>(new Map());

  useEffect(() => {
    const realtime = getRealtime();
    if (!realtime) return;

    const unsubscribers = eventTypes.map((type) =>
      realtime.subscribe<T>(type, (event) => {
        setEvents((prev) => new Map(prev).set(event.type, event.payload as T));
      })
    );

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [eventTypes.join(',')]);

  return events;
}
