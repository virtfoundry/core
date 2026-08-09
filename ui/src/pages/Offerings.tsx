import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Cpu } from 'lucide-react';
import {
  listAllServiceOfferings, createServiceOffering, updateServiceOffering, deleteServiceOffering,
} from '../lib/platform-api';
import type { ServiceOffering } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { useAppSelector } from '../store/hooks';
import { selectIsRoot } from '../store/authSlice';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, EmptyState, ResourceGridCard,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';
import { offeringLabel } from '../lib/offerings';

type OfferingForm = {
  name: string;
  display_name: string;
  cpu: number;
  memory_gi: number;
  dedicated_cpu: boolean;
  state: string;
};

const emptyForm = (): OfferingForm => ({
  name: '',
  display_name: '',
  cpu: 1,
  memory_gi: 1,
  dedicated_cpu: false,
  state: 'Active',
});

function fmtMemGi(mi: number) {
  return mi >= 1024 ? `${(mi / 1024).toFixed(1)} GiB` : `${mi} MiB`;
}

export function Offerings() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editOffering, setEditOffering] = useState<ServiceOffering | null>(null);
  const [deactivateTarget, setDeactivateTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState(emptyForm());
  const queryClient = useQueryClient();
  const isRoot = useAppSelector(selectIsRoot);

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.allOfferings,
    queryFn: listAllServiceOfferings,
    enabled: isRoot,
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.allOfferings });
    queryClient.invalidateQueries({ queryKey: queryKeys.offerings });
  };

  const createMutation = useMutation({
    mutationFn: createServiceOffering,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm(emptyForm());
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & Parameters<typeof updateServiceOffering>[1]) =>
      updateServiceOffering(id, payload),
    onSuccess: () => {
      invalidate();
      setEditOffering(null);
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: deleteServiceOffering,
    onSuccess: () => {
      invalidate();
      setDeactivateTarget(null);
    },
  });

  if (!isRoot) {
    return (
      <div className="text-center py-16 text-on-surface-variant">
        {t('offerings.rootOnly')}
      </div>
    );
  }

  const offerings = data?.service_offerings || [];
  const filtered = offerings.filter((o) =>
    o.name.toLowerCase().includes(search.toLowerCase()) ||
    o.display_name.toLowerCase().includes(search.toLowerCase())
  );

  const openEdit = (o: ServiceOffering) => {
    setForm({
      name: o.name,
      display_name: o.display_name,
      cpu: o.cpu,
      memory_gi: o.memory_mi / 1024,
      dedicated_cpu: !!o.dedicated_cpu,
      state: o.state,
    });
    setEditOffering(o);
  };

  const submitCreate = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate({
      name: form.name,
      display_name: form.display_name,
      cpu: form.cpu,
      memory_mi: Math.round(form.memory_gi * 1024),
      dedicated_cpu: form.dedicated_cpu,
    });
  };

  const submitEdit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editOffering) return;
    updateMutation.mutate({
      id: editOffering.id,
      display_name: form.display_name,
      cpu: form.cpu,
      memory_mi: Math.round(form.memory_gi * 1024),
      dedicated_cpu: form.dedicated_cpu,
      state: form.state,
    });
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('offerings.title')}
        subtitle={`${offerings.length} ${t('offerings.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => { setForm(emptyForm()); setCreateModal(true); }} className="btn-primary">
              <Plus size={18} /> {t('offerings.create')}
            </button>
          </>
        }
      />

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('offerings.searchPlaceholder')} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <EmptyState icon={<Cpu size={48} />} title={t('offerings.empty')} />
          ) : (
            filtered.map((o) => (
              <ResourceGridCard key={o.id}>
                <div className="flex items-start justify-between gap-2 mb-3">
                  <h3 className="font-headline text-headline-md font-semibold text-on-surface">{offeringLabel(o)}</h3>
                  <StatusBadge status={o.state?.toLowerCase() || 'active'} pulse={false} />
                </div>
                <p className="text-sm text-on-surface-variant mb-3 font-data-mono">{o.name}</p>
                <div className="text-sm space-y-1 mb-4">
                  <p><span className="text-on-surface-variant">{t('offerings.cpu')}:</span> {o.cpu} vCPU</p>
                  <p><span className="text-on-surface-variant">{t('offerings.memory')}:</span> {fmtMemGi(o.memory_mi)}</p>
                  <p>
                    <span className="text-on-surface-variant">{t('offerings.dedicatedCpu')}:</span>{' '}
                    {o.dedicated_cpu ? t('common.yes') : t('common.no')}
                  </p>
                </div>
                <ResourceActions
                  onEdit={() => openEdit(o)}
                  onDelete={o.state === 'Active' ? () => setDeactivateTarget({ id: o.id, name: o.name }) : undefined}
                  deleteLabel={t('offerings.deactivate')}
                />
              </ResourceGridCard>
            ))
          )}
        </div>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('offerings.modalCreate')}>
        <form onSubmit={submitCreate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input
              required
              pattern="[-a-z0-9]+"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
              className={formInputClass}
              placeholder="xlarge"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('offerings.displayName')}</label>
            <input
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              className={formInputClass}
              placeholder="XLarge (8 vCPU, 16 GiB)"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('offerings.cpu')}</label>
              <input
                required
                type="number"
                min={1}
                value={form.cpu}
                onChange={(e) => setForm({ ...form, cpu: Number(e.target.value) })}
                className={formInputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('offerings.memoryGi')}</label>
              <input
                required
                type="number"
                min={0.5}
                step={0.5}
                value={form.memory_gi}
                onChange={(e) => setForm({ ...form, memory_gi: Number(e.target.value) })}
                className={formInputClass}
              />
            </div>
          </div>
          <label className="flex items-start gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              className="mt-1"
              checked={form.dedicated_cpu}
              onChange={(e) => setForm({ ...form, dedicated_cpu: e.target.checked })}
            />
            <span>
              <span className="font-medium">{t('offerings.dedicatedCpu')}</span>
              <p className="text-xs text-on-surface-variant mt-0.5">{t('offerings.dedicatedCpuHint')}</p>
            </span>
          </label>
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

      <Modal isOpen={!!editOffering} onClose={() => setEditOffering(null)} title={t('offerings.modalEdit')}>
        <form onSubmit={submitEdit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input value={form.name} disabled className={`${formInputClass} opacity-60`} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('offerings.displayName')}</label>
            <input
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              className={formInputClass}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('offerings.cpu')}</label>
              <input
                required
                type="number"
                min={1}
                value={form.cpu}
                onChange={(e) => setForm({ ...form, cpu: Number(e.target.value) })}
                className={formInputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('offerings.memoryGi')}</label>
              <input
                required
                type="number"
                min={0.5}
                step={0.5}
                value={form.memory_gi}
                onChange={(e) => setForm({ ...form, memory_gi: Number(e.target.value) })}
                className={formInputClass}
              />
            </div>
          </div>
          <label className="flex items-start gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              className="mt-1"
              checked={form.dedicated_cpu}
              onChange={(e) => setForm({ ...form, dedicated_cpu: e.target.checked })}
            />
            <span>
              <span className="font-medium">{t('offerings.dedicatedCpu')}</span>
              <p className="text-xs text-on-surface-variant mt-0.5">{t('offerings.dedicatedCpuHint')}</p>
            </span>
          </label>
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.state')}</label>
            <select
              value={form.state}
              onChange={(e) => setForm({ ...form, state: e.target.value })}
              className={formSelectClass}
            >
              <option value="Active">{t('offerings.stateActive')}</option>
              <option value="Inactive">{t('offerings.stateInactive')}</option>
            </select>
          </div>
          {updateMutation.isError && (
            <p className="text-error text-sm">{(updateMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setEditOffering(null)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={updateMutation.isPending} className="btn-primary">
              {updateMutation.isPending ? t('common.loading') : t('common.save')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={!!deactivateTarget}
        onClose={() => setDeactivateTarget(null)}
        onConfirm={() => deactivateTarget && deactivateMutation.mutate(deactivateTarget.id)}
        title={t('offerings.deactivateTitle')}
        message={t('offerings.deactivateMessage')}
        resourceName={deactivateTarget?.name}
        confirmLabel={t('offerings.deactivate')}
        loading={deactivateMutation.isPending}
      />
    </div>
  );
}
