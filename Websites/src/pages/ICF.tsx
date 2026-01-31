import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
  Shield,
  Link as LinkIcon,
  Brain,
  Server,
  Lock,
  CheckCircle,
  Users,
  Vote,
  Eye,
  FileCheck,
  Crown,
  ArrowRight,
  Coins,
  Award,
  Star,
  Network,
  Scale,
  AlertTriangle
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

const poiBreakdown = [
  { label: 'Token Weight', value: '40%', icon: Coins, color: 'text-yellow-500', bg: 'bg-yellow-500/10', description: 'AURA token holdings determine voting power' },
  { label: 'Expertise Verification', value: '40%', icon: Award, color: 'text-blue-500', bg: 'bg-blue-500/10', description: 'Verified technical credentials and certifications' },
  { label: 'Reputation Score', value: '20%', icon: Star, color: 'text-purple-500', bg: 'bg-purple-500/10', description: 'Historical contribution and community standing' }
];

const expertCategories = [
  {
    icon: Shield,
    title: 'Cybersecurity Experts',
    description: 'Threat detection specialists protecting network integrity through advanced monitoring and incident response.',
    color: 'text-red-500',
    bg: 'bg-red-500/10',
    specs: ['Threat detection', 'Incident response', 'Vulnerability assessment', 'Security audits']
  },
  {
    icon: LinkIcon,
    title: 'Blockchain Specialists',
    description: 'DLT and consensus experts ensuring protocol integrity and optimal blockchain performance.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    specs: ['DLT architecture', 'Consensus protocols', 'Smart contracts', 'Chain analysis']
  },
  {
    icon: Brain,
    title: 'AI Researchers',
    description: 'ML anomaly detection specialists leveraging artificial intelligence for predictive security.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    specs: ['Anomaly detection', 'Pattern recognition', 'Predictive models', 'Neural networks']
  },
  {
    icon: Server,
    title: 'IT Infrastructure',
    description: 'Enterprise architects designing resilient, scalable systems for mission-critical operations.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    specs: ['System design', 'Cloud architecture', 'High availability', 'Disaster recovery']
  },
  {
    icon: Lock,
    title: 'Cryptography',
    description: 'Post-quantum specialists developing future-proof encryption and security protocols.',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
    specs: ['Post-quantum crypto', 'Key management', 'Zero-knowledge proofs', 'Encryption standards']
  },
  {
    icon: CheckCircle,
    title: 'Network Validators',
    description: 'Proven reliability validators maintaining network consensus through consistent uptime.',
    color: 'text-cyan-500',
    bg: 'bg-cyan-500/10',
    specs: ['Node operation', '99.9% uptime', 'Block validation', 'Network stability']
  }
];

const securityGuarantees = [
  {
    icon: Users,
    title: 'Multi-Signature Authority',
    description: 'Critical decisions require consensus from multiple verified ICF members, preventing single points of failure.',
    color: 'text-blue-500'
  },
  {
    icon: Eye,
    title: 'Transparent Auditing',
    description: 'All governance decisions and validator actions are publicly recorded on-chain for complete transparency.',
    color: 'text-emerald-500'
  },
  {
    icon: Scale,
    title: 'Accountability Mechanisms',
    description: 'Clear penalties and dispute resolution processes ensure member accountability and network integrity.',
    color: 'text-orange-500'
  }
];

const icfResponsibilities = [
  {
    icon: Shield,
    title: 'Security Oversight',
    description: 'Continuous monitoring and assessment of network security threats and vulnerabilities.',
    color: 'text-red-500',
    bg: 'bg-red-500/10'
  },
  {
    icon: Vote,
    title: 'Protocol Governance',
    description: 'Democratic decision-making on protocol upgrades, parameter changes, and network policies.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10'
  },
  {
    icon: FileCheck,
    title: 'Validation & Monitoring',
    description: 'Real-time network validation and performance monitoring to ensure optimal operation.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10'
  },
  {
    icon: Crown,
    title: 'Community Leadership',
    description: 'Guiding ecosystem development and fostering collaboration among network participants.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10'
  }
];

const joiningRequirements = [
  'Top 1% AURA token holders',
  'Verified technical expertise in relevant domain',
  'Nomination by existing ICF members',
  'Successful completion of expertise verification',
  'Commitment to network governance participation',
  'Clean security background verification'
];

export default function ICF() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-24 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/4 w-[600px] h-[600px] bg-indigo-500/10 rounded-full blur-3xl" />
          <div className="absolute top-1/4 right-0 w-[400px] h-[400px] bg-yellow-500/5 rounded-full blur-3xl" />
        </div>

        <div className="container-wide">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-4xl"
          >
            <motion.div variants={fadeInUp} className="mb-6">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 text-sm font-medium">
                <Network className="w-4 h-4" />
                Proof of Intelligence Consensus
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              Intelligence Consule Federation
              <span className="gradient-text"> — PoI Consensus</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mb-8"
            >
              An elite council of verified experts maintaining network security through Proof of Intelligence consensus.
              The ICF combines technical expertise, token economics, and reputation to ensure decentralized governance excellence.
            </motion.p>
            
            <motion.div variants={fadeInUp} className="flex flex-wrap gap-4">
              <a href="https://discord.gg/aura-genesis" target="_blank" rel="noopener noreferrer">
                <Button size="lg" className="group">
                  Join ICF Community
                  <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                </Button>
              </a>
              <Link to="/features">
                <Button variant="outline" size="lg">
                  Learn More
                </Button>
              </Link>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Proof of Intelligence Breakdown */}
      <section className="py-12 border-y border-asgard-100 dark:border-asgard-800 bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-10"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Proof of Intelligence
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto">
              A revolutionary consensus mechanism that weighs expertise alongside economic stake.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-8">
            {poiBreakdown.map((item, index) => (
              <motion.div
                key={item.label}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6 text-center">
                    <div className={cn('w-14 h-14 rounded-2xl flex items-center justify-center mx-auto mb-4', item.bg)}>
                      <item.icon className={cn('w-7 h-7', item.color)} />
                    </div>
                    <div className="text-4xl font-bold text-asgard-900 dark:text-white mb-2">
                      {item.value}
                    </div>
                    <div className="text-lg font-semibold text-asgard-700 dark:text-asgard-300 mb-2">
                      {item.label}
                    </div>
                    <p className="text-body text-asgard-500 dark:text-asgard-400">
                      {item.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Expert Categories Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Expert Categories
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              The ICF comprises specialists across six critical domains, each bringing unique expertise
              to maintain network security and governance excellence.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {expertCategories.map((expert, index) => (
              <motion.div
                key={expert.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6">
                    <div className={cn('w-12 h-12 rounded-xl flex items-center justify-center mb-4', expert.bg)}>
                      <expert.icon className={cn('w-6 h-6', expert.color)} />
                    </div>
                    <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                      {expert.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400 mb-4">
                      {expert.description}
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {expert.specs.map((spec) => (
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

      {/* Security Guarantees Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-indigo-500/20 via-transparent to-transparent" />
        
        <div className="container-wide relative">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-white mb-4">
              Security Guarantees
            </h2>
            <p className="text-lg text-asgard-300 max-w-2xl mx-auto">
              Built-in mechanisms ensure the ICF operates with integrity, transparency,
              and accountability at every level.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-6">
            {securityGuarantees.map((guarantee, index) => (
              <motion.div
                key={guarantee.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full bg-white/5 border-white/10 backdrop-blur-sm">
                  <CardContent className="p-6">
                    <guarantee.icon className={cn('w-8 h-8 mb-4', guarantee.color)} />
                    <h3 className="text-lg font-semibold text-white mb-2">
                      {guarantee.title}
                    </h3>
                    <p className="text-asgard-300">
                      {guarantee.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* ICF Responsibilities Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              ICF Responsibilities
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Federation members carry significant responsibility for the health and security
              of the entire network ecosystem.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {icfResponsibilities.map((responsibility, index) => (
              <motion.div
                key={responsibility.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6 text-center">
                    <div className={cn('w-14 h-14 rounded-2xl flex items-center justify-center mx-auto mb-4', responsibility.bg)}>
                      <responsibility.icon className={cn('w-7 h-7', responsibility.color)} />
                    </div>
                    <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                      {responsibility.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400">
                      {responsibility.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Joining Requirements Section */}
      <section className="section-padding bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="grid lg:grid-cols-2 gap-12 items-center"
          >
            <div>
              <h2 className="text-display text-asgard-900 dark:text-white mb-4">
                Joining Requirements
              </h2>
              <p className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-8">
                Membership in the ICF is highly selective, ensuring only the most qualified
                and committed experts participate in network governance.
              </p>
              <div className="space-y-4">
                {joiningRequirements.map((requirement, index) => (
                  <motion.div
                    key={requirement}
                    initial={{ opacity: 0, x: -20 }}
                    whileInView={{ opacity: 1, x: 0 }}
                    viewport={{ once: true }}
                    transition={{ delay: index * 0.1 }}
                    className="flex items-center gap-3"
                  >
                    <CheckCircle className="w-5 h-5 text-success flex-shrink-0" />
                    <span className="text-asgard-700 dark:text-asgard-300">{requirement}</span>
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
                  <div className="aspect-video bg-gradient-to-br from-indigo-500/20 to-purple-500/20 rounded-xl flex items-center justify-center">
                    <div className="text-center">
                      <div className="flex items-center justify-center gap-2 mb-4">
                        <Crown className="w-12 h-12 text-indigo-500" />
                        <Network className="w-12 h-12 text-purple-500" />
                      </div>
                      <p className="text-asgard-500 dark:text-asgard-400 font-medium">
                        Elite Governance Council
                      </p>
                      <p className="text-sm text-asgard-400 dark:text-asgard-500 mt-1">
                        Top 1% of AURA holders + Verified Expertise
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Important Notice */}
      <section className="py-8">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
          >
            <Card className="border-orange-500/20 bg-orange-500/5">
              <CardContent className="p-6">
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 rounded-xl bg-orange-500/10 flex items-center justify-center flex-shrink-0">
                    <AlertTriangle className="w-5 h-5 text-orange-500" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-asgard-900 dark:text-white mb-1">
                      Application Process
                    </h3>
                    <p className="text-body text-asgard-600 dark:text-asgard-400">
                      ICF membership applications are reviewed quarterly. Candidates must demonstrate exceptional expertise 
                      in their domain and receive nomination from at least two existing ICF members before their application 
                      can be considered by the council.
                    </p>
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
            className="rounded-3xl bg-gradient-to-br from-indigo-600 to-purple-700 p-10 md:p-14 text-center text-white overflow-hidden relative"
          >
            <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
            
            <div className="relative">
              <Network className="w-16 h-16 mx-auto mb-6 opacity-80" />
              <h2 className="text-display mb-4">Join the ICF Community</h2>
              <p className="text-indigo-100 max-w-2xl mx-auto mb-8">
                Connect with fellow experts, stay updated on governance proposals, and begin your path
                to becoming an ICF member. Our Discord is the central hub for all ICF discussions.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <a href="https://discord.gg/aura-genesis" target="_blank" rel="noopener noreferrer">
                  <Button size="lg" variant="secondary" className="group">
                    Join Discord Community
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </a>
                <Link to="/contact">
                  <Button size="lg" variant="outline" className="border-white/30 text-white hover:bg-white/10">
                    Contact ICF Council
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
