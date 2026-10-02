import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Download } from 'lucide-react';
import {
  listVKSClusters, createVKSCluster, deleteVKSCluster, downloadVKSKubeconfig,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { StatusBadge } from '../components/StatusBadge';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  vksCanDownloadKubeconfig, vksNodesLabel, vksPhaseBadgeStatus,
} from '../lib/vks-display';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, InfoBanner,
} from '../components/shell';

const DEFAULTS = {
  kubernetes_version: 'v1.36.5',
  template: 'ubuntu-node-1-36-5',
  offering: 'medium',
  network: 'default',
  workers: 1,
  port: '' as string,
};

export function Clusters() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({
    name: '',
    kubernetes_version: DEFAULTS.kubernetes_version,
    workers: DEFAULTS.workers,
    template: DEFAULTS.template,
    offering: DEFAULTS.offering,
    network: DEFAULTS.network,
    port: DEFAULTS.port,
  });
  const [deleteTarget, setDeleteTarget] = useState<{ name: string } | null>(null);
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vksClusters,
    queryFn: listVKSClusters,
    enabled: !needsTenant,
    refetchInterval: 10_000,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.vksClusters });
  };

  const resetForm = () => setForm({
    name: '',
    kubernetes_version: DEFAULTS.kubernetes_version,
    workers: DEFAULTS.workers,
    template: DEFAULTS.template,
    offering: DEFAULTS.offering,
    network: DEFAULTS.network,
    port: DEFAULTS.port,
  });

  const createMutation = useMutation({
    mutationFn: createVKSCluster,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      resetForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteVKSCluster(name),
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const downloadMutation = useMutation({
    mutationFn: async (name: string) => {
      const blob = await downloadVKSKubeconfig(name);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${name}.kubeconfig`;
      a.click();
      URL.revokeObjectURL(url);
    },
  });

  const clusters = data?.clusters || [];
  const filtered = clusters.filter((c) => {
    const q = search.toLowerCase();
    return (
      c.name?.toLowerCase().includes(q) ||
      c.phase?.toLowerCase().includes(q) ||
      c.kubernetes_version?.toLowerCase().includes(q) ||
      c.workers?.network_ref?.name?.toLowerCase().includes(q)
    );
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const port = form.port.trim() ? Number(form.port) : undefined;
    createMutation.mutate({
      name: form.name,
      kubernetes_version: form.kubernetes_version,
      control_plane: port
        ? { service_type: 'NodePort', port }
        : undefined,
      workers: {
        count: form.workers,
        template_ref: { name: form.template },
        offering_ref: { name: form.offering },
        network_ref: { name: form.network },
      },
    });
  };

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('vks.title')}
        subtitle={t('vks.subtitle')}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('vks.create')}
            </button>
          </>
        }
      />

      <InfoBanner>{t('vks.info')}</InfoBanner>

      <SurfaceCard>
        <div className="mb-4">
          <SearchField
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={`${t('common.search')}...`}
          />
        </div>
        <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
          {isLoading ? (
            <p className="text-sm text-on-surface-variant">{t('common.loading')}</p>
          ) : filtered.length === 0 ? (
            <p className="text-sm text-on-surface-variant">{t('vks.empty')}</p>
          ) : (
            <PageTable>
              <PageTableHead>
                <PageTableTh>{t('vks.col.name')}</PageTableTh>
                <PageTableTh>{t('vks.col.status')}</PageTableTh>
                <PageTableTh>{t('vks.col.version')}</PageTableTh>
                <PageTableTh>{t('vks.col.location')}</PageTableTh>
                <PageTableTh>{t('vks.col.nodes')}</PageTableTh>
                <PageTableTh>{t('vks.col.machineType')}</PageTableTh>
                <PageTableTh>{t('vks.col.network')}</PageTableTh>
                <PageTableTh>{t('vks.col.actions')}</PageTableTh>
              </PageTableHead>
              <PageTableBody>
                {filtered.map((c) => (
                  <PageTableRow key={c.name}>
                    <PageTableTd>
                      <Link
                        to={`/clusters/${encodeURIComponent(c.name)}`}
                        className="font-medium text-primary hover:underline"
                      >
                        {c.name}
                      </Link>
                    </PageTableTd>
                    <PageTableTd>
                      <StatusBadge status={vksPhaseBadgeStatus(c.phase)} />
                    </PageTableTd>
                    <PageTableTd className="font-mono text-sm">{c.kubernetes_version || '—'}</PageTableTd>
                    <PageTableTd className="font-mono text-sm">
                      {c.control_plane_endpoint || '—'}
                    </PageTableTd>
                    <PageTableTd>
                      {vksNodesLabel(c.ready_workers, c.workers?.count)}
                    </PageTableTd>
                    <PageTableTd className="font-mono text-sm">
                      {c.workers?.offering_ref?.name || '—'}
                    </PageTableTd>
                    <PageTableTd className="font-mono text-sm">
                      {c.workers?.network_ref?.name || '—'}
                    </PageTableTd>
                    <PageTableTd>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          className="inline-flex items-center gap-1 text-sm text-primary disabled:opacity-40"
                          disabled={!vksCanDownloadKubeconfig(c.phase) || downloadMutation.isPending}
                          onClick={() => downloadMutation.mutate(c.name)}
                          title={t('vks.downloadKubeconfig')}
                        >
                          <Download className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          className="inline-flex items-center gap-1 text-sm text-error"
                          onClick={() => setDeleteTarget({ name: c.name })}
                          title={t('common.delete')}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </PageTableTd>
                  </PageTableRow>
                ))}
              </PageTableBody>
            </PageTable>
          )}
        </RefreshingPanel>
      </SurfaceCard>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('vks.create')}>
        <form className="space-y-3" onSubmit={handleSubmit}>
          <label className="block text-sm">
            {t('vks.form.name')}
            <input
              className={formInputClass}
              required
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.version')}
            <input
              className={formInputClass}
              required
              value={form.kubernetes_version}
              onChange={(e) => setForm((f) => ({ ...f, kubernetes_version: e.target.value }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.workers')}
            <input
              type="number"
              min={1}
              max={3}
              className={formInputClass}
              required
              value={form.workers}
              onChange={(e) => setForm((f) => ({ ...f, workers: Number(e.target.value) }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.template')}
            <input
              className={formInputClass}
              required
              value={form.template}
              onChange={(e) => setForm((f) => ({ ...f, template: e.target.value }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.offering')}
            <input
              className={formInputClass}
              required
              value={form.offering}
              onChange={(e) => setForm((f) => ({ ...f, offering: e.target.value }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.network')}
            <input
              className={formInputClass}
              required
              value={form.network}
              onChange={(e) => setForm((f) => ({ ...f, network: e.target.value }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.nodePort')}
            <input
              type="number"
              min={30000}
              max={32767}
              className={formInputClass}
              placeholder={t('vks.form.nodePortHint')}
              value={form.port}
              onChange={(e) => setForm((f) => ({ ...f, port: e.target.value }))}
            />
            <span className="mt-1 block text-xs text-on-surface-variant">{t('vks.form.nodePortHelp')}</span>
          </label>
          {createMutation.error && (
            <p className="text-sm text-error">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" className="rounded-lg px-3 py-2 text-sm" onClick={() => setCreateModal(false)}>
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary disabled:opacity-50">
              {t('vks.create')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.name)}
        title={t('vks.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
