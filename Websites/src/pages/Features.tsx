import { motion } from 'framer-motion';
import { 
  Satellite, 
  Bot, 
  Shield, 
  Globe, 
  Network,
  Brain,
  Lock,
  Radio,
  Server,
  Monitor,
  Sparkles,
  ArrowRight,
  CheckCircle
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Card, CardContent } from '@/components/ui/Card';
import { cn } from '@/lib/utils';

const fadeInUp = {
  initial: { opacity: 0, y: 30 },
  animate: { 
    opacity: 1, 
    y: 0,
    transition: { duration: 0.6 }
  }
};

const staggerContainer = {
  animate: {
    transition: {
      staggerChildren: 0.1
    }
  }
};

const coreCapabilities = [
  {
    id: 'silenus',
    icon: Satellite,
    title: 'Silenus',
    subtitle: 'Orbital Intelligence',
    description: 'AI-powered satellite network providing real-time global monitoring with edge-computed threat detection.',
    features: [
      'YOLOv8 object detection at the edge',
      'TinyGo firmware for radiation tolerance',
      'Sub-second alert generation',
      'Thermal and multispectral imaging',
    ],
    color: 'blue',
  },
  {
    id: 'hunoid',
    icon: Bot,
    title: 'Hunoid',
    subtitle: 'Autonomous Response',
    description: 'Super-intelligent humanoid units capable of disaster response and humanitarian aid without human bias.',
    features: [
      'Vision-Language-Action (VLA) models',
      'Ethical pre-processor for all actions',
      'Mars-ready autarky mode',
      'ROS2 robotics middleware',
    ],
    color: 'emerald',
  },
  {
    id: 'nysus',
    icon: Brain,
    title: 'Nysus',
    subtitle: 'Nerve Center',
    description: 'Central orchestration hub that coordinates all systems using Model Context Protocol (MCP).',
    features: [
      'Real-time multi-agent coordination',
      'LLM-powered decision making',
      'Global situation awareness',
      'Predictive threat modeling',
    ],
    color: 'purple',
  },
  {
    id: 'giru',
    icon: Shield,
    title: 'Giru 2.0',
    subtitle: 'Adaptive Security',
    description: 'AI-driven defense system with continuous red-teaming and zero-day threat neutralization.',
    features: [
      'Autonomous penetration testing',
      'Parallel shadow engine',
      'Gaga Chat steganography',
      'Ethical offensive capabilities',
    ],
    color: 'red',
    image: '/giru.png',
  },
  {
    id: 'satnet',
    icon: Network,
    title: 'Sat_Net',
    subtitle: 'Interstellar Network',
    description: 'Delay-tolerant networking backbone using Bundle Protocol v7 for Earth-to-Mars communication.',
    features: [
      'RL-powered adaptive routing',
      'Energy-aware path selection',
      'Store-and-forward reliability',
      'Optical inter-satellite links',
    ],
    color: 'orange',
  },
  {
    id: 'hubs',
    icon: Monitor,
    title: 'Viewing Hubs',
    subtitle: '24/7 Visibility',
    description: 'Real-time streaming interfaces providing transparent access to operations worldwide.',
    features: [
      'WebRTC low-latency streaming',
      'Civilian/Military/Interstellar tiers',
      'Time-delayed Mars feeds',
      '3D reconstructed reality',
    ],
    color: 'pink',
  },
];

const technicalSpecs = [
  { label: 'Response Latency', value: '<500ms', description: 'LEO direct' },
  { label: 'Packet Delivery', value: '99.99%', description: 'DTN custody' },
  { label: 'Threat Neutralization', value: '<5s', description: 'Detection to block' },
  { label: 'System Uptime', value: '99.999%', description: 'Five nines SLA' },
];

const colorClasses: Record<string, { bg: string; text: string; border: string }> = {
  blue: { bg: 'bg-blue-500/10', text: 'text-blue-500', border: 'border-blue-500/20' },
  emerald: { bg: 'bg-emerald-500/10', text: 'text-emerald-500', border: 'border-emerald-500/20' },
  purple: { bg: 'bg-purple-500/10', text: 'text-purple-500', border: 'border-purple-500/20' },
  red: { bg: 'bg-red-500/10', text: 'text-red-500', border: 'border-red-500/20' },
  orange: { bg: 'bg-orange-500/10', text: 'text-orange-500', border: 'border-orange-500/20' },
  pink: { bg: 'bg-pink-500/10', text: 'text-pink-500', border: 'border-pink-500/20' },
};

export default function Features() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-32 pb-20 lg:pt-40 lg:pb-32">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 right-1/4 w-[600px] h-[600px] bg-purple-500/5 rounded-full blur-3xl" />
        </div>

        <div className="container-wide">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-3xl mx-auto text-center"
          >
            <motion.div variants={fadeInUp} className="mb-6">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium">
                <Sparkles className="w-4 h-4" />
                System Capabilities
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg text-asgard-900 dark:text-white mb-6"
            >
              Six Systems.{' '}
              <span className="gradient-text">One Organism.</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400"
            >
              Each component of ASGARD is a specialized organ in a larger living system. 
              Together, they form the most advanced planetary defense network ever conceived.
            </motion.p>
          </motion.div>
        </div>
      </section>

      {/* Technical Specs Bar */}
      <section className="py-8 border-y border-asgard-100 dark:border-asgard-800 bg-asgard-50/50 dark:bg-asgard-900/50">
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
                <div className="text-2xl sm:text-3xl font-bold text-primary mb-1">
                  {spec.value}
                </div>
                <div className="text-sm font-medium text-asgard-900 dark:text-white">
                  {spec.label}
                </div>
                <div className="text-xs text-asgard-500 dark:text-asgard-400">
                  {spec.description}
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Core Capabilities */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Core Capabilities
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto">
              Dive deep into each subsystem powering the ASGARD network.
            </p>
          </motion.div>

          <div className="space-y-8">
            {coreCapabilities.map((capability, index) => {
              const colors = colorClasses[capability.color];
              return (
                <motion.div
                  key={capability.id}
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ delay: index * 0.1 }}
                >
                  <Card className="overflow-hidden">
                    <CardContent className="p-0">
                      <div className="grid md:grid-cols-3 gap-0">
                        <div className={cn(
                          'p-8 flex flex-col justify-center',
                          colors.bg
                        )}>
                          {'image' in capability && capability.image ? (
                            <div className="w-24 h-24 mb-4">
                              <img 
                                src={capability.image} 
                                alt={capability.title}
                                className="w-full h-full object-contain"
                              />
                            </div>
                          ) : (
                            <div className={cn(
                              'w-16 h-16 rounded-2xl flex items-center justify-center mb-4',
                              'bg-white dark:bg-asgard-900'
                            )}>
                              <capability.icon className={cn('w-8 h-8', colors.text)} />
                            </div>
                          )}
                          <h3 className="text-headline text-asgard-900 dark:text-white">
                            {capability.title}
                          </h3>
                          <p className={cn('text-sm font-medium', colors.text)}>
                            {capability.subtitle}
                          </p>
                        </div>
                        <div className="md:col-span-2 p-8">
                          <p className="text-body-lg text-asgard-600 dark:text-asgard-300 mb-6">
                            {capability.description}
                          </p>
                          <ul className="grid sm:grid-cols-2 gap-3">
                            {capability.features.map((feature) => (
                              <li key={feature} className="flex items-center gap-2">
                                <CheckCircle className={cn('w-5 h-5 flex-shrink-0', colors.text)} />
                                <span className="text-sm text-asgard-700 dark:text-asgard-300">
                                  {feature}
                                </span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              );
            })}
          </div>
        </div>
      </section>

      {/* Infrastructure Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black">
        <div className="container-wide">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
            >
              <h2 className="text-display text-white mb-6">
                Built for the Stars
              </h2>
              <p className="text-lg text-asgard-300 mb-8 leading-relaxed">
                ASGARD isn't just designed for Earth. Every protocol, every algorithm, 
                every system is built to operate across interplanetary distances. 
                The same network protecting a city can coordinate a Mars colony.
              </p>
              <ul className="space-y-4">
                {[
                  { icon: Radio, text: 'Bundle Protocol v7 for delay-tolerant communication' },
                  { icon: Server, text: 'Edge-computed AI for autonomous operation' },
                  { icon: Lock, text: 'BPSec cryptographic security throughout' },
                  { icon: Globe, text: 'CRDT-based conflict resolution for data sync' },
                ].map((item) => (
                  <li key={item.text} className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-white/10 flex items-center justify-center">
                      <item.icon className="w-5 h-5 text-primary" />
                    </div>
                    <span className="text-asgard-200">{item.text}</span>
                  </li>
                ))}
              </ul>
            </motion.div>
            
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              className="relative"
            >
              <div className="aspect-square rounded-3xl bg-gradient-to-br from-primary/20 to-purple-500/20 p-8 flex items-center justify-center">
                <div className="relative">
                  <Globe className="w-48 h-48 text-white/20" />
                  <motion.div
                    className="absolute top-0 left-1/2 -translate-x-1/2 -translate-y-4"
                    animate={{ y: [-4, -8, -4] }}
                    transition={{ duration: 3, repeat: Infinity }}
                  >
                    <Satellite className="w-12 h-12 text-primary" />
                  </motion.div>
                  <motion.div
                    className="absolute bottom-1/4 right-0 translate-x-4"
                    animate={{ x: [4, 8, 4] }}
                    transition={{ duration: 4, repeat: Infinity }}
                  >
                    <Bot className="w-10 h-10 text-emerald-500" />
                  </motion.div>
                  <motion.div
                    className="absolute bottom-0 left-1/4"
                    animate={{ scale: [1, 1.1, 1] }}
                    transition={{ duration: 2, repeat: Infinity }}
                  >
                    <Shield className="w-10 h-10 text-red-500" />
                  </motion.div>
                </div>
              </div>
            </motion.div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="relative rounded-3xl bg-gradient-to-br from-primary to-primary-700 p-12 md:p-16 text-center overflow-hidden"
          >
            <div className="relative">
              <h2 className="text-display text-white mb-4">
                Ready to Experience ASGARD?
              </h2>
              <p className="text-lg text-primary-100 mb-8 max-w-xl mx-auto">
                Join the network protecting humanity across Earth and beyond.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <Link to="/signup">
                  <Button size="lg" variant="secondary" className="group">
                    Create Account
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </Link>
                <Link to="/pricing">
                  <Button size="lg" variant="outline" className="border-white/20 text-white hover:bg-white/10">
                    View Pricing
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
