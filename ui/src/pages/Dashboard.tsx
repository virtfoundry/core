import { HardDrive, Globe, Shield, Server, Cpu, MemoryStick, CheckCircle, AlertTriangle, RefreshCw } from 'lucide-react';
import clsx from 'clsx';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { getDashboardSummary } from '../lib/platform-api';
import { useNeedsTenant } from '../store/hooks';
import { queryKeys } from '../lib/query-keys';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { useI18n, type TranslationKey } from '../lib/i18n';
import { PageHeader, SurfaceCard } from '../components/shell';

import { OnboardingChecklist } from '../components/OnboardingChecklist';
import { getRecentActions } from '../lib/preview-prefs';
import { ComingSoonBadge } from '../components/ComingSoonBadge';
import { useMemo } from 'react';
import {
  useRealtimeConnected,
} from '../hooks/useRealtimeEvents';

function usageTone(pct: number): string {
  if (pct >= 85) return 'bg-error';
  if (pct >= 60) return 'bg-warning';
  return 'bg-primary';
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—';
  if (bytes === 0) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(value >= 100 || i === 0 ? 0 : 1)} ${units[i]}`;
}

function UsageBar({ pct, collectedAt }: { pct: number; collectedAt?: string }) {
  const clamped = Math.min(100, Math.max(0, pct));
  return (
    <div className="w-full mt-1" title={collectedAt}>
      <div className="w-full bg-surface-container h-1.5 rounded-full overflow-hidden">
        <div
          className={`${usageTone(clamped)} h-full transition-all duration-500`}
          style={{ width: `${clamped}%` }}
        />
      </div>
      <p className="text-[10px] text-on-surface-variant mt-1 font-data-mono text-center">
        {clamped}%
      </p>
    </div>
  );
}

type NodeHealthTone = {
  badge: string;
  label: TranslationKey;
};

function nodeHealth(total: number, ready: number): NodeHealthTone | null {
  if (total <= 0) return null;
  if (ready >= total) {
    return { badge: 'bg-success-muted text-success border-success/20', label: 'dashboard.hostsNodesAllReady' };
  }
  if (ready <= 0) {
    return { badge: 'bg-error-container/30 text-error border-error/30', label: 'dashboard.hostsNodesNoneReady' };
  }
  return { badge: 'bg-warning-muted text-warning border-warning/20', label: 'dashboard.hostsNodesSomeDegraded' };
}

function NodesHealthBadge({ total, ready }: { total: number; ready: number }) {
  const { t } = useI18n();
  const tone = nodeHealth(total, ready);
  if (!tone) return null;
  const label = tone.label === 'dashboard.hostsNodesSomeDegraded'
    ? t(tone.label).replace('{n}', String(total - ready))
    : t(tone.label);
  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded-full font-label-sm border whitespace-nowrap ${tone.badge}`}
      title={`${ready} / ${total}`}
    >
      {label}
    </span>
  );
}

const ADDON_LABEL_KEY: Record<string, TranslationKey> = {
  kubevirt: 'dashboard.addonsKubevirt',
  cdi: 'dashboard.addonsCdi',
  multus: 'dashboard.addonsMultus',
  'metrics-server': 'dashboard.addonsMetricsServer',
  networking: 'dashboard.addonsNetworking',
  'cert-manager': 'dashboard.addonsCertManager',
};

function addonTone(status: string): { dot: string; pulse: boolean; labelKey: TranslationKey } {
  if (status === 'ok') return { dot: 'bg-success', pulse: true, labelKey: 'dashboard.addonsStatusOk' };
  if (status === 'absent') return { dot: 'bg-transparent border border-on-surface-variant/60', pulse: false, labelKey: 'dashboard.addonsStatusAbsent' };
  return { dot: 'bg-warning', pulse: true, labelKey: 'dashboard.addonsStatusUnknown' };
}

function AddonsHealthStrip({ addons }: { addons: { addons: Array<{ name: string; status: string; detail?: string }>; checked_at: string } }) {
  const { t } = useI18n();
  const available = addons.addons.filter((a) => a.status === 'ok');
  if (available.length === 0) return null;
  return (
    <div
      className="mt-4 pt-4 border-t border-outline-variant"
      title={addons.checked_at}
    >
      <p className="text-[10px] font-label uppercase text-on-surface-variant mb-2">
        {t('dashboard.addonsTitle')}
      </p>
      <div className="flex flex-wrap gap-x-4 gap-y-2">
        {available.map((a) => {
          const tone = addonTone(a.status);
          const labelKey = ADDON_LABEL_KEY[a.name] ?? null;
          const label = labelKey ? t(labelKey) : a.name;
          const statusLabel = t(tone.labelKey);
          return (
            <div
              key={a.name}
              className="inline-flex items-center gap-1.5"
              title={a.detail ?? statusLabel}
            >
              <span className={`w-2 h-2 rounded-full ${tone.dot} ${tone.pulse ? 'animate-vf-pulse' : ''}`} />
              <span className="text-xs text-on-surface-variant font-data-mono">
                {label}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

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
    // Cluster usage comes from metrics.k8s.io polled every ~15s — there is no
    // platform event for it, so the dashboard must keep polling regardless of
    // /ws/events health. The WS still invalidates VM counts on transitions
    // (see lib/realtime-invalidation), so the two paths compose.
    refetchInterval: () => (wsConnected ? 10_000 : 5_000),
  });

  const vms = summary?.vms ?? { total: 0, running: 0, error: 0 };
  const running = vms.running ?? 0;
  const errors = vms.error ?? 0;
  const runningPct = vms.total ? Math.round((running / vms.total) * 100) : 0;

  const stats = [
    { label: t('nav.volumes'), value: summary?.volumes.total ?? 0, icon: HardDrive },
    { label: t('nav.vpcs'), value: summary?.vpcs.total ?? 0, icon: Globe },
    { label: t('nav.securityGroups'), value: summary?.security_groups.total ?? 0, icon: Shield },
  ];

  const recentVms = summary?.recent_activity ?? [];
  const hosts = summary?.hosts;
  const hostsUnavailable = hosts === undefined;
  const cpuCores = hosts ? (hosts.cpu_allocatable_millicores / 1000).toFixed(1) : '0.0';
  const storage = summary?.storage;
  const storageUsed = storage ? formatBytes(storage.used_bytes) : '—';
  const storageTotal = storage ? formatBytes(storage.total_bytes) : '—';
  const storagePct = storage && storage.total_bytes > 0
    ? Math.round((storage.used_bytes / storage.total_bytes) * 100)
    : null;
  const memGiB = hosts ? Math.round(hosts.memory_allocatable_bytes / (1024 * 1024 * 1024)) : 0;
  const kubeletSummary = hosts?.kubelet_versions?.length
    ? hosts.kubelet_versions.length === 1
      ? hosts.kubelet_versions[0]
      : `${hosts.kubelet_versions.length} versões`
    : '';
  const osImage = hosts?.os_images?.[0] ?? '';
  const arch = hosts?.os_architectures?.[0] ?? '';
  const usage = hosts?.usage;
  const cpuUsageCores = usage ? (usage.cpu_usage_millicores / 1000).toFixed(1) : null;
  const memUsageGiB = usage ? Math.round(usage.memory_usage_bytes / (1024 * 1024 * 1024)) : null;
  const cpuUsagePct = usage && hosts && hosts.cpu_allocatable_millicores > 0
    ? Math.round((usage.cpu_usage_millicores / hosts.cpu_allocatable_millicores) * 100)
    : null;
  const memUsagePct = usage && hosts && hosts.memory_allocatable_bytes > 0
    ? Math.round((usage.memory_usage_bytes / hosts.memory_allocatable_bytes) * 100)
    : null;

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

      <OnboardingChecklist />

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
            <SurfaceCard
              className="min-h-[120px] h-full [&>div:last-child]:h-full [&>div:last-child]:flex [&>div:last-child]:items-center [&>div:last-child]:justify-center"
              padding="md"
            >
              <div className="flex flex-col items-center text-center gap-1.5">
                <HardDrive size={20} className="text-primary" />
                {storage && storagePct !== null ? (
                  <>
                    <span className="font-label text-on-surface-variant text-[10px] leading-tight whitespace-nowrap">
                      {t('dashboard.storageUsedOf').replace('{used}', storageUsed).replace('{total}', storageTotal)}
                    </span>
                    <span className="font-headline text-headline-md font-bold text-on-surface">
                      {storagePct}%
                    </span>
                    <span className="text-[10px] text-on-surface-variant font-data-mono">
                      {t('dashboard.storageCount').replace('{n}', String(storage.count))}
                    </span>
                  </>
                ) : (
                  <>
                    <span className="font-label text-on-surface-variant text-[10px] leading-tight">
                      {t('dashboard.storageTitle')}
                    </span>
                    <span className="font-headline text-headline-md font-bold text-on-surface">—</span>
                  </>
                )}
              </div>
            </SurfaceCard>
          </div>

          <SurfaceCard className="md:col-span-5 flex flex-col overflow-hidden min-h-[280px]" padding="md" title="Recent activity">
            <div className="flex-1 overflow-y-auto space-y-2 -mx-1 px-1">
              {localRecent.length > 0 && (
                <div className="mb-3">
                  <p className="text-[10px] font-label uppercase text-on-surface-variant mb-1 flex items-center gap-2">
                    Local · ⌘K <ComingSoonBadge label={t('preview.localOnly')} />
                  </p>
                  {localRecent.map((r) => (
                    <Link
                      key={r.id}
                      to={r.path}
                      className="block bg-surface-container p-2 rounded-lg border border-outline-variant mb-1 text-sm hover:bg-surface-variant"
                    >
                      {r.label}
                    </Link>
                  ))}
                </div>
              )}
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

          <SurfaceCard className="md:col-span-12" padding="md" title={t('dashboard.hostsTitle')}>
            {hostsUnavailable ? (
              <p className="text-on-surface-variant text-sm py-4">
                {t('dashboard.hostsUnavailable')}
              </p>
            ) : (
              <>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-gutter items-stretch">
                  <div className="flex flex-col items-center text-center gap-1.5 h-full">
                    <Server size={20} className="text-primary" />
                    <span className="font-label text-on-surface-variant text-[10px] leading-tight">
                      {t('dashboard.hostsNodes')}
                    </span>
                    <span className="font-headline text-headline-md font-bold text-on-surface">
                      {hosts?.nodes ?? 0}
                    </span>
                    <div className="flex-1" />
                  </div>
                  <div className="flex flex-col items-center text-center gap-1.5 h-full">
                    <CheckCircle size={20} className="text-tertiary" />
                    <span className="font-label text-on-surface-variant text-[10px] leading-tight">
                      {t('dashboard.hostsNodesReady')}
                    </span>
                    <span className="font-headline text-headline-md font-bold text-on-surface">
                      {hosts?.nodes_ready ?? 0}
                    </span>
                    <NodesHealthBadge
                      total={hosts?.nodes ?? 0}
                      ready={hosts?.nodes_ready ?? 0}
                    />
                    <div className="flex-1" />
                  </div>
                  <div
                    className="flex flex-col items-center text-center gap-1.5 min-w-0 h-full"
                    title={t('dashboard.hostsAllocatableHint')}
                  >
                    <Cpu size={20} className="text-primary" />
                    <span className="font-label text-on-surface-variant text-[10px] leading-tight whitespace-nowrap">
                      {t('dashboard.hostsCpu')}
                    </span>
                    <span className="font-headline text-headline-md font-bold text-on-surface">
                      {cpuCores}
                    </span>
                    <div className="flex-1" />
                    {cpuUsageCores !== null ? (
                      <span className="text-[10px] text-on-surface-variant font-data-mono whitespace-nowrap">
                        {cpuUsageCores} {t('dashboard.hostsUsage')}
                      </span>
                    ) : (
                      <span className="text-[10px] text-on-surface-variant" />
                    )}
                    {cpuUsagePct !== null ? (
                      <UsageBar pct={cpuUsagePct} collectedAt={usage?.collected_at} />
                    ) : (
                      <span
                        className="text-[10px] text-on-surface-variant italic"
                        title={t('dashboard.hostsUsageUnavailable')}
                      >
                        {t('dashboard.hostsUsageUnavailable')}
                      </span>
                    )}
                  </div>
                  <div
                    className="flex flex-col items-center text-center gap-1.5 min-w-0 h-full"
                    title={t('dashboard.hostsAllocatableHint')}
                  >
                    <MemoryStick size={20} className="text-primary" />
                    <span className="font-label text-on-surface-variant text-[10px] leading-tight whitespace-nowrap">
                      {t('dashboard.hostsMemory')}
                    </span>
                    <span className="font-headline text-headline-md font-bold text-on-surface">
                      {memGiB} GiB
                    </span>
                    <div className="flex-1" />
                    {memUsageGiB !== null ? (
                      <span className="text-[10px] text-on-surface-variant font-data-mono whitespace-nowrap">
                        {memUsageGiB} GiB {t('dashboard.hostsUsage')}
                      </span>
                    ) : (
                      <span className="text-[10px] text-on-surface-variant" />
                    )}
                    {memUsagePct !== null ? (
                      <UsageBar pct={memUsagePct} collectedAt={usage?.collected_at} />
                    ) : (
                      <span
                        className="text-[10px] text-on-surface-variant italic"
                        title={t('dashboard.hostsUsageUnavailable')}
                      >
                        {t('dashboard.hostsUsageUnavailable')}
                      </span>
                    )}
                  </div>
                </div>
                {(kubeletSummary || osImage || arch) && (
                  <p
                    className="text-xs text-on-surface-variant mt-4 font-data-mono"
                    title={hosts?.collected_at}
                  >
                    {[
                      kubeletSummary && `${t('dashboard.hostsKubelet')}: ${kubeletSummary}`,
                      osImage && `${t('dashboard.hostsOs')}: ${osImage}`,
                      arch && `${t('dashboard.hostsArch')}: ${arch}`,
                    ]
                      .filter(Boolean)
                      .join(' · ')}
                  </p>
                )}
                {summary?.addons && <AddonsHealthStrip addons={summary.addons} />}
              </>
            )}
          </SurfaceCard>
        </div>
      </RefreshingPanel>
    </div>
  );
}

