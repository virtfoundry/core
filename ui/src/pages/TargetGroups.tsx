import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import {
  listTargetGroups, createTargetGroup, deleteTargetGroup, getTargetGroup,
  registerTarget, deregisterTarget, listVMs,
} from '../lib/platform-api';
import type { TargetGroup } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass,
} from '../components/shell';

export function TargetGroups() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [detail, setDetail] = useState<TargetGroup | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', port: 80 });
  const [vmId, setVmId] = useState('');
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.targetGroups,
    queryFn: listTargetGroups,
    enabled: !needsTenant,
  });

  const detailQuery = useQuery({
    queryKey: [...queryKeys.targetGroups, detail?.id],
    queryFn: () => getTargetGroup(detail!.id),
    enabled: !!detail,
  });

  const vms = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !!detail && !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.targetGroups });

  const createMutation = useMutation({
    mutationFn: () => createTargetGroup({ name: form.name, port: Number(form.port), protocol: 'tcp' }),
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', port: 80 });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTargetGroup,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
      setDetail(null);
    },
  });

  const addTarget = useMutation({
    mutationFn: () => registerTarget(detail!.id, { vm_id: vmId }),
    onSuccess: () => {
      void detailQuery.refetch();
      setVmId('');
    },
  });

  const removeTarget = useMutation({
    mutationFn: (tid: string) => deregisterTarget(detail!.id, tid),
    onSuccess: () => void detailQuery.refetch(),
  });

  const rows = data?.target_groups || [];
  const filtered = rows.filter((g) => g.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Target Groups"
        actions={(
          <>
            <RefreshButton onClick={() => void refetch()} refreshing={isRefetching} updatedAt={dataUpdatedAt} />
            <button type="button" className="btn-primary inline-flex items-center gap-2" onClick={() => setCreateModal(true)}>
              <Plus className="h-4 w-4" /> Create
            </button>
          </>
        )}
      />
      <RefreshingPanel fetching={isFetching && !isLoading}>
        <SurfaceCard>
          <div className="mb-4">
            <SearchField value={search} onChange={setSearch} placeholder={t('common.search')} />
          </div>
          <PageTable>
            <PageTableHead>
              <tr>
                <PageTableTh>Name</PageTableTh>
                <PageTableTh>Protocol</PageTableTh>
                <PageTableTh>Port</PageTableTh>
                <PageTableTh className="w-28" />
              </tr>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <PageTableRow><PageTableTd colSpan={4}>{t('common.loading')}</PageTableTd></PageTableRow>
              ) : filtered.length === 0 ? (
                <PageTableRow><PageTableTd colSpan={4}>{t('common.empty')}</PageTableTd></PageTableRow>
              ) : filtered.map((tg) => (
                <PageTableRow key={tg.id}>
                  <PageTableTd>
                    <button type="button" className="text-left font-medium hover:underline" onClick={() => setDetail(tg)}>
                      {tg.name}
                    </button>
                  </PageTableTd>
                  <PageTableTd>{tg.protocol}</PageTableTd>
                  <PageTableTd className="font-mono">{tg.port}</PageTableTd>
                  <PageTableTd>
                    <ResourceActions onDelete={() => setDeleteTarget({ id: tg.id, name: tg.name })} />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      </RefreshingPanel>

      <Modal open={createModal} onClose={() => setCreateModal(false)} title="Create target group">
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate();
          }}
        >
          <label className="block space-y-1">
            <span className="text-sm">Name</span>
            <input className={formInputClass} required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </label>
          <label className="block space-y-1">
            <span className="text-sm">Instance port</span>
            <input
              className={formInputClass}
              type="number"
              min={1}
              max={65535}
              required
              value={form.port}
              onChange={(e) => setForm({ ...form, port: Number(e.target.value) })}
            />
          </label>
          {createMutation.error && <p className="text-sm text-red-600">{(createMutation.error as Error).message}</p>}
          <button type="submit" className="btn-primary" disabled={createMutation.isPending}>Create</button>
        </form>
      </Modal>

      <Modal open={!!detail} onClose={() => setDetail(null)} title={detail?.name || 'Target group'}>
        {detail && (
          <div className="space-y-4">
            <p className="text-sm font-mono">TCP :{detail.port}</p>
            <div>
              <h3 className="mb-2 font-medium">Targets</h3>
              <ul className="mb-3 space-y-1 text-sm">
                {(detailQuery.data?.targets || []).map((tgt) => (
                  <li key={tgt.id} className="flex items-center justify-between gap-2">
                    <span className="font-mono">{tgt.vm_name || tgt.vm_id} → {tgt.ip}:{tgt.port || detail.port}</span>
                    <button type="button" className="text-red-600" onClick={() => removeTarget.mutate(tgt.id)}>Remove</button>
                  </li>
                ))}
              </ul>
              <form
                className="flex flex-wrap items-end gap-2"
                onSubmit={(e) => {
                  e.preventDefault();
                  addTarget.mutate();
                }}
              >
                <label className="space-y-1 min-w-[12rem] flex-1">
                  <span className="text-xs">VM</span>
                  <select className={formInputClass} required value={vmId} onChange={(e) => setVmId(e.target.value)}>
                    <option value="">Select…</option>
                    {(vms.data?.vms || []).map((vm) => (
                      <option key={vm.id} value={vm.id}>{vm.name} {vm.ip ? `(${vm.ip})` : ''}</option>
                    ))}
                  </select>
                </label>
                <button type="submit" className="btn-primary" disabled={addTarget.isPending || !vmId}>Register</button>
              </form>
              {addTarget.error && <p className="text-sm text-red-600">{(addTarget.error as Error).message}</p>}
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title="Delete target group"
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
      />
    </div>
  );
}
