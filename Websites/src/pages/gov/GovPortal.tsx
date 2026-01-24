import { useState } from 'react';
import { Routes, Route, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { 
  Shield, 
  Building2, 
  Lock, 
  Key, 
  FileText, 
  Users,
  AlertTriangle,
  CheckCircle,
  ArrowRight,
  Fingerprint
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { useAuth } from '@/providers/AuthProvider';
// Utils imported for future use

function GovLanding() {
  const [showLogin, setShowLogin] = useState(false);
  const [email, setEmail] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const { signInWithFido2, registerFido2, user, isAuthenticated } = useAuth();

  const handleFido2SignIn = async () => {
    if (!email.trim()) {
      setStatusMessage('Email is required for FIDO2 sign-in.');
      return;
    }
    setIsSubmitting(true);
    setStatusMessage(null);
    try {
      await signInWithFido2(email.trim());
      setStatusMessage('Authentication successful.');
      setShowLogin(false);
    } catch (err) {
      setStatusMessage((err as Error).message ?? 'FIDO2 authentication failed.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleFido2Register = async () => {
    setIsSubmitting(true);
    setStatusMessage(null);
    try {
      await registerFido2();
      setStatusMessage('Security key registered successfully.');
    } catch (err) {
      setStatusMessage((err as Error).message ?? 'FIDO2 registration failed.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="relative pt-32 pb-20 lg:pt-40 lg:pb-32">
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-b from-asgard-900 via-asgard-950 to-black" />
          <div className="absolute inset-0 bg-[url('/grid.svg')] opacity-10" />
        </div>

        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="max-w-3xl mx-auto text-center"
          >
            <div className="mb-6 flex justify-center">
              <div className="p-4 rounded-2xl bg-primary/20 border border-primary/30">
                <Building2 className="w-12 h-12 text-primary" />
              </div>
            </div>
            
            <h1 className="text-display-lg text-white mb-6">
              Government Portal
            </h1>
            
            <p className="text-body-lg text-asgard-300 mb-8">
              Secure access to ASGARD systems for authorized government entities. 
              FIDO2 hardware token required for authentication.
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 mb-12">
              <Button 
                size="lg" 
                onClick={() => setShowLogin(true)}
                className="group"
              >
                <Key className="w-5 h-5 mr-2" />
                Authenticate with FIDO2
              </Button>
              <Link to="/gov/request">
                <Button size="lg" variant="outline" className="border-white/20 text-white hover:bg-white/10">
                  Request Access
                </Button>
              </Link>
            </div>

            {/* Trust Badges */}
            <div className="flex flex-wrap items-center justify-center gap-6 text-asgard-400">
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span className="text-sm">FedRAMP High</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span className="text-sm">ITAR Compliant</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-success" />
                <span className="text-sm">IL5 Certified</span>
              </div>
            </div>
          </motion.div>
        </div>
      </section>

      {/* Login Modal */}
      {showLogin && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm">
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="w-full max-w-md mx-4"
          >
            <Card>
              <CardHeader className="text-center">
                <div className="mx-auto mb-4 w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center">
                  <Fingerprint className="w-8 h-8 text-primary" />
                </div>
                <CardTitle>FIDO2 Authentication</CardTitle>
                <CardDescription>
                  Insert your hardware security key and touch to authenticate
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <Input
                  label="Government Email"
                  placeholder="you@gov.example"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />

                <div className="p-4 rounded-xl bg-asgard-50 dark:bg-asgard-800 text-center">
                  <div className="text-asgard-500 dark:text-asgard-400">
                    Insert your hardware key and touch to continue.
                  </div>
                </div>

                {statusMessage && (
                  <div className="text-center text-sm text-asgard-500 dark:text-asgard-400">
                    {statusMessage}
                  </div>
                )}
                
                <div className="text-center text-sm text-asgard-500 dark:text-asgard-400">
                  Supported: YubiKey, CAC/PIV, FIDO2 keys
                </div>

                <div className="flex gap-3">
                  <Button 
                    variant="outline" 
                    className="flex-1"
                    onClick={() => setShowLogin(false)}
                  >
                    Cancel
                  </Button>
                  <Button className="flex-1" onClick={handleFido2SignIn} disabled={isSubmitting}>
                    {isSubmitting ? 'Authenticating...' : 'Authenticate'}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </motion.div>
        </div>
      )}

      {isAuthenticated && (
        <section className="container-wide pb-16">
          <Card>
            <CardHeader>
              <CardTitle>Security Key Enrollment</CardTitle>
              <CardDescription>
                Register a FIDO2 hardware key for {user?.email}.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {statusMessage && (
                <div className="text-sm text-asgard-500 dark:text-asgard-400">
                  {statusMessage}
                </div>
              )}
              <Button onClick={handleFido2Register} disabled={isSubmitting}>
                <Fingerprint className="w-4 h-4 mr-2" />
                {isSubmitting ? 'Registering...' : 'Register Security Key'}
              </Button>
            </CardContent>
          </Card>
        </section>
      )}

      {/* Features */}
      <section className="py-20 bg-asgard-950">
        <div className="container-wide">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-center mb-16"
          >
            <h2 className="text-display text-white mb-4">
              Enterprise-Grade Capabilities
            </h2>
            <p className="text-body-lg text-asgard-400 max-w-2xl mx-auto">
              Purpose-built for defense, intelligence, and emergency management agencies.
            </p>
          </motion.div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[
              {
                icon: Shield,
                title: 'Real-Time Intelligence',
                description: 'Access raw satellite feeds and AI-processed threat assessments in real-time.',
              },
              {
                icon: Users,
                title: 'Hunoid Coordination',
                description: 'Direct command and control of humanoid units for emergency response.',
              },
              {
                icon: AlertTriangle,
                title: 'Priority Alerting',
                description: 'Receive classified threat alerts with sub-second delivery.',
              },
              {
                icon: Lock,
                title: 'Secure Communications',
                description: 'End-to-end encrypted Gaga Chat for inter-agency coordination.',
              },
              {
                icon: FileText,
                title: 'Audit & Compliance',
                description: 'Complete audit trails for all operations and decisions.',
              },
              {
                icon: Key,
                title: 'API Integration',
                description: 'RESTful APIs for integration with existing defense systems.',
              },
            ].map((feature, index) => (
              <motion.div
                key={feature.title}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
              >
                <Card className="h-full bg-asgard-900 border-asgard-800">
                  <CardContent className="p-6">
                    <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-4">
                      <feature.icon className="w-6 h-6 text-primary" />
                    </div>
                    <h3 className="text-title text-white mb-2">{feature.title}</h3>
                    <p className="text-body text-asgard-400">{feature.description}</p>
                  </CardContent>
                </Card>
              </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Contact */}
      <section className="py-20 bg-black">
        <div className="container-wide">
          <div className="max-w-2xl mx-auto text-center">
            <h2 className="text-display text-white mb-4">
              Request Government Access
            </h2>
            <p className="text-body-lg text-asgard-400 mb-8">
              Contact our government relations team to begin the accreditation process.
            </p>
            <div className="grid sm:grid-cols-2 gap-4 text-left mb-8">
              <Card className="bg-asgard-900 border-asgard-800">
                <CardContent className="p-6">
                  <h3 className="text-title text-white mb-2">US Government</h3>
                  <p className="text-sm text-asgard-400 mb-1">govrelations@asgard.gov</p>
                  <p className="text-sm text-asgard-400">CAGE Code: XXXX1</p>
                </CardContent>
              </Card>
              <Card className="bg-asgard-900 border-asgard-800">
                <CardContent className="p-6">
                  <h3 className="text-title text-white mb-2">International</h3>
                  <p className="text-sm text-asgard-400 mb-1">international@asgard.dev</p>
                  <p className="text-sm text-asgard-400">NATO Stock Number available</p>
                </CardContent>
              </Card>
            </div>
            <Link to="/gov/request">
              <Button size="lg">
                Begin Request Process
                <ArrowRight className="w-4 h-4 ml-2" />
              </Button>
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}

function GovRequest() {
  return (
    <div className="min-h-screen pt-32 pb-20">
      <div className="container-wide">
        <div className="max-w-2xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            <Link 
              to="/gov" 
              className="inline-flex items-center gap-2 text-sm text-asgard-500 hover:text-asgard-900 dark:hover:text-white mb-8"
            >
              ← Back to Government Portal
            </Link>
            
            <Card>
              <CardHeader>
                <CardTitle>Request Government Access</CardTitle>
                <CardDescription>
                  Complete this form to begin the accreditation process. 
                  A government relations specialist will contact you within 2 business days.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form className="space-y-4">
                  <div className="grid sm:grid-cols-2 gap-4">
                    <Input label="First Name" placeholder="John" />
                    <Input label="Last Name" placeholder="Doe" />
                  </div>
                  <Input label="Official Email" type="email" placeholder="john.doe@agency.gov" />
                  <Input label="Agency/Organization" placeholder="Department of Defense" />
                  <Input label="Position/Title" placeholder="Program Manager" />
                  <div>
                    <label className="block text-sm font-medium text-asgard-700 dark:text-asgard-300 mb-2">
                      Use Case Description
                    </label>
                    <textarea
                      className="w-full h-32 rounded-xl border border-asgard-200 dark:border-asgard-700 bg-white dark:bg-asgard-900 px-4 py-3 text-base focus:outline-none focus:ring-2 focus:ring-primary/50"
                      placeholder="Please describe your intended use of ASGARD systems..."
                    />
                  </div>
                  <Button type="submit" className="w-full">
                    Submit Request
                    <ArrowRight className="w-4 h-4 ml-2" />
                  </Button>
                </form>
              </CardContent>
            </Card>
          </motion.div>
        </div>
      </div>
    </div>
  );
}

export default function GovPortal() {
  return (
    <Routes>
      <Route index element={<GovLanding />} />
      <Route path="request" element={<GovRequest />} />
    </Routes>
  );
}
