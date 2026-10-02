import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Download, Trash2, Copy, Check } from 'lucide-react';
import {
  getVKSCluster, deleteVKSCluster, downloadVKSKubeconfig,
} from '../lib/platform-api';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { StatusBadge } from '../components/StatusBadge';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { useCopyToClipboard } from '../hooks/useCopyToClipboard';
import {
  vksCanDownloadKubeconfig, vksNodesLabel, vksPhaseBadgeStatus,
} from '../lib/vks-display';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TabBar, InfoBanner,
} from '../components/shell';

type Tab = 'overview' | 'nodes' | 'networking' | 'status';

export function VKSClusterDetail() {
  const { name = '' } = useParams();
  const navigate = useNavigate();
  const { t, formatDate } = useI18n();
  const needsTenant = useNeedsTenant();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>('overview');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const { state: copyState, copy, reset: resetCopy } = useCopyToClipboard();

  const { data, isLoading, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vksCluster(name),
    queryFn: () => getVKSCluster(name),
    enabled: !needsTenant && !!name,
    refetchInterval: 10_000,
  });

  const cluster = data?.cluster;

  const downloadMutation = useMutation({
    mutationFn: async () => {
      const blob = await downloadVKSKubeconfig(name);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${name}.kubeconfig`;
      a.click();
      URL.revokeObjectURL(url);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteVKSCluster(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.vksClusters });
      navigate('/clusters');
    },
  });

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  if (isLoading && !cluster) {
    return <p className="text-sm text-on-surface-variant">{t('common.loading')}</p>;
  }

  if (error || !cluster) {
    return (
      <div className="space-y-4">
        <Link to="/clusters" className="inline-flex items-center gap-2 text-sm text-primary">
          <ArrowLeft size={16} /> {t('vks.title')}
        </Link>
        <p className="text-sm text-error">{(error as Error)?.message || t('vks.detail.notFound')}</p>
      </div>
    );
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: 'overview', label: t('vks.detail.tabOverview') },
    { id: 'nodes', label: t('vks.detail.tabNodes') },
    { id: 'networking', label: t('vks.detail.tabNetworking') },
    { id: 'status', label: t('vks.detail.tabStatus') },
  ];

  const endpoint = cluster.control_plane_endpoint || '—';
  const canKubeconfig = vksCanDownloadKubeconfig(cluster.phase);

  return (
    <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
      <div className="space-y-6">
        <div>
          <Link to="/clusters" className="mb-2 inline-flex items-center gap-2 text-sm text-primary-fixed-dim">
            <ArrowLeft size={16} /> {t('vks.title')}
          </Link>
          <PageHeader
            title={cluster.name}
            subtitle={`${t('vks.detail.product')} · ${cluster.kubernetes_version || '—'}${endpoint !== '—' ? ` · ${endpoint}` : ''}`}
            actions={
              <>
                <StatusBadge status={vksPhaseBadgeStatus(cluster.phase)} />
                <RefreshButton
                  compact
                  onRefresh={() => refetch()}
                  isFetching={isRefetching}
                  dataUpdatedAt={dataUpdatedAt}
                />
                <button
                  type="button"
                  className="btn-outline-sm"
                  disabled={!canKubeconfig || downloadMutation.isPending}
                  onClick={() => downloadMutation.mutate()}
                >
                  <Download size={16} /> {t('vks.downloadKubeconfig')}
                </button>
                <button type="button" className="btn-outline-sm text-error" onClick={() => setDeleteOpen(true)}>
                  <Trash2 size={16} /> {t('common.delete')}
                </button>
              </>
            }
          />
        </div>

        {!canKubeconfig && (
          <InfoBanner variant="warning">{t('vks.detail.kubeconfigWait')}</InfoBanner>
        )}

        <TabBar tabs={tabs} active={tab} onChange={setTab} />

        {tab === 'overview' && (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <SurfaceCard className="lg:col-span-2">
              <h2 className="mb-4 font-headline text-headline-md font-semibold text-on-surface">
                {t('vks.detail.clusterDetails')}
              </h2>
              <dl className="grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2">
                <div>
                  <dt className="text-on-surface-variant">{t('vks.col.name')}</dt>
                  <dd className="font-mono text-on-surface">{cluster.name}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.col.status')}</dt>
                  <dd className="text-on-surface">{cluster.phase || '—'}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.k8sVersion')}</dt>
                  <dd className="font-mono text-on-surface">{cluster.kubernetes_version || '—'}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.controlPlane')}</dt>
                  <dd className="text-on-surface">{t('vks.detail.controlPlaneManaged')}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.totalNodes')}</dt>
                  <dd className="text-on-surface">
                    {vksNodesLabel(cluster.ready_workers, cluster.workers?.count)}
                  </dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.tenantNs')}</dt>
                  <dd className="font-mono text-xs text-on-surface">{cluster.namespace || '—'}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.createdAt')}</dt>
                  <dd className="text-on-surface">{formatDate(cluster.created_at)}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.kubeconfigSecret')}</dt>
                  <dd className="font-mono text-xs text-on-surface">
                    {cluster.kubeconfig_secret_ref || '—'}
                  </dd>
                </div>
              </dl>
            </SurfaceCard>

            <SurfaceCard>
              <h2 className="mb-4 font-headline text-headline-md font-semibold text-on-surface">
                {t('vks.detail.credentials')}
              </h2>
              <p className="mb-4 text-sm text-on-surface-variant">{t('vks.detail.credentialsHint')}</p>
              <button
                type="button"
                className="btn-primary w-full justify-center"
                disabled={!canKubeconfig || downloadMutation.isPending}
                onClick={() => downloadMutation.mutate()}
              >
                <Download size={16} /> {t('vks.downloadKubeconfig')}
              </button>
              {endpoint !== '—' && (
                <div className="mt-4">
                  <p className="mb-1 text-xs text-on-surface-variant">{t('vks.col.location')}</p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 truncate rounded bg-surface-container px-2 py-1 font-mono text-xs">
                      {endpoint}
                    </code>
                    <button
                      type="button"
                      className="btn-outline-sm"
                      onClick={() => {
                        resetCopy();
                        void copy(endpoint);
                      }}
                      title={t('common.copy')}
                    >
                      {copyState === 'copied' ? <Check size={14} /> : <Copy size={14} />}
                    </button>
                  </div>
                </div>
              )}
            </SurfaceCard>
          </div>
        )}

        {tab === 'nodes' && (
          <SurfaceCard>
            <h2 className="mb-2 font-headline text-headline-md font-semibold text-on-surface">
              {t('vks.detail.nodePool')}
            </h2>
            <p className="mb-4 text-sm text-on-surface-variant">{t('vks.detail.nodePoolHint')}</p>
            <dl className="grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2">
              <div>
                <dt className="text-on-surface-variant">{t('vks.detail.poolName')}</dt>
                <dd className="font-mono text-on-surface">default-pool</dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('vks.col.nodes')}</dt>
                <dd className="text-on-surface">
                  {vksNodesLabel(cluster.ready_workers, cluster.workers?.count)}
                </dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('vks.col.machineType')}</dt>
                <dd className="font-mono text-on-surface">
                  {cluster.workers?.offering_ref?.name || '—'}
                </dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('vks.detail.nodeImage')}</dt>
                <dd className="font-mono text-on-surface">
                  {cluster.workers?.template_ref?.name || '—'}
                </dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('vks.form.workers')}</dt>
                <dd className="text-on-surface">{cluster.workers?.count ?? '—'}</dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('vks.detail.sshKeys')}</dt>
                <dd className="font-mono text-xs text-on-surface">
                  {(cluster.workers?.ssh_key_refs || []).map((r) => r.name).join(', ') || '—'}
                </dd>
              </div>
            </dl>
          </SurfaceCard>
        )}

        {tab === 'networking' && (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <SurfaceCard>
              <h2 className="mb-4 font-headline text-headline-md font-semibold text-on-surface">
                {t('vks.detail.apiAccess')}
              </h2>
              <dl className="grid grid-cols-1 gap-y-4 text-sm">
                <div>
                  <dt className="text-on-surface-variant">{t('vks.col.location')}</dt>
                  <dd className="font-mono text-on-surface">{endpoint}</dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.serviceType')}</dt>
                  <dd className="text-on-surface">
                    {cluster.control_plane?.service_type || 'NodePort'}
                  </dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.advertisedAddress')}</dt>
                  <dd className="font-mono text-on-surface">
                    {cluster.control_plane?.address || '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.form.nodePort')}</dt>
                  <dd className="font-mono text-on-surface">
                    {cluster.control_plane?.port || '—'}
                  </dd>
                </div>
              </dl>
            </SurfaceCard>
            <SurfaceCard>
              <h2 className="mb-4 font-headline text-headline-md font-semibold text-on-surface">
                {t('vks.detail.vpcNetwork')}
              </h2>
              <dl className="grid grid-cols-1 gap-y-4 text-sm">
                <div>
                  <dt className="text-on-surface-variant">{t('vks.col.network')}</dt>
                  <dd className="font-mono text-on-surface">
                    {cluster.workers?.network_ref?.name || '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.tcpHost')}</dt>
                  <dd className="font-mono text-xs text-on-surface">
                    {cluster.tcp_namespace && cluster.tcp_name
                      ? `${cluster.tcp_namespace}/${cluster.tcp_name}`
                      : '—'}
                  </dd>
                </div>
                <div>
                  <dt className="text-on-surface-variant">{t('vks.detail.cni')}</dt>
                  <dd className="text-on-surface">{t('vks.detail.cniFlannel')}</dd>
                </div>
              </dl>
            </SurfaceCard>
          </div>
        )}

        {tab === 'status' && (
          <SurfaceCard>
            <h2 className="mb-4 font-headline text-headline-md font-semibold text-on-surface">
              {t('vks.detail.conditions')}
            </h2>
            {(cluster.conditions || []).length === 0 ? (
              <p className="text-sm text-on-surface-variant">{t('vks.detail.noConditions')}</p>
            ) : (
              <ul className="space-y-3">
                {(cluster.conditions || []).map((cond) => (
                  <li
                    key={cond.type}
                    className="rounded-lg border border-outline-variant/40 bg-surface-container/40 px-4 py-3"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-sm font-medium text-on-surface">{cond.type}</span>
                      <StatusBadge
                        status={cond.status === 'True' ? 'active' : cond.status === 'False' ? 'error' : 'inactive'}
                        pulse={false}
                      />
                      {cond.reason && (
                        <span className="text-xs text-on-surface-variant">{cond.reason}</span>
                      )}
                    </div>
                    {cond.message && (
                      <p className="mt-1 text-sm text-on-surface-variant">{cond.message}</p>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </SurfaceCard>
        )}

        <ConfirmDialog
          open={deleteOpen}
          onClose={() => setDeleteOpen(false)}
          onConfirm={() => deleteMutation.mutate()}
          title={t('vks.deleteTitle')}
          message={t('common.confirmDeleteMessage')}
          resourceName={cluster.name}
          confirmLabel={t('common.delete')}
          loading={deleteMutation.isPending}
        />
      </div>
    </RefreshingPanel>
  );
}
