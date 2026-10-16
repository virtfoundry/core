import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { listSnapshots, createSnapshot, listVolumes } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
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

  const volById = Object.fromEntries(volumes.map((v) => [v.id, v]));

  return (
    <div className="flex flex-col gap-6">
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

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <TableToolbar>
            <TableFilter
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('table.filter')}
              aria-label={t('table.filter')}
            />
          </TableToolbar>
          <PageTable minWidth={600}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('common.volume')}</PageTableTh>
              <PageTableTh>{t('volumes.size')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={4}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={4}>{t('snapshots.empty')}</TableMessageRow>
              ) : (
                filtered.map((snap) => {
                  const vol = volById[snap.volume_id];
                  return (
                    <PageTableRow key={snap.id}>
                      <PageTableTd className="font-semibold">{snap.name}</PageTableTd>
                      <PageTableTd mono>{vol?.name ?? `${snap.volume_id.slice(0, 8)}…`}</PageTableTd>
                      <PageTableTd mono>{vol ? `${vol.size_gi} GiB` : '—'}</PageTableTd>
                      <PageTableTd>
                        <StatusBadge status={snap.state === 'ready' ? 'active' : 'starting'} pulse={false} />
                      </PageTableTd>
                    </PageTableRow>
                  );
                })
              )}
            </PageTableBody>
          </PageTable>
          <TablePager shown={filtered.length} total={snapshots.length} />
        </SurfaceCard>
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
