import { Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { useState } from 'react';
import { 
  LayoutDashboard, 
  Satellite, 
  Bot, 
  Bell, 
  Settings, 
  CreditCard,
  Activity,
  Globe,
  Shield,
  TerminalSquare,
  AlertTriangle,
  Clock,
  MapPin,
  Search,
  ChevronRight,
  Play,
  Eye,
  CheckCircle2,
  AlertCircle,
  Loader2,
  User,
  Lock,
  Moon,
  Sun,
  BellRing,
  Receipt,
  ExternalLink,
  Zap,
  Check,
  Crown,
  Sparkles
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { useAuth } from '@/providers/AuthProvider';
import { useTheme } from '@/providers/ThemeProvider';
import { cn } from '@/lib/utils';
import AdminHub from '@/pages/dashboard/AdminHub';
import CommandHub from '@/pages/dashboard/CommandHub';

// Navigation items
const baseNavItems = [
  { icon: LayoutDashboard, label: 'Overview', path: '/dashboard' },
  { icon: Satellite, label: 'Satellite Feeds', path: '/dashboard/feeds' },
  { icon: Bell, label: 'Alerts', path: '/dashboard/alerts' },
  { icon: Activity, label: 'Activity', path: '/dashboard/activity' },
  { icon: CreditCard, label: 'Subscription', path: '/dashboard/subscription' },
  { icon: Settings, label: 'Settings', path: '/dashboard/settings' },
];

const adminNavItems = [
  { icon: Shield, label: 'Admin Access', path: '/dashboard/admin' },
  { icon: TerminalSquare, label: 'Command Hub', path: '/dashboard/command' },
];

// Stats data
const stats = [
  { icon: Globe, label: 'Active Satellites', value: '156', change: '+3', trend: 'up' },
  { icon: Bot, label: 'Hunoid Units', value: '48', change: '+2', trend: 'up' },
  { icon: Shield, label: 'Threats Blocked', value: '1,247', change: '+89', trend: 'up' },
  { icon: AlertTriangle, label: 'Active Alerts', value: '3', change: '-2', trend: 'down' },
];

// Recent alerts data
const alertsData = [
  {
    id: 'ALT-001',
    type: 'tsunami',
    severity: 'critical',
    location: 'Pacific Ocean, Near Japan',
    coordinates: { lat: 35.6762, lng: 139.6503 },
    confidence: 0.94,
    timestamp: new Date(Date.now() - 120000),
    status: 'active',
    satellite: 'Silenus-47',
    description: 'Seismic activity detected indicating potential tsunami event.',
    responseTeam: 'Hunoid Squad Alpha',
  },
  {
    id: 'ALT-002',
    type: 'wildfire',
    severity: 'high',
    location: 'California, USA',
    coordinates: { lat: 36.7783, lng: -119.4179 },
    confidence: 0.87,
    timestamp: new Date(Date.now() - 900000),
    status: 'dispatched',
    satellite: 'Silenus-23',
    description: 'Thermal anomaly detected in forested region.',
    responseTeam: 'Hunoid Squad Delta',
  },
  {
    id: 'ALT-003',
    type: 'flood',
    severity: 'medium',
    location: 'Bangladesh',
    coordinates: { lat: 23.685, lng: 90.3563 },
    confidence: 0.91,
    timestamp: new Date(Date.now() - 3600000),
    status: 'resolved',
    satellite: 'Silenus-56',
    description: 'Rising water levels in delta region.',
    responseTeam: 'Hunoid Squad Gamma',
  },
  {
    id: 'ALT-004',
    type: 'earthquake',
    severity: 'high',
    location: 'Indonesia',
    coordinates: { lat: -6.2088, lng: 106.8456 },
    confidence: 0.89,
    timestamp: new Date(Date.now() - 7200000),
    status: 'monitoring',
    satellite: 'Silenus-78',
    description: 'Magnitude 5.2 earthquake detected, monitoring aftershocks.',
    responseTeam: null,
  },
  {
    id: 'ALT-005',
    type: 'storm',
    severity: 'medium',
    location: 'Caribbean Sea',
    coordinates: { lat: 18.2208, lng: -66.5901 },
    confidence: 0.82,
    timestamp: new Date(Date.now() - 14400000),
    status: 'monitoring',
    satellite: 'Silenus-34',
    description: 'Tropical depression forming, tracking trajectory.',
    responseTeam: null,
  },
];

// Activity log data
const activityData = [
  { id: 1, action: 'Hunoid H-047 deployed to search area', type: 'deployment', time: new Date(Date.now() - 300000), icon: Bot },
  { id: 2, action: 'Satellite S-089 entered eclipse mode', type: 'satellite', time: new Date(Date.now() - 720000), icon: Satellite },
  { id: 3, action: 'Alert ALT-003 auto-resolved - flood waters receding', type: 'alert', time: new Date(Date.now() - 2040000), icon: CheckCircle2 },
  { id: 4, action: 'Giru blocked 23 intrusion attempts from 185.x.x.x', type: 'security', time: new Date(Date.now() - 3600000), icon: Shield },
  { id: 5, action: 'New firmware v2.4.1 deployed to Silenus fleet', type: 'update', time: new Date(Date.now() - 7200000), icon: Zap },
  { id: 6, action: 'Hunoid H-102 completed medical supply delivery', type: 'mission', time: new Date(Date.now() - 10800000), icon: CheckCircle2 },
  { id: 7, action: 'Alert ALT-006 generated - vessel distress signal', type: 'alert', time: new Date(Date.now() - 14400000), icon: AlertTriangle },
  { id: 8, action: 'Satellite S-056 orbital adjustment completed', type: 'satellite', time: new Date(Date.now() - 18000000), icon: Satellite },
  { id: 9, action: 'Nysus core processed 1.2M events this hour', type: 'system', time: new Date(Date.now() - 21600000), icon: Activity },
  { id: 10, action: 'Monthly audit completed - all systems nominal', type: 'audit', time: new Date(Date.now() - 86400000), icon: CheckCircle2 },
];

// Satellite feeds data
const satelliteFeeds = [
  { id: 'SAT-001', name: 'Silenus-47', location: 'Pacific Ocean', status: 'live', viewers: 12453, coverage: 'Maritime' },
  { id: 'SAT-002', name: 'Silenus-23', location: 'North America', status: 'live', viewers: 8721, coverage: 'Continental' },
  { id: 'SAT-003', name: 'Silenus-56', location: 'South America', status: 'live', viewers: 6743, coverage: 'Amazon Basin' },
  { id: 'SAT-004', name: 'Silenus-12', location: 'Arctic Circle', status: 'live', viewers: 2156, coverage: 'Polar' },
  { id: 'SAT-005', name: 'Silenus-78', location: 'Southeast Asia', status: 'live', viewers: 9234, coverage: 'Regional' },
  { id: 'SAT-006', name: 'Silenus-34', location: 'Africa', status: 'maintenance', viewers: 0, coverage: 'Continental' },
];

// Subscription plans
const subscriptionPlans = [
  {
    id: 'free',
    name: 'Free',
    price: 0,
    period: 'forever',
    features: ['Access to public streams', 'Basic alert notifications', '5 saved locations', 'Community support'],
    current: false,
  },
  {
    id: 'pro',
    name: 'Pro',
    price: 29,
    period: 'month',
    features: ['All Free features', 'HD stream quality', 'Unlimited saved locations', 'Custom alert criteria', 'API access (1000 calls/day)', 'Priority support'],
    current: true,
    popular: true,
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    price: 199,
    period: 'month',
    features: ['All Pro features', '4K stream quality', 'Dedicated satellite access', 'Custom integrations', 'Unlimited API calls', '24/7 dedicated support', 'SLA guarantee'],
    current: false,
  },
];

// Helper functions
function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  if (diff < 60000) return 'Just now';
  if (diff < 3600000) return `${Math.floor(diff / 60000)} min ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours ago`;
  return `${Math.floor(diff / 86400000)} days ago`;
}

function getSeverityColor(severity: string): string {
  switch (severity) {
    case 'critical': return 'text-red-500 bg-red-500/10';
    case 'high': return 'text-orange-500 bg-orange-500/10';
    case 'medium': return 'text-yellow-500 bg-yellow-500/10';
    default: return 'text-blue-500 bg-blue-500/10';
  }
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'active': return 'bg-red-500';
    case 'dispatched': return 'bg-yellow-500';
    case 'resolved': return 'bg-green-500';
    case 'monitoring': return 'bg-blue-500';
    default: return 'bg-gray-500';
  }
}

// Dashboard Overview Component
function DashboardOverview() {
  return (
    <div className="space-y-8">
      {/* Stats Grid */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
          >
            <Card className="hover:shadow-lg transition-shadow">
              <CardContent className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
                    <stat.icon className="w-6 h-6 text-primary" />
                  </div>
                  <span className={cn(
                    'text-sm font-medium px-2 py-0.5 rounded-full',
                    stat.trend === 'up' ? 'text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-900/30' : 'text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/30'
                  )}>
                    {stat.change}
                  </span>
                </div>
                <div className="text-2xl font-bold text-asgard-900 dark:text-white mb-1">
                  {stat.value}
                </div>
                <div className="text-sm text-asgard-500 dark:text-asgard-400">
                  {stat.label}
                </div>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      {/* Main Content Grid */}
      <div className="grid lg:grid-cols-3 gap-6">
        {/* Recent Alerts */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="lg:col-span-2"
        >
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-warning" />
                Recent Alerts
              </CardTitle>
              <CardDescription>Real-time detections from Silenus network</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {alertsData.slice(0, 3).map((alert) => (
                  <div
                    key={alert.id}
                    className="flex items-center gap-4 p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50 hover:bg-asgard-100 dark:hover:bg-asgard-800 transition-colors cursor-pointer"
                  >
                    <div className={cn('w-3 h-3 rounded-full flex-shrink-0', getStatusColor(alert.status))} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium text-asgard-900 dark:text-white capitalize">
                          {alert.type}
                        </span>
                        <span className={cn('px-2 py-0.5 rounded-full text-xs font-medium', getSeverityColor(alert.severity))}>
                          {alert.severity}
                        </span>
                        <span className="px-2 py-0.5 rounded-full bg-primary/10 text-primary text-xs">
                          {Math.round(alert.confidence * 100)}% confidence
                        </span>
                      </div>
                      <div className="flex items-center gap-4 text-sm text-asgard-500 dark:text-asgard-400">
                        <span className="flex items-center gap-1">
                          <MapPin className="w-3 h-3" />
                          {alert.location}
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {formatRelativeTime(alert.timestamp)}
                        </span>
                      </div>
                    </div>
                    <ChevronRight className="w-5 h-5 text-asgard-400" />
                  </div>
                ))}
              </div>
              <Link to="/dashboard/alerts" className="block mt-4">
                <Button variant="outline" className="w-full">
                  View All Alerts
                </Button>
              </Link>
            </CardContent>
          </Card>
        </motion.div>

        {/* Activity Feed */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
        >
          <Card className="h-full">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="w-5 h-5 text-primary" />
                Activity Feed
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {activityData.slice(0, 4).map((item) => (
                  <div key={item.id} className="flex gap-3">
                    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                      <item.icon className="w-4 h-4 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-asgard-700 dark:text-asgard-300 line-clamp-2">
                        {item.action}
                      </p>
                      <p className="text-xs text-asgard-400 mt-1">{formatRelativeTime(item.time)}</p>
                    </div>
                  </div>
                ))}
              </div>
              <Link to="/dashboard/activity" className="block mt-4">
                <Button variant="ghost" className="w-full text-sm">
                  View All Activity
                </Button>
              </Link>
            </CardContent>
          </Card>
        </motion.div>
      </div>

      {/* System Status */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.6 }}
      >
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CheckCircle2 className="w-5 h-5 text-green-500" />
              System Status
            </CardTitle>
            <CardDescription>All systems operational</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
              {[
                { name: 'Silenus Network', status: 'operational', uptime: '99.99%' },
                { name: 'Hunoid Fleet', status: 'operational', uptime: '99.97%' },
                { name: 'Nysus Core', status: 'operational', uptime: '100%' },
                { name: 'Giru Security', status: 'operational', uptime: '99.99%' },
              ].map((system) => (
                <div
                  key={system.name}
                  className="flex items-center gap-3 p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50"
                >
                  <span className="w-2.5 h-2.5 rounded-full bg-green-500 animate-pulse" />
                  <div className="flex-1">
                    <span className="text-sm font-medium text-asgard-700 dark:text-asgard-300 block">
                      {system.name}
                    </span>
                    <span className="text-xs text-asgard-500">
                      {system.uptime} uptime
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}

// Dashboard Feeds Component
function DashboardFeeds() {
  const [selectedFeed, setSelectedFeed] = useState<string | null>(null);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Satellite className="w-5 h-5 text-primary" />
            Satellite Feeds
          </CardTitle>
          <CardDescription>Live streams from Silenus constellation</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {satelliteFeeds.map((feed, index) => (
              <motion.div
                key={feed.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.05 }}
                className={cn(
                  'relative rounded-xl overflow-hidden cursor-pointer group',
                  'border-2 transition-all duration-300',
                  selectedFeed === feed.id 
                    ? 'border-primary ring-2 ring-primary/20' 
                    : 'border-transparent hover:border-asgard-200 dark:hover:border-asgard-700'
                )}
                onClick={() => setSelectedFeed(feed.id === selectedFeed ? null : feed.id)}
              >
                {/* Feed Preview */}
                <div className="aspect-video bg-gradient-to-br from-asgard-100 to-asgard-200 dark:from-asgard-800 dark:to-asgard-900 relative">
                  {/* Simulated satellite view */}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="relative">
                      <Globe className="w-16 h-16 text-asgard-400 dark:text-asgard-600" />
                      <div className="absolute inset-0 flex items-center justify-center">
                        {feed.status === 'live' && (
                          <div className="w-3 h-3 rounded-full bg-green-500 animate-ping" />
                        )}
                      </div>
                    </div>
                  </div>
                  
                  {/* Status Badge */}
                  <div className="absolute top-3 left-3">
                    {feed.status === 'live' ? (
                      <span className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-red-500 text-white text-xs font-bold">
                        <span className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
                        LIVE
                      </span>
                    ) : (
                      <span className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-yellow-500 text-white text-xs font-bold">
                        MAINTENANCE
                      </span>
                    )}
                  </div>

                  {/* Viewers */}
                  {feed.status === 'live' && (
                    <div className="absolute top-3 right-3">
                      <span className="flex items-center gap-1 px-2 py-1 rounded-full bg-black/50 text-white text-xs">
                        <Eye className="w-3 h-3" />
                        {feed.viewers.toLocaleString()}
                      </span>
                    </div>
                  )}

                  {/* Play Overlay */}
                  <div className="absolute inset-0 flex items-center justify-center bg-black/0 group-hover:bg-black/30 transition-colors">
                    <div className="w-12 h-12 rounded-full bg-white/0 group-hover:bg-white/90 flex items-center justify-center transition-all transform scale-0 group-hover:scale-100">
                      <Play className="w-5 h-5 text-asgard-900 ml-1" fill="currentColor" />
                    </div>
                  </div>
                </div>

                {/* Feed Info */}
                <div className="p-4 bg-white dark:bg-asgard-800">
                  <h3 className="font-semibold text-asgard-900 dark:text-white mb-1">
                    {feed.name}
                  </h3>
                  <div className="flex items-center justify-between text-sm text-asgard-500 dark:text-asgard-400">
                    <span className="flex items-center gap-1">
                      <MapPin className="w-3 h-3" />
                      {feed.location}
                    </span>
                    <span>{feed.coverage}</span>
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Dashboard Alerts Component
function DashboardAlerts() {
  const [filter, setFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');

  const filteredAlerts = alertsData.filter(alert => {
    if (filter !== 'all' && alert.status !== filter) return false;
    if (searchQuery && !alert.location.toLowerCase().includes(searchQuery.toLowerCase()) && 
        !alert.type.toLowerCase().includes(searchQuery.toLowerCase())) return false;
    return true;
  });

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-warning" />
                All Alerts
              </CardTitle>
              <CardDescription>Detection history and active alerts</CardDescription>
            </div>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-asgard-400" />
                <input
                  type="text"
                  placeholder="Search alerts..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full sm:w-64 h-10 pl-10 pr-4 rounded-xl bg-asgard-50 dark:bg-asgard-800 border border-asgard-200 dark:border-asgard-700 text-asgard-900 dark:text-white placeholder-asgard-400 focus:outline-none focus:ring-2 focus:ring-primary/50"
                />
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {/* Filter Tabs */}
          <div className="flex items-center gap-2 mb-6 overflow-x-auto pb-2">
            {['all', 'active', 'dispatched', 'monitoring', 'resolved'].map((status) => (
              <button
                key={status}
                onClick={() => setFilter(status)}
                className={cn(
                  'px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors',
                  filter === status
                    ? 'bg-primary text-white'
                    : 'bg-asgard-100 dark:bg-asgard-800 text-asgard-600 dark:text-asgard-400 hover:bg-asgard-200 dark:hover:bg-asgard-700'
                )}
              >
                {status.charAt(0).toUpperCase() + status.slice(1)}
              </button>
            ))}
          </div>

          {/* Alerts List */}
          <div className="space-y-4">
            <AnimatePresence mode="popLayout">
              {filteredAlerts.map((alert, index) => (
                <motion.div
                  key={alert.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  transition={{ delay: index * 0.05 }}
                  className="p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50 hover:bg-asgard-100 dark:hover:bg-asgard-800 transition-colors"
                >
                  <div className="flex items-start gap-4">
                    <div className={cn('w-3 h-3 rounded-full mt-1.5 flex-shrink-0', getStatusColor(alert.status))} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center flex-wrap gap-2 mb-2">
                        <span className="font-semibold text-asgard-900 dark:text-white">
                          {alert.id}
                        </span>
                        <span className="font-medium text-asgard-900 dark:text-white capitalize">
                          {alert.type}
                        </span>
                        <span className={cn('px-2 py-0.5 rounded-full text-xs font-medium', getSeverityColor(alert.severity))}>
                          {alert.severity}
                        </span>
                        <span className="px-2 py-0.5 rounded-full bg-primary/10 text-primary text-xs">
                          {Math.round(alert.confidence * 100)}% confidence
                        </span>
                      </div>
                      <p className="text-sm text-asgard-600 dark:text-asgard-400 mb-2">
                        {alert.description}
                      </p>
                      <div className="flex flex-wrap items-center gap-4 text-xs text-asgard-500">
                        <span className="flex items-center gap-1">
                          <MapPin className="w-3 h-3" />
                          {alert.location}
                        </span>
                        <span className="flex items-center gap-1">
                          <Satellite className="w-3 h-3" />
                          {alert.satellite}
                        </span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {formatRelativeTime(alert.timestamp)}
                        </span>
                        {alert.responseTeam && (
                          <span className="flex items-center gap-1">
                            <Bot className="w-3 h-3" />
                            {alert.responseTeam}
                          </span>
                        )}
                      </div>
                    </div>
                    <Button variant="ghost" size="sm">
                      View Details
                    </Button>
                  </div>
                </motion.div>
              ))}
            </AnimatePresence>

            {filteredAlerts.length === 0 && (
              <div className="text-center py-12">
                <AlertCircle className="w-12 h-12 text-asgard-300 dark:text-asgard-600 mx-auto mb-4" />
                <p className="text-asgard-500 dark:text-asgard-400">No alerts found matching your criteria</p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Dashboard Activity Component
function DashboardActivity() {
  const [filter, setFilter] = useState('all');

  const typeColors: Record<string, string> = {
    deployment: 'bg-blue-500',
    satellite: 'bg-purple-500',
    alert: 'bg-yellow-500',
    security: 'bg-red-500',
    update: 'bg-green-500',
    mission: 'bg-emerald-500',
    system: 'bg-gray-500',
    audit: 'bg-indigo-500',
  };

  const filteredActivity = filter === 'all' 
    ? activityData 
    : activityData.filter(a => a.type === filter);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-primary" />
            Activity Log
          </CardTitle>
          <CardDescription>Complete system activity history</CardDescription>
        </CardHeader>
        <CardContent>
          {/* Filter Tabs */}
          <div className="flex items-center gap-2 mb-6 overflow-x-auto pb-2">
            {['all', 'deployment', 'satellite', 'alert', 'security', 'mission'].map((type) => (
              <button
                key={type}
                onClick={() => setFilter(type)}
                className={cn(
                  'px-4 py-2 rounded-full text-sm font-medium whitespace-nowrap transition-colors',
                  filter === type
                    ? 'bg-primary text-white'
                    : 'bg-asgard-100 dark:bg-asgard-800 text-asgard-600 dark:text-asgard-400 hover:bg-asgard-200 dark:hover:bg-asgard-700'
                )}
              >
                {type.charAt(0).toUpperCase() + type.slice(1)}
              </button>
            ))}
          </div>

          {/* Activity Timeline */}
          <div className="relative">
            <div className="absolute left-4 top-0 bottom-0 w-px bg-asgard-200 dark:bg-asgard-700" />
            
            <div className="space-y-6">
              {filteredActivity.map((item, index) => (
                <motion.div
                  key={item.id}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.05 }}
                  className="relative pl-10"
                >
                  <div className={cn(
                    'absolute left-0 w-8 h-8 rounded-full flex items-center justify-center',
                    typeColors[item.type] || 'bg-gray-500'
                  )}>
                    <item.icon className="w-4 h-4 text-white" />
                  </div>
                  <div className="p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50">
                    <p className="text-sm text-asgard-900 dark:text-white mb-1">
                      {item.action}
                    </p>
                    <div className="flex items-center gap-3 text-xs text-asgard-500">
                      <span className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {formatRelativeTime(item.time)}
                      </span>
                      <span className={cn(
                        'px-2 py-0.5 rounded-full capitalize',
                        'bg-asgard-200 dark:bg-asgard-700 text-asgard-600 dark:text-asgard-300'
                      )}>
                        {item.type}
                      </span>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Dashboard Subscription Component
function DashboardSubscription() {
  // Auth context available for subscription tier info
  useAuth();

  return (
    <div className="space-y-6">
      {/* Current Plan */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Crown className="w-5 h-5 text-yellow-500" />
            Current Plan
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between p-4 rounded-xl bg-gradient-to-r from-primary to-primary-700 text-white">
            <div>
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xl font-bold">Pro Plan</span>
                <Sparkles className="w-5 h-5" />
              </div>
              <p className="text-primary-100 text-sm">$29/month • Renews Jan 20, 2026</p>
            </div>
            <Button variant="secondary" size="sm">
              Manage
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* All Plans */}
      <Card>
        <CardHeader>
          <CardTitle>Available Plans</CardTitle>
          <CardDescription>Choose the plan that fits your needs</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid md:grid-cols-3 gap-6">
            {subscriptionPlans.map((plan, index) => (
              <motion.div
                key={plan.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.1 }}
                className={cn(
                  'relative p-6 rounded-2xl border-2 transition-all',
                  plan.current
                    ? 'border-primary bg-primary/5'
                    : 'border-asgard-200 dark:border-asgard-700 hover:border-asgard-300 dark:hover:border-asgard-600'
                )}
              >
                {plan.popular && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                    <span className="px-3 py-1 rounded-full bg-primary text-white text-xs font-medium">
                      Most Popular
                    </span>
                  </div>
                )}
                
                <div className="text-center mb-6">
                  <h3 className="text-lg font-semibold text-asgard-900 dark:text-white mb-2">
                    {plan.name}
                  </h3>
                  <div className="flex items-baseline justify-center gap-1">
                    <span className="text-3xl font-bold text-asgard-900 dark:text-white">
                      ${plan.price}
                    </span>
                    {plan.price > 0 && (
                      <span className="text-asgard-500">/{plan.period}</span>
                    )}
                  </div>
                </div>

                <ul className="space-y-3 mb-6">
                  {plan.features.map((feature, i) => (
                    <li key={i} className="flex items-start gap-2 text-sm text-asgard-600 dark:text-asgard-400">
                      <Check className="w-4 h-4 text-green-500 flex-shrink-0 mt-0.5" />
                      {feature}
                    </li>
                  ))}
                </ul>

                <Button 
                  variant={plan.current ? 'outline' : 'primary'} 
                  className="w-full"
                  disabled={plan.current}
                >
                  {plan.current ? 'Current Plan' : plan.price === 0 ? 'Downgrade' : 'Upgrade'}
                </Button>
              </motion.div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Billing History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Receipt className="w-5 h-5 text-asgard-500" />
            Billing History
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[
              { date: 'Dec 20, 2025', amount: '$29.00', status: 'Paid', invoice: 'INV-2025-012' },
              { date: 'Nov 20, 2025', amount: '$29.00', status: 'Paid', invoice: 'INV-2025-011' },
              { date: 'Oct 20, 2025', amount: '$29.00', status: 'Paid', invoice: 'INV-2025-010' },
            ].map((item, index) => (
              <div
                key={index}
                className="flex items-center justify-between p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50"
              >
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
                    <CheckCircle2 className="w-5 h-5 text-green-500" />
                  </div>
                  <div>
                    <p className="font-medium text-asgard-900 dark:text-white">{item.amount}</p>
                    <p className="text-sm text-asgard-500">{item.date}</p>
                  </div>
                </div>
                <Button variant="ghost" size="sm" className="text-primary">
                  <ExternalLink className="w-4 h-4 mr-2" />
                  Download
                </Button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Dashboard Settings Component
function DashboardSettings() {
  const { user } = useAuth();
  const { theme, setTheme } = useTheme();
  const [notifications, setNotifications] = useState({
    email: true,
    push: true,
    alerts: true,
    updates: false,
  });

  return (
    <div className="space-y-6">
      {/* Profile Settings */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="w-5 h-5 text-primary" />
            Profile Settings
          </CardTitle>
          <CardDescription>Manage your account information</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4 p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50">
            <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <User className="w-8 h-8 text-primary" />
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-asgard-900 dark:text-white">
                {user?.fullName || 'User'}
              </h3>
              <p className="text-sm text-asgard-500">{user?.email}</p>
            </div>
            <Button variant="outline" size="sm">
              Change Photo
            </Button>
          </div>

          <div className="grid sm:grid-cols-2 gap-4">
            <Input
              label="Full Name"
              defaultValue={user?.fullName || ''}
              placeholder="Enter your name"
            />
            <Input
              label="Email"
              type="email"
              defaultValue={user?.email || ''}
              placeholder="Enter your email"
            />
          </div>

          <Button>Save Changes</Button>
        </CardContent>
      </Card>

      {/* Appearance */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            {theme === 'dark' ? <Moon className="w-5 h-5" /> : <Sun className="w-5 h-5" />}
            Appearance
          </CardTitle>
          <CardDescription>Customize how ASGARD looks on your device</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            {(
              [
                { value: 'light', label: 'Light', icon: Sun },
                { value: 'dark', label: 'Dark', icon: Moon },
                { value: 'system', label: 'System', icon: Settings },
              ] as const
            ).map((option) => (
              <button
                key={option.value}
                onClick={() => setTheme(option.value)}
                className={cn(
                  'flex-1 p-4 rounded-xl border-2 transition-all',
                  theme === option.value
                    ? 'border-primary bg-primary/5'
                    : 'border-asgard-200 dark:border-asgard-700 hover:border-asgard-300 dark:hover:border-asgard-600'
                )}
              >
                <option.icon className={cn(
                  'w-6 h-6 mx-auto mb-2',
                  theme === option.value ? 'text-primary' : 'text-asgard-500'
                )} />
                <span className={cn(
                  'text-sm font-medium',
                  theme === option.value ? 'text-primary' : 'text-asgard-600 dark:text-asgard-400'
                )}>
                  {option.label}
                </span>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Notifications */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BellRing className="w-5 h-5 text-primary" />
            Notifications
          </CardTitle>
          <CardDescription>Configure how you receive updates</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[
              { key: 'email', label: 'Email Notifications', description: 'Receive updates via email' },
              { key: 'push', label: 'Push Notifications', description: 'Browser push notifications' },
              { key: 'alerts', label: 'Alert Notifications', description: 'Critical alert notifications' },
              { key: 'updates', label: 'Product Updates', description: 'News and feature announcements' },
            ].map((item) => (
              <div
                key={item.key}
                className="flex items-center justify-between p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50"
              >
                <div>
                  <p className="font-medium text-asgard-900 dark:text-white">{item.label}</p>
                  <p className="text-sm text-asgard-500">{item.description}</p>
                </div>
                <button
                  onClick={() => setNotifications(prev => ({ ...prev, [item.key]: !prev[item.key as keyof typeof prev] }))}
                  className={cn(
                    'w-12 h-7 rounded-full transition-colors relative',
                    notifications[item.key as keyof typeof notifications]
                      ? 'bg-primary'
                      : 'bg-asgard-300 dark:bg-asgard-600'
                  )}
                >
                  <div className={cn(
                    'absolute top-1 w-5 h-5 rounded-full bg-white transition-transform',
                    notifications[item.key as keyof typeof notifications]
                      ? 'translate-x-6'
                      : 'translate-x-1'
                  )} />
                </button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Security */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Lock className="w-5 h-5 text-primary" />
            Security
          </CardTitle>
          <CardDescription>Protect your account</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50">
            <div>
              <p className="font-medium text-asgard-900 dark:text-white">Password</p>
              <p className="text-sm text-asgard-500">Last changed 30 days ago</p>
            </div>
            <Button variant="outline" size="sm">
              Change Password
            </Button>
          </div>

          <div className="flex items-center justify-between p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50">
            <div>
              <p className="font-medium text-asgard-900 dark:text-white">Two-Factor Authentication</p>
              <p className="text-sm text-asgard-500">Add an extra layer of security</p>
            </div>
            <Button variant="outline" size="sm">
              Enable 2FA
            </Button>
          </div>

          <div className="flex items-center justify-between p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50">
            <div>
              <p className="font-medium text-asgard-900 dark:text-white">Active Sessions</p>
              <p className="text-sm text-asgard-500">Manage devices logged into your account</p>
            </div>
            <Button variant="outline" size="sm">
              View Sessions
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-red-200 dark:border-red-900">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-red-500">
            <AlertTriangle className="w-5 h-5" />
            Danger Zone
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between p-4 rounded-xl bg-red-50 dark:bg-red-900/20">
            <div>
              <p className="font-medium text-red-600 dark:text-red-400">Delete Account</p>
              <p className="text-sm text-red-500/80">Permanently delete your account and all data</p>
            </div>
            <Button variant="outline" size="sm" className="border-red-500 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/30">
              Delete Account
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Main Dashboard Component
export default function Dashboard() {
  const location = useLocation();
  const { user, isAuthenticated, isLoading } = useAuth();
  const isAdmin = Boolean(user?.isGovernment || user?.subscriptionTier === 'commander');
  const navItems = isAdmin ? [...baseNavItems, ...adminNavItems] : baseNavItems;

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/signin" replace />;
  }

  return (
    <div className="min-h-screen pt-20">
      <div className="container-wide py-8">
        <div className="flex gap-8">
          {/* Sidebar */}
          <aside className="hidden lg:block w-64 flex-shrink-0">
            <div className="sticky top-24">
              <div className="mb-6">
                <h2 className="text-sm font-semibold text-asgard-400 uppercase tracking-wider mb-2">
                  Dashboard
                </h2>
                <p className="text-sm text-asgard-500">
                  Welcome back, {user?.fullName?.split(' ')[0] || 'User'}
                </p>
              </div>
              <nav className="space-y-1">
                {navItems.map((item) => {
                  const isActive = location.pathname === item.path || 
                    (item.path === '/dashboard' && location.pathname === '/dashboard');
                  return (
                    <Link
                      key={item.path}
                      to={item.path}
                      className={cn(
                        'flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-medium transition-all',
                        isActive
                          ? 'bg-primary text-white shadow-lg shadow-primary/25'
                          : 'text-asgard-600 hover:bg-asgard-100 dark:text-asgard-400 dark:hover:bg-asgard-800'
                      )}
                    >
                      <item.icon className="w-5 h-5" />
                      {item.label}
                    </Link>
                  );
                })}
              </nav>
            </div>
          </aside>

          {/* Main Content */}
          <main className="flex-1 min-w-0">
            <Routes>
              <Route index element={<DashboardOverview />} />
              <Route path="feeds" element={<DashboardFeeds />} />
              <Route path="alerts" element={<DashboardAlerts />} />
              <Route path="activity" element={<DashboardActivity />} />
              <Route path="subscription" element={<DashboardSubscription />} />
              <Route path="settings" element={<DashboardSettings />} />
              {isAdmin && <Route path="admin" element={<AdminHub />} />}
              {isAdmin && <Route path="command" element={<CommandHub />} />}
            </Routes>
          </main>
        </div>
      </div>
    </div>
  );
}
