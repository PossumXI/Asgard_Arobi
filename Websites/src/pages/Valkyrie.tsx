import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
  Plane,
  Cpu,
  Shield,
  Radio,
  AlertTriangle,
  Activity,
  Radar,
  Navigation,
  Gauge,
  Lock,
  ArrowRight,
  CheckCircle,
  Zap,
  Eye
} from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Card, CardContent } from '@/components/ui/Card';
import { cn } from '@/lib/utils';

const fadeInUp = {
  initial: { opacity: 0, y: 24 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.6 } }
};

const staggerContainer = {
  animate: {
    transition: {
      staggerChildren: 0.1
    }
  }
};

const coreFeatures = [
  {
    icon: Radar,
    title: 'Sensor Fusion',
    description: '100Hz Extended Kalman Filter fusing GPS, INS, RADAR, and LIDAR for unparalleled situational awareness.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    specs: ['GPS + INS fusion', 'RADAR tracking', 'LIDAR mapping', '100Hz update rate']
  },
  {
    icon: Cpu,
    title: 'AI Decision Engine',
    description: '50Hz reinforcement learning-based control system for adaptive flight decisions in real-time.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    specs: ['RL-based control', 'Adaptive planning', 'Predictive modeling', '50Hz processing']
  },
  {
    icon: Shield,
    title: 'Security Monitoring',
    description: 'Shadow stack verification and continuous threat detection powered by Giru integration.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    specs: ['Shadow stack', 'Threat detection', 'Intrusion prevention', 'Giru integration']
  },
  {
    icon: AlertTriangle,
    title: 'Fail-Safe Systems',
    description: 'Triple redundancy architecture with automated emergency procedures for mission-critical reliability.',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
    specs: ['Triple redundancy', 'Auto-recovery', 'Emergency RTB', 'Safe landing']
  },
  {
    icon: Radio,
    title: 'Real-time Telemetry',
    description: 'WebSocket streaming delivers live flight data to ground stations and ASGARD control centers.',
    color: 'text-pink-500',
    bg: 'bg-pink-500/10',
    specs: ['WebSocket streams', 'Live telemetry', 'Ground station link', 'Cloud backup']
  },
  {
    icon: Navigation,
    title: 'Autonomous Navigation',
    description: 'Advanced waypoint management with dynamic re-routing based on real-time conditions.',
    color: 'text-cyan-500',
    bg: 'bg-cyan-500/10',
    specs: ['Waypoint planning', 'Dynamic routing', 'Obstacle avoidance', 'Terrain following']
  }
];

const technicalSpecs = [
  { label: 'Sensor Fusion Rate', value: '100 Hz', icon: Gauge },
  { label: 'Control Loop', value: '50 Hz', icon: Activity },
  { label: 'Telemetry Latency', value: '<50ms', icon: Radio },
  { label: 'Redundancy Level', value: 'Triple', icon: Shield }
];

const integrations = [
  {
    title: 'Pricilla Guidance',
    description: 'Precision payload delivery with multi-agent trajectory optimization.',
    icon: Eye,
    color: 'text-purple-500'
  },
  {
    title: 'Giru Security',
    description: 'Continuous threat monitoring and intrusion prevention.',
    icon: Lock,
    color: 'text-emerald-500'
  },
  {
    title: 'Nysus Orchestration',
    description: 'Real-time coordination with ASGARD command infrastructure.',
    icon: Zap,
    color: 'text-yellow-500'
  }
];

const capabilities = [
  'Fully autonomous takeoff and landing',
  'All-weather operation capability',
  'Beyond visual line of sight (BVLOS) certified',
  'Multi-aircraft swarm coordination',
  'Encrypted command and control',
  'Mission abort and safe return protocols'
];

export default function Valkyrie() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-24 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/4 w-[600px] h-[600px] bg-cyan-500/10 rounded-full blur-3xl" />
          <div className="absolute top-1/4 right-0 w-[400px] h-[400px] bg-purple-500/5 rounded-full blur-3xl" />
        </div>

        <div className="container-wide">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-4xl"
          >
            <motion.div variants={fadeInUp} className="mb-6">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 text-sm font-medium">
                <Plane className="w-4 h-4" />
                Autonomous Flight System
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              VALKYRIE — The
              <span className="gradient-text"> Tesla Autopilot for Aircraft</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mb-8"
            >
              Valkyrie delivers next-generation autonomous flight capabilities through advanced sensor fusion, 
              AI-powered decision making, and seamless integration with the ASGARD defense ecosystem.
            </motion.p>
            
            <motion.div variants={fadeInUp} className="flex flex-wrap gap-4">
              <Link to="/signup">
                <Button size="lg" className="group">
                  Get Started
                  <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                </Button>
              </Link>
              <Link to="/features">
                <Button variant="outline" size="lg">
                  Technical Docs
                </Button>
              </Link>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Technical Specs Bar */}
      <section className="py-12 border-y border-asgard-100 dark:border-asgard-800 bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            {technicalSpecs.map((spec, index) => (
              <motion.div
                key={spec.label}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className="text-center"
              >
                <div className="flex items-center justify-center mb-3">
                  <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                    <spec.icon className="w-5 h-5 text-primary" />
                  </div>
                </div>
                <div className="text-2xl sm:text-3xl font-bold text-asgard-900 dark:text-white mb-1">
                  {spec.value}
                </div>
                <div className="text-sm text-asgard-500 dark:text-asgard-400">
                  {spec.label}
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Core Features Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Core Flight Systems
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Six integrated systems work in harmony to deliver autonomous flight capability 
              that exceeds human pilot performance in safety and precision.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {coreFeatures.map((feature, index) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6">
                    <div className={cn('w-12 h-12 rounded-xl flex items-center justify-center mb-4', feature.bg)}>
                      <feature.icon className={cn('w-6 h-6', feature.color)} />
                    </div>
                    <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                      {feature.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400 mb-4">
                      {feature.description}
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {feature.specs.map((spec) => (
                        <span
                          key={spec}
                          className="inline-flex items-center px-2.5 py-1 rounded-lg bg-asgard-100 dark:bg-asgard-800 text-xs font-medium text-asgard-600 dark:text-asgard-300"
                        >
                          {spec}
                        </span>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* ASGARD Integration Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-cyan-500/20 via-transparent to-transparent" />
        
        <div className="container-wide relative">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-white mb-4">
              Integrated with ASGARD Ecosystem
            </h2>
            <p className="text-lg text-asgard-300 max-w-2xl mx-auto">
              Valkyrie seamlessly connects with other ASGARD systems for complete 
              mission coordination and enhanced capabilities.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-6">
            {integrations.map((integration, index) => (
              <motion.div
                key={integration.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full bg-white/5 border-white/10 backdrop-blur-sm">
                  <CardContent className="p-6">
                    <integration.icon className={cn('w-8 h-8 mb-4', integration.color)} />
                    <h3 className="text-lg font-semibold text-white mb-2">
                      {integration.title}
                    </h3>
                    <p className="text-asgard-300">
                      {integration.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Capabilities Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="grid lg:grid-cols-2 gap-12 items-center"
          >
            <div>
              <h2 className="text-display text-asgard-900 dark:text-white mb-4">
                Operational Capabilities
              </h2>
              <p className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-8">
                Valkyrie is designed for demanding operational environments where reliability, 
                safety, and autonomous decision-making are paramount.
              </p>
              <div className="space-y-4">
                {capabilities.map((capability, index) => (
                  <motion.div
                    key={capability}
                    initial={{ opacity: 0, x: -20 }}
                    whileInView={{ opacity: 1, x: 0 }}
                    viewport={{ once: true }}
                    transition={{ delay: index * 0.1 }}
                    className="flex items-center gap-3"
                  >
                    <CheckCircle className="w-5 h-5 text-success flex-shrink-0" />
                    <span className="text-asgard-700 dark:text-asgard-300">{capability}</span>
                  </motion.div>
                ))}
              </div>
            </div>
            
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              className="relative"
            >
              <Card className="overflow-hidden">
                <CardContent className="p-8">
                  <div className="aspect-video bg-gradient-to-br from-cyan-500/20 to-purple-500/20 rounded-xl flex items-center justify-center">
                    <div className="text-center">
                      <Plane className="w-16 h-16 text-cyan-500 mx-auto mb-4" />
                      <p className="text-asgard-500 dark:text-asgard-400">
                        Valkyrie Flight Visualization
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="rounded-3xl bg-gradient-to-br from-cyan-600 to-blue-700 p-10 md:p-14 text-center text-white overflow-hidden relative"
          >
            <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
            
            <div className="relative">
              <h2 className="text-display mb-4">Ready to Deploy Valkyrie?</h2>
              <p className="text-cyan-100 max-w-2xl mx-auto mb-8">
                Experience autonomous flight technology that redefines what's possible in aviation. 
                Contact us to discuss deployment options for your organization.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <Link to="/signup">
                  <Button size="lg" variant="secondary" className="group">
                    Get Started
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </Link>
                <Link to="/contact">
                  <Button size="lg" variant="outline" className="border-white/30 text-white hover:bg-white/10">
                    Contact Sales
                  </Button>
                </Link>
              </div>
            </div>
          </motion.div>
        </div>
      </section>
    </div>
  );
}
