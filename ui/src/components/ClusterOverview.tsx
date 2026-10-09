import { useI18n, type TranslationKey } from '../lib/i18n';
import type { DashboardSummary } from '../lib/platform-api';
import { SurfaceCard } from './shell';
import { StatsCard } from './StatsCard';
import { StatusBadge, type StatusTone } from './StatusBadge';

type Tone = 'success' | 'warning' | 'danger';

const GiB = 1024 ** 3;

function usageVariant(pct: number): Tone {
  if (pct >= 85) return 'danger';
  if (pct >= 60) return 'warning';
  return 'success';
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
  const i = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  const v = bytes / 1024 ** i;
  return `${v.toFixed(v >= 100 || i === 0 ? 0 : 1)} ${units[i]}`;
}

const ADDON_LABEL: Record<string, TranslationKey> = {
  kubevirt: 'dashboard.addonsKubevirt',
  cdi: 'dashboard.addonsCdi',
  multus: 'dashboard.addonsMultus',
  'metrics-server': 'dashboard.addonsMetricsServer',
  networking: 'dashboard.addonsNetworking',
  'cert-manager': 'dashboard.addonsCertManager',
};

/** Platform-wide node, storage and addon telemetry. Root only; renders nothing without data. */
export function ClusterOverview({ summary }: { summary: DashboardSummary }) {
  const { t } = useI18n();
  const { hosts, storage, addons } = summary;
  const okAddons = addons?.addons.filter((a) => a.status === 'ok') ?? [];
  if (!hosts && !storage && okAddons.length === 0) return null;

  const usage = hosts?.usage;
  const cpuPct = usage && hosts && hosts.cpu_allocatable_millicores > 0
    ? Math.round((usage.cpu_usage_millicores / hosts.cpu_allocatable_millicores) * 100) : undefined;
  const memPct = usage && hosts && hosts.memory_allocatable_bytes > 0
    ? Math.round((usage.memory_usage_bytes / hosts.memory_allocatable_bytes) * 100) : undefined;
  const storagePct = storage && storage.total_bytes > 0
    ? Math.round((storage.used_bytes / storage.total_bytes) * 100) : undefined;

  let nodeBadge: { tone: StatusTone; label: string } | null = null;
  if (hosts && hosts.nodes > 0) {
    nodeBadge = hosts.nodes_ready >= hosts.nodes
      ? { tone: 'ok', label: t('dashboard.hostsNodesAllReady') }
      : hosts.nodes_ready <= 0
        ? { tone: 'bad', label: t('dashboard.hostsNodesNoneReady') }
        : { tone: 'warn', label: t('dashboard.hostsNodesSomeDegraded').replace('{n}', String(hosts.nodes - hosts.nodes_ready)) };
  }

  const meta = hosts && [
    hosts.kubelet_versions?.length && `${t('dashboard.hostsKubelet')}: ${hosts.kubelet_versions.join(', ')}`,
    hosts.os_images?.[0] && `${t('dashboard.hostsOs')}: ${hosts.os_images[0]}`,
    hosts.os_architectures?.[0] && `${t('dashboard.hostsArch')}: ${hosts.os_architectures[0]}`,
  ].filter(Boolean).join(' · ');

  return (
    <SurfaceCard title={t('dashboard.hostsTitle')} headerAction={nodeBadge && <StatusBadge status="nodes" tone={nodeBadge.tone} label={nodeBadge.label} />}>
      <div className="flex flex-col gap-4">
        <div className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
          {hosts && (
            <>
              <StatsCard title={t('dashboard.hostsNodesReady')} value={hosts.nodes_ready} of={`/ ${hosts.nodes}`}
                progress={hosts.nodes ? Math.round((hosts.nodes_ready / hosts.nodes) * 100) : 0}
                variant={hosts.nodes_ready >= hosts.nodes ? 'success' : hosts.nodes_ready <= 0 ? 'danger' : 'warning'} />
              <StatsCard title={t('dashboard.hostsCpu')} value={(hosts.cpu_allocatable_millicores / 1000).toFixed(1)}
                of={usage ? `${(usage.cpu_usage_millicores / 1000).toFixed(1)} ${t('dashboard.hostsUsage').toLowerCase()}` : undefined}
                subtitle={usage ? undefined : t('dashboard.hostsUsageUnavailable')}
                progress={cpuPct} variant={cpuPct === undefined ? 'default' : usageVariant(cpuPct)} />
              <StatsCard title={t('dashboard.hostsMemory')} value={`${Math.round(hosts.memory_allocatable_bytes / GiB)} GiB`}
                of={usage ? `${Math.round(usage.memory_usage_bytes / GiB)} GiB ${t('dashboard.hostsUsage').toLowerCase()}` : undefined}
                subtitle={usage ? undefined : t('dashboard.hostsUsageUnavailable')}
                progress={memPct} variant={memPct === undefined ? 'default' : usageVariant(memPct)} />
            </>
          )}
          {storage && storagePct !== undefined && (
            <StatsCard title={t('dashboard.storageTitle')} value={`${storagePct}%`}
              subtitle={`${t('dashboard.storageUsedOf').replace('{used}', formatBytes(storage.used_bytes)).replace('{total}', formatBytes(storage.total_bytes))} · ${t('dashboard.storageCount').replace('{n}', String(storage.count))}`}
              progress={storagePct} variant={usageVariant(storagePct)} />
          )}
        </div>
        {meta && <p className="m-0 text-[13px] text-on-surface-variant font-data-mono" title={hosts?.collected_at}>{meta}</p>}
        {okAddons.length > 0 && (
          <div className="pt-4 border-t border-outline-variant" title={addons?.checked_at}>
            <p className="m-0 mb-2 text-[13px] text-on-surface-variant">{t('dashboard.addonsTitle')}</p>
            <div className="flex flex-wrap gap-2">
              {okAddons.map((a) => (
                <StatusBadge key={a.name} status="addon" tone="ok" pulse={false} label={ADDON_LABEL[a.name] ? t(ADDON_LABEL[a.name]) : a.name} />
              ))}
            </div>
          </div>
        )}
      </div>
    </SurfaceCard>
  );
}
