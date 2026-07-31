import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Network as NetworkIcon } from 'lucide-react';
import {
  listNetworks, createNetwork, updateNetwork, deleteNetwork, listVPCs, fetchNetworkCIDRPlan,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { CIDRPicker, SubnetPrefixSelect } from '../components/CIDRPicker';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';
import type { Network } from '../lib/platform-api';

type DeleteTarget = { id: string; name: string };

export function Networks() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editNet, setEditNet] = useState<Network | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [form, setForm] = useState({ name: '', cidr: '', vpc_id: '' });
  const [cidrMode, setCidrMode] = useState<'auto' | 'custom'>('auto');
  const [prefix, setPrefix] = useState(24);
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant,
    refetchInterval: 12_000,
  });

  const { data: vpcData } = useQuery({
    queryKey: queryKeys.vpcs,
    queryFn: listVPCs,
    enabled: !needsTenant,
  });

  const { data: subnetPlan, isLoading: planLoading } = useQuery({
    queryKey: [...queryKeys.networks, 'cidr-plan', form.vpc_id, prefix],
    queryFn: () => fetchNetworkCIDRPlan(form.vpc_id, prefix),
    enabled: createModal && !!form.vpc_id,
  });

  const selectedVPC = vpcData?.vpcs.find((v) => v.id === form.vpc_id);
  const autoCIDR = subnetPlan?.auto || '';

  useEffect(() => {
    if (cidrMode === 'auto' && autoCIDR) {
      setForm((f) => ({ ...f, cidr: autoCIDR }));
    }
  }, [cidrMode, autoCIDR, prefix]);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.networks });

  const createMutation = useMutation({
    mutationFn: createNetwork,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', cidr: '', vpc_id: '' });
      setCidrMode('auto');
      setPrefix(24);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => updateNetwork(id, { name }),
    onSuccess: () => {
      invalidate();
      setEditNet(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteNetwork,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const networks = data?.networks ?? [];
  const vpcs = vpcData?.vpcs ?? [];
  const filtered = networks.filter((n) => n.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('common.selectTenant')}</div>;
  }

  if (error) {
    return (
      <div className="text-center py-12 space-y-3">
        <p className="text-red-600">{t('common.errorLoad')}: {(error as Error).message}</p>
        <button type="button" onClick={() => refetch()} className="btn-primary">{t('common.retry')}</button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('networks.title')}</h1>
          <p className="text-gray-500">{networks.length} {t('networks.subtitle')}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary" disabled={vpcs.length === 0}>
            <Plus size={18} /> {t('networks.create')}
          </button>
        </div>
      </div>

      {vpcs.length === 0 && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 dark:bg-amber-900/20 px-4 py-3 text-sm text-amber-800 dark:text-amber-200">
          {t('networks.emptyHint')}{' '}
          <Link to="/vpcs" className="font-medium underline">{t('vpcs.create')}</Link>
        </div>
      )}

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
            <NetworkIcon size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">{t('networks.empty')}</p>
            <p className="text-sm text-gray-400 mt-2 max-w-md mx-auto">{t('networks.emptyHint')}</p>
          </div>
        ) : (
          filtered.map((net) => (
            <div key={net.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5 hover:shadow-lg transition">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
                    <NetworkIcon size={20} className="text-blue-500" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold">{net.name}</h3>
                      {net.name === 'default' && (
                        <span className="px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300">
                          {t('networks.defaultBadge')}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-gray-500">
                      {vpcs.find((v) => v.id === net.vpc_id)?.name || 'VPC'}
                    </p>
                  </div>
                </div>
                <span className="px-2 py-1 rounded text-xs font-medium bg-green-100 text-green-700">{net.state}</span>
              </div>
              {net.name === 'default' && (
                <p className="text-xs text-gray-500 mb-3">{t('networks.defaultHint')}</p>
              )}
              <div className="text-sm">
                <p className="text-gray-500">{t('networks.cidr')}</p>
                <p className="font-medium font-mono">{net.cidr}</p>
              </div>
              <ResourceActions
                editLabel={t('common.edit')}
                deleteLabel={t('common.delete')}
                onEdit={() => setEditNet(net)}
                onDelete={() => setDeleteTarget({ id: net.id, name: net.name })}
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
        title={t('networks.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        loading={deleteMutation.isPending}
      />

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('networks.modalCreate')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (!form.vpc_id) return;
            createMutation.mutate({
              name: form.name,
              vpc_id: form.vpc_id,
              prefix,
              ...(cidrMode === 'custom' ? { cidr: form.cidr } : {}),
            });
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('networks.vpc')}</label>
            <select required value={form.vpc_id} onChange={(e) => setForm({ ...form, vpc_id: e.target.value, cidr: '' })}
              className="w-full px-4 py-2 border rounded-lg">
              <option value="">{t('networks.selectVpc')}</option>
              {vpcs.map((v) => (
                <option key={v.id} value={v.id}>{v.name} ({v.cidr})</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg" placeholder="private-net" />
          </div>
          {form.vpc_id && (
            <>
              <SubnetPrefixSelect prefix={prefix} onChange={setPrefix} />
              <div>
                <label className="block text-sm font-medium mb-2">{t('networks.ipRange')}</label>
                {selectedVPC && (
                  <p className="text-xs text-gray-500 mb-2">
                    {t('networks.insideVpc')} {selectedVPC.name} ({selectedVPC.cidr})
                  </p>
                )}
                <CIDRPicker
                  mode={cidrMode}
                  onModeChange={setCidrMode}
                  value={form.cidr}
                  onChange={(cidr) => setForm({ ...form, cidr })}
                  suggestions={subnetPlan?.suggestions || []}
                  autoValue={autoCIDR}
                  loading={planLoading}
                  customPlaceholder="10.0.1.0/24"
                />
              </div>
            </>
          )}
          {createMutation.isError && (
            <p className="text-red-500 text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending || !form.vpc_id} className="btn-primary">{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={!!editNet} onClose={() => setEditNet(null)} title={t('networks.modalEdit')}>
        {editNet && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              updateMutation.mutate({ id: editNet.id, name: editNet.name });
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
              <input required value={editNet.name} onChange={(e) => setEditNet({ ...editNet, name: e.target.value })}
                className="w-full px-4 py-2 border rounded-lg" />
            </div>
            <p className="text-sm text-gray-500">{t('networks.cidr')}: {editNet.cidr}</p>
            {updateMutation.isError && (
              <p className="text-red-500 text-sm">{(updateMutation.error as Error).message}</p>
            )}
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={() => setEditNet(null)} className="btn-secondary">{t('common.cancel')}</button>
              <button type="submit" disabled={updateMutation.isPending} className="btn-primary">{t('common.save')}</button>
            </div>
          </form>
        )}
      </Modal>
    </div>
  );
}
