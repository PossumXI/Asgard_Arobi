import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Eye, EyeOff, ArrowRight, Fingerprint } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Card, CardContent } from '@/components/ui/Card';
import { useAuth } from '@/providers/AuthProvider';
import { useToast } from '@/components/ui/Toaster';

const signInSchema = z.object({
  email: z.string().email('Please enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
  accessCode: z.string().optional(),
});

type SignInFormData = z.infer<typeof signInSchema>;

export default function SignIn() {
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [requiresFido2, setRequiresFido2] = useState(false);
  const { signIn, signInWithFido2 } = useAuth();
  const { error: showError } = useToast();

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<SignInFormData>({
    resolver: zodResolver(signInSchema),
  });
  const emailValue = watch('email');

  const onSubmit = async (data: SignInFormData) => {
    setIsLoading(true);
    try {
      setRequiresFido2(false);
      await signIn(data.email, data.password, data.accessCode);
    } catch (err) {
      const code = (err as { code?: string })?.code;
      if (code === 'FIDO2_REQUIRED') {
        setRequiresFido2(true);
        showError('Security key required', 'Use your FIDO2 device to complete sign in.');
      } else if (code === 'ACCESS_CODE_REQUIRED') {
        showError('Access code required', 'Enter your clearance code to continue.');
      } else if (code === 'ACCESS_CODE_EXPIRED') {
        showError('Access code expired', 'Request a new clearance code.');
      } else if (code === 'ACCESS_CODE_REVOKED') {
        showError('Access code revoked', 'Contact administration for a new code.');
      } else if (code === 'ACCESS_CODE_EXHAUSTED') {
        showError('Access code exhausted', 'Request a new clearance code.');
      } else if (code === 'ACCESS_CODE_SCOPE') {
        showError('Access code scope mismatch', 'Use the correct clearance code for this portal.');
      } else if (code === 'ACCESS_CODE_INVALID') {
        showError('Invalid access code', 'Check your code and try again.');
      } else if (code === 'EMAIL_NOT_VERIFIED') {
        showError('Email verification required', 'Check your inbox to verify your email address.');
      } else {
        showError('Sign in failed', 'Please check your credentials and try again.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  const handleFido2 = async () => {
    if (!emailValue) {
      showError('Email required', 'Enter your email to continue with FIDO2.');
      return;
    }
    setIsLoading(true);
    try {
      await signInWithFido2(emailValue);
    } catch (err) {
      showError('FIDO2 sign-in failed', (err as Error).message ?? 'Try again.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center pt-16 pb-12 px-4">
      <div className="absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-b from-asgard-50 via-white to-white dark:from-asgard-950 dark:via-asgard-950 dark:to-asgard-950" />
        <div className="absolute top-1/4 left-1/4 w-[400px] h-[400px] bg-primary/5 rounded-full blur-3xl" />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full max-w-md"
      >
        <div className="text-center mb-8">
          <Link to="/" className="inline-flex items-center gap-2 mb-6">
            <svg
              viewBox="0 0 36 36"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              className="w-10 h-10 text-primary"
            >
              <circle cx="18" cy="18" r="16" stroke="currentColor" strokeWidth="2" />
              <path d="M18 6L26 22H10L18 6Z" fill="currentColor" />
              <circle cx="18" cy="28" r="3" fill="currentColor" />
            </svg>
            <span className="font-semibold text-xl text-asgard-900 dark:text-white">
              ASGARD
            </span>
          </Link>
          <h1 className="text-headline text-asgard-900 dark:text-white mb-2">
            Welcome back
          </h1>
          <p className="text-body text-asgard-500 dark:text-asgard-400">
            Sign in to access your dashboard
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div className="relative">
                <Input
                  {...register('email')}
                  type="email"
                  label="Email"
                  placeholder="you@example.com"
                  error={errors.email?.message}
                  autoComplete="email"
                />
              </div>

              <div className="relative">
                <Input
                  {...register('password')}
                  type={showPassword ? 'text' : 'password'}
                  label="Password"
                  placeholder="Enter your password"
                  error={errors.password?.message}
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-9 text-asgard-400 hover:text-asgard-600 dark:hover:text-asgard-300"
                >
                  {showPassword ? (
                    <EyeOff className="w-5 h-5" />
                  ) : (
                    <Eye className="w-5 h-5" />
                  )}
                </button>
              </div>

              <div>
                <Input
                  {...register('accessCode')}
                  type="text"
                  label="Access Code (Gov/Admin)"
                  placeholder="AG-XXXX-XXXX"
                />
                <p className="mt-1 text-xs text-asgard-500">
                  Required for clearance-based access.
                </p>
              </div>

              <div className="flex items-center justify-between">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    className="w-4 h-4 rounded border-asgard-300 text-primary focus:ring-primary"
                  />
                  <span className="text-sm text-asgard-600 dark:text-asgard-400">
                    Remember me
                  </span>
                </label>
                <Link
                  to="/forgot-password"
                  className="text-sm text-primary hover:text-primary-600"
                >
                  Forgot password?
                </Link>
              </div>

              <Button type="submit" className="w-full" isLoading={isLoading}>
                Sign In
                <ArrowRight className="w-4 h-4 ml-2" />
              </Button>
            </form>

            <div className="mt-4 space-y-3">
              {requiresFido2 && (
                <div className="rounded-lg border border-primary/30 bg-primary/5 px-3 py-2 text-sm text-primary">
                  Government access requires FIDO2 authentication.
                </div>
              )}
              <Button
                type="button"
                variant="outline"
                className="w-full"
                onClick={handleFido2}
                disabled={!emailValue || isLoading}
              >
                <Fingerprint className="w-4 h-4 mr-2" />
                Use Security Key
              </Button>
            </div>

            <div className="relative my-6">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-asgard-200 dark:border-asgard-700" />
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-white dark:bg-asgard-900 text-asgard-500">
                  or continue with
                </span>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <Button variant="outline" type="button">
                <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24">
                  <path
                    fill="currentColor"
                    d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  />
                  <path
                    fill="currentColor"
                    d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  />
                  <path
                    fill="currentColor"
                    d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                  />
                  <path
                    fill="currentColor"
                    d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                  />
                </svg>
                Google
              </Button>
              <Button variant="outline" type="button">
                <svg className="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
                </svg>
                GitHub
              </Button>
            </div>
          </CardContent>
        </Card>

        <p className="text-center text-sm text-asgard-500 dark:text-asgard-400 mt-6">
          Don't have an account?{' '}
          <Link to="/signup" className="text-primary hover:text-primary-600 font-medium">
            Sign up
          </Link>
        </p>
      </motion.div>
    </div>
  );
}
