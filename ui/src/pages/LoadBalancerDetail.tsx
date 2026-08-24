import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import {
  getLoadBalancer, deleteLoadBalancer, createLBListener, deleteLBListener, listTargetGroups,
} from '../lib/platform-api';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { StatusBadge } from '../components/StatusBadge';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';

export function LoadBalancerDetail() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [listenerForm, setListenerForm] = useState({ port: 80, target_group_id: '' });

  const { data, isLoading, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.loadBalancer(id),
    queryFn: () => getLoadBalancer(id),
    enabled: !needsTenant && !!id,
    refetchInterval: (q) => {
      const state = q.state.data?.load_balancer.state?.toLowerCase();
      return state === 'creating' ? 3000 : false;
    },
  });

  const tgs = useQuery({
    queryKey: queryKeys.targetGroups,
    queryFn: listTargetGroups,
    enabled: !needsTenant,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.loadBalancer(id) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.loadBalancers });
  };

  const deleteMutation = useMutation({
    mutationFn: () => deleteLoadBalancer(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.loadBalancers });
      navigate('/load-balancers');
    },
  });

  const addListener = useMutation({
    mutationFn: () => createLBListener(id, {
      port: Number(listenerForm.port),
      target_group_id: listenerForm.target_group_id,
      protocol: 'tcp',
    }),
    onSuccess: () => {
      invalidate();
      setListenerForm({ port: 80, target_group_id: '' });
    },
  });

  const removeListener = useMutation({
    mutationFn: (lid: string) => deleteLBListener(id, lid),
    onSuccess: () => invalidate(),
  });

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  if (isLoading) {
    return <p className="text-on-surface-variant">{t('common.loading')}</p>;
  }

  if (error || !data?.load_balancer) {
    return (
      <div className="space-y-4">
        <Link to="/load-balancers" className="inline-flex items-center gap-2 text-sm text-on-surface-variant hover:underline">
          <ArrowLeft size={16} /> {t('lb.title')}
        </Link>
        <InfoBanner variant="warning">{(error as Error)?.message || t('lb.empty')}</InfoBanner>
      </div>
    );
  }

  const lb = data.load_balancer;
  const listeners = data.listeners || [];
  const tgById = Object.fromEntries((tgs.data?.target_groups || []).map((tg) => [tg.id, tg]));

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <button type="button" className="btn-secondary p-2" onClick={() => navigate('/load-balancers')} aria-label={t('lb.title')}>
          <ArrowLeft size={18} />
        </button>
        <PageHeader
          className="flex-1"
          breadcrumb={t('lb.title')}
          title={lb.name}
          subtitle={lb.description || undefined}
          actions={(
            <>
              <RefreshButton onRefresh={() => void refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
              <button type="button" className="btn-danger-outline" onClick={() => setDeleteOpen(true)}>
                <Trash2 size={16} /> {t('common.delete')}
              </button>
            </>
          )}
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <SurfaceCard>
          <p className="text-xs text-on-surface-variant mb-1">{t('lb.vip')}</p>
          <p className="font-data-mono text-lg">{lb.vip || '—'}</p>
        </SurfaceCard>
        <SurfaceCard>
          <p className="text-xs text-on-surface-variant mb-1">{t('common.state')}</p>
          <StatusBadge status={lb.state} />
        </SurfaceCard>
        <SurfaceCard>
          <p className="text-xs text-on-surface-variant mb-1">{t('lb.listeners')}</p>
          <p className="text-lg font-medium">{listeners.length}</p>
        </SurfaceCard>
      </div>

      {lb.state.toLowerCase() === 'creating' && (
        <InfoBanner>{t('lb.creatingVip')}</InfoBanner>
      )}

      <RefreshingPanel isFetching={isRefetching} isLoading={false}>
        <SurfaceCard>
          <div className="flex items-center justify-between gap-3 mb-4">
            <h2 className="font-headline text-headline-md font-semibold">{t('lb.listeners')}</h2>
          </div>

          {listeners.length === 0 ? (
            <p className="text-sm text-on-surface-variant mb-4">{t('lb.listenersEmpty')}</p>
          ) : (
            <PageTable>
              <PageTableHead>
                <PageTableTh>{t('lb.protocol')}</PageTableTh>
                <PageTableTh>{t('lb.frontPort')}</PageTableTh>
                <PageTableTh>{t('lb.forwardTo')}</PageTableTh>
                <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
              </PageTableHead>
              <PageTableBody>
                {listeners.map((l) => {
                  const tg = tgById[l.target_group_id];
                  return (
                    <PageTableRow key={l.id}>
                      <PageTableTd className="uppercase font-data-mono text-sm">{l.protocol}</PageTableTd>
                      <PageTableTd className="font-data-mono">{l.port}</PageTableTd>
                      <PageTableTd>
                        {tg ? (
                          <Link to={`/target-groups/${tg.id}`} className="hover:underline">
                            {tg.name} <span className="text-on-surface-variant font-data-mono text-xs">:{tg.port}</span>
                          </Link>
                        ) : (
                          <span className="font-data-mono text-xs">{l.target_group_id.slice(0, 8)}…</span>
                        )}
                      </PageTableTd>
                      <PageTableTd className="text-right">
                        <button
                          type="button"
                          className="text-sm text-error hover:underline"
                          disabled={removeListener.isPending}
                          onClick={() => removeListener.mutate(l.id)}
                        >
                          {t('lb.removeListener')}
                        </button>
                      </PageTableTd>
                    </PageTableRow>
                  );
                })}
              </PageTableBody>
            </PageTable>
          )}

          <div className="border-t border-outline-variant/40 pt-4">
            <h3 className="text-sm font-medium mb-3">{t('lb.addListener')}</h3>
            {(tgs.data?.target_groups || []).length === 0 ? (
              <div className="flex flex-wrap items-center gap-3">
                <p className="text-sm text-on-surface-variant">{t('lb.noTg')}</p>
                <Link to="/target-groups" className="btn-secondary text-sm">{t('lb.goTg')}</Link>
              </div>
            ) : (
              <form
                className="flex flex-wrap items-end gap-3"
                onSubmit={(e) => {
                  e.preventDefault();
                  addListener.mutate();
                }}
              >
                <label className="space-y-1">
                  <span className="text-xs text-on-surface-variant">{t('lb.frontPort')}</span>
                  <input
                    className={formInputClass}
                    type="number"
                    min={1}
                    max={65534}
                    required
                    value={listenerForm.port}
                    onChange={(e) => setListenerForm({ ...listenerForm, port: Number(e.target.value) })}
                  />
                </label>
                <label className="space-y-1 min-w-[14rem] flex-1">
                  <span className="text-xs text-on-surface-variant">{t('lb.targetGroup')}</span>
                  <select
                    className={formSelectClass}
                    required
                    value={listenerForm.target_group_id}
                    onChange={(e) => setListenerForm({ ...listenerForm, target_group_id: e.target.value })}
                  >
                    <option value="">{t('lb.selectTg')}</option>
                    {(tgs.data?.target_groups || []).map((tg) => (
                      <option key={tg.id} value={tg.id}>{tg.name} (TCP :{tg.port})</option>
                    ))}
                  </select>
                </label>
                <button
                  type="submit"
                  className="btn-primary"
                  disabled={addListener.isPending || !listenerForm.target_group_id}
                >
                  <Plus size={16} /> {t('lb.addListener')}
                </button>
              </form>
            )}
            {addListener.error && (
              <p className="mt-2 text-sm text-error">{(addListener.error as Error).message}</p>
            )}
            {removeListener.error && (
              <p className="mt-2 text-sm text-error">{(removeListener.error as Error).message}</p>
            )}
          </div>
        </SurfaceCard>
      </RefreshingPanel>

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate()}
        title={t('lb.deleteTitle')}
        message={t('lb.deleteMessage')}
        resourceName={lb.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
      />
    </div>
  );
}
