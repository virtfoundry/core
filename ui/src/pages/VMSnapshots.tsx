import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, RotateCcw, Trash2 } from 'lucide-react';
import {
  listVMSnapshots, createVMSnapshot, deleteVMSnapshot, restoreVMSnapshot, listVMs,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function VMSnapshots() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ vm_name: '', name: '' });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vmSnapshots,
    queryFn: listVMSnapshots,
    enabled: !needsTenant,
  });

  const { data: vmData } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.vmSnapshots });

  const createMutation = useMutation({
    mutationFn: createVMSnapshot,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({ vm_name: '', name: '' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteVMSnapshot,
    onSuccess: invalidate,
  });

  const restoreMutation = useMutation({
    mutationFn: restoreVMSnapshot,
    onSuccess: invalidate,
  });

  const snapshots = data?.vm_snapshots || [];
  const vms = vmData?.vms || [];
  const filtered = snapshots.filter((s) =>
    s.name?.toLowerCase().includes(search.toLowerCase()) ||
    s.vm_name?.toLowerCase().includes(search.toLowerCase())
  );

  const phaseStatus = (phase: string) => {
    switch (phase?.toLowerCase()) {
      case 'ready':
      case 'succeeded':
        return 'active';
      case 'failed':
        return 'error';
      default:
        return 'starting';
    }
  };

  if (needsTenant) {
    return <TenantRequiredNotice message={t('vmSnapshots.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('nav.vmSnapshots')}
        subtitle={`${snapshots.length} ${t('vmSnapshots.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('vmSnapshots.create')}
            </button>
          </>
        }
      />

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('vmSnapshots.searchPlaceholder')} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>VM</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <tr><td colSpan={4} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={4} className="text-center py-12 text-on-surface-variant">{t('vmSnapshots.empty')}</td></tr>
              ) : (
                filtered.map((snap) => (
                  <PageTableRow key={snap.id}>
                    <PageTableTd className="font-medium">{snap.name}</PageTableTd>
                    <PageTableTd className="font-data-mono text-sm">{snap.vm_name}</PageTableTd>
                    <PageTableTd>
                      <StatusBadge status={phaseStatus(snap.phase)} pulse={false} />
                    </PageTableTd>
                    <PageTableTd>
                      <div className="flex justify-end gap-1">
                        <button
                          type="button"
                          onClick={() => restoreMutation.mutate({ name: snap.name, vm_name: snap.vm_name })}
                          disabled={restoreMutation.isPending || snap.phase !== 'ready'}
                          className="btn-icon-neutral"
                          title={t('vmSnapshots.restore')}
                        >
                          <RotateCcw size={16} />
                        </button>
                        <button
                          type="button"
                          onClick={() => deleteMutation.mutate({ name: snap.name })}
                          className="btn-icon-danger"
                          title={t('common.delete')}
                        >
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </PageTableTd>
                  </PageTableRow>
                ))
              )}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('vmSnapshots.modalTitle')}>
        <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(form); }} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">VM</label>
            <select required value={form.vm_name} onChange={(e) => setForm({ ...form, vm_name: e.target.value })}
              className={formSelectClass}>
              <option value="">{t('common.select')}</option>
              {vms.map((v) => (
                <option key={v.id || v.name} value={v.name}>{v.name} ({v.state})</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('vmSnapshots.snapshotName')}</label>
            <input required pattern="[-a-z0-9]+" value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
              className={formInputClass} placeholder="snap-before-upgrade" />
          </div>
          {createMutation.isError && (
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">{t('common.create')}</button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
