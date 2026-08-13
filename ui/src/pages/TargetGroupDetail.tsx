import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Plus, Trash2 } from 'lucide-react';
import {
  getTargetGroup, deleteTargetGroup, registerTarget, deregisterTarget, listVMs,
} from '../lib/platform-api';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';

export function TargetGroupDetail() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [vmId, setVmId] = useState('');
  const [portOverride, setPortOverride] = useState('');

  const { data, isLoading, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.targetGroup(id),
    queryFn: () => getTargetGroup(id),
    enabled: !needsTenant && !!id,
  });

  const vms = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.targetGroup(id) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.targetGroups });
    void queryClient.invalidateQueries({ queryKey: queryKeys.loadBalancers });
  };

  const deleteMutation = useMutation({
    mutationFn: () => deleteTargetGroup(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.targetGroups });
      navigate('/target-groups');
    },
  });

  const addTarget = useMutation({
    mutationFn: () => registerTarget(id, {
      vm_id: vmId,
      ...(portOverride ? { port: Number(portOverride) } : {}),
    }),
    onSuccess: () => {
      invalidate();
      setVmId('');
      setPortOverride('');
    },
  });

  const removeTarget = useMutation({
    mutationFn: (tid: string) => deregisterTarget(id, tid),
    onSuccess: () => invalidate(),
  });

  const targets = data?.targets || [];
  const registered = useMemo(() => new Set(targets.map((t) => t.vm_id)), [targets]);
  const availableVms = (vms.data?.vms || []).filter((vm) => !registered.has(vm.id));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  if (isLoading) {
    return <p className="text-on-surface-variant">{t('common.loading')}</p>;
  }

  if (error || !data?.target_group) {
    return (
      <div className="space-y-4">
        <Link to="/target-groups" className="inline-flex items-center gap-2 text-sm text-on-surface-variant hover:underline">
          <ArrowLeft size={16} /> {t('tg.title')}
        </Link>
        <InfoBanner variant="warning">{(error as Error)?.message || t('tg.empty')}</InfoBanner>
      </div>
    );
  }

  const tg = data.target_group;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <button type="button" className="btn-secondary p-2" onClick={() => navigate('/target-groups')} aria-label={t('tg.title')}>
          <ArrowLeft size={18} />
        </button>
        <PageHeader
          className="flex-1"
          breadcrumb={t('tg.title')}
          title={tg.name}
          subtitle={`${tg.protocol.toUpperCase()} :${tg.port}`}
          actions={(
            <>
              <RefreshButton onRefresh={() => void refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
              <Link to="/load-balancers" className="btn-secondary text-sm">{t('lb.title')}</Link>
              <button type="button" className="btn-danger-outline" onClick={() => setDeleteOpen(true)}>
                <Trash2 size={16} /> {t('common.delete')}
              </button>
            </>
          )}
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <SurfaceCard>
          <p className="text-xs text-on-surface-variant mb-1">{t('lb.protocol')}</p>
          <p className="font-data-mono uppercase">{tg.protocol}</p>
        </SurfaceCard>
        <SurfaceCard>
          <p className="text-xs text-on-surface-variant mb-1">{t('tg.instancePort')}</p>
          <p className="font-data-mono text-lg">{tg.port}</p>
        </SurfaceCard>
        <SurfaceCard>
          <p className="text-xs text-on-surface-variant mb-1">{t('tg.targets')}</p>
          <p className="text-lg font-medium">{targets.length}</p>
        </SurfaceCard>
      </div>

      <RefreshingPanel isFetching={isRefetching} isLoading={false}>
        <SurfaceCard>
          <h2 className="font-headline text-headline-md font-semibold mb-4">{t('tg.targets')}</h2>

          {targets.length === 0 ? (
            <p className="text-sm text-on-surface-variant mb-4">{t('tg.targetsEmpty')}</p>
          ) : (
            <div className="mb-6">
              <PageTable>
                <PageTableHead>
                  <PageTableTh>{t('tg.vm')}</PageTableTh>
                  <PageTableTh>{t('tg.ip')}</PageTableTh>
                  <PageTableTh>{t('tg.port')}</PageTableTh>
                  <PageTableTh>{t('common.state')}</PageTableTh>
                  <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
                </PageTableHead>
                <PageTableBody>
                  {targets.map((tgt) => (
                    <PageTableRow key={tgt.id}>
                      <PageTableTd>
                        {tgt.vm_name ? (
                          <Link to={`/vms/${encodeURIComponent(tgt.vm_name)}`} className="hover:underline font-medium">
                            {tgt.vm_name}
                          </Link>
                        ) : (
                          <span className="font-data-mono text-xs">{tgt.vm_id.slice(0, 8)}…</span>
                        )}
                      </PageTableTd>
                      <PageTableTd className="font-data-mono text-sm">{tgt.ip}</PageTableTd>
                      <PageTableTd className="font-data-mono">{tgt.port || tg.port}</PageTableTd>
                      <PageTableTd className="text-sm capitalize">{tgt.state}</PageTableTd>
                      <PageTableTd className="text-right">
                        <button
                          type="button"
                          className="text-sm text-error hover:underline"
                          disabled={removeTarget.isPending}
                          onClick={() => removeTarget.mutate(tgt.id)}
                        >
                          {t('tg.deregister')}
                        </button>
                      </PageTableTd>
                    </PageTableRow>
                  ))}
                </PageTableBody>
              </PageTable>
            </div>
          )}

          <div className="border-t border-outline-variant/40 pt-4">
            <h3 className="text-sm font-medium mb-3">{t('tg.register')}</h3>
            {availableVms.length === 0 ? (
              <p className="text-sm text-on-surface-variant">{t('tg.noVm')}</p>
            ) : (
              <form
                className="flex flex-wrap items-end gap-3"
                onSubmit={(e) => {
                  e.preventDefault();
                  addTarget.mutate();
                }}
              >
                <label className="space-y-1 min-w-[16rem] flex-1">
                  <span className="text-xs text-on-surface-variant">{t('tg.vm')}</span>
                  <select
                    className={formSelectClass}
                    required
                    value={vmId}
                    onChange={(e) => setVmId(e.target.value)}
                  >
                    <option value="">{t('tg.selectVm')}</option>
                    {availableVms.map((vm) => {
                      const nicIp = vm.nics?.find((n) => n.ip)?.ip || vm.ip;
                      return (
                        <option key={vm.id} value={vm.id}>
                          {vm.display_name || vm.name}{nicIp ? ` (${nicIp})` : ''}
                        </option>
                      );
                    })}
                  </select>
                </label>
                <label className="space-y-1 w-36">
                  <span className="text-xs text-on-surface-variant">{t('tg.overridePort')}</span>
                  <input
                    className={formInputClass}
                    type="number"
                    min={1}
                    max={65535}
                    placeholder={String(tg.port)}
                    value={portOverride}
                    onChange={(e) => setPortOverride(e.target.value)}
                  />
                </label>
                <button type="submit" className="btn-primary" disabled={addTarget.isPending || !vmId}>
                  <Plus size={16} /> {t('tg.register')}
                </button>
              </form>
            )}
            {addTarget.error && (
              <p className="mt-2 text-sm text-error">{(addTarget.error as Error).message}</p>
            )}
            {removeTarget.error && (
              <p className="mt-2 text-sm text-error">{(removeTarget.error as Error).message}</p>
            )}
            {deleteMutation.error && (
              <p className="mt-2 text-sm text-error">{(deleteMutation.error as Error).message}</p>
            )}
          </div>
        </SurfaceCard>
      </RefreshingPanel>

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate()}
        title={t('tg.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={tg.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
      />
    </div>
  );
}
