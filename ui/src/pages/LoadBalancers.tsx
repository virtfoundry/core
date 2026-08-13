import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import {
  listLoadBalancers, createLoadBalancer, deleteLoadBalancer, getLoadBalancer,
  createLBListener, deleteLBListener, listTargetGroups,
} from '../lib/platform-api';
import type { LoadBalancer } from '../lib/platform-api';
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
  formInputClass, formTextareaClass,
} from '../components/shell';

export function LoadBalancers() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [detail, setDetail] = useState<LoadBalancer | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', description: '' });
  const [listenerForm, setListenerForm] = useState({ port: 80, target_group_id: '' });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.loadBalancers,
    queryFn: listLoadBalancers,
    enabled: !needsTenant,
  });

  const detailQuery = useQuery({
    queryKey: [...queryKeys.loadBalancers, detail?.id],
    queryFn: () => getLoadBalancer(detail!.id),
    enabled: !!detail,
  });

  const tgs = useQuery({
    queryKey: queryKeys.targetGroups,
    queryFn: listTargetGroups,
    enabled: !!detail && !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.loadBalancers });

  const createMutation = useMutation({
    mutationFn: createLoadBalancer,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', description: '' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteLoadBalancer,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
      setDetail(null);
    },
  });

  const addListener = useMutation({
    mutationFn: () => createLBListener(detail!.id, {
      port: Number(listenerForm.port),
      target_group_id: listenerForm.target_group_id,
      protocol: 'tcp',
    }),
    onSuccess: () => {
      void detailQuery.refetch();
      setListenerForm({ port: 80, target_group_id: '' });
    },
  });

  const removeListener = useMutation({
    mutationFn: (lid: string) => deleteLBListener(detail!.id, lid),
    onSuccess: () => void detailQuery.refetch(),
  });

  const rows = data?.load_balancers || [];
  const filtered = rows.filter((g) => g.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Load Balancers"
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
                <PageTableTh>VIP</PageTableTh>
                <PageTableTh>State</PageTableTh>
                <PageTableTh className="w-28" />
              </tr>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <PageTableRow><PageTableTd colSpan={4}>{t('common.loading')}</PageTableTd></PageTableRow>
              ) : filtered.length === 0 ? (
                <PageTableRow><PageTableTd colSpan={4}>{t('common.empty')}</PageTableTd></PageTableRow>
              ) : filtered.map((lb) => (
                <PageTableRow key={lb.id}>
                  <PageTableTd>
                    <button type="button" className="text-left font-medium hover:underline" onClick={() => setDetail(lb)}>
                      {lb.name}
                    </button>
                  </PageTableTd>
                  <PageTableTd className="font-mono text-sm">{lb.vip || '—'}</PageTableTd>
                  <PageTableTd>{lb.state}</PageTableTd>
                  <PageTableTd>
                    <ResourceActions onDelete={() => setDeleteTarget({ id: lb.id, name: lb.name })} />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      </RefreshingPanel>

      <Modal open={createModal} onClose={() => setCreateModal(false)} title="Create load balancer">
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
        >
          <label className="block space-y-1">
            <span className="text-sm">Name</span>
            <input className={formInputClass} required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </label>
          <label className="block space-y-1">
            <span className="text-sm">Description</span>
            <textarea className={formTextareaClass} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </label>
          {createMutation.error && <p className="text-sm text-red-600">{(createMutation.error as Error).message}</p>}
          <button type="submit" className="btn-primary" disabled={createMutation.isPending}>
            {createMutation.isPending ? 'Creating…' : 'Create'}
          </button>
        </form>
      </Modal>

      <Modal open={!!detail} onClose={() => setDetail(null)} title={detail?.name || 'Load balancer'}>
        {detail && (
          <div className="space-y-4">
            <p className="font-mono text-sm">VIP: {detailQuery.data?.load_balancer.vip || detail.vip || '—'}</p>
            <p className="text-sm">State: {detailQuery.data?.load_balancer.state || detail.state}</p>
            <div>
              <h3 className="mb-2 font-medium">Listeners</h3>
              <ul className="mb-3 space-y-1 text-sm">
                {(detailQuery.data?.listeners || []).map((l) => (
                  <li key={l.id} className="flex items-center justify-between gap-2">
                    <span className="font-mono">TCP :{l.port} → TG {l.target_group_id.slice(0, 8)}</span>
                    <button type="button" className="text-red-600" onClick={() => removeListener.mutate(l.id)}>Remove</button>
                  </li>
                ))}
              </ul>
              <form
                className="flex flex-wrap items-end gap-2"
                onSubmit={(e) => {
                  e.preventDefault();
                  addListener.mutate();
                }}
              >
                <label className="space-y-1">
                  <span className="text-xs">Port</span>
                  <input
                    className={formInputClass}
                    type="number"
                    min={1}
                    max={65534}
                    value={listenerForm.port}
                    onChange={(e) => setListenerForm({ ...listenerForm, port: Number(e.target.value) })}
                  />
                </label>
                <label className="space-y-1 min-w-[12rem] flex-1">
                  <span className="text-xs">Target group</span>
                  <select
                    className={formInputClass}
                    required
                    value={listenerForm.target_group_id}
                    onChange={(e) => setListenerForm({ ...listenerForm, target_group_id: e.target.value })}
                  >
                    <option value="">Select…</option>
                    {(tgs.data?.target_groups || []).map((tg) => (
                      <option key={tg.id} value={tg.id}>{tg.name} (:{tg.port})</option>
                    ))}
                  </select>
                </label>
                <button type="submit" className="btn-primary" disabled={addListener.isPending || !listenerForm.target_group_id}>
                  Add
                </button>
              </form>
              {(tgs.data?.target_groups || []).length === 0 && (
                <p className="mt-2 text-sm opacity-70">
                  Create a <Link className="underline" to="/target-groups">target group</Link> first.
                </p>
              )}
              {addListener.error && <p className="text-sm text-red-600">{(addListener.error as Error).message}</p>}
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title="Delete load balancer"
        message="VIP will be released."
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
      />
    </div>
  );
}
