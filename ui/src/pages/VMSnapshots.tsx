import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Camera, Plus, RotateCcw, Trash2 } from 'lucide-react';
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
  PageHeader, SearchField, EmptyState, ResourceGridCard, TenantRequiredNotice,
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
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <EmptyState icon={<Camera size={48} />} title={t('vmSnapshots.empty')} />
          ) : (
            filtered.map((snap) => (
              <ResourceGridCard key={snap.id}>
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 bg-primary-container/20 rounded-lg flex items-center justify-center shrink-0">
                    <Camera size={20} className="text-primary-fixed-dim" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="font-headline text-headline-md font-semibold text-on-surface">{snap.name}</h3>
                    <p className="text-sm text-on-surface-variant font-data-mono">VM: {snap.vm_name}</p>
                  </div>
                </div>
                <StatusBadge status={phaseStatus(snap.phase)} pulse={false} />
                <div className="flex gap-2 mt-4">
                  <button
                    type="button"
                    onClick={() => restoreMutation.mutate({ name: snap.name, vm_name: snap.vm_name })}
                    disabled={restoreMutation.isPending || snap.phase !== 'ready'}
                    className="btn-action-row"
                  >
                    <RotateCcw size={14} /> {t('vmSnapshots.restore')}
                  </button>
                  <button
                    type="button"
                    onClick={() => deleteMutation.mutate({ name: snap.name })}
                    className="btn-action-danger"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </ResourceGridCard>
            ))
          )}
        </div>
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
