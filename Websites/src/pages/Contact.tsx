import { useState } from 'react';
import { motion } from 'framer-motion';
import {
  Mail,
  MapPin,
  Phone,
  Send,
  User,
  Building2,
  Globe,
  Linkedin,
  Twitter,
  Github,
  MessageSquare,
  CheckCircle
} from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
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

const contactInfo = [
  {
    icon: Mail,
    title: 'Email',
    value: 'Gaetano@aura-genesis.org',
    link: 'mailto:Gaetano@aura-genesis.org',
    color: 'text-blue-500',
    bg: 'bg-blue-500/10'
  },
  {
    icon: Building2,
    title: 'Company',
    value: 'Arobi - ASGARD Product Line',
    link: null,
    color: 'text-purple-500',
    bg: 'bg-purple-500/10'
  },
  {
    icon: User,
    title: 'Founder & CEO',
    value: 'Gaetano Comparcola',
    link: null,
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10'
  }
];

const offices = [
  {
    city: 'San Francisco',
    country: 'United States',
    type: 'Global Headquarters'
  },
  {
    city: 'London',
    country: 'United Kingdom',
    type: 'European Operations'
  },
  {
    city: 'Singapore',
    country: 'Singapore',
    type: 'Asia-Pacific Hub'
  },
  {
    city: 'Dubai',
    country: 'UAE',
    type: 'Middle East Office'
  }
];

const socialLinks = [
  { icon: Linkedin, label: 'LinkedIn', href: '#' },
  { icon: Twitter, label: 'Twitter', href: '#' },
  { icon: Github, label: 'GitHub', href: '#' }
];

const subjectOptions = [
  'General Inquiry',
  'Product Demo Request',
  'Partnership Opportunities',
  'Government & Defense',
  'Technical Support',
  'Media & Press',
  'Careers'
];

export default function Contact() {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    subject: 'General Inquiry',
    message: ''
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSubmitted, setIsSubmitted] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    
    // Simulate form submission
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    setIsSubmitting(false);
    setIsSubmitted(true);
    setFormData({ name: '', email: '', subject: 'General Inquiry', message: '' });
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    setFormData(prev => ({
      ...prev,
      [e.target.name]: e.target.value
    }));
  };

  return (
    <div className="overflow-hidden">
      {/* Hero Section */}
      <section className="relative pt-24 pb-16">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
          <div className="absolute top-0 right-1/4 w-[500px] h-[500px] bg-primary/10 rounded-full blur-3xl" />
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
                <MessageSquare className="w-4 h-4" />
                Get in Touch
              </span>
            </motion.div>
            
            <motion.h1
              variants={fadeInUp}
              className="text-display-lg sm:text-display-xl text-asgard-900 dark:text-white mb-6"
            >
              Contact <span className="gradient-text">ASGARD</span>
            </motion.h1>
            
            <motion.p
              variants={fadeInUp}
              className="text-body-lg text-asgard-500 dark:text-asgard-400 max-w-2xl mx-auto"
            >
              Have questions about our autonomous systems? Want to explore partnership opportunities? 
              We'd love to hear from you.
            </motion.p>
          </motion.div>
        </div>
      </section>

      {/* Contact Info Cards */}
      <section className="py-12">
        <div className="container-wide">
          <div className="grid md:grid-cols-3 gap-6">
            {contactInfo.map((info, index) => (
              <motion.div
                key={info.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full hover-lift">
                  <CardContent className="p-6">
                    <div className={cn('w-12 h-12 rounded-xl flex items-center justify-center mb-4', info.bg)}>
                      <info.icon className={cn('w-6 h-6', info.color)} />
                    </div>
                    <h3 className="text-sm font-medium text-asgard-500 dark:text-asgard-400 mb-1">
                      {info.title}
                    </h3>
                    {info.link ? (
                      <a
                        href={info.link}
                        className="text-lg font-semibold text-asgard-900 dark:text-white hover:text-primary transition-colors"
                      >
                        {info.value}
                      </a>
                    ) : (
                      <p className="text-lg font-semibold text-asgard-900 dark:text-white">
                        {info.value}
                      </p>
                    )}
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Contact Form & Info Section */}
      <section className="section-padding">
        <div className="container-wide">
          <div className="grid lg:grid-cols-2 gap-12">
            {/* Contact Form */}
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
            >
              <Card>
                <CardContent className="p-8">
                  <h2 className="text-2xl font-bold text-asgard-900 dark:text-white mb-6">
                    Send us a Message
                  </h2>
                  
                  {isSubmitted ? (
                    <motion.div
                      initial={{ opacity: 0, scale: 0.95 }}
                      animate={{ opacity: 1, scale: 1 }}
                      className="text-center py-12"
                    >
                      <div className="w-16 h-16 rounded-full bg-success/10 flex items-center justify-center mx-auto mb-4">
                        <CheckCircle className="w-8 h-8 text-success" />
                      </div>
                      <h3 className="text-xl font-semibold text-asgard-900 dark:text-white mb-2">
                        Message Sent!
                      </h3>
                      <p className="text-asgard-500 dark:text-asgard-400 mb-6">
                        Thank you for reaching out. We'll get back to you within 24-48 hours.
                      </p>
                      <Button onClick={() => setIsSubmitted(false)} variant="outline">
                        Send Another Message
                      </Button>
                    </motion.div>
                  ) : (
                    <form onSubmit={handleSubmit} className="space-y-6">
                      <div className="grid sm:grid-cols-2 gap-6">
                        <Input
                          label="Your Name"
                          name="name"
                          placeholder="John Doe"
                          value={formData.name}
                          onChange={handleChange}
                          required
                        />
                        <Input
                          label="Email Address"
                          name="email"
                          type="email"
                          placeholder="john@example.com"
                          value={formData.email}
                          onChange={handleChange}
                          required
                        />
                      </div>
                      
                      <div className="space-y-2">
                        <label
                          htmlFor="subject"
                          className="block text-sm font-medium text-asgard-700 dark:text-asgard-300"
                        >
                          Subject
                        </label>
                        <select
                          id="subject"
                          name="subject"
                          value={formData.subject}
                          onChange={handleChange}
                          className={cn(
                            'flex h-11 w-full rounded-xl border bg-white px-4 py-2 text-base',
                            'border-asgard-200 dark:border-asgard-700',
                            'dark:bg-asgard-900',
                            'focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary',
                            'transition-all duration-200'
                          )}
                        >
                          {subjectOptions.map((option) => (
                            <option key={option} value={option}>
                              {option}
                            </option>
                          ))}
                        </select>
                      </div>
                      
                      <div className="space-y-2">
                        <label
                          htmlFor="message"
                          className="block text-sm font-medium text-asgard-700 dark:text-asgard-300"
                        >
                          Message
                        </label>
                        <textarea
                          id="message"
                          name="message"
                          rows={5}
                          placeholder="Tell us about your inquiry..."
                          value={formData.message}
                          onChange={handleChange}
                          required
                          className={cn(
                            'flex w-full rounded-xl border bg-white px-4 py-3 text-base resize-none',
                            'border-asgard-200 dark:border-asgard-700',
                            'dark:bg-asgard-900',
                            'placeholder:text-asgard-400 dark:placeholder:text-asgard-500',
                            'focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary',
                            'transition-all duration-200'
                          )}
                        />
                      </div>
                      
                      <Button type="submit" size="lg" className="w-full group" isLoading={isSubmitting}>
                        {!isSubmitting && (
                          <>
                            Send Message
                            <Send className="w-4 h-4 ml-2 group-hover:translate-x-1 transition-transform" />
                          </>
                        )}
                      </Button>
                    </form>
                  )}
                </CardContent>
              </Card>
            </motion.div>

            {/* Office Locations & Social */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              className="space-y-8"
            >
              {/* Worldwide Presence */}
              <div>
                <h2 className="text-2xl font-bold text-asgard-900 dark:text-white mb-6 flex items-center gap-3">
                  <Globe className="w-6 h-6 text-primary" />
                  Worldwide Presence
                </h2>
                <div className="grid sm:grid-cols-2 gap-4">
                  {offices.map((office, index) => (
                    <motion.div
                      key={office.city}
                      initial={{ opacity: 0, y: 10 }}
                      whileInView={{ opacity: 1, y: 0 }}
                      viewport={{ once: true }}
                      transition={{ delay: index * 0.1 }}
                    >
                      <Card>
                        <CardContent className="p-5">
                          <div className="flex items-start gap-3">
                            <MapPin className="w-5 h-5 text-primary mt-0.5 flex-shrink-0" />
                            <div>
                              <h3 className="font-semibold text-asgard-900 dark:text-white">
                                {office.city}
                              </h3>
                              <p className="text-sm text-asgard-500 dark:text-asgard-400">
                                {office.country}
                              </p>
                              <p className="text-xs text-primary mt-1">
                                {office.type}
                              </p>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    </motion.div>
                  ))}
                </div>
              </div>

              {/* About Arobi */}
              <Card className="bg-asgard-50/50 dark:bg-asgard-800/50">
                <CardContent className="p-6">
                  <h3 className="text-lg font-semibold text-asgard-900 dark:text-white mb-3">
                    About Arobi
                  </h3>
                  <p className="text-asgard-600 dark:text-asgard-300 mb-4">
                    Arobi is the technology company behind the ASGARD product line — a comprehensive suite 
                    of autonomous systems designed for planetary defense, humanitarian aid, and space exploration. 
                    Our mission is to protect humanity and extend our reach to the stars.
                  </p>
                  <div className="flex items-center gap-2 text-sm text-asgard-500 dark:text-asgard-400">
                    <Phone className="w-4 h-4" />
                    <span>Available 24/7 for enterprise inquiries</span>
                  </div>
                </CardContent>
              </Card>

              {/* Social Links */}
              <div>
                <h3 className="text-lg font-semibold text-asgard-900 dark:text-white mb-4">
                  Connect with Us
                </h3>
                <div className="flex gap-3">
                  {socialLinks.map((social) => (
                    <a
                      key={social.label}
                      href={social.href}
                      aria-label={social.label}
                      className={cn(
                        'w-12 h-12 rounded-xl flex items-center justify-center',
                        'bg-asgard-100 dark:bg-asgard-800',
                        'text-asgard-600 dark:text-asgard-300',
                        'hover:bg-primary hover:text-white',
                        'transition-all duration-200'
                      )}
                    >
                      <social.icon className="w-5 h-5" />
                    </a>
                  ))}
                </div>
              </div>
            </motion.div>
          </div>
        </div>
      </section>

      {/* Map/CTA Section */}
      <section className="section-padding bg-asgard-900 dark:bg-black">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center"
          >
            <h2 className="text-display text-white mb-4">
              Global Operations, Local Support
            </h2>
            <p className="text-lg text-asgard-300 max-w-2xl mx-auto mb-8">
              With offices across four continents, ASGARD provides round-the-clock support 
              for our partners and clients worldwide.
            </p>
            <div className="flex flex-wrap items-center justify-center gap-6 text-asgard-400">
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span>24/7 Support</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span>Multi-language Teams</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span>Government Certified</span>
              </div>
            </div>
          </motion.div>
        </div>
      </section>
    </div>
  );
}
