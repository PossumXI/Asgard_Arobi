import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
  Gavel,
  Code,
  Users,
  TrendingUp,
  FileText,
  MessageSquare,
  Vote,
  CheckCircle,
  ArrowRight,
  Settings,
  Wallet,
  Sliders,
  Compass,
  DollarSign,
  Lightbulb,
  Award,
  UserPlus,
  Landmark
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

const foundationPillars = [
  {
    icon: Gavel,
    title: 'Governance',
    description: 'Token holders vote on protocol upgrades, ensuring decentralized decision-making and community ownership.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    specs: ['Token voting', 'Protocol upgrades', 'Community ownership', 'Transparent decisions']
  },
  {
    icon: Code,
    title: 'Developer Programs',
    description: 'Comprehensive grants and resources for developers building on the ASGARD ecosystem.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    specs: ['Grant funding', 'Technical support', 'Documentation', 'Dev resources']
  },
  {
    icon: Users,
    title: 'Community Projects',
    description: 'Funding and support for community-driven initiatives that expand the ecosystem.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    specs: ['Project funding', 'Community support', 'Initiative grants', 'Ecosystem growth']
  },
  {
    icon: TrendingUp,
    title: 'Network Growth',
    description: 'Strategic partnerships and initiatives to expand the network reach and adoption.',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
    specs: ['Partnerships', 'Market expansion', 'Adoption programs', 'Strategic alliances']
  }
];

const governanceSteps = [
  {
    step: 1,
    icon: FileText,
    title: 'Proposal Submission',
    description: 'Community members submit detailed proposals for protocol changes, new features, or treasury allocations.',
    color: 'text-cyan-500',
    bg: 'bg-cyan-500/10'
  },
  {
    step: 2,
    icon: MessageSquare,
    title: 'Community Discussion',
    description: 'Open forum for community members to discuss, debate, and refine proposals before voting begins.',
    color: 'text-yellow-500',
    bg: 'bg-yellow-500/10'
  },
  {
    step: 3,
    icon: Vote,
    title: 'Democratic Voting',
    description: 'Token-weighted voting where all community members can participate in shaping the future.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10'
  },
  {
    step: 4,
    icon: CheckCircle,
    title: 'Implementation',
    description: 'Approved proposals are implemented by the development team with full transparency.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10'
  }
];

const governanceAreas = [
  { title: 'Protocol Upgrades', icon: Settings, description: 'Core protocol improvements and feature additions' },
  { title: 'Treasury Management', icon: Wallet, description: 'Allocation of community funds and resources' },
  { title: 'Network Parameters', icon: Sliders, description: 'Adjustments to network settings and configurations' },
  { title: 'Strategic Direction', icon: Compass, description: 'Long-term vision and roadmap decisions' }
];

const programs = [
  {
    icon: DollarSign,
    title: 'Developer Grants',
    amount: 'Up to $50k',
    description: 'Funding for developers building innovative applications, tools, and infrastructure on ASGARD.',
    color: 'text-green-500',
    bg: 'bg-green-500/10',
    features: ['Project funding', 'Technical mentorship', 'Marketing support', 'Network access']
  },
  {
    icon: Lightbulb,
    title: 'Innovation Fund',
    amount: 'Research Partnerships',
    description: 'Collaborative research initiatives with universities and research institutions.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    features: ['Research grants', 'Academic partnerships', 'Publication support', 'Innovation labs']
  },
  {
    icon: Award,
    title: 'Community Rewards',
    amount: 'Bug Bounty & Ambassador',
    description: 'Recognition and rewards for community members who contribute to security and growth.',
    color: 'text-amber-500',
    bg: 'bg-amber-500/10',
    features: ['Bug bounty program', 'Ambassador rewards', 'Content creation', 'Community events']
  }
];

const communityStats = [
  { label: 'Community Members', value: '10K+', icon: Users },
  { label: 'Proposals Passed', value: '150+', icon: Vote },
  { label: 'Grants Awarded', value: '$2M+', icon: DollarSign },
  { label: 'Active Developers', value: '500+', icon: Code }
];

export default function Foundation() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-24 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/4 w-[600px] h-[600px] bg-purple-500/10 rounded-full blur-3xl" />
          <div className="absolute top-1/4 right-0 w-[400px] h-[400px] bg-emerald-500/5 rounded-full blur-3xl" />
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
                <Landmark className="w-4 h-4" />
                Community Governance & Innovation
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              Aura Genesis Foundation —
              <span className="gradient-text"> Decentralized Future</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mb-8"
            >
              A community-driven organization fostering innovation and decentralized governance.
              Together, we shape the future of the ASGARD ecosystem through transparent, democratic processes.
            </motion.p>
            
            <motion.div variants={fadeInUp} className="flex flex-wrap gap-4">
              <a href="https://discord.gg/auragenesis" target="_blank" rel="noopener noreferrer">
                <Button size="lg" className="group">
                  Join the Foundation
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

      {/* Community Stats Bar */}
      <section className="py-12 border-y border-asgard-100 dark:border-asgard-800 bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            {communityStats.map((stat, index) => (
              <motion.div
                key={stat.label}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className="text-center"
              >
                <div className="flex items-center justify-center mb-3">
                  <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center">
                    <stat.icon className="w-5 h-5 text-primary" />
                  </div>
                </div>
                <div className="text-2xl sm:text-3xl font-bold text-asgard-900 dark:text-white mb-1">
                  {stat.value}
                </div>
                <div className="text-sm text-asgard-500 dark:text-asgard-400">
                  {stat.label}
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Foundation Pillars Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Foundation Pillars
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Four core pillars drive our mission to build a decentralized, community-owned ecosystem
              that empowers innovation and sustainable growth.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {foundationPillars.map((pillar, index) => (
              <motion.div
                key={pillar.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6">
                    <div className={cn('w-12 h-12 rounded-xl flex items-center justify-center mb-4', pillar.bg)}>
                      <pillar.icon className={cn('w-6 h-6', pillar.color)} />
                    </div>
                    <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                      {pillar.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400 mb-4">
                      {pillar.description}
                    </p>
                    <div className="flex flex-wrap gap-2">
                      {pillar.specs.map((spec) => (
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

      {/* Governance Process Section */}
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
              Governance Process
            </h2>
            <p className="text-lg text-asgard-300 max-w-2xl mx-auto">
              Our transparent, four-step governance process ensures every community member 
              has a voice in shaping the future of the ecosystem.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {governanceSteps.map((step, index) => (
              <motion.div
                key={step.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full bg-white/5 border-white/10 backdrop-blur-sm">
                  <CardContent className="p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <div className={cn('w-10 h-10 rounded-xl flex items-center justify-center', step.bg)}>
                        <step.icon className={cn('w-5 h-5', step.color)} />
                      </div>
                      <span className="text-2xl font-bold text-white/30">0{step.step}</span>
                    </div>
                    <h3 className="text-lg font-semibold text-white mb-2">
                      {step.title}
                    </h3>
                    <p className="text-asgard-300">
                      {step.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Governance Areas Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Governance Areas
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Community governance extends across all critical aspects of the ecosystem,
              ensuring decentralized control and transparent decision-making.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {governanceAreas.map((area, index) => (
              <motion.div
                key={area.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift text-center">
                  <CardContent className="p-6">
                    <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center mx-auto mb-4">
                      <area.icon className="w-7 h-7 text-primary" />
                    </div>
                    <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                      {area.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400">
                      {area.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Programs Section */}
      <section className="section-padding bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Foundation Programs
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Multiple programs designed to support developers, researchers, and community members
              in building and growing the ASGARD ecosystem.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-3 gap-8">
            {programs.map((program, index) => (
              <motion.div
                key={program.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-8">
                    <div className={cn('w-14 h-14 rounded-2xl flex items-center justify-center mb-6', program.bg)}>
                      <program.icon className={cn('w-7 h-7', program.color)} />
                    </div>
                    <div className="mb-4">
                      <h3 className="text-xl font-bold text-asgard-900 dark:text-white mb-1">
                        {program.title}
                      </h3>
                      <span className={cn('text-sm font-semibold', program.color)}>
                        {program.amount}
                      </span>
                    </div>
                    <p className="text-body text-asgard-500 dark:text-asgard-400 mb-6">
                      {program.description}
                    </p>
                    <div className="space-y-3">
                      {program.features.map((feature) => (
                        <div key={feature} className="flex items-center gap-3">
                          <CheckCircle className={cn('w-5 h-5 flex-shrink-0', program.color)} />
                          <span className="text-asgard-700 dark:text-asgard-300">{feature}</span>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
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
            className="rounded-3xl bg-gradient-to-br from-purple-600 to-indigo-700 p-10 md:p-14 text-center text-white overflow-hidden relative"
          >
            <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
            
            <div className="relative">
              <Landmark className="w-16 h-16 mx-auto mb-6 opacity-80" />
              <h2 className="text-display mb-4">Join the Foundation</h2>
              <p className="text-purple-100 max-w-2xl mx-auto mb-8">
                Be part of a community-driven movement shaping the future of decentralized technology.
                Your voice matters in building the ecosystem of tomorrow.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <a href="https://discord.gg/auragenesis" target="_blank" rel="noopener noreferrer">
                  <Button size="lg" variant="secondary" className="group">
                    <UserPlus className="w-4 h-4 mr-2" />
                    Join Discord Community
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </a>
                <Link to="/signup">
                  <Button size="lg" variant="outline" className="border-white/30 text-white hover:bg-white/10">
                    Create Account
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
