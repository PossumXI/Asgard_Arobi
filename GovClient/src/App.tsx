/**
 * ASGARD Government Client - Main Application
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AccessCodeGate } from './pages/auth/AccessCodeGate';
import { GovLogin } from './pages/auth/GovLogin';
import { GovDashboard } from './pages/dashboard/GovDashboard';
import { CommandHub } from './pages/command/CommandHub';
import { MissionHub } from './pages/mission/MissionHub';
import { MilitaryHub } from './pages/military/MilitaryHub';
import { AdminHub } from './pages/admin/AdminHub';
import { SecurityMonitor } from './pages/security/SecurityMonitor';
import { ValkyrieControl } from './pages/valkyrie/ValkyrieControl';
import { HunoidControl } from './pages/hunoid/HunoidControl';
import { PricillaGuidance } from './pages/pricilla/PricillaGuidance';
import { Layout } from './components/layout/Layout';
import { useAuthStore } from './stores/authStore';
import { LoadingScreen } from './components/LoadingScreen';
import { Toaster } from './components/ui/Toaster';

function App() {
  const [isLoading, setIsLoading] = useState(true);
  const { isAuthenticated, hasAccessCode, checkStoredAuth } = useAuthStore();

  useEffect(() => {
    const init = async () => {
      await checkStoredAuth();
      setIsLoading(false);
    };
    init();
  }, [checkStoredAuth]);

  if (isLoading) {
    return <LoadingScreen />;
  }

  // Access code gate - first layer of security
  if (!hasAccessCode) {
    return (
      <BrowserRouter>
        <AccessCodeGate />
        <Toaster />
      </BrowserRouter>
    );
  }

  // Authentication gate - second layer
  if (!isAuthenticated) {
    return (
      <BrowserRouter>
        <GovLogin />
        <Toaster />
      </BrowserRouter>
    );
  }

  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<GovDashboard />} />
          <Route path="/command" element={<CommandHub />} />
          <Route path="/mission" element={<MissionHub />} />
          <Route path="/military" element={<MilitaryHub />} />
          <Route path="/admin" element={<AdminHub />} />
          <Route path="/security" element={<SecurityMonitor />} />
          <Route path="/valkyrie" element={<ValkyrieControl />} />
          <Route path="/hunoid" element={<HunoidControl />} />
          <Route path="/pricilla" element={<PricillaGuidance />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </Layout>
      <Toaster />
    </BrowserRouter>
  );
}

export default App;
