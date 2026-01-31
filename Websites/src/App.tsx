import { Routes, Route } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { Toaster } from '@/components/ui/Toaster';
import { ThemeProvider } from '@/providers/ThemeProvider';
import { AuthProvider } from '@/providers/AuthProvider';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import Layout from '@/components/layout/Layout';
import LoadingScreen from '@/components/ui/LoadingScreen';

// Lazy-loaded pages for optimal performance
const Landing = lazy(() => import('@/pages/Landing'));
const About = lazy(() => import('@/pages/About'));
const Features = lazy(() => import('@/pages/Features'));
const Pricing = lazy(() => import('@/pages/Pricing'));
const Pricilla = lazy(() => import('@/pages/Pricilla'));
const Valkyrie = lazy(() => import('@/pages/Valkyrie'));
const Contact = lazy(() => import('@/pages/Contact'));
const Giru = lazy(() => import('@/pages/Giru'));
// Aura Genesis ecosystem pages
const ApexOS = lazy(() => import('@/pages/ApexOS'));
const Foundation = lazy(() => import('@/pages/Foundation'));
const ICF = lazy(() => import('@/pages/ICF'));
// Auth and dashboard
const SignIn = lazy(() => import('@/pages/auth/SignIn'));
const SignUp = lazy(() => import('@/pages/auth/SignUp'));
const Dashboard = lazy(() => import('@/pages/dashboard/Dashboard'));
const GovPortal = lazy(() => import('@/pages/gov/GovPortal'));
const NotFound = lazy(() => import('@/pages/NotFound'));

// Route-level error boundary wrapper for lazy-loaded components
const withErrorBoundary = (Component: React.ComponentType) => (
  <ErrorBoundary>
    <Component />
  </ErrorBoundary>
);

export default function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <AuthProvider>
          <Suspense fallback={<LoadingScreen />}>
            <Routes>
              <Route path="/" element={<Layout />}>
                <Route index element={withErrorBoundary(Landing)} />
                <Route path="about" element={withErrorBoundary(About)} />
                <Route path="features" element={withErrorBoundary(Features)} />
                <Route path="pricilla" element={withErrorBoundary(Pricilla)} />
                <Route path="valkyrie" element={withErrorBoundary(Valkyrie)} />
                <Route path="contact" element={withErrorBoundary(Contact)} />
                <Route path="giru" element={withErrorBoundary(Giru)} />
                {/* Aura Genesis ecosystem routes */}
                <Route path="apex-os" element={withErrorBoundary(ApexOS)} />
                <Route path="foundation" element={withErrorBoundary(Foundation)} />
                <Route path="icf" element={withErrorBoundary(ICF)} />
                <Route path="pricing" element={withErrorBoundary(Pricing)} />
                <Route path="signin" element={withErrorBoundary(SignIn)} />
                <Route path="signup" element={withErrorBoundary(SignUp)} />
                <Route path="dashboard/*" element={withErrorBoundary(Dashboard)} />
                <Route path="gov/*" element={withErrorBoundary(GovPortal)} />
                <Route path="*" element={withErrorBoundary(NotFound)} />
              </Route>
            </Routes>
          </Suspense>
          <Toaster />
        </AuthProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}
