import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, HardDrive, Trash2 } from 'lucide-react';
import { listVolumes, createVolume, deleteVolume, listVMs } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function Volumes() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', size_gi: 10 });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.volumes,
    queryFn: listVolumes,
    enabled: !needsTenant,
  });

  const { data: vmData } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
  });

  const createMutation = useMutation({
    mutationFn: createVolume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.volumes });
      setCreateModal(false);
      setForm({ name: '', size_gi: 10 });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteVolume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.volumes });
      setDeleteTarget(null);
    },
  });

  const volumes = data?.volumes || [];
  const vmById = Object.fromEntries((vmData?.vms || []).map((vm) => [vm.id, vm]));
  const filtered = volumes.filter((v) => v.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('volumes.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('nav.volumes')}
        subtitle={`${volumes.length} ${t('volumes.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('volumes.create')}
            </button>
          </>
        }
      />

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('volumes.searchPlaceholder')} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('volumes.col.volume')}</PageTableTh>
              <PageTableTh>{t('volumes.size')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh>{t('volumes.attachedVm')}</PageTableTh>
              <PageTableTh>PVC</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <tr><td colSpan={6} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={6} className="text-center py-12 text-on-surface-variant">{t('volumes.empty')}</td></tr>
              ) : (
                filtered.map((vol) => {
                  const attachedVm = vol.vm_id ? vmById[vol.vm_id] : undefined;
                  return (
                  <PageTableRow key={vol.id}>
                    <PageTableTd>
                      <div className="flex items-center gap-3">
                        <HardDrive size={16} className="text-primary-fixed-dim" />
                        <span className="font-medium">{vol.name}</span>
                      </div>
                    </PageTableTd>
                    <PageTableTd>{vol.size_gi} Gi</PageTableTd>
                    <PageTableTd>
                      <StatusBadge status={vol.state || 'active'} pulse={false} />
                    </PageTableTd>
                    <PageTableTd>
                      {attachedVm ? (
                        <Link to={`/vms/${attachedVm.name}`} className="text-primary hover:text-primary-fixed-dim hover:underline">
                          {attachedVm.display_name || attachedVm.name}
                        </Link>
                      ) : (
                        <span className="text-on-surface-variant">{t('volumes.unattached')}</span>
                      )}
                    </PageTableTd>
                    <PageTableTd className="font-data-mono text-xs text-on-surface-variant">{vol.pvc_name}</PageTableTd>
                    <PageTableTd className="text-right">
                      {!vol.vm_id && (
                        <button
                          type="button"
                          onClick={() => setDeleteTarget({ id: vol.id, name: vol.name })}
                          className="btn-icon-danger"
                          title={t('volumes.deleteTitle')}
                        >
                          <Trash2 size={16} />
                        </button>
                      )}
                    </PageTableTd>
                  </PageTableRow>
                  );
                })
              )}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      </RefreshingPanel>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('volumes.deleteTitle')}
        message={t('volumes.deleteConfirm')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        loading={deleteMutation.isPending}
      />

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('volumes.modalTitle')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required pattern="[-a-z0-9]+" value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
              className={formInputClass} placeholder="data-disk-01" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('volumes.sizeGi')}</label>
            <input type="number" min={1} required value={form.size_gi}
              onChange={(e) => setForm({ ...form, size_gi: parseInt(e.target.value, 10) })}
              className={formInputClass} />
          </div>
          {createMutation.isError && (
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
