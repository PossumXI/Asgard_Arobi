import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { 
  Satellite, 
  Bot, 
  Shield, 
  Globe, 
  Zap, 
  Eye,
  ArrowRight,
  Play,
  CheckCircle,
  Users,
  Building2,
  Sparkles
} from 'lucide-react';
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

const features = [
  {
    icon: Satellite,
    title: 'Orbital Intelligence',
    description: 'Silenus satellite network provides real-time global monitoring with AI-powered threat detection.',
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
  },
  {
    icon: Bot,
    title: 'Humanoid Response',
    description: 'Hunoid autonomous units deliver immediate humanitarian aid without human bias.',
    color: 'text-emerald-500',
    bgColor: 'bg-emerald-500/10',
  },
  {
    icon: Shield,
    title: 'Adaptive Security',
    description: 'Giru continuously red-teams infrastructure, neutralizing threats before impact.',
    color: 'text-purple-500',
    bgColor: 'bg-purple-500/10',
    image: '/giru.png',
  },
  {
    icon: Globe,
    title: 'Interstellar Ready',
    description: 'Delay-tolerant networking extends operations from Earth orbit to Mars and beyond.',
    color: 'text-orange-500',
    bgColor: 'bg-orange-500/10',
  },
  {
    icon: Zap,
    title: 'Real-time Orchestration',
    description: 'Nysus nerve center coordinates all systems with sub-second response times.',
    color: 'text-yellow-500',
    bgColor: 'bg-yellow-500/10',
  },
  {
    icon: Eye,
    title: '24/7 Visibility',
    description: 'Live streaming hubs provide transparent access to operations worldwide.',
    color: 'text-pink-500',
    bgColor: 'bg-pink-500/10',
  },
];

const stats = [
  { value: '150+', label: 'Satellites Active' },
  { value: '24/7', label: 'Global Coverage' },
  { value: '<500ms', label: 'Response Time' },
  { value: '99.999%', label: 'System Uptime' },
];

const testimonials = [
  {
    quote: "ASGARD's rapid response system saved countless lives during the Pacific tsunami. The coordination was unprecedented.",
    author: 'Dr. Elena Vasquez',
    role: 'Director, Global Disaster Relief Initiative',
  },
  {
    quote: "The transparency and real-time visibility gives our citizens confidence in humanitarian operations.",
    author: 'Ambassador James Chen',
    role: 'United Nations Humanitarian Affairs',
  },
];

export default function Landing() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative min-h-screen flex items-center justify-center pt-16">
        {/* Background */}
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[1000px] h-[600px] bg-primary/5 rounded-full blur-3xl" />
          <div className="absolute top-1/4 right-0 w-[400px] h-[400px] bg-purple-500/5 rounded-full blur-3xl" />
        </div>

        <div className="container-wide py-20 lg:py-32">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-4xl mx-auto text-center"
          >
            {/* Badge */}
            <motion.div variants={fadeInUp} className="mb-8">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium">
                <Sparkles className="w-4 h-4" />
                Humanity's Guardian Network
              </span>
            </motion.div>

            {/* Headline */}
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6 text-balance"
            >
              Protecting Earth.{' '}
              <span className="gradient-text">Reaching Stars.</span>
            </motion.h1>

            {/* Subheadline */}
            <motion.p
              variants={fadeInUp}
              className="text-body-lg sm:text-xl text-asgard-500 dark:text-asgard-400 mb-10 max-w-2xl mx-auto text-balance"
            >
              ASGARD unifies orbital surveillance, autonomous robotics, and intelligent security 
              into a single planetary defense and humanitarian aid system.
            </motion.p>

            {/* CTAs */}
            <motion.div
              variants={fadeInUp}
              className="flex flex-col sm:flex-row items-center justify-center gap-4"
            >
              <Link to="/signup">
                <Button size="lg" className="group">
                  Get Started
                  <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                </Button>
              </Link>
              <Link to="/demo">
                <Button variant="outline" size="lg" className="group">
                  <Play className="w-4 h-4 mr-2" />
                  Watch Demo
                </Button>
              </Link>
            </motion.div>

            {/* Trust Badges */}
            <motion.div
              variants={fadeInUp}
              className="mt-16 flex flex-wrap items-center justify-center gap-8 text-asgard-400"
            >
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span className="text-sm">SOC 2 Certified</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span className="text-sm">UN Partnership</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span className="text-sm">ISO 27001</span>
              </div>
            </motion.div>
          </motion.div>
        </div>

        {/* Scroll indicator */}
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 1.5 }}
          className="absolute bottom-8 left-1/2 -translate-x-1/2"
        >
          <motion.div
            animate={{ y: [0, 8, 0] }}
            transition={{ duration: 1.5, repeat: Infinity }}
            className="w-6 h-10 rounded-full border-2 border-asgard-300 dark:border-asgard-600 flex items-start justify-center p-2"
          >
            <motion.div className="w-1 h-2 rounded-full bg-asgard-400" />
          </motion.div>
        </motion.div>
      </section>

      {/* Stats Section */}
      <section className="py-16 border-y border-asgard-100 dark:border-asgard-800 bg-asgard-50/50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
            {stats.map((stat, index) => (
              <motion.div
                key={stat.label}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className="text-center"
              >
                <div className="text-3xl sm:text-4xl font-bold text-asgard-900 dark:text-white mb-2">
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

      {/* Features Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Integrated Defense Architecture
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto">
              Six interconnected systems working as one organism to protect and serve humanity.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {features.map((feature, index) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6">
                    {'image' in feature && feature.image ? (
                      <div className="w-16 h-16 mb-4">
                        <img 
                          src={feature.image} 
                          alt={feature.title}
                          className="w-full h-full object-contain"
                        />
                      </div>
                    ) : (
                      <div className={cn('w-12 h-12 rounded-xl flex items-center justify-center mb-4', feature.bgColor)}>
                        <feature.icon className={cn('w-6 h-6', feature.color)} />
                      </div>
                    )}
                    <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                      {feature.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400">
                      {feature.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Mission Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black relative overflow-hidden">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-primary/20 via-transparent to-transparent" />
        
        <div className="container-wide relative">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="max-w-3xl mx-auto text-center"
          >
            <h2 className="text-display text-white mb-6">
              Our Mission
            </h2>
            <p className="text-xl text-asgard-300 mb-8 leading-relaxed">
              To create an autonomous system that aids humanity without bias, 
              defends against threats with precision, and extends our reach 
              to the stars—all while maintaining complete transparency.
            </p>
            <div className="flex flex-wrap items-center justify-center gap-6">
              <div className="flex items-center gap-2 text-asgard-400">
                <Users className="w-5 h-5" />
                <span>Humanity First</span>
              </div>
              <div className="flex items-center gap-2 text-asgard-400">
                <Shield className="w-5 h-5" />
                <span>Ethical Defense</span>
              </div>
              <div className="flex items-center gap-2 text-asgard-400">
                <Globe className="w-5 h-5" />
                <span>Global Access</span>
              </div>
            </div>
          </motion.div>
        </div>
      </section>

      {/* Testimonials */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Trusted Worldwide
            </h2>
          </motion.div>

          <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
            {testimonials.map((testimonial, index) => (
              <motion.div
                key={index}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full">
                  <CardContent className="p-8">
                    <blockquote className="text-lg text-asgard-700 dark:text-asgard-300 mb-6 leading-relaxed">
                      "{testimonial.quote}"
                    </blockquote>
                    <div>
                      <div className="font-semibold text-asgard-900 dark:text-white">
                        {testimonial.author}
                      </div>
                      <div className="text-sm text-asgard-500">
                        {testimonial.role}
                      </div>
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
            className="relative rounded-3xl bg-gradient-to-br from-primary to-primary-700 p-12 md:p-16 text-center overflow-hidden"
          >
            <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
            
            <div className="relative">
              <h2 className="text-display text-white mb-4">
                Ready to Join the Mission?
              </h2>
              <p className="text-lg text-primary-100 mb-8 max-w-xl mx-auto">
                Whether you're an individual, organization, or government entity, 
                there's a role for you in humanity's defense network.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <Link to="/signup">
                  <Button size="lg" variant="secondary" className="group">
                    Create Account
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </Link>
                <Link to="/gov">
                  <Button size="lg" variant="outline" className="border-white/20 text-white hover:bg-white/10">
                    <Building2 className="w-4 h-4 mr-2" />
                    Government Portal
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
