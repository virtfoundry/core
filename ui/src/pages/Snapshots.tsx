import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Search, Camera, Plus } from 'lucide-react';
import { listSnapshots, createSnapshot, listVolumes } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';

export function Snapshots() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ volume_id: '', name: '' });
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.snapshots,
    queryFn: listSnapshots,
    enabled: !needsTenant,
    refetchInterval: 10_000,
  });

  const { data: volData } = useQuery({
    queryKey: queryKeys.volumes,
    queryFn: listVolumes,
    enabled: !needsTenant,
  });

  const createMutation = useMutation({
    mutationFn: createSnapshot,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.snapshots });
      setCreateModal(false);
      setForm({ volume_id: '', name: '' });
    },
  });

  const snapshots = data?.snapshots || [];
  const volumes = volData?.volumes || [];
  const filtered = snapshots.filter((s) => s.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('snapshots.selectTenant')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('nav.volumeSnapshots')}</h1>
          <p className="text-gray-500">{snapshots.length} {t('nav.volumeSnapshots').toLowerCase()}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary">
            <Plus size={18} /> {t('snapshots.create')}
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('snapshots.searchPlaceholder')}
          className="w-full pl-10 pr-4 py-3 border rounded-lg" />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full text-center py-12">{t('common.loading')}</div>
        ) : filtered.length === 0 ? (
          <div className="col-span-full text-center py-12">
            <Camera size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">{t('snapshots.empty')}</p>
          </div>
        ) : (
          filtered.map((snap) => (
            <div key={snap.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
                  <Camera size={20} className="text-blue-500" />
                </div>
                <div>
                  <h3 className="font-semibold">{snap.name}</h3>
                  <p className="text-sm text-gray-500">{t('common.volume')}: {snap.volume_id.slice(0, 8)}...</p>
                </div>
              </div>
              <span className={`px-2 py-1 rounded text-xs font-medium ${snap.state === 'ready' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'}`}>
                {snap.state}
              </span>
            </div>
          ))
        )}
      </div>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('snapshots.modalTitle')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.volume')}</label>
            <select required value={form.volume_id} onChange={(e) => setForm({ ...form, volume_id: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg">
              <option value="">{t('common.select')}</option>
              {volumes.map((v) => (
                <option key={v.id} value={v.id}>{v.name} ({v.size_gi} Gi)</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg" placeholder="snap-2026-01-01" />
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
