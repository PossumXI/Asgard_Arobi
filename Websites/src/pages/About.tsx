import { motion } from 'framer-motion';
import { 
  Shield, 
  Globe, 
  Users, 
  Heart, 
  Target, 
  Sparkles,
  ArrowRight,
  Building2,
  Rocket,
  Eye
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Card, CardContent } from '@/components/ui/Card';

const fadeInUp = {
  initial: { opacity: 0, y: 30 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.6 }
};

const staggerContainer = {
  animate: {
    transition: {
      staggerChildren: 0.1
    }
  }
};

const values = [
  {
    icon: Heart,
    title: 'Humanity First',
    description: 'Every decision, every algorithm, every action is designed to protect and serve human life above all else.',
  },
  {
    icon: Shield,
    title: 'Ethical Defense',
    description: 'Our AI operates within strict ethical boundaries, ensuring responses are proportional and minimize harm.',
  },
  {
    icon: Eye,
    title: 'Radical Transparency',
    description: 'All operations are logged, auditable, and accessible. We believe accountability builds trust.',
  },
  {
    icon: Globe,
    title: 'Universal Access',
    description: 'Protection and aid should not be privileges. We work to serve all nations and peoples equally.',
  },
];

const timeline = [
  {
    year: '2024',
    title: 'Foundation',
    description: 'ASGARD initiative launched with core architecture design and initial team formation.',
  },
  {
    year: '2025',
    title: 'Silenus Alpha',
    description: 'First orbital perception satellite network deployed with real-time threat detection.',
  },
  {
    year: '2025',
    title: 'Hunoid Genesis',
    description: 'Initial deployment of autonomous humanoid units for disaster response testing.',
  },
  {
    year: '2026',
    title: 'Full Integration',
    description: 'Complete Nysus nerve center activation unifying all systems into single organism.',
  },
];

const leadership = [
  {
    name: 'Dr. Helena Vance',
    role: 'Chief Executive Officer',
    bio: 'Former NASA Mission Director with 20 years in autonomous systems.',
  },
  {
    name: 'Marcus Chen',
    role: 'Chief Technology Officer',
    bio: 'Pioneer in delay-tolerant networking and distributed AI architectures.',
  },
  {
    name: 'Dr. Amara Okonkwo',
    role: 'Chief Ethics Officer',
    bio: 'Leading researcher in AI ethics and autonomous decision-making frameworks.',
  },
  {
    name: 'Gen. James Reeves (Ret.)',
    role: 'Chief Security Officer',
    bio: 'Former CYBERCOM commander, expert in defensive cyber operations.',
  },
];

export default function About() {
  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-32 pb-20 lg:pt-40 lg:pb-32">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/4 w-[600px] h-[600px] bg-primary/5 rounded-full blur-3xl" />
        </div>

        <div className="container-wide">
          <motion.div
            initial="initial"
            animate="animate"
            variants={staggerContainer}
            className="max-w-3xl"
          >
            <motion.div variants={fadeInUp} className="mb-6">
              <span className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium">
                <Sparkles className="w-4 h-4" />
                Our Story
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg text-asgard-900 dark:text-white mb-6"
            >
              Building Humanity's Guardian System
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-8"
            >
              ASGARD represents a fundamental shift in how we protect and serve humanity. 
              By unifying orbital intelligence, ground robotics, and adaptive security into 
              a single conscious system, we're creating the nervous system Earth deserves.
            </motion.p>
          </motion.div>
        </div>
      </section>

      {/* Mission Section */}
      <section className="py-20 bg-asgard-900 dark:bg-black">
        <div className="container-wide">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
            >
              <h2 className="text-display text-white mb-6">Our Mission</h2>
              <p className="text-lg text-asgard-300 mb-6 leading-relaxed">
                To create an autonomous defense and humanitarian aid system that operates 
                without bias, responds to threats with precision, and extends humanity's 
                reach beyond Earth—all while maintaining complete transparency and ethical integrity.
              </p>
              <p className="text-lg text-asgard-300 leading-relaxed">
                We believe that advanced technology, properly constrained by ethics and 
                accountability, can be a force for unprecedented good. ASGARD is our 
                commitment to proving that belief.
              </p>
            </motion.div>
            
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              className="grid grid-cols-2 gap-4"
            >
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <Target className="w-10 h-10 text-primary mb-4" />
                <div className="text-3xl font-bold text-white mb-1">99.999%</div>
                <div className="text-sm text-asgard-400">System Uptime</div>
              </div>
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <Rocket className="w-10 h-10 text-emerald-500 mb-4" />
                <div className="text-3xl font-bold text-white mb-1">&lt;500ms</div>
                <div className="text-sm text-asgard-400">Response Time</div>
              </div>
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <Users className="w-10 h-10 text-purple-500 mb-4" />
                <div className="text-3xl font-bold text-white mb-1">150+</div>
                <div className="text-sm text-asgard-400">Partner Nations</div>
              </div>
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <Building2 className="w-10 h-10 text-orange-500 mb-4" />
                <div className="text-3xl font-bold text-white mb-1">12</div>
                <div className="text-sm text-asgard-400">Global Hubs</div>
              </div>
            </motion.div>
          </div>
        </div>
      </section>

      {/* Values Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Our Core Values
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto">
              These principles guide every decision we make and every system we build.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 gap-6">
            {values.map((value, index) => (
              <motion.div
                key={value.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-8">
                    <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center mb-6">
                      <value.icon className="w-7 h-7 text-primary" />
                    </div>
                    <h3 className="text-headline text-asgard-900 dark:text-white mb-3">
                      {value.title}
                    </h3>
                    <p className="text-body text-asgard-500 dark:text-asgard-400">
                      {value.description}
                    </p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Timeline Section */}
      <section className="section-padding bg-asgard-50 dark:bg-asgard-900/50">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Our Journey
            </h2>
          </motion.div>

          <div className="max-w-3xl mx-auto">
            {timeline.map((item, index) => (
              <motion.div
                key={item.year + item.title}
                initial={{ opacity: 0, x: -20 }}
                whileInView={{ opacity: 1, x: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className="relative pl-8 pb-12 last:pb-0 border-l-2 border-asgard-200 dark:border-asgard-700"
              >
                <div className="absolute left-0 top-0 w-4 h-4 -translate-x-[9px] rounded-full bg-primary" />
                <div className="text-sm font-semibold text-primary mb-1">{item.year}</div>
                <h3 className="text-title text-asgard-900 dark:text-white mb-2">{item.title}</h3>
                <p className="text-body text-asgard-500 dark:text-asgard-400">{item.description}</p>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Leadership Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Leadership Team
            </h2>
            <p className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto">
              World-class experts united by a common vision.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {leadership.map((person, index) => (
              <motion.div
                key={person.name}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full text-center">
                  <CardContent className="p-6">
                    <div className="w-20 h-20 mx-auto mb-4 rounded-full bg-gradient-to-br from-primary/20 to-purple-500/20 flex items-center justify-center">
                      <Users className="w-10 h-10 text-primary" />
                    </div>
                    <h3 className="text-title text-asgard-900 dark:text-white mb-1">
                      {person.name}
                    </h3>
                    <p className="text-sm text-primary mb-3">{person.role}</p>
                    <p className="text-caption text-asgard-500 dark:text-asgard-400">
                      {person.bio}
                    </p>
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
            <div className="relative">
              <h2 className="text-display text-white mb-4">
                Join the Mission
              </h2>
              <p className="text-lg text-primary-100 mb-8 max-w-xl mx-auto">
                Whether you're a developer, researcher, or organization, 
                there's a place for you in building humanity's defense network.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <Link to="/signup">
                  <Button size="lg" variant="secondary" className="group">
                    Get Started
                    <ArrowRight className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                  </Button>
                </Link>
                <Link to="/features">
                  <Button size="lg" variant="outline" className="border-white/20 text-white hover:bg-white/10">
                    View Capabilities
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
