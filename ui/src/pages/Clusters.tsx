import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Download } from 'lucide-react';
import {
  listVKSClusters, createVKSCluster, deleteVKSCluster, downloadVKSKubeconfig,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
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

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.vksClusters });

  const createMutation = useMutation({
    mutationFn: createVKSCluster,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({
        name: '',
        kubernetes_version: DEFAULTS.kubernetes_version,
        workers: DEFAULTS.workers,
        template: DEFAULTS.template,
        offering: DEFAULTS.offering,
        network: DEFAULTS.network,
      });
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
  const filtered = clusters.filter((c) =>
    c.name?.toLowerCase().includes(search.toLowerCase()) ||
    c.phase?.toLowerCase().includes(search.toLowerCase())
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate({
      name: form.name,
      kubernetes_version: form.kubernetes_version,
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
          <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />
        </div>
        <RefreshingPanel refreshing={isFetching && !isLoading} updatedAt={dataUpdatedAt}>
          {isLoading ? (
            <p className="text-sm text-on-surface-variant">{t('common.loading')}</p>
          ) : filtered.length === 0 ? (
            <p className="text-sm text-on-surface-variant">{t('vks.empty')}</p>
          ) : (
            <PageTable>
              <PageTableHead>
                <PageTableTh>{t('vks.col.name')}</PageTableTh>
                <PageTableTh>{t('vks.col.phase')}</PageTableTh>
                <PageTableTh>{t('vks.col.endpoint')}</PageTableTh>
                <PageTableTh>{t('vks.col.workers')}</PageTableTh>
                <PageTableTh>{t('vks.col.actions')}</PageTableTh>
              </PageTableHead>
              <PageTableBody>
                {filtered.map((c) => (
                  <PageTableRow key={c.name}>
                    <PageTableTd className="font-mono">{c.name}</PageTableTd>
                    <PageTableTd>{c.phase || '—'}</PageTableTd>
                    <PageTableTd className="font-mono text-sm">{c.control_plane_endpoint || '—'}</PageTableTd>
                    <PageTableTd>
                      {c.ready_workers ?? 0}/{c.workers?.count ?? 0}
                    </PageTableTd>
                    <PageTableTd>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          className="inline-flex items-center gap-1 text-sm text-primary disabled:opacity-40"
                          disabled={c.phase !== 'Ready' && c.phase !== 'ControlPlaneReady'}
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
