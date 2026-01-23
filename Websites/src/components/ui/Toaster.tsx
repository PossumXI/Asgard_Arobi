import { useEffect, useState, createContext, useContext, ReactNode, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';
import { cn } from '@/lib/utils';

type ToastType = 'success' | 'error' | 'info' | 'warning';

interface Toast {
  id: string;
  type: ToastType;
  title: string;
  description?: string;
  duration?: number;
}

interface ToastContextValue {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, 'id'>) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const addToast = useCallback((toast: Omit<Toast, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9);
    const newToast = { ...toast, id };
    
    setToasts((prev) => [...prev, newToast]);

    const duration = toast.duration ?? 5000;
    if (duration > 0) {
      setTimeout(() => removeToast(id), duration);
    }
  }, [removeToast]);

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast }}>
      {children}
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }

  return {
    toast: context.addToast,
    dismiss: context.removeToast,
    success: (title: string, description?: string) =>
      context.addToast({ type: 'success', title, description }),
    error: (title: string, description?: string) =>
      context.addToast({ type: 'error', title, description }),
    info: (title: string, description?: string) =>
      context.addToast({ type: 'info', title, description }),
    warning: (title: string, description?: string) =>
      context.addToast({ type: 'warning', title, description }),
  };
}

const icons: Record<ToastType, typeof CheckCircle> = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
  warning: AlertTriangle,
};

const styles: Record<ToastType, string> = {
  success: 'border-success/20 bg-success/5 text-success',
  error: 'border-danger/20 bg-danger/5 text-danger',
  info: 'border-primary/20 bg-primary/5 text-primary',
  warning: 'border-warning/20 bg-warning/5 text-warning',
};

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const Icon = icons[toast.type];

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 50, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className={cn(
        'pointer-events-auto flex w-full max-w-md items-start gap-3 rounded-xl border p-4 shadow-medium',
        'bg-white dark:bg-asgard-900',
        styles[toast.type]
      )}
    >
      <Icon className="h-5 w-5 flex-shrink-0 mt-0.5" />
      <div className="flex-1">
        <p className="font-medium text-asgard-900 dark:text-white">{toast.title}</p>
        {toast.description && (
          <p className="mt-1 text-sm text-asgard-500 dark:text-asgard-400">
            {toast.description}
          </p>
        )}
      </div>
      <button
        onClick={onDismiss}
        className="flex-shrink-0 rounded-lg p-1 hover:bg-asgard-100 dark:hover:bg-asgard-800 transition-colors"
      >
        <X className="h-4 w-4 text-asgard-400" />
      </button>
    </motion.div>
  );
}

export function Toaster() {
  const [mounted, setMounted] = useState(false);
  const context = useContext(ToastContext);

  useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted || !context) return null;

  return (
    <div className="fixed bottom-0 right-0 z-50 flex flex-col gap-3 p-4 sm:p-6 pointer-events-none">
      <AnimatePresence mode="popLayout">
        {context.toasts.map((toast) => (
          <ToastItem
            key={toast.id}
            toast={toast}
            onDismiss={() => context.removeToast(toast.id)}
          />
        ))}
      </AnimatePresence>
    </div>
  );
}
