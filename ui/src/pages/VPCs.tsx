import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Globe } from 'lucide-react';
import { listVPCs, createVPC, updateVPC, deleteVPC, fetchVPCCIDRPlan } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { CIDRPicker } from '../components/CIDRPicker';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';
import type { VPC } from '../lib/platform-api';

export function VPCs() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editVpc, setEditVpc] = useState<VPC | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', cidr: '' });
  const [cidrMode, setCidrMode] = useState<'auto' | 'custom'>('auto');
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vpcs,
    queryFn: listVPCs,
    enabled: !needsTenant,
    refetchInterval: 12_000,
  });

  const { data: cidrPlan, isLoading: cidrLoading } = useQuery({
    queryKey: [...queryKeys.vpcs, 'cidr-plan'],
    queryFn: fetchVPCCIDRPlan,
    enabled: createModal && !needsTenant,
  });

  const autoCIDR = cidrPlan?.suggestions.find((s) => s.available)?.cidr || '';

  useEffect(() => {
    if (createModal && cidrMode === 'auto' && autoCIDR) {
      setForm((f) => ({ ...f, cidr: autoCIDR }));
    }
  }, [createModal, cidrMode, autoCIDR]);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.vpcs });
    queryClient.invalidateQueries({ queryKey: queryKeys.networks });
  };

  const createMutation = useMutation({
    mutationFn: createVPC,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', cidr: '' });
      setCidrMode('auto');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updateVPC(id, { name }),
    onSuccess: () => {
      invalidate();
      setEditVpc(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteVPC,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const vpcs = data?.vpcs || [];
  const filtered = vpcs.filter((v) => v.name.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('common.selectTenant')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('vpcs.title')}</h1>
          <p className="text-gray-500">{vpcs.length} {t('vpcs.subtitle')}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary">
            <Plus size={18} /> {t('vpcs.create')}
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`}
          className="w-full pl-10 pr-4 py-3 border rounded-lg" />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full text-center py-12">{t('common.loading')}</div>
        ) : filtered.length === 0 ? (
          <div className="col-span-full text-center py-12">
            <Globe size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">{t('vpcs.empty')}</p>
          </div>
        ) : (
          filtered.map((vpc) => (
            <div key={vpc.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5">
              <h3 className="font-semibold text-lg">{vpc.name}</h3>
              <p className="text-sm text-gray-500 mb-2">{t('vpcs.privateNet')} · {vpc.cidr}</p>
              <p className="text-sm"><span className="text-gray-500">{t('common.state')}:</span> {vpc.state}</p>
              <ResourceActions
                editLabel={t('common.edit')}
                deleteLabel={t('common.delete')}
                onEdit={() => setEditVpc(vpc)}
                onDelete={() => setDeleteTarget({ id: vpc.id, name: vpc.name })}
              />
            </div>
          ))
        )}
      </div>
      </RefreshingPanel>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('vpcs.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        loading={deleteMutation.isPending}
      />

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('vpcs.modalCreate')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate({
              name: form.name,
              ...(cidrMode === 'custom' ? { cidr: form.cidr } : {}),
            });
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t('vpcs.cidrLabel')}</label>
            <p className="text-xs text-gray-500 mb-2">{t('vpcs.cidrHint')}</p>
            <div className="rounded-lg border border-blue-200 bg-blue-50 dark:bg-blue-900/20 px-3 py-2 text-xs text-blue-800 dark:text-blue-200 mb-3">
              {t('vpcs.defaultSubnetInfo')}
            </div>
            <CIDRPicker
              mode={cidrMode}
              onModeChange={setCidrMode}
              value={form.cidr}
              onChange={(cidr) => setForm({ ...form, cidr })}
              suggestions={cidrPlan?.suggestions || []}
              autoValue={autoCIDR}
              loading={cidrLoading}
              customPlaceholder="10.0.0.0/16"
            />
          </div>
          {createMutation.isError && (
            <p className="text-red-500 text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={!!editVpc} onClose={() => setEditVpc(null)} title={t('vpcs.modalEdit')}>
        {editVpc && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              updateMutation.mutate({ id: editVpc.id, name: editVpc.name });
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
              <input required value={editVpc.name} onChange={(e) => setEditVpc({ ...editVpc, name: e.target.value })}
                className="w-full px-4 py-2 border rounded-lg" />
            </div>
            <p className="text-sm text-gray-500">CIDR: {editVpc.cidr}</p>
            {updateMutation.isError && (
              <p className="text-red-500 text-sm">{(updateMutation.error as Error).message}</p>
            )}
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={() => setEditVpc(null)} className="btn-secondary">{t('common.cancel')}</button>
              <button type="submit" disabled={updateMutation.isPending} className="btn-primary">{t('common.save')}</button>
            </div>
          </form>
        )}
      </Modal>
    </div>
  );
}
