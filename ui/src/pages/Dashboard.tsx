import { Plus, Rocket } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getDashboardSummary, listServiceOfferings, listVKSClusters, listVMs, listVolumes } from '../lib/platform-api';
import { useNeedsTenant } from '../store/hooks';
import { queryKeys } from '../lib/query-keys';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
} from '../components/shell';
import { StatusBadge, toneClass, type StatusTone } from '../components/StatusBadge';
import clsx from 'clsx';
import { StatsCard } from '../components/StatsCard';
import { ClusterOverview } from '../components/ClusterOverview';

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

  // Same keys as the list pages (shared cache); no extra polling here.
  const vmsQ = useQuery({ queryKey: queryKeys.vms, queryFn: listVMs, enabled });
  const offQ = useQuery({ queryKey: queryKeys.offerings, queryFn: listServiceOfferings, enabled });
  const volQ = useQuery({ queryKey: queryKeys.volumes, queryFn: listVolumes, enabled });
  const vksQ = useQuery({ queryKey: queryKeys.vksClusters, queryFn: listVKSClusters, enabled });

  const vmList = vmsQ.data?.vms;
  const volList = volQ.data?.volumes ?? [];
  const clusters = vksQ.data?.clusters ?? [];

  const d = useMemo(() => {
    const offs = new Map((offQ.data?.service_offerings ?? []).map((o) => [o.name, o]));
    const list = vmList ?? [];
    const by = { running: 0, stopped: 0, error: 0, other: 0 };
    let cpuRun = 0, cpuAll = 0, memRun = 0, memAll = 0;
    for (const v of list) {
      const o = v.service_offering_id ? offs.get(v.service_offering_id) : undefined;
      const cpu = v.cpu || o?.cpu || 0;
      const mem = v.memory_mi || o?.memory_mi || 0;
      const st = v.state.toLowerCase();
      cpuAll += cpu; memAll += mem;
      if (st === 'running') { by.running++; cpuRun += cpu; memRun += mem; }
      else if (st === 'stopped') by.stopped++;
      else if (st === 'error') by.error++;
      else by.other++;
    }
    const volAll = volList.reduce((a, v) => a + (v.size_gi || 0), 0);
    const volUsed = volList.reduce((a, v) => a + (v.vm_id ? v.size_gi || 0 : 0), 0);
    return { total: list.length, by, cpuRun, cpuAll, memRun: memRun / 1024, memAll: memAll / 1024, volAll, volUsed };
  }, [vmList, offQ.data, volList]);

  const vms = summary?.vms ?? { total: 0, running: 0, error: 0 };
  const total = vmList ? d.total : vms.total;
  const running = vmList ? d.by.running : vms.running ?? 0;
  const errors = vmList ? d.by.error : vms.error ?? 0;
  const runningPct = total ? Math.round((running / total) * 100) : 0;
  const ready = clusters.filter((c) => c.phase === 'Ready' || c.phase === 'ControlPlaneReady').length;
  const gi = (n: number) => `${Math.round(n * 10) / 10}`;
  const pct = (a: number, b: number) => (b ? Math.round((a / b) * 100) : 0);

  const counts = [
    { label: t('nav.volumes'), value: summary?.volumes.total ?? volList.length },
    { label: t('nav.vpcs'), value: summary?.vpcs.total ?? 0 },
    { label: t('nav.networks'), value: summary?.networks.total ?? 0 },
    { label: t('nav.securityGroups'), value: summary?.security_groups.total ?? 0 },
  ].filter((c) => c.value > 0);

  const bars = [
    d.cpuAll > 0 && { label: 'vCPU', val: `${d.cpuRun} / ${d.cpuAll}`, pct: pct(d.cpuRun, d.cpuAll) },
    d.memAll > 0 && { label: t('dashboard.memory'), val: `${gi(d.memRun)} / ${gi(d.memAll)} GiB`, pct: pct(d.memRun, d.memAll) },
    d.volAll > 0 && { label: t('dashboard.storageAttached'), val: `${d.volUsed} / ${d.volAll} GiB`, pct: pct(d.volUsed, d.volAll) },
  ].filter(Boolean) as Array<{ label: string; val: string; pct: number }>;

  const segs = [
    { key: 'running', n: d.by.running, cls: 'bg-ok-fg', label: t('status.running') },
    { key: 'other', n: d.by.other, cls: 'bg-accent', label: t('dashboard.stateOther') },
    { key: 'stopped', n: d.by.stopped, cls: 'bg-off-fg', label: t('status.stopped') },
    { key: 'error', n: d.by.error, cls: 'bg-bad-fg', label: t('status.error') },
  ].filter((x) => x.n > 0);

  // Recent activity derived from real created/updated timestamps (summary rows carry none).
  const when = (a?: string, b?: string) => {
    const t1 = Date.parse(a ?? ''), t2 = Date.parse(b ?? '');
    return Math.max(t1 > 9e11 ? t1 : 0, t2 > 9e11 ? t2 : 0);
  };
  const recent = [
    ...(vmList ?? []).map((v) => ({
      key: `vm-${v.name}`, name: v.display_name || v.name, sub: v.display_name && v.display_name !== v.name ? v.name : '',
      type: 'vm', path: `/vms/${v.name}`, at: when(v.updated_at, v.created_at), state: v.state, tone: undefined as StatusTone | undefined,
    })),
    ...clusters.map((c) => ({
      key: `vks-${c.name}`, name: c.name, sub: '', type: 'vks', path: `/clusters/${encodeURIComponent(c.name)}`,
      at: when(undefined, c.created_at), state: c.phase ?? '', tone: (c.phase === 'Ready' || c.phase === 'ControlPlaneReady' ? 'ok' : 'warn') as StatusTone,
    })),
  ].sort((a, b) => b.at - a.at).slice(0, 6);

  const isEmpty = !!summary && !!vmsQ.data && total === 0 && clusters.length === 0 && volList.length === 0 && counts.length === 0;

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

  const ago = (ms: number) => (ms ? formatDistanceToNow(new Date(ms), { addSuffix: true, locale: dateLocale }) : '—');

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
        {isEmpty ? (
          <SurfaceCard>
            <div className="flex flex-col items-center text-center gap-3 py-10">
              <span className="p-3 rounded-full bg-accent-soft text-accent"><Rocket size={22} aria-hidden /></span>
              <h2 className="m-0 text-lg font-semibold">{t('dashboard.emptyTitle')}</h2>
              <p className="m-0 max-w-md text-on-surface-variant">{t('dashboard.emptyBody')}</p>
              <Link to="/vms" className="btn-primary mt-1">
                <Plus size={16} strokeWidth={2.2} aria-hidden />
                {t('dashboard.newVm')}
              </Link>
            </div>
          </SurfaceCard>
        ) : (
        <div className="flex flex-col gap-6">
          <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
            {total > 0 && (
              <StatsCard
                title={t('dashboard.runningVms')}
                value={running}
                of={t('dashboard.of').replace('{n}', String(total))}
                progress={runningPct}
                variant={errors > 0 ? 'warning' : 'success'}
              />
            )}
            {d.cpuAll > 0 && (
              <StatsCard title={t('dashboard.vcpuInUse')} value={d.cpuRun} of={`/ ${d.cpuAll}`} progress={pct(d.cpuRun, d.cpuAll)} />
            )}
            {d.memAll > 0 && (
              <StatsCard title={t('dashboard.memory')} value={gi(d.memRun)} of={`/ ${gi(d.memAll)} GiB`} progress={pct(d.memRun, d.memAll)} />
            )}
            {clusters.length > 0 && (
              <StatsCard
                title={t('nav.clusters')}
                value={ready}
                of={t('dashboard.of').replace('{n}', String(clusters.length))}
                progress={pct(ready, clusters.length)}
                variant="success"
              />
            )}
            {counts.map((c) => (
              <StatsCard key={c.label} title={c.label} value={c.value} />
            ))}
          </section>

          {(segs.length > 0 || bars.length > 0) && (
            <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(320px,1fr))]">
              {segs.length > 0 && (
                <SurfaceCard title={t('dashboard.statesTitle')}>
                  <div className="flex h-3 rounded-full overflow-hidden bg-track" role="img" aria-label={segs.map((x) => `${x.label} ${x.n}`).join(', ')}>
                    {segs.map((x) => (
                      <div key={x.key} className={x.cls} style={{ width: `${(x.n / d.total) * 100}%` }} />
                    ))}
                  </div>
                  <ul className="m-0 mt-4 p-0 list-none flex flex-col gap-2 text-sm">
                    {segs.map((x) => (
                      <li key={x.key} className="flex items-center gap-2">
                        <span className={clsx('w-2 h-2 rounded-full', x.cls)} aria-hidden />
                        <span className="flex-1">{x.label}</span>
                        <span className="tabular-nums text-on-surface-variant">{x.n}</span>
                      </li>
                    ))}
                  </ul>
                </SurfaceCard>
              )}
              {bars.length > 0 && (
                <SurfaceCard title={t('dashboard.allocTitle')}>
                  <div className="flex flex-col gap-4">
                    {bars.map((b) => (
                      <div key={b.label} className="flex flex-col gap-1.5">
                        <div className="flex justify-between text-sm">
                          <span>{b.label}</span>
                          <span className="tabular-nums text-on-surface-variant">{b.val}</span>
                        </div>
                        <div className="h-1.5 rounded-full bg-track" role="progressbar" aria-label={b.label} aria-valuenow={b.pct} aria-valuemin={0} aria-valuemax={100}>
                          <div className={clsx('h-1.5 rounded-full', b.pct >= 85 ? 'bg-warn-fg' : 'bg-accent')} style={{ width: `${b.pct}%` }} />
                        </div>
                      </div>
                    ))}
                  </div>
                </SurfaceCard>
              )}
            </section>
          )}

          {summary && <ClusterOverview summary={summary} />}

          {(recent.length > 0 || localRecent.length > 0) && (
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
                  {recent.map((r) => (
                    <PageTableRow key={r.key}>
                      <PageTableTd>
                        <Link to={r.path} className="font-medium hover:underline">{r.name}</Link>
                        {r.sub && <div className="font-mono text-xs text-on-surface-variant">{r.sub}</div>}
                      </PageTableTd>
                      <PageTableTd mono className="text-on-surface-variant uppercase">{r.type}</PageTableTd>
                      <PageTableTd mono className="text-on-surface-variant">{ago(r.at)}</PageTableTd>
                      <PageTableTd><StatusBadge status={r.state} tone={r.tone} /></PageTableTd>
                    </PageTableRow>
                  ))}
                </PageTableBody>
              </PageTable>
            </SurfaceCard>
          )}
        </div>
        )}
      </RefreshingPanel>
    </div>
  );
}
