import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, RotateCcw, Trash2 } from 'lucide-react';
import {
  listVMSnapshots, createVMSnapshot, updateVMSnapshot, deleteVMSnapshot, restoreVMSnapshot, listVMs,
} from '../lib/platform-api';
import type { VMSnapshot } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { OverflowMenu } from '../components/OverflowMenu';
import { ago } from '../lib/ago';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { TagsInput, TagPills, mergeTags } from '../components/TagsInput';
import { TagsEditorModal } from '../components/TagsEditorModal';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function VMSnapshots() {
  const { t, dateLocale } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editSnap, setEditSnap] = useState<VMSnapshot | null>(null);
  const [form, setForm] = useState<{ vm_name: string; name: string; tags: string[] }>({ vm_name: '', name: '', tags: [] });
  const [tagDraft, setTagDraft] = useState('');
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
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
      setForm({ vm_name: '', name: '', tags: [] });
      setTagDraft('');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, tags }: { id: string; tags: string[] }) => updateVMSnapshot(id, { tags }),
    onSuccess: () => {
      invalidate();
      setEditSnap(null);
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

  const { rows, pager } = usePagination(filtered, search);

  if (needsTenant) {
    return <TenantRequiredNotice message={t('vmSnapshots.selectTenant')} />;
  }

  return (
    <div className="flex flex-col gap-6">
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
          <PageTable minWidth={820}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>VM</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh>{t('tags.column')}</PageTableTh>
              <PageTableTh>{t('common.created')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={6}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={6}>{t('vmSnapshots.empty')}</TableMessageRow>
              ) : (
                rows.map((snap) => (
                  <PageTableRow key={snap.id}>
                    <PageTableTd className="font-semibold">{snap.name}</PageTableTd>
                    <PageTableTd mono>{snap.vm_name}</PageTableTd>
                    <PageTableTd>
                      <StatusBadge status={phaseStatus(snap.phase)} pulse={false} />
                    </PageTableTd>
                    <PageTableTd><TagPills tags={snap.tags} /></PageTableTd>
                    <PageTableTd className="text-on-surface-variant">{ago(snap.created_at, dateLocale)}</PageTableTd>
                    <PageTableTd className="text-right">
                      <OverflowMenu
                        variant="dots"
                        label={t('table.moreActions')}
                        items={[
                          {
                            id: 'edit-tags',
                            label: t('tags.editTitle'),
                            icon: <Pencil size={14} />,
                            onSelect: () => setEditSnap(snap),
                          },
                          {
                            id: 'restore',
                            label: t('vmSnapshots.restore'),
                            icon: <RotateCcw size={14} />,
                            disabled: restoreMutation.isPending || snap.phase !== 'ready',
                            onSelect: () => restoreMutation.mutate({ name: snap.name, vm_name: snap.vm_name }),
                          },
                          {
                            id: 'delete',
                            label: t('common.delete'),
                            icon: <Trash2 size={14} />,
                            danger: true,
                            onSelect: () => deleteMutation.mutate({ name: snap.name }),
                          },
                        ]}
                      />
                    </PageTableTd>
                  </PageTableRow>
                ))
              )}
            </PageTableBody>
          </PageTable>
          <TablePager pager={pager} />
        </SurfaceCard>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('vmSnapshots.modalTitle')}>
        <form onSubmit={(e) => { e.preventDefault(); createMutation.mutate({ ...form, tags: mergeTags(form.tags, tagDraft) }); }} className="space-y-4">
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
