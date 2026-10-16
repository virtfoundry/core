import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Download, Trash2, Copy, Check } from 'lucide-react';
import {
  getVKSCluster, getVKSClusterSummary, deleteVKSCluster, downloadVKSKubeconfig, listVMs,
} from '../lib/platform-api';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { StatusBadge } from '../components/StatusBadge';
import { StatsCard } from '../components/StatsCard';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { useCopyToClipboard } from '../hooks/useCopyToClipboard';
import {
  vksCanDownloadKubeconfig, isVKSWorkerVM, vksNodesLabel, vksPhaseBadgeStatus,
} from '../lib/vks-display';
import { effectiveVmState, formatVmOffering } from '../lib/vm-display';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TabBar, InfoBanner, KeyValueList, TerminalBlock,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
} from '../components/shell';

type Tab = 'overview' | 'nodes' | 'workloads' | 'networking' | 'status';

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

  const {
    data: vmsData,
    isLoading: vmsLoading,
    isError: vmsIsError,
    error: vmsError,
  } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant && !!name && tab === 'nodes',
  });

  const workerVMs = useMemo(
    () => (vmsData?.vms || []).filter((vm) => isVKSWorkerVM(name, vm.name)),
    [vmsData?.vms, name],
  );

  const canWorkloads = vksCanDownloadKubeconfig(cluster?.phase);

  const {
    data: summaryData,
    isLoading: summaryLoading,
    isError: summaryIsError,
    error: summaryError,
  } = useQuery({
    queryKey: queryKeys.vksClusterSummary(name),
    queryFn: () => getVKSClusterSummary(name),
    enabled: !needsTenant && !!name && tab === 'workloads' && canWorkloads,
    refetchInterval: tab === 'workloads' && canWorkloads ? 15_000 : false,
  });

  const summary = summaryData?.summary;

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
    { id: 'workloads', label: t('vks.detail.tabWorkloads') },
    { id: 'networking', label: t('vks.detail.tabNetworking') },
    { id: 'status', label: t('vks.detail.tabStatus') },
  ];

  const endpoint = cluster.control_plane_endpoint || '—';
  const canKubeconfig = vksCanDownloadKubeconfig(cluster.phase);
  const ageDays = cluster.created_at
    ? Math.max(0, Math.floor((Date.now() - new Date(cluster.created_at).getTime()) / 86_400_000))
    : null;
  const kv = (k: string, v: string | number | undefined, mono = false) => ({ k, v: v || '—', mono });

  return (
    <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
      <div className="space-y-6">
        <PageHeader
          breadcrumb={`${t('vks.title')} / ${cluster.name}`}
          title={cluster.name}
          subtitle={`${t('vks.detail.product')} · ${cluster.kubernetes_version || '—'}`}
          status={<StatusBadge status={vksPhaseBadgeStatus(cluster.phase)} />}
          actions={
            <>
              <RefreshButton
                compact
                onRefresh={() => refetch()}
                isFetching={isRefetching}
                dataUpdatedAt={dataUpdatedAt}
              />
              <button
                type="button"
                className="btn-secondary"
                disabled={!canKubeconfig || downloadMutation.isPending}
                onClick={() => downloadMutation.mutate()}
              >
                <Download size={16} /> {t('vks.downloadKubeconfig')}
              </button>
              <button type="button" className="btn-secondary !border-bad-fg !text-bad-fg" onClick={() => setDeleteOpen(true)}>
                <Trash2 size={16} /> {t('common.delete')}
              </button>
            </>
          }
        />

        <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
          <StatsCard
            title={t('vks.detail.readyWorkers')}
            value={cluster.ready_workers ?? 0}
            of={`/ ${cluster.workers?.count ?? 0}`}
            variant={cluster.workers?.count && cluster.ready_workers === cluster.workers.count ? 'success' : 'warning'}
            progress={cluster.workers?.count ? ((cluster.ready_workers ?? 0) / cluster.workers.count) * 100 : 0}
          />
          <StatsCard title={t('vks.detail.k8sVersion')} value={cluster.kubernetes_version || '—'} />
          {ageDays !== null && (
            <StatsCard title={t('vks.detail.age')} value={ageDays} of={t('vks.detail.days')} />
          )}
        </section>

        {!canKubeconfig && (
          <InfoBanner variant="warning">{t('vks.detail.kubeconfigWait')}</InfoBanner>
        )}

        <TabBar tabs={tabs} active={tab} onChange={setTab} />

        {tab === 'overview' && (
          <div className="flex flex-wrap gap-4 items-stretch [&>*]:flex-1 [&>*]:basis-[340px] [&>*]:min-w-0">
            <SurfaceCard title={t('vks.detail.clusterDetails')}>
              <KeyValueList
                rows={[
                  kv(t('vks.col.name'), cluster.name, true),
                  kv(t('vks.col.status'), cluster.phase),
                  kv(t('vks.detail.k8sVersion'), cluster.kubernetes_version, true),
                  kv(t('vks.detail.controlPlane'), t('vks.detail.controlPlaneManaged')),
                  kv(t('vks.detail.totalNodes'), vksNodesLabel(cluster.ready_workers, cluster.workers?.count)),
                  kv(t('vks.detail.tenantNs'), cluster.namespace, true),
                  kv(t('vks.detail.createdAt'), formatDate(cluster.created_at)),
                  kv(t('vks.detail.kubeconfigSecret'), cluster.kubeconfig_secret_ref, true),
                ]}
              />
            </SurfaceCard>

            <SurfaceCard title={t('vks.detail.credentials')}>
              <p className="mt-0 mb-4 text-sm text-on-surface-variant">{t('vks.detail.credentialsHint')}</p>
              <button
                type="button"
                className="btn-primary w-full"
                disabled={!canKubeconfig || downloadMutation.isPending}
                onClick={() => downloadMutation.mutate()}
              >
                <Download size={16} /> {t('vks.downloadKubeconfig')}
              </button>
              {endpoint !== '—' && (
                <div className="mt-4">
                  <p className="mt-0 mb-1.5 text-[13px] text-on-surface-variant">{t('vks.col.location')}</p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 truncate rounded-lg bg-sunk px-3 py-2 font-mono text-[13px]">
                      {endpoint}
                    </code>
                    <button
                      type="button"
                      className="btn-secondary !px-3"
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

            {canKubeconfig && (
              <SurfaceCard title={t('vks.detail.access')} className="!basis-full">
                <TerminalBlock
                  lines={[
                    { text: `$ kubectl --kubeconfig ${cluster.name}.kubeconfig get nodes` },
                    { text: 'NAME   STATUS   ROLES   AGE   VERSION', tone: 'muted' },
                  ]}
                />
              </SurfaceCard>
            )}
          </div>
        )}

        {tab === 'nodes' && (
          <div className="space-y-4">
            <SurfaceCard title={t('vks.detail.nodePool')}>
              <p className="mt-0 mb-4 text-sm text-on-surface-variant">{t('vks.detail.nodePoolHint')}</p>
              <KeyValueList
                rows={[
                  kv(t('vks.detail.poolName'), 'default-pool', true),
                  kv(t('vks.col.nodes'), vksNodesLabel(cluster.ready_workers, cluster.workers?.count)),
                  kv(t('vks.col.machineType'), cluster.workers?.offering_ref?.name, true),
                  kv(t('vks.detail.nodeImage'), cluster.workers?.template_ref?.name, true),
                  kv(t('vks.form.workers'), cluster.workers?.count ?? '—'),
                  kv(t('vks.col.network'), cluster.workers?.network_ref?.name, true),
                  kv(t('vks.detail.sshKeys'), (cluster.workers?.ssh_key_refs || []).map((r) => r.name).join(', '), true),
                ]}
              />
            </SurfaceCard>

            <SurfaceCard padding="none" title={t('vks.detail.workerInstances')}>
              {vmsLoading ? (
                <p className="px-6 py-8 text-sm text-on-surface-variant">{t('common.loading')}</p>
              ) : vmsIsError ? (
                <p className="px-6 py-8 text-sm text-error">
                  {t('common.errorLoad')}: {(vmsError as Error)?.message}
                </p>
              ) : workerVMs.length === 0 ? (
                <p className="px-6 py-8 text-sm text-on-surface-variant">{t('vks.detail.noWorkers')}</p>
              ) : (
                <PageTable>
                  <PageTableHead>
                    <PageTableTh>{t('vks.col.name')}</PageTableTh>
                    <PageTableTh>{t('vks.col.status')}</PageTableTh>
                    <PageTableTh>{t('vks.detail.colOffering')}</PageTableTh>
                    <PageTableTh>{t('vks.detail.colIp')}</PageTableTh>
                  </PageTableHead>
                  <PageTableBody>
                    {workerVMs.map((vm) => {
                      const displayState = effectiveVmState(vm);
                      const offering = vm.service_offering_id || formatVmOffering(vm);
                      const ip = vm.ip || vm.nics?.find((n) => n.ip)?.ip || '—';
                      return (
                        <PageTableRow key={vm.id || vm.name}>
                          <PageTableTd>
                            <Link
                              to={`/vms/${encodeURIComponent(vm.name)}`}
                              className="font-mono text-sm text-primary hover:underline"
                            >
                              {vm.name}
                            </Link>
                          </PageTableTd>
                          <PageTableTd>
                            <StatusBadge status={(displayState || 'inactive').toLowerCase()} pulse={false} />
                          </PageTableTd>
                          <PageTableTd className="font-mono text-sm">{offering}</PageTableTd>
                          <PageTableTd className="font-mono text-sm">{ip}</PageTableTd>
                        </PageTableRow>
                      );
                    })}
                  </PageTableBody>
                </PageTable>
              )}
            </SurfaceCard>
          </div>
        )}

        {tab === 'workloads' && (
          <div className="space-y-4">
            {!canKubeconfig ? (
              <InfoBanner variant="warning">{t('vks.detail.workloadsWait')}</InfoBanner>
            ) : summaryIsError ? (
              <SurfaceCard>
                <p className="text-sm text-error">
                  {t('vks.detail.workloadsError')}: {(summaryError as Error)?.message}
                </p>
                <p className="mt-2 text-sm text-on-surface-variant">{t('vks.detail.workloadsErrorHint')}</p>
                <button
                  type="button"
                  className="btn-primary mt-4"
                  disabled={downloadMutation.isPending}
                  onClick={() => downloadMutation.mutate()}
                >
                  <Download size={16} /> {t('vks.downloadKubeconfig')}
                </button>
              </SurfaceCard>
            ) : summaryLoading && !summary ? (
              <p className="text-sm text-on-surface-variant">{t('common.loading')}</p>
            ) : (
              <>
                {summary?.message && (
                  <InfoBanner variant="warning">{summary.message}</InfoBanner>
                )}
                <SurfaceCard title={t('vks.detail.podTotals')}>
                  <p className="mt-0 mb-4 text-sm text-on-surface-variant">{t('vks.detail.workloadsHint')}</p>
                  <dl className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
                    <div>
                      <dt className="text-on-surface-variant">{t('vks.detail.podRunning')}</dt>
                      <dd className="text-lg font-semibold text-on-surface">
                        {summary?.pod_totals?.running ?? 0}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-on-surface-variant">{t('vks.detail.podPending')}</dt>
                      <dd className="text-lg font-semibold text-on-surface">
                        {summary?.pod_totals?.pending ?? 0}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-on-surface-variant">{t('vks.detail.podFailed')}</dt>
                      <dd className="text-lg font-semibold text-on-surface">
                        {summary?.pod_totals?.failed ?? 0}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-on-surface-variant">{t('vks.detail.podOther')}</dt>
                      <dd className="text-lg font-semibold text-on-surface">
                        {summary?.pod_totals?.other ?? 0}
                      </dd>
                    </div>
                  </dl>
                </SurfaceCard>

                <SurfaceCard padding="none" title={t('vks.detail.namespaces')}>
                  {(summary?.namespaces || []).length === 0 ? (
                    <p className="px-6 py-8 text-sm text-on-surface-variant">
                      {t('vks.detail.noNamespaces')}
                    </p>
                  ) : (
                    <PageTable>
                      <PageTableHead>
                        <PageTableTh>{t('vks.detail.colNamespace')}</PageTableTh>
                        <PageTableTh>{t('vks.detail.colPodCount')}</PageTableTh>
                      </PageTableHead>
                      <PageTableBody>
                        {(summary?.namespaces || []).map((ns) => (
                          <PageTableRow key={ns.name}>
                            <PageTableTd className="font-mono text-sm">{ns.name}</PageTableTd>
                            <PageTableTd>{ns.pod_count}</PageTableTd>
                          </PageTableRow>
                        ))}
                      </PageTableBody>
                    </PageTable>
                  )}
                </SurfaceCard>

                <SurfaceCard padding="none" title={t('vks.detail.guestNodes')}>
                  {(summary?.guest_nodes || []).length === 0 ? (
                    <p className="px-6 py-8 text-sm text-on-surface-variant">
                      {t('vks.detail.noGuestNodes')}
                    </p>
                  ) : (
                    <PageTable>
                      <PageTableHead>
                        <PageTableTh>{t('vks.col.name')}</PageTableTh>
                        <PageTableTh>{t('vks.detail.colNodeReady')}</PageTableTh>
                      </PageTableHead>
                      <PageTableBody>
                        {(summary?.guest_nodes || []).map((node) => (
                          <PageTableRow key={node.name}>
                            <PageTableTd className="font-mono text-sm">{node.name}</PageTableTd>
                            <PageTableTd>
                              <StatusBadge
                                status={node.ready ? 'active' : 'inactive'}
                                pulse={false}
                              />
                            </PageTableTd>
                          </PageTableRow>
                        ))}
                      </PageTableBody>
                    </PageTable>
                  )}
                </SurfaceCard>
              </>
            )}
          </div>
        )}

        {tab === 'networking' && (
          <div className="flex flex-wrap gap-4 items-stretch [&>*]:flex-1 [&>*]:basis-[340px] [&>*]:min-w-0">
            <SurfaceCard title={t('vks.detail.apiAccess')}>
              <KeyValueList
                rows={[
                  kv(t('vks.col.location'), endpoint, true),
                  kv(t('vks.detail.serviceType'), cluster.control_plane?.service_type || 'LoadBalancer'),
                  kv(t('vks.detail.advertisedAddress'), cluster.control_plane?.address, true),
                  kv(t('vks.form.nodePort'), cluster.control_plane?.port, true),
                ]}
              />
            </SurfaceCard>
            <SurfaceCard title={t('vks.detail.vpcNetwork')}>
              <KeyValueList
                rows={[
                  kv(t('vks.col.network'), cluster.workers?.network_ref?.name, true),
                  kv(
                    t('vks.detail.tcpHost'),
                    cluster.tcp_namespace && cluster.tcp_name ? `${cluster.tcp_namespace}/${cluster.tcp_name}` : undefined,
                    true,
                  ),
                  kv(t('vks.detail.cni'), t('vks.detail.cniFlannel')),
                ]}
              />
            </SurfaceCard>
          </div>
        )}

        {tab === 'status' && (
          <SurfaceCard title={t('vks.detail.conditions')}>
            {(cluster.conditions || []).length === 0 ? (
              <p className="text-sm text-on-surface-variant">{t('vks.detail.noConditions')}</p>
            ) : (
              <ul className="space-y-3">
                {(cluster.conditions || []).map((cond) => (
                  <li
                    key={cond.type}
                    className="rounded-[10px] border border-outline-variant bg-sunk px-4 py-3"
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
