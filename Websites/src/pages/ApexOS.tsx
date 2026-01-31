import { motion } from 'framer-motion';
import {
  Shield,
  Coins,
  Network,
  Atom,
  Lock,
  Layers,
  ArrowRight,
  CheckCircle,
  Monitor,
  Apple,
  Terminal,
  ExternalLink
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
    icon: Shield,
    title: 'AI Security',
    description: 'Real-time threat detection with ML-based anomaly detection that learns and adapts to emerging threats.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    specs: ['ML detection', 'Real-time alerts', 'Adaptive learning', 'Zero-day protection']
  },
  {
    icon: Coins,
    title: 'Earn Rewards',
    description: 'Generate AURA tokens while using APEX-OS-LQ. Contribute to the network and earn passive income.',
    color: 'text-yellow-500',
    bg: 'bg-yellow-500/10',
    specs: ['AURA tokens', 'Passive income', 'Staking rewards', 'Contribution bonuses']
  },
  {
    icon: Network,
    title: 'Decentralized',
    description: 'Community-governed ecosystem with token holder participation in key decisions and upgrades.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    specs: ['DAO governance', 'Token voting', 'Proposal system', 'Community driven']
  },
  {
    icon: Atom,
    title: 'Quantum-Safe',
    description: 'Post-quantum cryptography protection ensures your data remains secure against future quantum threats.',
    color: 'text-cyan-500',
    bg: 'bg-cyan-500/10',
    specs: ['PQC algorithms', 'Lattice-based', 'Future-proof', 'NIST compliant']
  },
  {
    icon: Lock,
    title: 'seL4 Microkernel',
    description: 'Built on the formally verified seL4 microkernel—the most secure operating system foundation available.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    specs: ['Formally verified', 'Minimal attack surface', 'Proven secure', 'Military-grade']
  },
  {
    icon: Layers,
    title: 'Arobi Network',
    description: 'Full blockchain integration with the Arobi Network for decentralized storage, transactions, and identity.',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
    specs: ['Blockchain native', 'Smart contracts', 'DeFi ready', 'Web3 integration']
  }
];

const architectureLayers = [
  {
    title: 'Arobi Network Integration',
    description: 'Blockchain layer for transactions, governance, and decentralized services',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    borderColor: 'border-purple-500/30'
  },
  {
    title: 'AISA Security System',
    description: 'AI-powered security layer with real-time threat detection and response',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    borderColor: 'border-blue-500/30'
  },
  {
    title: 'seL4 Microkernel Foundation',
    description: 'Formally verified secure foundation ensuring system integrity',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    borderColor: 'border-emerald-500/30'
  }
];

const platforms = [
  { name: 'Windows 11', icon: Monitor, description: 'Full native support' },
  { name: 'macOS', icon: Apple, description: 'Apple Silicon optimized' },
  { name: 'Linux', icon: Terminal, description: 'Kernel-level integration' }
];

const keyBenefits = [
  'Formally verified security foundation',
  'Quantum-resistant cryptography',
  'Earn AURA tokens while computing',
  'Community governance participation',
  'AI-powered threat detection',
  'Full blockchain integration'
];

export default function ApexOS() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-24 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/4 w-[600px] h-[600px] bg-purple-500/10 rounded-full blur-3xl" />
          <div className="absolute top-1/4 right-0 w-[400px] h-[400px] bg-cyan-500/5 rounded-full blur-3xl" />
        </div>

        <div className="container-wide">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-4xl"
          >
            <motion.div variants={fadeInUp} className="mb-6">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-purple-500/10 text-purple-600 dark:text-purple-400 text-sm font-medium">
                <Lock className="w-4 h-4" />
                Autonomous Computing Platform
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              APEX-OS-LQ — The Future of
              <span className="gradient-text"> Autonomous Computing</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mb-8"
            >
              Built on the formally verified seL4 microkernel with AI-augmented security and blockchain integration.
              Experience computing that's secure by design and rewarding by nature.
            </motion.p>
            
            <motion.div variants={fadeInUp} className="flex flex-wrap gap-4">
              <a href="https://discord.gg/h64Cg8c6" target="_blank" rel="noopener noreferrer">
                <Button size="lg" className="group">
                  Get Notified on Discord
                  <ExternalLink className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                </Button>
              </a>
              <Button variant="outline" size="lg" disabled>
                Download (Coming Soon)
              </Button>
            </motion.div>
          </motion.div>
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
              Revolutionary Features
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Six integrated systems combine to create the most secure, rewarding, and 
              future-proof computing platform ever built.
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

      {/* Architecture Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-purple-500/20 via-transparent to-transparent" />
        
        <div className="container-wide relative">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-white mb-4">
              Layered Architecture
            </h2>
            <p className="text-lg text-asgard-300 max-w-2xl mx-auto">
              APEX-OS-LQ's three-layer architecture provides security from the kernel up,
              with each layer reinforcing the others.
            </p>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            whileInView={{ opacity: 1, scale: 1 }}
            viewport={{ once: true }}
            className="max-w-3xl mx-auto"
          >
            <Card className="bg-white/5 border-white/10 backdrop-blur-sm overflow-hidden">
              <CardContent className="p-8">
                <div className="space-y-4">
                  {architectureLayers.map((layer, index) => (
                    <motion.div
                      key={layer.title}
                      initial={{ opacity: 0, x: -20 }}
                      whileInView={{ opacity: 1, x: 0 }}
                      viewport={{ once: true }}
                      transition={{ delay: index * 0.15 }}
                      className={cn(
                        'p-6 rounded-xl border-2 transition-all duration-300',
                        layer.bg,
                        layer.borderColor,
                        'hover:scale-[1.02]'
                      )}
                    >
                      <div className="flex items-center gap-4">
                        <div className={cn(
                          'w-10 h-10 rounded-lg flex items-center justify-center',
                          layer.bg
                        )}>
                          <Layers className={cn('w-5 h-5', layer.color)} />
                        </div>
                        <div>
                          <h3 className={cn('text-lg font-semibold', layer.color)}>
                            {layer.title}
                          </h3>
                          <p className="text-asgard-300 text-sm">
                            {layer.description}
                          </p>
                        </div>
                      </div>
                    </motion.div>
                  ))}
                </div>

                {/* Connection Indicators */}
                <div className="flex items-center justify-center gap-4 mt-8 pt-6 border-t border-white/10">
                  <div className="flex items-center gap-2 text-asgard-400">
                    <Shield className="w-4 h-4" />
                    <span className="text-sm">Verified Security</span>
                  </div>
                  <div className="w-px h-4 bg-asgard-600" />
                  <div className="flex items-center gap-2 text-asgard-400">
                    <Network className="w-4 h-4" />
                    <span className="text-sm">Blockchain Native</span>
                  </div>
                  <div className="w-px h-4 bg-asgard-600" />
                  <div className="flex items-center gap-2 text-asgard-400">
                    <Atom className="w-4 h-4" />
                    <span className="text-sm">Quantum-Safe</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </motion.div>
        </div>
      </section>

      {/* Platforms Section */}
      <section className="section-padding bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Available Platforms
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto">
              APEX-OS-LQ runs natively on all major platforms with optimized performance for each.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-6 max-w-4xl mx-auto">
            {platforms.map((platform, index) => (
              <motion.div
                key={platform.name}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full text-center hover-lift">
                  <CardContent className="p-8">
                    <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4">
                      <platform.icon className="w-8 h-8 text-primary" />
                    </div>
                    <h3 className="text-xl font-semibold text-asgard-900 dark:text-white mb-2">
                      {platform.name}
                    </h3>
                    <p className="text-asgard-500 dark:text-asgard-400">
                      {platform.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Benefits & Pricing Section */}
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
                Why Choose APEX-OS-LQ?
              </h2>
              <p className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-8">
                A computing platform that pays you while protecting you—built on the most 
                secure foundation available.
              </p>
              <div className="space-y-4">
                {keyBenefits.map((benefit, index) => (
                  <motion.div
                    key={benefit}
                    initial={{ opacity: 0, x: -20 }}
                    whileInView={{ opacity: 1, x: 0 }}
                    viewport={{ once: true }}
                    transition={{ delay: index * 0.1 }}
                    className="flex items-center gap-3"
                  >
                    <CheckCircle className="w-5 h-5 text-success flex-shrink-0" />
                    <span className="text-asgard-700 dark:text-asgard-300">{benefit}</span>
                  </motion.div>
                ))}
              </div>
            </div>
            
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
            >
              <Card className="overflow-hidden">
                <CardContent className="p-8">
                  <div className="text-center">
                    <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 text-sm font-medium mb-6">
                      <Coins className="w-4 h-4" />
                      Coming Soon
                    </div>
                    <h3 className="text-2xl font-bold text-asgard-900 dark:text-white mb-2">
                      APEX-OS-LQ License
                    </h3>
                    <div className="flex items-baseline justify-center gap-1 mb-4">
                      <span className="text-5xl font-bold text-asgard-900 dark:text-white">$49</span>
                      <span className="text-asgard-500 dark:text-asgard-400">/one-time</span>
                    </div>
                    <p className="text-asgard-500 dark:text-asgard-400 mb-6">
                      Lifetime license with free updates and AURA token rewards
                    </p>
                    <div className="space-y-3 text-left mb-8">
                      {[
                        'Full APEX-OS-LQ installation',
                        'AURA token earning enabled',
                        'Lifetime security updates',
                        'Community governance access',
                        'Priority support'
                      ].map((item) => (
                        <div key={item} className="flex items-center gap-2 text-sm text-asgard-600 dark:text-asgard-300">
                          <CheckCircle className="w-4 h-4 text-emerald-500 flex-shrink-0" />
                          {item}
                        </div>
                      ))}
                    </div>
                    <a href="https://discord.gg/h64Cg8c6" target="_blank" rel="noopener noreferrer" className="block">
                      <Button size="lg" className="w-full group">
                        Get Notified on Discord
                        <ExternalLink className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                      </Button>
                    </a>
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
            className="rounded-3xl bg-gradient-to-br from-purple-600 to-indigo-700 p-10 md:p-14 text-center text-white overflow-hidden relative"
          >
            <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
            
            <div className="relative">
              <Lock className="w-16 h-16 mx-auto mb-6 opacity-80" />
              <h2 className="text-display mb-4">Join the Future of Computing</h2>
              <p className="text-purple-100 max-w-2xl mx-auto mb-8">
                Be among the first to experience APEX-OS-LQ. Join our Discord community 
                for early access, updates, and exclusive rewards.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <a href="https://discord.gg/h64Cg8c6" target="_blank" rel="noopener noreferrer">
                  <Button size="lg" variant="secondary" className="group">
                    Join Discord Community
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </a>
              </div>
            </div>
          </motion.div>
        </div>
      </section>
    </div>
  );
}
