import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
  Shield,
  Bot,
  Brain,
  Mic,
  Activity,
  Globe,
  Network,
  Crosshair,
  Eye,
  Cpu,
  Monitor,
  Layers,
  ArrowRight,
  CheckCircle,
  Zap
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

const giruSecurityFeatures = [
  'Network Intrusion Detection System (IDS)',
  'Automated Red Team/Blue Team exercises',
  'Real-time Threat Intelligence feeds',
  'Shadow Stack memory protection',
  'PCAP analysis and deep packet inspection',
  'Anomaly detection with ML models'
];

const jarvisFeatures = [
  'Voice-activated with "Giru" wake word',
  'Multi-model AI (Groq, Gemini, Claude)',
  'Real-time activity monitoring dashboard',
  'Natural language command processing',
  'Full ASGARD ecosystem integration',
  'Automated security response actions'
];

const featureCards = [
  {
    icon: Network,
    title: 'Network Security',
    description: 'Advanced PCAP analysis and intrusion detection with real-time packet inspection and threat identification.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    specs: ['PCAP analysis', 'Deep inspection', 'IDS/IPS', 'Flow analysis']
  },
  {
    icon: Crosshair,
    title: 'Red Team / Blue Team',
    description: 'Automated security testing with offensive and defensive simulation for continuous vulnerability assessment.',
    color: 'text-red-500',
    bg: 'bg-red-500/10',
    specs: ['Pen testing', 'Attack simulation', 'Defense drills', 'Auto remediation']
  },
  {
    icon: Brain,
    title: 'AI Models',
    description: '10+ AI models including free options. Seamlessly switch between Groq, Gemini, Claude, and more.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    specs: ['Groq Llama', 'Gemini Pro', 'Claude', 'Local models']
  },
  {
    icon: Mic,
    title: 'Voice Control',
    description: 'Wake word "Giru" activates natural language processing for hands-free operation and commands.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    specs: ['Wake word', 'NLP engine', 'Voice synthesis', 'Multi-language']
  },
  {
    icon: Activity,
    title: 'Real-time Monitor',
    description: 'Comprehensive activity dashboard tracking model usage, system health, and security events.',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
    specs: ['Live metrics', 'Usage tracking', 'Alert system', 'Performance stats']
  },
  {
    icon: Globe,
    title: 'ASGARD Integration',
    description: 'Connects all ASGARD systems—Valkyrie, Pricilla, Nysus—for unified security and control.',
    color: 'text-cyan-500',
    bg: 'bg-cyan-500/10',
    specs: ['Unified API', 'Cross-system', 'Event routing', 'Central control']
  }
];

const architectureNodes = [
  { id: 'electron', label: 'Electron UI', icon: Monitor, color: 'text-blue-500', bg: 'bg-blue-500/10' },
  { id: 'python', label: 'Python Backend', icon: Cpu, color: 'text-yellow-500', bg: 'bg-yellow-500/10' },
  { id: 'ai', label: 'AI Providers', icon: Brain, color: 'text-purple-500', bg: 'bg-purple-500/10' },
  { id: 'monitor', label: 'Monitor Dashboard', icon: Activity, color: 'text-emerald-500', bg: 'bg-emerald-500/10' }
];

const aiProviders = [
  { name: 'Groq', description: 'Ultra-fast inference with Llama models' },
  { name: 'Gemini', description: 'Google\'s multimodal AI capabilities' },
  { name: 'Claude', description: 'Anthropic\'s advanced reasoning' },
  { name: 'Local', description: 'Privacy-first on-device models' }
];

export default function Giru() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-24 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/3 w-[600px] h-[600px] bg-emerald-500/10 rounded-full blur-3xl" />
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
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-sm font-medium">
                <Shield className="w-4 h-4" />
                Adaptive Security & AI Assistant
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              GIRU — Your Intelligent
              <span className="gradient-text"> Security Guardian</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mb-8"
            >
              GIRU combines advanced network security with JARVIS, a voice-activated AI assistant.
              Protect your systems, automate responses, and command your infrastructure with natural language.
            </motion.p>
            
            <motion.div variants={fadeInUp} className="flex flex-wrap gap-4">
              <Link to="/signup">
                <Button size="lg" className="group">
                  Get Started with GIRU
                  <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                </Button>
              </Link>
              <Link to="/features">
                <Button variant="outline" size="lg">
                  Explore Features
                </Button>
              </Link>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Two Main Sections: GIRU Security & JARVIS */}
      <section className="section-padding bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <div className="grid lg:grid-cols-2 gap-8">
            {/* GIRU Security */}
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
            >
              <Card className="h-full">
                <CardContent className="p-8">
                  <div className="w-14 h-14 rounded-2xl bg-red-500/10 flex items-center justify-center mb-6">
                    <Shield className="w-7 h-7 text-red-500" />
                  </div>
                  <h2 className="text-2xl font-bold text-asgard-900 dark:text-white mb-4">
                    GIRU Security
                  </h2>
                  <p className="text-body text-asgard-500 dark:text-asgard-400 mb-6">
                    Enterprise-grade security suite with network monitoring, automated testing, 
                    and intelligent threat response capabilities.
                  </p>
                  <div className="space-y-3">
                    {giruSecurityFeatures.map((feature, index) => (
                      <motion.div
                        key={feature}
                        initial={{ opacity: 0, x: -10 }}
                        whileInView={{ opacity: 1, x: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: index * 0.05 }}
                        className="flex items-center gap-3"
                      >
                        <CheckCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
                        <span className="text-asgard-700 dark:text-asgard-300">{feature}</span>
                      </motion.div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </motion.div>

            {/* GIRU JARVIS */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
            >
              <Card className="h-full">
                <CardContent className="p-8">
                  <div className="w-14 h-14 rounded-2xl bg-purple-500/10 flex items-center justify-center mb-6">
                    <Bot className="w-7 h-7 text-purple-500" />
                  </div>
                  <h2 className="text-2xl font-bold text-asgard-900 dark:text-white mb-4">
                    GIRU JARVIS v2.0
                  </h2>
                  <p className="text-body text-asgard-500 dark:text-asgard-400 mb-6">
                    Voice-activated AI assistant powered by multiple LLM providers, offering 
                    real-time monitoring and seamless ASGARD integration.
                  </p>
                  <div className="space-y-3">
                    {jarvisFeatures.map((feature, index) => (
                      <motion.div
                        key={feature}
                        initial={{ opacity: 0, x: -10 }}
                        whileInView={{ opacity: 1, x: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: index * 0.05 }}
                        className="flex items-center gap-3"
                      >
                        <CheckCircle className="w-5 h-5 text-purple-500 flex-shrink-0" />
                        <span className="text-asgard-700 dark:text-asgard-300">{feature}</span>
                      </motion.div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Comprehensive Security Features
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Six integrated capabilities work together to deliver complete security coverage
              and intelligent automation for your infrastructure.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {featureCards.map((feature, index) => (
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

      {/* Technical Architecture Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-emerald-500/20 via-transparent to-transparent" />
        
        <div className="container-wide relative">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-white mb-4">
              Technical Architecture
            </h2>
            <p className="text-lg text-asgard-300 max-w-2xl mx-auto">
              GIRU's modular architecture enables seamless communication between the user interface,
              backend processing, and AI providers.
            </p>
          </motion.div>

          {/* Architecture Diagram */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            whileInView={{ opacity: 1, scale: 1 }}
            viewport={{ once: true }}
            className="max-w-4xl mx-auto"
          >
            <Card className="bg-white/5 border-white/10 backdrop-blur-sm">
              <CardContent className="p-8">
                {/* Main Flow */}
                <div className="flex flex-col md:flex-row items-center justify-between gap-6 mb-8">
                  {architectureNodes.map((node, index) => (
                    <motion.div
                      key={node.id}
                      initial={{ opacity: 0, y: 20 }}
                      whileInView={{ opacity: 1, y: 0 }}
                      viewport={{ once: true }}
                      transition={{ delay: index * 0.1 }}
                      className="flex flex-col items-center"
                    >
                      <div className={cn('w-16 h-16 rounded-2xl flex items-center justify-center mb-3', node.bg)}>
                        <node.icon className={cn('w-8 h-8', node.color)} />
                      </div>
                      <span className="text-white font-medium text-center">{node.label}</span>
                      {index < architectureNodes.length - 1 && (
                        <div className="hidden md:block absolute">
                          <ArrowRight className="w-5 h-5 text-asgard-500" />
                        </div>
                      )}
                    </motion.div>
                  ))}
                </div>

                {/* Connection Lines (Visual) */}
                <div className="hidden md:flex items-center justify-center gap-4 mb-8">
                  <div className="flex items-center gap-2 text-asgard-400">
                    <Layers className="w-4 h-4" />
                    <span className="text-sm">WebSocket</span>
                  </div>
                  <div className="w-px h-4 bg-asgard-600" />
                  <div className="flex items-center gap-2 text-asgard-400">
                    <Zap className="w-4 h-4" />
                    <span className="text-sm">REST API</span>
                  </div>
                  <div className="w-px h-4 bg-asgard-600" />
                  <div className="flex items-center gap-2 text-asgard-400">
                    <Eye className="w-4 h-4" />
                    <span className="text-sm">Event Stream</span>
                  </div>
                </div>

                {/* AI Providers */}
                <div className="border-t border-white/10 pt-8">
                  <h3 className="text-lg font-semibold text-white mb-4 text-center">
                    AI Provider Integration
                  </h3>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    {aiProviders.map((provider, index) => (
                      <motion.div
                        key={provider.name}
                        initial={{ opacity: 0, y: 10 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: index * 0.05 }}
                        className="text-center p-4 rounded-xl bg-white/5 border border-white/10"
                      >
                        <div className="text-white font-medium mb-1">{provider.name}</div>
                        <div className="text-xs text-asgard-400">{provider.description}</div>
                      </motion.div>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>
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
            className="rounded-3xl bg-gradient-to-br from-emerald-600 to-teal-700 p-10 md:p-14 text-center text-white overflow-hidden relative"
          >
            <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
            
            <div className="relative">
              <Bot className="w-16 h-16 mx-auto mb-6 opacity-80" />
              <h2 className="text-display mb-4">Get Started with GIRU</h2>
              <p className="text-emerald-100 max-w-2xl mx-auto mb-8">
                Deploy intelligent security and voice-controlled AI assistance for your infrastructure.
                GIRU adapts to your environment and grows with your needs.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <Link to="/signup">
                  <Button size="lg" variant="secondary" className="group">
                    Start Free Trial
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
