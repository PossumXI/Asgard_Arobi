import { useState } from 'react';
import { motion } from 'framer-motion';
import { 
  Check, 
  X, 
  Sparkles,
  ArrowRight,
  Building2,
  Shield,
  Zap
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';
import { cn } from '@/lib/utils';

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

const plans = [
  {
    id: 'observer',
    name: 'Observer',
    description: 'For individuals monitoring global events',
    price: { monthly: 0, yearly: 0 },
    features: [
      { text: 'Public civilian feeds', included: true },
      { text: 'Basic alert notifications', included: true },
      { text: 'Community forums access', included: true },
      { text: 'Weekly digest emails', included: true },
      { text: 'Real-time satellite feeds', included: false },
      { text: 'Priority support', included: false },
      { text: 'API access', included: false },
      { text: 'Custom integrations', included: false },
    ],
    cta: 'Get Started Free',
    highlighted: false,
  },
  {
    id: 'supporter',
    name: 'Supporter',
    description: 'For researchers and organizations',
    price: { monthly: 49, yearly: 490 },
    features: [
      { text: 'All Observer features', included: true },
      { text: 'Real-time satellite feeds', included: true },
      { text: 'Advanced alert filtering', included: true },
      { text: 'Historical data access (30 days)', included: true },
      { text: 'Priority email support', included: true },
      { text: 'Basic API access', included: true },
      { text: 'Custom integrations', included: false },
      { text: 'Dedicated account manager', included: false },
    ],
    cta: 'Start Free Trial',
    highlighted: true,
  },
  {
    id: 'commander',
    name: 'Commander',
    description: 'For enterprise and government',
    price: { monthly: 299, yearly: 2990 },
    features: [
      { text: 'All Supporter features', included: true },
      { text: 'Full API access', included: true },
      { text: 'Historical data access (unlimited)', included: true },
      { text: 'Custom alert configurations', included: true },
      { text: 'Custom integrations', included: true },
      { text: 'Dedicated account manager', included: true },
      { text: '24/7 phone support', included: true },
      { text: 'SLA guarantees', included: true },
    ],
    cta: 'Contact Sales',
    highlighted: false,
  },
];

const faqs = [
  {
    question: 'What payment methods do you accept?',
    answer: 'We accept all major credit cards, wire transfers for enterprise plans, and government procurement systems (CAGE code available on request).',
  },
  {
    question: 'Can I upgrade or downgrade my plan?',
    answer: 'Yes, you can change your plan at any time. Upgrades take effect immediately, and downgrades take effect at the end of your billing period.',
  },
  {
    question: 'Is there a free trial?',
    answer: 'Yes, Supporter plans include a 14-day free trial with full access to all features. No credit card required.',
  },
  {
    question: 'Do you offer discounts for non-profits?',
    answer: 'Yes, we offer 50% discounts for verified non-profit organizations and NGOs. Contact our sales team for details.',
  },
  {
    question: 'What are the government portal requirements?',
    answer: 'Government entities require FIDO2 hardware tokens for authentication. We support CAC/PIV cards and are FedRAMP compliant.',
  },
  {
    question: 'How do I cancel my subscription?',
    answer: 'You can cancel anytime from your account settings. You\'ll retain access until the end of your billing period.',
  },
];

export default function Pricing() {
  const [billingPeriod, setBillingPeriod] = useState<'monthly' | 'yearly'>('yearly');

  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-32 pb-20 lg:pt-40 lg:pb-24">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[400px] bg-primary/5 rounded-full blur-3xl" />
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
                Simple Pricing
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg text-asgard-900 dark:text-white mb-6"
            >
              Choose Your Access Level
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 mb-8"
            >
              From free community access to enterprise-grade capabilities. 
              Transparent pricing with no hidden fees.
            </motion.p>

            {/* Billing Toggle */}
            <motion.div variants={fadeInUp} className="flex items-center justify-center gap-4">
              <span className={cn(
                'text-sm font-medium transition-colors',
                billingPeriod === 'monthly' ? 'text-asgard-900 dark:text-white' : 'text-asgard-400'
              )}>
                Monthly
              </span>
              <button
                onClick={() => setBillingPeriod(billingPeriod === 'monthly' ? 'yearly' : 'monthly')}
                className={cn(
                  'relative w-14 h-8 rounded-full transition-colors',
                  billingPeriod === 'yearly' ? 'bg-primary' : 'bg-asgard-200 dark:bg-asgard-700'
                )}
              >
                <span className={cn(
                  'absolute top-1 w-6 h-6 rounded-full bg-white shadow transition-transform',
                  billingPeriod === 'yearly' ? 'translate-x-7' : 'translate-x-1'
                )} />
              </button>
              <span className={cn(
                'text-sm font-medium transition-colors',
                billingPeriod === 'yearly' ? 'text-asgard-900 dark:text-white' : 'text-asgard-400'
              )}>
                Yearly
              </span>
              {billingPeriod === 'yearly' && (
                <span className="px-2 py-0.5 rounded-full bg-success/10 text-success text-xs font-medium">
                  Save 17%
                </span>
              )}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Pricing Cards */}
      <section className="pb-20">
        <div className="container-wide">
          <div className="grid md:grid-cols-3 gap-8 max-w-6xl mx-auto">
            {plans.map((plan, index) => (
              <motion.div
                key={plan.id}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className={cn(
                  'relative',
                  plan.highlighted && 'md:-mt-4 md:mb-4'
                )}
              >
                {plan.highlighted && (
                  <div className="absolute -top-4 left-0 right-0 flex justify-center">
                    <span className="px-4 py-1 rounded-full bg-primary text-white text-xs font-medium">
                      Most Popular
                    </span>
                  </div>
                )}
                <Card className={cn(
                  'h-full',
                  plan.highlighted && 'border-primary shadow-glow'
                )}>
                  <CardHeader className="text-center pb-0">
                    <CardTitle className="text-headline">{plan.name}</CardTitle>
                    <CardDescription>{plan.description}</CardDescription>
                  </CardHeader>
                  <CardContent className="pt-6">
                    <div className="text-center mb-6">
                      <div className="flex items-baseline justify-center gap-1">
                        <span className="text-4xl font-bold text-asgard-900 dark:text-white">
                          ${billingPeriod === 'monthly' ? plan.price.monthly : Math.round(plan.price.yearly / 12)}
                        </span>
                        <span className="text-asgard-500">/month</span>
                      </div>
                      {plan.price.yearly > 0 && billingPeriod === 'yearly' && (
                        <p className="text-sm text-asgard-400 mt-1">
                          ${plan.price.yearly} billed yearly
                        </p>
                      )}
                    </div>

                    <ul className="space-y-3 mb-8">
                      {plan.features.map((feature) => (
                        <li key={feature.text} className="flex items-center gap-3">
                          {feature.included ? (
                            <Check className="w-5 h-5 text-success flex-shrink-0" />
                          ) : (
                            <X className="w-5 h-5 text-asgard-300 dark:text-asgard-600 flex-shrink-0" />
                          )}
                          <span className={cn(
                            'text-sm',
                            feature.included 
                              ? 'text-asgard-700 dark:text-asgard-300' 
                              : 'text-asgard-400 dark:text-asgard-500'
                          )}>
                            {feature.text}
                          </span>
                        </li>
                      ))}
                    </ul>

                    <Link to={plan.id === 'commander' ? '/gov' : '/signup'}>
                      <Button 
                        className="w-full" 
                        variant={plan.highlighted ? 'primary' : 'outline'}
                      >
                        {plan.cta}
                        <ArrowRight className="w-4 h-4 ml-2" />
                      </Button>
                    </Link>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Enterprise Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black">
        <div className="container-wide">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
            >
              <h2 className="text-display text-white mb-6">
                Enterprise & Government
              </h2>
              <p className="text-lg text-asgard-300 mb-8 leading-relaxed">
                Need a custom solution? We work with defense contractors, 
                government agencies, and large enterprises to build tailored 
                integrations with dedicated support and SLA guarantees.
              </p>
              <ul className="space-y-4 mb-8">
                {[
                  { icon: Shield, text: 'FedRAMP and SOC 2 compliant' },
                  { icon: Building2, text: 'On-premise deployment options' },
                  { icon: Zap, text: 'Custom API rate limits' },
                ].map((item) => (
                  <li key={item.text} className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-white/10 flex items-center justify-center">
                      <item.icon className="w-5 h-5 text-primary" />
                    </div>
                    <span className="text-asgard-200">{item.text}</span>
                  </li>
                ))}
              </ul>
              <Link to="/gov">
                <Button size="lg" variant="secondary">
                  Contact Enterprise Sales
                  <ArrowRight className="w-4 h-4 ml-2" />
                </Button>
              </Link>
            </motion.div>
            
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              className="grid grid-cols-2 gap-4"
            >
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <div className="text-3xl font-bold text-white mb-1">24/7</div>
                <div className="text-sm text-asgard-400">Dedicated Support</div>
              </div>
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <div className="text-3xl font-bold text-white mb-1">99.99%</div>
                <div className="text-sm text-asgard-400">Uptime SLA</div>
              </div>
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <div className="text-3xl font-bold text-white mb-1">∞</div>
                <div className="text-sm text-asgard-400">API Calls</div>
              </div>
              <div className="p-6 rounded-2xl bg-white/5 border border-white/10">
                <div className="text-3xl font-bold text-white mb-1">ITAR</div>
                <div className="text-sm text-asgard-400">Compliant</div>
              </div>
            </motion.div>
          </div>
        </div>
      </section>

      {/* FAQ Section */}
      <section className="section-padding">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-asgard-900 dark:text-white mb-4">
              Frequently Asked Questions
            </h2>
          </motion.div>

          <div className="max-w-3xl mx-auto">
            <div className="grid gap-6">
              {faqs.map((faq, index) => (
                <motion.div
                  key={faq.question}
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ delay: index * 0.1 }}
                >
                  <Card>
                    <CardContent className="p-6">
                      <h3 className="text-title text-asgard-900 dark:text-white mb-2">
                        {faq.question}
                      </h3>
                      <p className="text-body text-asgard-500 dark:text-asgard-400">
                        {faq.answer}
                      </p>
                    </CardContent>
                  </Card>
                </motion.div>
              ))}
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
