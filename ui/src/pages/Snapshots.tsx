import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Camera, Plus } from 'lucide-react';
import { listSnapshots, createSnapshot, listVolumes } from '../lib/platform-api';
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

export function Snapshots() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ volume_id: '', name: '' });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.snapshots,
    queryFn: listSnapshots,
    enabled: !needsTenant,
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
    return <TenantRequiredNotice message={t('snapshots.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('nav.volumeSnapshots')}
        subtitle={`${snapshots.length} ${t('nav.volumeSnapshots').toLowerCase()}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('snapshots.create')}
            </button>
          </>
        }
      />

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('snapshots.searchPlaceholder')} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <EmptyState icon={<Camera size={48} />} title={t('snapshots.empty')} />
          ) : (
            filtered.map((snap) => (
              <ResourceGridCard key={snap.id}>
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 bg-primary-container/20 rounded-lg flex items-center justify-center shrink-0">
                    <Camera size={20} className="text-primary-fixed-dim" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="font-headline text-headline-md font-semibold text-on-surface">{snap.name}</h3>
                    <p className="text-sm text-on-surface-variant">{t('common.volume')}: {snap.volume_id.slice(0, 8)}...</p>
                  </div>
                </div>
                <StatusBadge status={snap.state === 'ready' ? 'active' : 'starting'} pulse={false} />
              </ResourceGridCard>
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
              className={formSelectClass}>
              <option value="">{t('common.select')}</option>
              {volumes.map((v) => (
                <option key={v.id} value={v.id}>{v.name} ({v.size_gi} Gi)</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={formInputClass} placeholder="snap-2026-01-01" />
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
