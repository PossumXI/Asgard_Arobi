/**
 * ASGARD Government Client - Toast Notifications
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import * as Toast from '@radix-ui/react-toast';
import { motion, AnimatePresence } from 'framer-motion';
import { X, CheckCircle, AlertTriangle, Info, XCircle } from 'lucide-react';
import { create } from 'zustand';

interface ToastItem {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  description?: string;
}

interface ToastStore {
  toasts: ToastItem[];
  addToast: (toast: Omit<ToastItem, 'id'>) => void;
  removeToast: (id: string) => void;
}

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],
  addToast: (toast) =>
    set((state) => ({
      toasts: [...state.toasts, { ...toast, id: Math.random().toString(36) }],
    })),
  removeToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),
}));

export function toast(toast: Omit<ToastItem, 'id'>) {
  useToastStore.getState().addToast(toast);
}

const icons = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
};

const colors = {
  success: 'bg-emerald-500/20 border-emerald-500/50 text-emerald-400',
  error: 'bg-red-500/20 border-red-500/50 text-red-400',
  warning: 'bg-amber-500/20 border-amber-500/50 text-amber-400',
  info: 'bg-blue-500/20 border-blue-500/50 text-blue-400',
};

export function Toaster() {
  const { toasts, removeToast } = useToastStore();

  return (
    <Toast.Provider swipeDirection="right">
      <AnimatePresence>
        {toasts.map((toast) => {
          const Icon = icons[toast.type];
          return (
            <Toast.Root
              key={toast.id}
              duration={5000}
              onOpenChange={(open) => !open && removeToast(toast.id)}
              asChild
            >
              <motion.div
                initial={{ opacity: 0, x: 100 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: 100 }}
                className={`flex items-start gap-3 p-4 rounded-lg border backdrop-blur-xl ${colors[toast.type]}`}
              >
                <Icon className="w-5 h-5 flex-shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0">
                  <Toast.Title className="font-medium">{toast.title}</Toast.Title>
                  {toast.description && (
                    <Toast.Description className="text-sm opacity-80 mt-1">
                      {toast.description}
                    </Toast.Description>
                  )}
                </div>
                <Toast.Close asChild>
                  <button className="p-1 hover:bg-white/10 rounded transition-colors">
                    <X className="w-4 h-4" />
                  </button>
                </Toast.Close>
              </motion.div>
            </Toast.Root>
          );
        })}
      </AnimatePresence>

      <Toast.Viewport className="fixed bottom-4 right-4 flex flex-col gap-2 w-96 max-w-[calc(100vw-2rem)] z-[100]" />
    </Toast.Provider>
  );
}
