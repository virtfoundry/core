import { Plus } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getDashboardSummary } from '../lib/platform-api';
import { useNeedsTenant } from '../store/hooks';
import { queryKeys } from '../lib/query-keys';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
} from '../components/shell';
import { StatusBadge, toneClass, type StatusTone } from '../components/StatusBadge';
import { StatsCard } from '../components/StatsCard';

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
  const { t, dateLocale } = useI18n();
  const needsTenant = useNeedsTenant();
  const enabled = !needsTenant;
  const wsConnected = useRealtimeConnected();
  const localRecent = useMemo(() => getRecentActions(), []);

  const { data: summary, isLoading } = useQuery({
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
    { label: t('nav.volumes'), value: summary?.volumes.total ?? 0 },
    { label: t('nav.vpcs'), value: summary?.vpcs.total ?? 0 },
    { label: t('nav.securityGroups'), value: summary?.security_groups.total ?? 0 },
    { label: t('nav.networks'), value: summary?.networks.total ?? 0 },
  ];

  const recentVms = summary?.recent_activity ?? [];

  if (needsTenant) {
    return (
      <div className="text-center py-16 text-on-error-container">
        {t('dashboard.selectTenant')}
      </div>
    );
  }

  const health: { tone: StatusTone; label: string } | null = !summary
    ? null
    : summary.health === 'critical' || errors > 0
      ? { tone: 'bad', label: t('dashboard.healthCritical').replace('{n}', String(errors)) }
      : summary.health === 'warning'
        ? { tone: 'warn', label: t('dashboard.healthWarning') }
        : { tone: 'ok', label: t('dashboard.healthOk') };

  const ago = (iso?: string) => {
    if (!iso) return '—';
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? '—' : formatDistanceToNow(d, { addSuffix: true, locale: dateLocale });
  };

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('dashboard.title')}
        subtitle={t('dashboard.subtitle')}
        status={
          health && (
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-[3px] rounded-full text-[13px] font-medium ${toneClass[health.tone]}`}>
              <span className="w-1.5 h-1.5 rounded-full bg-current" aria-hidden />
              {health.label}
            </span>
          )
        }
        actions={
          <Link to="/vms" className="btn-primary">
            <Plus size={16} strokeWidth={2.2} aria-hidden />
            {t('dashboard.newVm')}
          </Link>
        }
      />

      <OnboardingChecklist />

      <RefreshingPanel isLoading={isLoading}>
        <div className="flex flex-col gap-6">
          <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
            <StatsCard
              title={t('dashboard.runningVms')}
              value={running}
              of={t('dashboard.of').replace('{n}', String(vms.total))}
              progress={runningPct}
              variant="success"
            />
            {stats.map((stat) => (
              <StatsCard key={stat.label} title={stat.label} value={stat.value} />
            ))}
          </section>

          <SurfaceCard padding="none" title={t('dashboard.recentActivity')}>
            <PageTable minWidth={560}>
              <PageTableHead>
                <PageTableTh>{t('dashboard.colResource')}</PageTableTh>
                <PageTableTh>{t('dashboard.colType')}</PageTableTh>
                <PageTableTh>{t('dashboard.colWhen')}</PageTableTh>
                <PageTableTh>Status</PageTableTh>
              </PageTableHead>
              <PageTableBody>
                {localRecent.map((r) => (
                  <PageTableRow key={r.id}>
                    <PageTableTd className="font-medium">
                      <Link to={r.path} className="hover:underline">{r.label}</Link>
                    </PageTableTd>
                    <PageTableTd className="text-on-surface-variant">—</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant">—</PageTableTd>
                    <PageTableTd><ComingSoonBadge label={t('preview.localOnly')} /></PageTableTd>
                  </PageTableRow>
                ))}
                {recentVms.map((vm) => (
                  <PageTableRow key={vm.name}>
                    <PageTableTd>
                      <Link to={vm.path} className="font-medium hover:underline">{vm.display_name || vm.name}</Link>
                      {vm.display_name && vm.display_name !== vm.name && (
                        <div className="font-mono text-xs text-on-surface-variant">{vm.name}</div>
                      )}
                    </PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant uppercase">{vm.type || 'vm'}</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant">{ago(vm.updated_at)}</PageTableTd>
                    <PageTableTd><StatusBadge status={vm.state ?? ''} /></PageTableTd>
                  </PageTableRow>
                ))}
                {recentVms.length === 0 && localRecent.length === 0 && (
                  <PageTableRow>
                    <PageTableTd colSpan={4} className="py-8 text-on-surface-variant">{t('dashboard.subtitle')}</PageTableTd>
                  </PageTableRow>
                )}
              </PageTableBody>
            </PageTable>
          </SurfaceCard>
        </div>
      </RefreshingPanel>
    </div>
  );
}
