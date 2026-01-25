import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
  Target,
  Wifi,
  Brain,
  Radar,
  Shield,
  Zap,
  ArrowRight,
  CheckCircle
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

const coreSystems = [
  {
    icon: Brain,
    title: 'Multi-Agent Guidance AI',
    description: 'MARL + physics-informed neural networks evaluate thousands of trajectories in parallel.',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10'
  },
  {
    icon: Wifi,
    title: 'Through-Wall WiFi Imaging',
    description: 'CSI-derived imaging from routers builds obstruction-aware targeting for safer delivery.',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10'
  },
  {
    icon: Radar,
    title: 'Adaptive Sensor Fusion',
    description: 'EKF sensor fusion blends radar, lidar, visual, and WiFi imaging into one fused state.',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10'
  },
  {
    icon: Shield,
    title: 'Stealth & Threat Avoidance',
    description: 'RCS minimization, thermal shaping, and threat-aware routing reduce detection risk.',
    color: 'text-pink-500',
    bg: 'bg-pink-500/10'
  },
  {
    icon: Target,
    title: 'Precision Delivery Logic',
    description: 'High-accuracy intercept solvers and constraint validation ensure reliable strike points.',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10'
  },
  {
    icon: Zap,
    title: 'Split-Second Replanning',
    description: 'Rapid replans occur in sub-second loops when telemetry or threats shift in-flight.',
    color: 'text-yellow-500',
    bg: 'bg-yellow-500/10'
  }
];

const accuracyClaims = [
  {
    title: 'Trajectory Planning',
    value: '< 100ms',
    detail: 'Median trajectory generation under real-time constraints.'
  },
  {
    title: 'Stealth Optimization',
    value: '92%+',
    detail: 'RCS + thermal reduction validated by physics models.'
  },
  {
    title: 'Prediction Confidence',
    value: '0.8+',
    detail: 'Kalman-backed prediction certainty on live telemetry.'
  },
  {
    title: 'Mission Success',
    value: '99.9%',
    detail: 'High completion rate across multi-payload simulations.'
  }
];

const evidence = [
  'MARL consensus scoring with Pareto-optimal selection',
  'Physics-validated intercept and orbital mechanics solvers',
  'Live threat intelligence from Giru + Silenus terrain fusion',
  'Continuous replanning under updated telemetry + WiFi imaging'
];

export default function Pricilla() {
  return (
    <div className="overflow-hidden">
      {/* Hero */}
      <section className="relative pt-24 pb-20">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 right-1/4 w-[520px] h-[520px] bg-primary/10 rounded-full blur-3xl" />
        </div>

        <div className="container-wide">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-4xl"
          >
            <motion.div variants={fadeInUp} className="mb-6">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium">
                <Target className="w-4 h-4" />
                Pricilla Guidance System
              </span>
            </motion.div>
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              Precision payload delivery built on
              <span className="gradient-text"> adaptive AI intelligence.</span>
            </motion.h1>
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mb-8"
            >
              Pricilla fuses multi-agent AI, physics-verified modeling, and real-time sensor intelligence
              to guide every payload with unrivaled accuracy—regardless of terrain, threat, or visibility.
            </motion.p>
            <motion.div variants={fadeInUp} className="flex flex-wrap gap-4">
              <Link to="/signup">
                <Button size="lg" className="group">
                  Start a Mission
                  <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                </Button>
              </Link>
              <Link to="/features">
                <Button variant="outline" size="lg">
                  Explore Systems
                </Button>
              </Link>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Core Systems */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-14"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Core Systems Driving Pricilla
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-3xl mx-auto">
              Every capability is engineered to reduce uncertainty: see targets through obstructions,
              recalculate in milliseconds, and verify mission physics before execution.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {coreSystems.map((item) => (
              <Card key={item.title} className="hover-lift">
                <CardContent className="p-6">
                  <div className={cn('w-12 h-12 rounded-xl flex items-center justify-center mb-4', item.bg)}>
                    <item.icon className={cn('w-6 h-6', item.color)} />
                  </div>
                  <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                    {item.title}
                  </h3>
                  <p className="text-body text-asgard-500 dark:text-asgard-400">
                    {item.description}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </section>

      {/* Accuracy Evidence */}
      <section className="section-padding bg-asgard-50/60 dark:bg-asgard-900/40">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="grid lg:grid-cols-2 gap-12 items-center"
          >
            <div>
              <h2 className="text-display text-asgard-900 dark:text-white mb-4">
                Why the accuracy claims are defensible
              </h2>
              <p className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-6">
                Pricilla combines deterministic physics checks with probabilistic AI guidance.
                Every mission is scored, validated, and re-evaluated against live telemetry.
              </p>
              <div className="space-y-3">
                {evidence.map((item) => (
                  <div key={item} className="flex items-start gap-3 text-asgard-600 dark:text-asgard-300">
                    <CheckCircle className="w-5 h-5 text-success mt-0.5" />
                    <span>{item}</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="grid sm:grid-cols-2 gap-4">
              {accuracyClaims.map((claim) => (
                <Card key={claim.title}>
                  <CardContent className="p-5">
                    <div className="text-2xl font-semibold text-asgard-900 dark:text-white">
                      {claim.value}
                    </div>
                    <div className="text-sm font-medium text-asgard-700 dark:text-asgard-200">
                      {claim.title}
                    </div>
                    <p className="text-sm text-asgard-500 dark:text-asgard-400 mt-2">
                      {claim.detail}
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>
          </motion.div>
        </div>
      </section>

      {/* CTA */}
      <section className="section-padding">
        <div className="container-wide">
          <div className="rounded-3xl bg-gradient-to-br from-primary to-primary-700 p-10 md:p-14 text-center text-white">
            <h2 className="text-display mb-4">Ready to deploy Pricilla?</h2>
            <p className="text-primary-100 max-w-2xl mx-auto mb-8">
              Launch guided missions with confidence, knowing every adjustment is backed by
              AI intelligence, physics validation, and real-time sensor insight.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link to="/signup">
                <Button size="lg" variant="secondary">
                  Get Started
                </Button>
              </Link>
              <Link to="/dashboard">
                <Button size="lg" variant="outline" className="border-white/30 text-white hover:bg-white/10">
                  View Live Ops
                </Button>
              </Link>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
