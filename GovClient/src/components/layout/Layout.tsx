/**
 * ASGARD Government Client - Main Layout
 * Navigation and structure for the application
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import {
  LayoutDashboard,
  Radio,
  Target,
  Shield,
  Settings,
  Plane,
  Bot,
  Crosshair,
  ChevronLeft,
  ChevronRight,
  LogOut,
  Bell,
  User,
} from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';

interface LayoutProps {
  children: React.ReactNode;
}

const navItems = [
  { path: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { path: '/command', icon: Radio, label: 'Command Hub' },
  { path: '/mission', icon: Target, label: 'Mission Hub' },
  { path: '/military', icon: Shield, label: 'Military Ops' },
  { path: '/security', icon: Shield, label: 'Security' },
  { path: '/valkyrie', icon: Plane, label: 'Valkyrie' },
  { path: '/hunoid', icon: Bot, label: 'Hunoid' },
  { path: '/pricilla', icon: Crosshair, label: 'Pricilla' },
  { path: '/admin', icon: Settings, label: 'Admin' },
];

export function Layout({ children }: LayoutProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);
  const location = useLocation();
  const { user, logout } = useAuthStore();

  return (
    <div className="min-h-screen bg-slate-950 flex">
      {/* Sidebar */}
      <motion.aside
        initial={false}
        animate={{ width: isCollapsed ? 72 : 256 }}
        className="fixed left-0 top-0 h-full bg-slate-900 border-r border-slate-800 z-50 flex flex-col"
      >
        {/* Logo */}
        <div className="h-16 flex items-center px-4 border-b border-slate-800">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-amber-500 to-orange-600 flex items-center justify-center">
              <Shield className="w-6 h-6 text-white" />
            </div>
            <AnimatePresence>
              {!isCollapsed && (
                <motion.span
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  className="font-bold text-white whitespace-nowrap"
                >
                  ASGARD Command
                </motion.span>
              )}
            </AnimatePresence>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 py-4 overflow-y-auto">
          <ul className="space-y-1 px-2">
            {navItems.map((item) => {
              const isActive = location.pathname === item.path;
              return (
                <li key={item.path}>
                  <Link
                    to={item.path}
                    className={`flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all ${
                      isActive
                        ? 'bg-amber-500/20 text-amber-500'
                        : 'text-slate-400 hover:bg-slate-800 hover:text-white'
                    }`}
                  >
                    <item.icon className="w-5 h-5 flex-shrink-0" />
                    <AnimatePresence>
                      {!isCollapsed && (
                        <motion.span
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          exit={{ opacity: 0 }}
                          className="text-sm font-medium whitespace-nowrap"
                        >
                          {item.label}
                        </motion.span>
                      )}
                    </AnimatePresence>
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* User section */}
        <div className="p-4 border-t border-slate-800">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-slate-700 flex items-center justify-center">
              <User className="w-5 h-5 text-slate-400" />
            </div>
            <AnimatePresence>
              {!isCollapsed && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  exit={{ opacity: 0 }}
                  className="flex-1 min-w-0"
                >
                  <p className="text-sm font-medium text-white truncate">
                    {user?.name || 'Operator'}
                  </p>
                  <p className="text-xs text-slate-500 truncate capitalize">
                    {user?.role?.replace('_', ' ') || 'Unknown'}
                  </p>
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          {!isCollapsed && (
            <button
              onClick={logout}
              className="mt-3 w-full flex items-center justify-center gap-2 py-2 text-sm text-slate-400 hover:text-red-400 transition-colors"
            >
              <LogOut className="w-4 h-4" />
              Sign Out
            </button>
          )}
        </div>

        {/* Collapse button */}
        <button
          onClick={() => setIsCollapsed(!isCollapsed)}
          className="absolute -right-3 top-20 w-6 h-6 bg-slate-800 border border-slate-700 rounded-full flex items-center justify-center text-slate-400 hover:text-white transition-colors"
        >
          {isCollapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <ChevronLeft className="w-4 h-4" />
          )}
        </button>
      </motion.aside>

      {/* Main content */}
      <div
        className="flex-1 transition-all duration-300"
        style={{ marginLeft: isCollapsed ? 72 : 256 }}
      >
        {/* Top bar */}
        <header className="h-16 bg-slate-900/50 backdrop-blur-xl border-b border-slate-800 flex items-center justify-between px-6 sticky top-0 z-40">
          <div className="flex items-center gap-4">
            <h1 className="text-lg font-semibold text-white">
              {navItems.find((item) => item.path === location.pathname)?.label || 'ASGARD'}
            </h1>
            <span className="px-2 py-0.5 bg-emerald-500/20 text-emerald-400 text-xs font-medium rounded">
              ONLINE
            </span>
          </div>

          <div className="flex items-center gap-4">
            <button className="relative p-2 text-slate-400 hover:text-white transition-colors">
              <Bell className="w-5 h-5" />
              <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
            </button>
            <div className="text-right">
              <p className="text-xs text-slate-500">Clearance Level</p>
              <p className="text-sm font-medium text-amber-500">
                {user?.clearanceLevel || 0}
              </p>
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="p-6">{children}</main>
      </div>
    </div>
  );
}
