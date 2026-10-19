import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil } from 'lucide-react';
import { listSnapshots, createSnapshot, updateSnapshot, listVolumes } from '../lib/platform-api';
import type { Snapshot } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { OverflowMenu } from '../components/OverflowMenu';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { TagsInput, TagPills, mergeTags } from '../components/TagsInput';
import { TagsEditorModal } from '../components/TagsEditorModal';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { ago } from '../lib/ago';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function Snapshots() {
  const { t, dateLocale } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editSnap, setEditSnap] = useState<Snapshot | null>(null);
  const [form, setForm] = useState<{ volume_id: string; name: string; tags: string[] }>({ volume_id: '', name: '', tags: [] });
  const [tagDraft, setTagDraft] = useState('');
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
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
      setForm({ volume_id: '', name: '', tags: [] });
      setTagDraft('');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, tags }: { id: string; tags: string[] }) => updateSnapshot(id, { tags }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.snapshots });
      setEditSnap(null);
    },
  });

  const snapshots = data?.snapshots || [];
  const volumes = volData?.volumes || [];
  const filtered = snapshots.filter((s) => s.name?.toLowerCase().includes(search.toLowerCase()));

  const { rows, pager } = usePagination(filtered, search);

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
          <PageTable minWidth={760}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('common.volume')}</PageTableTh>
              <PageTableTh>{t('volumes.size')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh>{t('tags.column')}</PageTableTh>
              <PageTableTh>{t('common.created')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={7}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={7}>{t('snapshots.empty')}</TableMessageRow>
              ) : (
                rows.map((snap) => {
                  const vol = volById[snap.volume_id];
                  return (
                    <PageTableRow key={snap.id}>
                      <PageTableTd className="font-semibold">{snap.name}</PageTableTd>
                      <PageTableTd mono>{vol?.name ?? `${snap.volume_id.slice(0, 8)}…`}</PageTableTd>
                      <PageTableTd mono>{vol ? `${vol.size_gi} GiB` : '—'}</PageTableTd>
                      <PageTableTd>
                        <StatusBadge status={snap.state === 'ready' ? 'active' : 'starting'} pulse={false} />
                      </PageTableTd>
                      <PageTableTd><TagPills tags={snap.tags} /></PageTableTd>
                      <PageTableTd className="text-on-surface-variant whitespace-nowrap">{ago(snap.created_at, dateLocale)}</PageTableTd>
                      <PageTableTd className="text-right">
                        <OverflowMenu
                          variant="dots"
                          label={t('table.moreActions')}
                          items={[{
                            id: 'edit-tags',
                            label: t('tags.editTitle'),
                            icon: <Pencil size={14} />,
                            onSelect: () => setEditSnap(snap),
                          }]}
                        />
                      </PageTableTd>
                    </PageTableRow>
                  );
                })
              )}
            </PageTableBody>
          </PageTable>
          <TablePager pager={pager} />
        </SurfaceCard>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('snapshots.modalTitle')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate({ ...form, tags: mergeTags(form.tags, tagDraft) });
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
          <div>
            <label className="block text-sm font-medium mb-1">{t('tags.column')}</label>
            <TagsInput
              tags={form.tags}
              onTagsChange={(tags) => setForm({ ...form, tags })}
              draft={tagDraft}
              onDraftChange={setTagDraft}
              placeholder={t('common.tagsPlaceholder')}
            />
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

      <TagsEditorModal
        key={editSnap?.id ?? 'none'}
        isOpen={!!editSnap}
        onClose={() => setEditSnap(null)}
        initialTags={editSnap?.tags ?? []}
        onSubmit={(tags) => editSnap && updateMutation.mutate({ id: editSnap.id, tags })}
        loading={updateMutation.isPending}
        error={updateMutation.isError ? (updateMutation.error as Error).message : null}
      />
    </div>
  );
}
