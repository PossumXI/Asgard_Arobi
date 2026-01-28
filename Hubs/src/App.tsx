import { Routes, Route } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { ErrorBoundary } from './components/ErrorBoundary';
import Layout from './components/Layout';
import LoadingScreen from './components/LoadingScreen';

const HubsHome = lazy(() => import('./pages/HubsHome'));
const CivilianHub = lazy(() => import('./pages/CivilianHub'));
const MilitaryHub = lazy(() => import('./pages/MilitaryHub'));
const InterstellarHub = lazy(() => import('./pages/InterstellarHub'));
const StreamView = lazy(() => import('./pages/StreamView'));
const MissionHub = lazy(() => import('./pages/MissionHub'));

// Route-level error boundary wrapper for lazy-loaded components
const withErrorBoundary = (Component: React.ComponentType) => (
  <ErrorBoundary>
    <Component />
  </ErrorBoundary>
);

export default function App() {
  return (
    <ErrorBoundary>
      <Suspense fallback={<LoadingScreen />}>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={withErrorBoundary(HubsHome)} />
            <Route path="civilian/*" element={withErrorBoundary(CivilianHub)} />
            <Route path="military/*" element={withErrorBoundary(MilitaryHub)} />
            <Route path="interstellar/*" element={withErrorBoundary(InterstellarHub)} />
            <Route path="stream/:streamId" element={withErrorBoundary(StreamView)} />
          </Route>
          {/* Mission Hub - Standalone with tiered access */}
          <Route path="/missions/*" element={withErrorBoundary(MissionHub)} />
        </Routes>
      </Suspense>
    </ErrorBoundary>
  );
}
