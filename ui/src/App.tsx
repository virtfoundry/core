import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Layout } from './components/Layout';
import { Login } from './pages/Login';
import { Dashboard } from './pages/Dashboard';
import { VMs } from './pages/VMs';
import { VMDetail } from './pages/VMDetail';
import { Volumes } from './pages/Volumes';
import { Networks } from './pages/Networks';
import { PublicNetwork } from './pages/PublicNetwork';
import { SecurityGroups } from './pages/SecurityGroups';
import { Snapshots } from './pages/Snapshots';
import { VMSnapshots } from './pages/VMSnapshots';
import { Tenants } from './pages/Tenants';
import { VPCs } from './pages/VPCs';
import { Templates } from './pages/Templates';
import { SSHKeys } from './pages/SSHKeys';
import { IAM } from './pages/IAM';
import { VMConsole } from './pages/VMConsole';

import { I18nProvider } from './lib/i18n';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 0,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      retry: 1,
    },
  },
});

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = localStorage.getItem('jwt_token') !== null;
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/console" element={
            <ProtectedRoute>
              <VMConsole />
            </ProtectedRoute>
          } />
          <Route path="/" element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="tenants" element={<Tenants />} />
            <Route path="vms" element={<VMs />} />
            <Route path="vms/:name" element={<VMDetail />} />
            <Route path="templates" element={<Templates />} />
            <Route path="ssh-keys" element={<SSHKeys />} />
            <Route path="volumes" element={<Volumes />} />
            <Route path="vpcs" element={<VPCs />} />
            <Route path="networks/public" element={<PublicNetwork />} />
            <Route path="networks" element={<Networks />} />
            <Route path="security-groups" element={<SecurityGroups />} />
            <Route path="snapshots" element={<Snapshots />} />
            <Route path="vm-snapshots" element={<VMSnapshots />} />
            <Route path="iam" element={<IAM />} />
          </Route>
        </Routes>
      </BrowserRouter>
      </I18nProvider>
    </QueryClientProvider>
  );
}
