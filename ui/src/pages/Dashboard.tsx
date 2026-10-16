import { HardDrive, Globe, Shield, Network, CheckCircle, AlertTriangle, RefreshCw } from 'lucide-react';
import clsx from 'clsx';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getDashboardSummary } from '../lib/platform-api';
import { useNeedsTenant } from '../store/hooks';
import { queryKeys } from '../lib/query-keys';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { useI18n } from '../lib/i18n';
import { PageHeader, SurfaceCard } from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

import { OnboardingChecklist } from '../components/OnboardingChecklist';
import { getRecentActions } from '../lib/preview-prefs';
import { ComingSoonBadge } from '../components/ComingSoonBadge';
import { useMemo } from 'react';
import {
  POLL_WS_HEALTHY_SLOW_MS,
  realtimePollInterval,
  useRealtimeConnected,
} from '../hooks/useRealtimeEvents';

export function Dashboard() {
  const { t } = useI18n();
  const needsTenant = useNeedsTenant();
  const enabled = !needsTenant;
  const wsConnected = useRealtimeConnected();
  const localRecent = useMemo(() => getRecentActions(), []);

  const { data: summary, isFetching, isLoading } = useQuery({
    queryKey: queryKeys.dashboardSummary,
    queryFn: getDashboardSummary,
    enabled,
    refetchInterval: (q) => {
      const health = q.state.data?.health;
      const unhealthy = health === 'warning' || health === 'critical';
      return realtimePollInterval(wsConnected, unhealthy, {
        downMs: 5_000,
        healthyMs: POLL_WS_HEALTHY_SLOW_MS,
      });
    },
  });

  const vms = summary?.vms ?? { total: 0, running: 0, error: 0 };
  const running = vms.running ?? 0;
  const errors = vms.error ?? 0;
  const runningPct = vms.total ? Math.round((running / vms.total) * 100) : 0;

  const stats = [
    { label: t('nav.volumes'), value: summary?.volumes.total ?? 0, icon: HardDrive },
    { label: t('nav.vpcs'), value: summary?.vpcs.total ?? 0, icon: Globe },
    { label: t('nav.securityGroups'), value: summary?.security_groups.total ?? 0, icon: Shield },
    { label: t('nav.networks'), value: summary?.networks.total ?? 0, icon: Network },
  ];

  const recentVms = summary?.recent_activity ?? [];

  if (needsTenant) {
    return (
      <div className="text-center py-16 text-on-error-container">
        {t('dashboard.selectTenant')}
      </div>
    );
  }

  const healthLabel =
    summary?.health === 'critical'
      ? `${errors} VM errors`
      : summary?.health === 'warning'
        ? 'Transitions in progress'
        : 'All systems nominal';

  return (
    <div className="space-y-6 md:space-y-8">
      <PageHeader
        title={t('nav.dashboard')}
        subtitle={t('dashboard.subtitle')}
        actions={
          <>
            <Link to="/vms" className="btn-secondary">{t('dashboard.quickActions')}</Link>
            <Link to="/vms" className="btn-primary">{t('dashboard.deployVm')}</Link>
          </>
        }
      />

      <OnboardingChecklist />

      <RefreshingPanel isLoading={isLoading}>
        <section className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-5 gap-4">
          <div className="vf-card p-5 flex flex-col gap-3">
            <p className="text-body-sm text-on-surface-variant">{t('nav.vms')}</p>
            <p className="flex items-baseline gap-2">
              <span className="font-mono text-3xl font-medium tracking-tight text-on-surface">{running}</span>
              <span className="text-body-sm text-on-surface-variant">/ {vms.total} {t('dashboard.running')}</span>
            </p>
            <div className="h-1.5 rounded-full bg-surface-container-high overflow-hidden">
              <div className="h-full bg-success transition-all duration-500" style={{ width: `${runningPct}%` }} />
            </div>
          </div>
          {stats.map((stat) => (
            <div key={stat.label} className="vf-card p-5 flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <p className="text-body-sm text-on-surface-variant">{stat.label}</p>
                <span className="p-2 rounded-lg bg-primary-container/10 text-primary"><stat.icon size={16} /></span>
              </div>
              <p className="font-mono text-3xl font-medium tracking-tight text-on-surface">{stat.value}</p>
            </div>
          ))}
        </section>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mt-4">
          <SurfaceCard className="lg:col-span-2" padding="none" title={t('nav.vms')}>
            <ul className="divide-y divide-card-border">
              {localRecent.map((r) => (
                <li key={r.id}>
                  <Link to={r.path} className="flex items-center gap-3 px-6 py-3 text-sm hover:bg-surface-container-low">
                    <span className="w-2 h-2 rounded-full bg-primary-container shrink-0" />
                    <span className="flex-1 truncate">{r.label}</span>
                    <ComingSoonBadge label={t('preview.localOnly')} />
                  </Link>
                </li>
              ))}
              {recentVms.length === 0 && localRecent.length === 0 ? (
                <li className="px-6 py-8 text-sm text-on-surface-variant">{t('dashboard.subtitle')}</li>
              ) : (
                recentVms.map((vm) => {
                  const isError = vm.state?.toLowerCase() === 'error';
                  const isRunning = vm.state?.toLowerCase() === 'running';
                  const Icon = isError ? AlertTriangle : isRunning ? CheckCircle : RefreshCw;
                  const iconClass = isError ? 'text-error' : isRunning ? 'text-success' : 'text-warning';
                  return (
                    <li key={vm.name}>
                      <Link to={vm.path} className="flex items-center gap-3 px-6 py-3 hover:bg-surface-container-low transition-colors">
                        <Icon size={16} className={clsx('shrink-0', iconClass)} />
                        <span className="min-w-0 flex-1">
                          <span className="block font-medium text-on-surface truncate">{vm.display_name || vm.name}</span>
                          <span className="block font-mono text-xs text-on-surface-variant truncate">{vm.name}</span>
                        </span>
                        <StatusBadge status={vm.state ?? ''} />
                      </Link>
                    </li>
                  );
                })
              )}
            </ul>
          </SurfaceCard>

          <div className="flex flex-col gap-4">
            <SurfaceCard padding="lg" title="Health">
              <div className="flex items-center gap-3">
                <span
                  className={clsx(
                    'w-2.5 h-2.5 rounded-full',
                    summary?.health === 'critical' || errors > 0 ? 'bg-error' : summary?.health === 'warning' ? 'bg-warning' : 'bg-success',
                  )}
                />
                <span className="font-medium text-on-surface">{healthLabel}</span>
              </div>
              <Link to="/vms" className="mt-4 text-primary text-sm font-medium hover:underline inline-flex">
                {t('nav.vms')} →
              </Link>
            </SurfaceCard>
            <SurfaceCard padding="lg" title={t('dashboard.quickActions')}>
              <div className="flex flex-wrap gap-2">
                {[
                  { to: '/vms', label: t('dashboard.deployVm') },
                  { to: '/volumes', label: t('dashboard.createVolume') },
                  { to: '/vpcs', label: t('dashboard.newVpc') },
                  { to: '/snapshots', label: t('dashboard.snapshot') },
                ].map((a) => (
                  <Link key={a.to} to={a.to} className="btn-secondary h-9 text-sm">
                    {a.label}
                  </Link>
                ))}
              </div>
            </SurfaceCard>
          </div>
        </div>
      </RefreshingPanel>
    </div>
  );
}

