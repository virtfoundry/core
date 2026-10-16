import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2 } from 'lucide-react';
import { listVolumes, createVolume, deleteVolume, listVMs } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { OverflowMenu } from '../components/OverflowMenu';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { ago } from '../lib/ago';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';
import { StatsCard } from '../components/StatsCard';

export function Volumes() {
  const { t, dateLocale } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', size_gi: 10 });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.volumes,
    queryFn: listVolumes,
    enabled: !needsTenant,
  });

  const { data: vmData } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
  });

  const createMutation = useMutation({
    mutationFn: createVolume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.volumes });
      setCreateModal(false);
      setForm({ name: '', size_gi: 10 });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteVolume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.volumes });
      setDeleteTarget(null);
    },
  });

  const volumes = data?.volumes || [];
  const vmById = Object.fromEntries((vmData?.vms || []).map((vm) => [vm.id, vm]));
  const filtered = volumes.filter((v) => v.name?.toLowerCase().includes(search.toLowerCase()));

  const { rows, pager } = usePagination(filtered, search);

  if (needsTenant) {
    return <TenantRequiredNotice message={t('volumes.selectTenant')} />;
  }

  const attachedCount = volumes.filter((v) => v.vm_id).length;
  const totalGi = volumes.reduce((n, v) => n + (v.size_gi || 0), 0);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('nav.volumes')}
        subtitle={`${volumes.length} ${t('volumes.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('volumes.create')}
            </button>
          </>
        }
      />

      {volumes.length > 0 && (
        <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(220px,1fr))]">
          <StatsCard
            title={t('volumes.stat.attached')}
            value={attachedCount}
            of={t('dashboard.of').replace('{n}', String(volumes.length))}
            progress={Math.round((attachedCount / volumes.length) * 100)}
          />
          <StatsCard title={t('volumes.stat.capacity')} value={`${totalGi} GiB`} />
        </section>
      )}

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
              <PageTableTh>{t('volumes.col.volume')}</PageTableTh>
              <PageTableTh>{t('volumes.size')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh>{t('volumes.attachedVm')}</PageTableTh>
              <PageTableTh>PVC</PageTableTh>
              <PageTableTh>{t('vms.col.age')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={7}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={7}>{t('volumes.empty')}</TableMessageRow>
              ) : (
                rows.map((vol) => {
                  const attachedVm = vol.vm_id ? vmById[vol.vm_id] : undefined;
                  return (
                    <PageTableRow key={vol.id}>
                      <PageTableTd className="font-semibold">{vol.name}</PageTableTd>
                      <PageTableTd mono>{vol.size_gi} GiB</PageTableTd>
                      <PageTableTd>
                        <StatusBadge status={vol.state || 'active'} pulse={false} />
                      </PageTableTd>
                      <PageTableTd mono>
                        {attachedVm ? (
                          <Link to={`/vms/${attachedVm.name}`} className="text-link hover:underline">
                            {attachedVm.display_name || attachedVm.name}
                          </Link>
                        ) : (
                          <span className="text-on-surface-variant">{t('volumes.unattached')}</span>
                        )}
                      </PageTableTd>
                      <PageTableTd mono className="text-on-surface-variant">{vol.pvc_name}</PageTableTd>
                      <PageTableTd className="text-on-surface-variant whitespace-nowrap">{ago(vol.created_at, dateLocale)}</PageTableTd>
                      <PageTableTd className="text-right">
                        <OverflowMenu
                          variant="dots"
                          label={t('table.moreActions')}
                          items={[
                            ...(vol.vm_id
                              ? []
                              : [{
                                  id: 'delete',
                                  label: t('common.delete'),
                                  icon: <Trash2 size={14} />,
                                  danger: true,
                                  onSelect: () => setDeleteTarget({ id: vol.id, name: vol.name }),
                                }]),
                          ]}
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

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('volumes.deleteTitle')}
        message={t('volumes.deleteConfirm')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        loading={deleteMutation.isPending}
      />

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('volumes.modalTitle')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required pattern="[-a-z0-9]+" value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
              className={formInputClass} placeholder="data-disk-01" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('volumes.sizeGi')}</label>
            <input type="number" min={1} required value={form.size_gi}
              onChange={(e) => setForm({ ...form, size_gi: parseInt(e.target.value, 10) })}
              className={formInputClass} />
          </div>
          {createMutation.isError && (
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
