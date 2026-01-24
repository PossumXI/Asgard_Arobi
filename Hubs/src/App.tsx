import { Routes, Route } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import Layout from './components/Layout';
import LoadingScreen from './components/LoadingScreen';

const HubsHome = lazy(() => import('./pages/HubsHome'));
const CivilianHub = lazy(() => import('./pages/CivilianHub'));
const MilitaryHub = lazy(() => import('./pages/MilitaryHub'));
const InterstellarHub = lazy(() => import('./pages/InterstellarHub'));
const StreamView = lazy(() => import('./pages/StreamView'));
const MissionHub = lazy(() => import('./pages/MissionHub'));

export default function App() {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<HubsHome />} />
          <Route path="civilian/*" element={<CivilianHub />} />
          <Route path="military/*" element={<MilitaryHub />} />
          <Route path="interstellar/*" element={<InterstellarHub />} />
          <Route path="stream/:streamId" element={<StreamView />} />
        </Route>
        {/* Mission Hub - Standalone with tiered access */}
        <Route path="/missions/*" element={<MissionHub />} />
      </Routes>
    </Suspense>
  );
}
