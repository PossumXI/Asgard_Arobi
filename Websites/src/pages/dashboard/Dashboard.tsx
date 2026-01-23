import { Routes, Route, Link, useLocation, Navigate } from 'react-router-dom';
import { motion } from 'framer-motion';
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
  AlertTriangle,
  TrendingUp,
  Users,
  Clock,
  MapPin
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { useAuth } from '@/providers/AuthProvider';
import { cn } from '@/lib/utils';

const navItems = [
  { icon: LayoutDashboard, label: 'Overview', path: '/dashboard' },
  { icon: Satellite, label: 'Satellite Feeds', path: '/dashboard/feeds' },
  { icon: Bell, label: 'Alerts', path: '/dashboard/alerts' },
  { icon: Activity, label: 'Activity', path: '/dashboard/activity' },
  { icon: CreditCard, label: 'Subscription', path: '/dashboard/subscription' },
  { icon: Settings, label: 'Settings', path: '/dashboard/settings' },
];

const stats = [
  { icon: Globe, label: 'Active Satellites', value: '156', change: '+3' },
  { icon: Bot, label: 'Hunoid Units', value: '48', change: '+2' },
  { icon: Shield, label: 'Threats Blocked', value: '1,247', change: '+89' },
  { icon: AlertTriangle, label: 'Active Alerts', value: '3', change: '-2' },
];

const recentAlerts = [
  {
    id: '1',
    type: 'tsunami',
    location: 'Pacific Ocean, Near Japan',
    confidence: 0.94,
    timestamp: '2 minutes ago',
    status: 'active',
  },
  {
    id: '2',
    type: 'wildfire',
    location: 'California, USA',
    confidence: 0.87,
    timestamp: '15 minutes ago',
    status: 'dispatched',
  },
  {
    id: '3',
    type: 'flood',
    location: 'Bangladesh',
    confidence: 0.91,
    timestamp: '1 hour ago',
    status: 'resolved',
  },
];

const activityFeed = [
  { action: 'Hunoid H-047 deployed to search area', time: '5 min ago' },
  { action: 'Satellite S-089 entered eclipse mode', time: '12 min ago' },
  { action: 'Alert #1247 auto-resolved', time: '34 min ago' },
  { action: 'New firmware v2.4.1 deployed to Silenus fleet', time: '2 hrs ago' },
];

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
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
                    <stat.icon className="w-6 h-6 text-primary" />
                  </div>
                  <span className={cn(
                    'text-sm font-medium',
                    stat.change.startsWith('+') ? 'text-success' : 'text-danger'
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
                {recentAlerts.map((alert) => (
                  <div
                    key={alert.id}
                    className="flex items-center gap-4 p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50"
                  >
                    <div className={cn(
                      'w-3 h-3 rounded-full',
                      alert.status === 'active' ? 'bg-danger animate-pulse' :
                      alert.status === 'dispatched' ? 'bg-warning' : 'bg-success'
                    )} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium text-asgard-900 dark:text-white capitalize">
                          {alert.type}
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
                          {alert.timestamp}
                        </span>
                      </div>
                    </div>
                    <Button variant="ghost" size="sm">View</Button>
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
                {activityFeed.map((item, index) => (
                  <div key={index} className="flex gap-3">
                    <div className="w-2 h-2 mt-2 rounded-full bg-primary flex-shrink-0" />
                    <div>
                      <p className="text-sm text-asgard-700 dark:text-asgard-300">
                        {item.action}
                      </p>
                      <p className="text-xs text-asgard-400">{item.time}</p>
                    </div>
                  </div>
                ))}
              </div>
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
            <CardTitle>System Status</CardTitle>
            <CardDescription>All systems operational</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
              {[
                { name: 'Silenus Network', status: 'operational' },
                { name: 'Hunoid Fleet', status: 'operational' },
                { name: 'Nysus Core', status: 'operational' },
                { name: 'Giru Security', status: 'operational' },
              ].map((system) => (
                <div
                  key={system.name}
                  className="flex items-center gap-3 p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800/50"
                >
                  <span className="w-2 h-2 rounded-full bg-success" />
                  <span className="text-sm font-medium text-asgard-700 dark:text-asgard-300">
                    {system.name}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}

function DashboardFeeds() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Satellite Feeds</CardTitle>
        <CardDescription>Live streams from Silenus constellation</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="aspect-video rounded-xl bg-asgard-100 dark:bg-asgard-800 flex items-center justify-center">
              <div className="text-center">
                <Satellite className="w-8 h-8 text-asgard-400 mx-auto mb-2" />
                <p className="text-sm text-asgard-500">SAT-{String(i + 1).padStart(3, '0')}</p>
                <p className="text-xs text-asgard-400">Stream available</p>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function DashboardAlerts() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>All Alerts</CardTitle>
        <CardDescription>Detection history and active alerts</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-asgard-500 dark:text-asgard-400">
          Alert management interface will be displayed here.
        </p>
      </CardContent>
    </Card>
  );
}

function DashboardActivity() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Activity Log</CardTitle>
        <CardDescription>Complete system activity history</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-asgard-500 dark:text-asgard-400">
          Detailed activity log will be displayed here.
        </p>
      </CardContent>
    </Card>
  );
}

function DashboardSubscription() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Subscription</CardTitle>
        <CardDescription>Manage your subscription and billing</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-asgard-500 dark:text-asgard-400">
          Subscription management interface will be displayed here.
        </p>
      </CardContent>
    </Card>
  );
}

function DashboardSettings() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Settings</CardTitle>
        <CardDescription>Configure your account preferences</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-asgard-500 dark:text-asgard-400">
          Settings interface will be displayed here.
        </p>
      </CardContent>
    </Card>
  );
}

export default function Dashboard() {
  const location = useLocation();
  const { user, isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
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
                  Welcome back, {user?.fullName?.split(' ')[0]}
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
                        'flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-medium transition-colors',
                        isActive
                          ? 'bg-primary text-white'
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
            </Routes>
          </main>
        </div>
      </div>
    </div>
  );
}
