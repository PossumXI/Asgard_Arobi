import { useEffect, useMemo, useRef, useState } from 'react';
import { TerminalSquare, Loader2, Send, RefreshCw } from 'lucide-react';
import { controlPlaneApi, type ControlPlaneStatusResponse } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';

type ControlEvent = {
  id?: string;
  timestamp?: string;
  targetDomain?: string;
  targetSystem?: string;
  commandType?: string;
  parameters?: Record<string, unknown>;
  priority?: number;
};

const defaultParams = '{\n  "mode": "diagnostic"\n}';

export default function CommandHub() {
  const [targetDomain, setTargetDomain] = useState('controlplane');
  const [targetSystem, setTargetSystem] = useState('');
  const [commandType, setCommandType] = useState('status.check');
  const [priority, setPriority] = useState(5);
  const [params, setParams] = useState(defaultParams);
  const [events, setEvents] = useState<ControlEvent[]>([]);
  const [status, setStatus] = useState<ControlPlaneStatusResponse | null>(null);
  const [isSending, setIsSending] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const wsUrl = useMemo(() => {
    const base = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws/events';
    const url = new URL(base);
    const token = localStorage.getItem('asgard-auth-token');
    if (token) {
      url.searchParams.set('token', token);
    }
    url.searchParams.set('access', 'government');
    return url.toString();
  }, []);

  const fetchStatus = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await controlPlaneApi.getStatus();
      setStatus(data);
    } catch {
      setError('Failed to load control plane status');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, []);

  useEffect(() => {
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      ws.send(JSON.stringify({ type: 'subscribe', filters: ['control_command'] }));
    };

    ws.onmessage = (event) => {
      try {
        const raw = event.data as string;
        const frames = raw.includes('\n') ? raw.split('\n') : [raw];
        frames.forEach((frame) => {
          if (!frame.trim()) return;
          const message = JSON.parse(frame);
          if (message.type === 'event' && message.eventType === 'control_command') {
            const payload = message.event?.payload ?? {};
            setEvents((prev) => [payload as ControlEvent, ...prev].slice(0, 50));
          }
        });
      } catch {
        // ignore parse errors
      }
    };

    return () => {
      ws.close();
    };
  }, [wsUrl]);

  const handleSend = async () => {
    setIsSending(true);
    setError(null);
    try {
      const parsedParams = params.trim() ? JSON.parse(params) : {};
      await controlPlaneApi.sendCommand({
        targetDomain,
        targetSystem: targetSystem || undefined,
        commandType,
        parameters: parsedParams,
        priority,
      });
    } catch (err) {
      setError('Failed to send command (check JSON parameters)');
    } finally {
      setIsSending(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
            <TerminalSquare className="w-5 h-5 text-primary" />
          </div>
          <div>
            <CardTitle>Command Hub</CardTitle>
            <p className="text-sm text-asgard-500">
              Issue real-time control commands and monitor responses.
            </p>
          </div>
        </CardHeader>
      </Card>

      <div className="grid gap-6 lg:grid-cols-[1.2fr_1fr]">
        <Card>
          <CardHeader>
            <CardTitle>Send Command</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                  Target Domain
                </label>
                <Input value={targetDomain} onChange={(e) => setTargetDomain(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                  Target System
                </label>
                <Input value={targetSystem} onChange={(e) => setTargetSystem(e.target.value)} />
              </div>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                  Command Type
                </label>
                <Input value={commandType} onChange={(e) => setCommandType(e.target.value)} />
              </div>
              <div>
                <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                  Priority (0-10)
                </label>
                <Input
                  type="number"
                  min={0}
                  max={10}
                  value={priority}
                  onChange={(e) => setPriority(Number(e.target.value))}
                />
              </div>
            </div>
            <div>
              <label className="text-xs font-semibold uppercase tracking-wide text-asgard-500">
                Parameters (JSON)
              </label>
              <textarea
                value={params}
                onChange={(e) => setParams(e.target.value)}
                className="mt-1 w-full min-h-[140px] rounded-xl border border-asgard-200 dark:border-asgard-700 bg-white dark:bg-asgard-900 px-3 py-2 text-sm"
              />
            </div>
            <div className="flex items-center gap-3">
              <Button onClick={handleSend} disabled={isSending}>
                {isSending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
                <span className="ml-2">Send Command</span>
              </Button>
              {error && <span className="text-sm text-red-500">{error}</span>}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Control Plane Status</CardTitle>
            <Button variant="secondary" onClick={fetchStatus} disabled={isLoading}>
              {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            </Button>
          </CardHeader>
          <CardContent>
            {status ? (
              <pre className="text-xs whitespace-pre-wrap text-asgard-500">{JSON.stringify(status, null, 2)}</pre>
            ) : (
              <div className="text-sm text-asgard-500">No status available.</div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Recent Control Commands</CardTitle>
        </CardHeader>
        <CardContent>
          {events.length === 0 ? (
            <div className="text-sm text-asgard-500">No commands received yet.</div>
          ) : (
            <div className="space-y-3">
              {events.map((event, idx) => (
                <div key={`${event.id ?? idx}`} className="rounded-xl border border-asgard-200 dark:border-asgard-700 p-3">
                  <div className="text-sm text-asgard-600 dark:text-asgard-300">
                    {event.commandType || 'command'}
                  </div>
                  <div className="text-xs text-asgard-500">
                    {event.targetDomain} {event.targetSystem ? `• ${event.targetSystem}` : ''} • priority {event.priority ?? 0}
                  </div>
                  <pre className="text-xs text-asgard-500 mt-2 whitespace-pre-wrap">
                    {JSON.stringify(event.parameters ?? {}, null, 2)}
                  </pre>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
