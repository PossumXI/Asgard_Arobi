import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import {
  Play,
  ArrowLeft,
  Satellite,
  Bot,
  Shield,
  Crosshair,
  Lock,
  Radio,
  Cpu,
  Bell,
} from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Card, CardContent } from '@/components/ui/Card';

const demos = [
  {
    id: 'flight-simulation',
    title: 'Complete Flight Simulation',
    description:
      'Full 8-phase mission demonstration including system initialization, flight execution, weather response, defense engagement, through-wall imaging, and autonomous rescue operations.',
    videoSrc: '/demos/flight-simulation-demo.webm',
    duration: '~5 min',
    systems: [
      { name: 'Valkyrie', icon: Crosshair },
      { name: 'GIRU', icon: Shield },
      { name: 'Hunoid', icon: Bot },
      { name: 'Pricilla', icon: Satellite },
      { name: 'Vault', icon: Lock },
      { name: 'Silenus', icon: Radio },
      { name: 'Nysus', icon: Cpu },
      { name: 'Notifications', icon: Bell },
    ],
    phases: [
      'System Initialization & Health Verification',
      'Mission Planning & Trajectory Calculation',
      'Autonomous Flight Execution',
      'Dynamic Weather Event Response',
      'Defense Target Engagement',
      'WiFi Through-Wall Imaging',
      'Hunoid Rescue Operations',
      'Mission Complete & Debrief',
    ],
  },
];

export default function Demo() {
  return (
    <div className="min-h-screen pt-24 pb-16">
      <div className="container-wide">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-12"
        >
          <Link to="/">
            <Button variant="ghost" size="sm" className="mb-6">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Home
            </Button>
          </Link>

          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
              <Play className="w-5 h-5 text-primary" />
            </div>
            <h1 className="text-display text-asgard-900 dark:text-white">
              System Demos
            </h1>
          </div>
          <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl">
            Watch ASGARD's integrated defense systems in action. Each demo showcases
            real service orchestration across all subsystems.
          </p>
        </motion.div>

        {/* Demo Cards */}
        {demos.map((demo, index) => (
          <motion.div
            key={demo.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 + 0.2 }}
            className="mb-16"
          >
            {/* Video Player */}
            <Card className="overflow-hidden mb-8">
              <div className="relative bg-black rounded-t-xl">
                <video
                  controls
                  preload="metadata"
                  className="w-full aspect-video"
                  poster=""
                >
                  <source src={demo.videoSrc} type="video/webm" />
                  Your browser does not support WebM video playback.
                </video>
              </div>
              <CardContent className="p-6">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                  <div>
                    <h2 className="text-xl font-semibold text-asgard-900 dark:text-white mb-1">
                      {demo.title}
                    </h2>
                    <p className="text-body text-asgard-500 dark:text-asgard-400">
                      {demo.description}
                    </p>
                  </div>
                  <span className="text-sm text-asgard-400 whitespace-nowrap">
                    {demo.duration}
                  </span>
                </div>
              </CardContent>
            </Card>

            {/* Demo Details Grid */}
            <div className="grid md:grid-cols-2 gap-6">
              {/* Systems Involved */}
              <Card>
                <CardContent className="p-6">
                  <h3 className="text-title text-asgard-900 dark:text-white mb-4">
                    Systems Demonstrated
                  </h3>
                  <div className="grid grid-cols-2 gap-3">
                    {demo.systems.map((system) => (
                      <div
                        key={system.name}
                        className="flex items-center gap-2 text-sm text-asgard-600 dark:text-asgard-300"
                      >
                        <system.icon className="w-4 h-4 text-primary" />
                        {system.name}
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              {/* Mission Phases */}
              <Card>
                <CardContent className="p-6">
                  <h3 className="text-title text-asgard-900 dark:text-white mb-4">
                    Mission Phases
                  </h3>
                  <ol className="space-y-2">
                    {demo.phases.map((phase, i) => (
                      <li
                        key={i}
                        className="flex items-start gap-3 text-sm text-asgard-600 dark:text-asgard-300"
                      >
                        <span className="flex-shrink-0 w-5 h-5 rounded-full bg-primary/10 text-primary text-xs flex items-center justify-center font-medium mt-0.5">
                          {i + 1}
                        </span>
                        {phase}
                      </li>
                    ))}
                  </ol>
                </CardContent>
              </Card>
            </div>
          </motion.div>
        ))}

        {/* More Demos Coming */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="text-center py-12 border-t border-asgard-100 dark:border-asgard-800"
        >
          <p className="text-asgard-400 dark:text-asgard-500 mb-4">
            More demos are being generated from live system tests.
          </p>
          <Link to="/contact">
            <Button variant="outline">Request a Custom Demo</Button>
          </Link>
        </motion.div>
      </div>
    </div>
  );
}
