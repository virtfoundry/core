import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Search, Camera, Plus, RotateCcw, Trash2 } from 'lucide-react';
import {
  listVMSnapshots, createVMSnapshot, deleteVMSnapshot, restoreVMSnapshot, listVMs,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';

export function VMSnapshots() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ vm_name: '', name: '' });
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vmSnapshots,
    queryFn: listVMSnapshots,
    enabled: !needsTenant,
    refetchInterval: 10_000,
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

  const phaseColor = (phase: string) => {
    switch (phase?.toLowerCase()) {
      case 'ready': case 'succeeded': return 'bg-green-100 text-green-700';
      case 'failed': return 'bg-red-100 text-red-700';
      default: return 'bg-yellow-100 text-yellow-700';
    }
  };

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('vmSnapshots.selectTenant')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('nav.vmSnapshots')}</h1>
          <p className="text-gray-500">{snapshots.length} {t('vmSnapshots.subtitle')}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary">
            <Plus size={18} /> {t('vmSnapshots.create')}
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('vmSnapshots.searchPlaceholder')}
          className="w-full pl-10 pr-4 py-3 border rounded-lg" />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full text-center py-12">{t('common.loading')}</div>
        ) : filtered.length === 0 ? (
          <div className="col-span-full text-center py-12">
            <Camera size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">{t('vmSnapshots.empty')}</p>
          </div>
        ) : (
          filtered.map((snap) => (
            <div key={snap.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center">
                  <Camera size={20} className="text-purple-500" />
                </div>
                <div>
                  <h3 className="font-semibold">{snap.name}</h3>
                  <p className="text-sm text-gray-500 font-mono">VM: {snap.vm_name}</p>
                </div>
              </div>
              <span className={`px-2 py-1 rounded text-xs font-medium ${phaseColor(snap.phase)}`}>
                {snap.phase}
              </span>
              <div className="flex gap-2 mt-4">
                <button
                  onClick={() => restoreMutation.mutate({ name: snap.name, vm_name: snap.vm_name })}
                  disabled={restoreMutation.isPending || snap.phase !== 'ready'}
                  className="btn-action-row"
                >
                  <RotateCcw size={14} /> {t('vmSnapshots.restore')}
                </button>
                <button
                  onClick={() => deleteMutation.mutate({ name: snap.name })}
                  className="btn-action-danger"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))
        )}
      </div>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('vmSnapshots.modalTitle')}>
        <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate(form); }} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">VM</label>
            <select required value={form.vm_name} onChange={(e) => setForm({ ...form, vm_name: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg">
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
              className="w-full px-4 py-2 border rounded-lg" placeholder="snap-before-upgrade" />
          </div>
          {createMutation.isError && (
            <p className="text-red-500 text-sm">{(createMutation.error as Error).message}</p>
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
