import { Outlet, Link, useLocation } from 'react-router-dom';
import { motion } from 'framer-motion';
import { 
  Eye, 
  Globe, 
  Shield, 
  Rocket, 
  Settings, 
  Bell,
  User,
  Menu,
  X
} from 'lucide-react';
import { useState } from 'react';
import { cn } from '@/lib/utils';

const navItems = [
  { href: '/', icon: Eye, label: 'All Feeds' },
  { href: '/civilian', icon: Globe, label: 'Civilian', color: 'text-civilian' },
  { href: '/military', icon: Shield, label: 'Military', color: 'text-military' },
  { href: '/interstellar', icon: Rocket, label: 'Interstellar', color: 'text-interstellar' },
];

export default function Layout() {
  const location = useLocation();
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);

  return (
    <div className="min-h-screen bg-hub-dark flex">
      {/* Sidebar */}
      <motion.aside
        initial={false}
        animate={{ width: isSidebarOpen ? 240 : 72 }}
        className="fixed left-0 top-0 bottom-0 z-40 bg-hub-darker border-r border-hub-border flex flex-col"
      >
        {/* Logo */}
        <div className="h-16 flex items-center justify-between px-4 border-b border-hub-border">
          <Link to="/" className="flex items-center gap-3">
            <svg
              viewBox="0 0 36 36"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              className="w-8 h-8 text-hub-accent"
            >
              <circle cx="18" cy="18" r="16" stroke="currentColor" strokeWidth="2" />
              <path d="M18 6L26 22H10L18 6Z" fill="currentColor" />
              <circle cx="18" cy="28" r="3" fill="currentColor" />
            </svg>
            {isSidebarOpen && (
              <motion.span
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="font-semibold text-white"
              >
                HUBS
              </motion.span>
            )}
          </Link>
          <button
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="p-2 rounded-lg hover:bg-hub-surface transition-colors"
          >
            {isSidebarOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-3 space-y-1">
          {navItems.map((item) => {
            const isActive = item.href === '/' 
              ? location.pathname === '/'
              : location.pathname.startsWith(item.href);
            
            return (
              <Link
                key={item.href}
                to={item.href}
                className={cn(
                  'flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200',
                  isActive
                    ? 'bg-hub-accent/10 text-hub-accent'
                    : 'text-gray-400 hover:text-white hover:bg-hub-surface'
                )}
              >
                <item.icon className={cn('w-5 h-5 flex-shrink-0', item.color)} />
                {isSidebarOpen && (
                  <motion.span
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="text-sm font-medium"
                  >
                    {item.label}
                  </motion.span>
                )}
              </Link>
            );
          })}
        </nav>

        {/* Bottom Actions */}
        <div className="p-3 border-t border-hub-border space-y-1">
          <Link
            to="/settings"
            className="flex items-center gap-3 px-3 py-2.5 rounded-xl text-gray-400 hover:text-white hover:bg-hub-surface transition-colors"
          >
            <Settings className="w-5 h-5" />
            {isSidebarOpen && <span className="text-sm font-medium">Settings</span>}
          </Link>
        </div>
      </motion.aside>

      {/* Main Content */}
      <main
        className={cn(
          'flex-1 transition-all duration-300',
          isSidebarOpen ? 'ml-60' : 'ml-[72px]'
        )}
      >
        {/* Top Bar */}
        <header className="h-16 bg-hub-darker/80 backdrop-blur-xl border-b border-hub-border sticky top-0 z-30 flex items-center justify-between px-6">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="status-indicator status-live" />
              <span className="text-sm text-gray-400">
                <span className="text-white font-medium">247</span> active streams
              </span>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <button type="button" aria-label="Notifications" className="relative p-2 rounded-lg hover:bg-hub-surface transition-colors">
              <Bell className="w-5 h-5 text-gray-400" />
              <span className="absolute top-1 right-1 w-2 h-2 rounded-full bg-red-500" />
            </button>
            <button type="button" aria-label="User profile" className="p-2 rounded-lg hover:bg-hub-surface transition-colors">
              <User className="w-5 h-5 text-gray-400" />
            </button>
          </div>
        </header>

        {/* Page Content */}
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
