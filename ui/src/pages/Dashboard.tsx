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

export function Dashboard() {
  const { t } = useI18n();
  const needsTenant = useNeedsTenant();
  const enabled = !needsTenant;

  const { data: summary, isFetching, isLoading } = useQuery({
    queryKey: queryKeys.dashboardSummary,
    queryFn: getDashboardSummary,
    enabled,
    refetchInterval: (q) => {
      const health = q.state.data?.health;
      if (health === 'warning' || health === 'critical') return 5_000;
      return false;
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
        hero
        title={t('nav.dashboard')}
        subtitle={t('dashboard.subtitle')}
        actions={
          <>
            <Link to="/vms" className="btn-secondary">{t('dashboard.quickActions')}</Link>
            <Link to="/vms" className="btn-primary">{t('dashboard.deployVm')}</Link>
          </>
        }
      />

      <RefreshingPanel isLoading={isLoading}>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-gutter">
          <SurfaceCard className="flex flex-col justify-between" padding="lg">
            <div>
              <p className="font-label text-on-surface-variant">{t('nav.vms')}</p>
              <h3 className="font-headline text-headline-lg font-bold mt-2 text-on-surface">{vms.total}</h3>
            </div>
            <div className="mt-4 flex items-center gap-2">
              <span className="bg-success-muted text-success px-2 py-1 rounded-full font-label-sm flex items-center gap-1 border border-success/20">
                <span className="w-2 h-2 rounded-full bg-success animate-vf-pulse" />
                {runningPct}%
              </span>
              <span className="text-body-base text-on-surface-variant">{t('dashboard.running')}</span>
            </div>
          </SurfaceCard>

          <SurfaceCard className="flex flex-col justify-between" padding="lg">
            <div>
              <p className="font-label text-on-surface-variant">{t('nav.networks')}</p>
              <h3 className="font-headline text-headline-lg font-bold mt-2 text-on-surface">
                {summary?.networks.total ?? 0}
              </h3>
            </div>
            <div className="mt-4">
              <div className="w-full bg-surface-container h-2 rounded-full overflow-hidden">
                <div
                  className="bg-primary h-full transition-all duration-500"
                  style={{
                    width: `${Math.min(100, ((summary?.vpcs.total ?? 0) / Math.max(1, summary?.networks.total ?? 1)) * 100)}%`,
                  }}
                />
              </div>
              <p className="text-body-base text-on-surface-variant mt-2 text-right">
                {summary?.vpcs.total ?? 0} VPCs
              </p>
            </div>
          </SurfaceCard>

          <SurfaceCard className="flex flex-col justify-between sm:col-span-2 lg:col-span-1" padding="lg">
            <div>
              <div className="flex justify-between items-start">
                <p className="font-label text-on-surface-variant">Health</p>
                {summary?.health === 'critical' || errors > 0 ? (
                  <span className="bg-error-container/30 text-error px-2 py-1 rounded-full font-label-sm border border-error/30">ALERT</span>
                ) : summary?.health === 'warning' ? (
                  <span className="bg-warning-muted text-warning px-2 py-1 rounded-full font-label-sm border border-warning/20">SYNC</span>
                ) : (
                  <span className="bg-success-muted text-success px-2 py-1 rounded-full font-label-sm border border-success/20">OK</span>
                )}
              </div>
              <h3 className="font-headline text-headline-lg font-bold mt-2 text-on-surface">{healthLabel}</h3>
            </div>
            <Link to="/vms" className="mt-4 text-primary font-label-md hover:underline inline-flex items-center gap-1">
              View compute
            </Link>
          </SurfaceCard>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-12 gap-gutter mt-gutter auto-rows-[minmax(180px,auto)]">
          <div className="md:col-span-12 grid grid-cols-2 md:grid-cols-4 gap-gutter items-stretch">
            {stats.map((stat) => (
              <SurfaceCard
                key={stat.label}
                className="min-h-[120px] h-full [&>div:last-child]:h-full [&>div:last-child]:flex [&>div:last-child]:items-center [&>div:last-child]:justify-center"
                padding="md"
              >
                <div className="flex flex-col items-center text-center gap-1.5">
                  <stat.icon size={20} className="text-primary" />
                  <span className="font-label text-on-surface-variant text-[10px] leading-tight">{stat.label}</span>
                  <span className="font-headline text-headline-md font-bold text-on-surface">{stat.value}</span>
                </div>
              </SurfaceCard>
            ))}
          </div>

          <SurfaceCard className="md:col-span-5 flex flex-col overflow-hidden min-h-[280px]" padding="md" title="Recent activity">
            <div className="flex-1 overflow-y-auto space-y-2 -mx-1 px-1">
              {recentVms.length === 0 ? (
                <p className="text-on-surface-variant text-sm">{t('dashboard.subtitle')}</p>
              ) : (
                recentVms.map((vm) => {
                  const isError = vm.state?.toLowerCase() === 'error';
                  const isRunning = vm.state?.toLowerCase() === 'running';
                  const Icon = isError ? AlertTriangle : isRunning ? CheckCircle : RefreshCw;
                  const iconClass = isError ? 'text-error' : isRunning ? 'text-tertiary' : 'text-secondary';
                  return (
                    <Link
                      key={vm.name}
                      to={vm.path}
                      className="bg-surface-container p-3 rounded-lg border border-outline-variant flex gap-3 items-start hover:bg-surface-variant transition-colors"
                    >
                      <Icon size={18} className={clsx('mt-0.5 shrink-0', iconClass)} />
                      <div className="min-w-0">
                        <div className="font-semibold text-body-semibold text-on-surface truncate">{vm.display_name || vm.name}</div>
                        <div className="font-data-mono text-on-surface-variant text-xs truncate">
                          {vm.name} · {vm.state}
                        </div>
                      </div>
                    </Link>
                  );
                })
              )}
            </div>
          </SurfaceCard>

          <SurfaceCard className="md:col-span-7" padding="lg" title={t('dashboard.quickActions')}>
            <div className="flex flex-wrap gap-3">
              {[
                { to: '/vms', label: t('dashboard.deployVm') },
                { to: '/volumes', label: t('dashboard.createVolume') },
                { to: '/vpcs', label: t('dashboard.newVpc') },
                { to: '/snapshots', label: t('dashboard.snapshot') },
              ].map((a) => (
                <Link key={a.to} to={a.to} className="btn-secondary text-body-semibold">
                  {a.label}
                </Link>
              ))}
            </div>
          </SurfaceCard>
        </div>
      </RefreshingPanel>
    </div>
  );
}
