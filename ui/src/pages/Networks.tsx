import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Network as NetworkIcon } from 'lucide-react';
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
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { isIsolatedNetwork } from '../lib/networks';
import type { Network, VPC } from '../lib/platform-api';
import {
  PageHeader, SearchField, EmptyState, ResourceGridCard, TenantRequiredNotice,
  formInputClass, formSelectClass, InfoBanner,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

type DeleteTarget = { id: string; name: string };

function NetworkCard({
  net,
  vpcs,
  onEdit,
  onDelete,
  t,
}: {
  net: Network;
  vpcs: VPC[];
  onEdit: (net: Network) => void;
  onDelete: (target: DeleteTarget) => void;
  t: (key: import('../lib/i18n').TranslationKey) => string;
}) {
  const vpc = vpcs.find((v) => v.id === net.vpc_id);

  return (
    <ResourceGridCard>
      <div className="flex items-start justify-between gap-2 mb-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="w-10 h-10 bg-primary-container/20 rounded-lg flex items-center justify-center shrink-0">
            <NetworkIcon size={20} className="text-primary-fixed-dim" />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="font-headline text-headline-md font-semibold text-on-surface">{net.name}</h3>
              <span className="px-2 py-0.5 rounded text-xs font-medium bg-surface-container-high text-on-surface-variant">
                {t('networks.typeIsolated')}
              </span>
              {net.name === 'default' && (
                <span className="px-2 py-0.5 rounded text-xs font-medium bg-primary-container/20 text-primary-fixed-dim">
                  {t('networks.defaultBadge')}
                </span>
              )}
            </div>
            <p className="text-sm text-on-surface-variant">
              {vpc ? t('networks.vpcLabel').replace('{name}', vpc.name) : t('networks.vpc')}
            </p>
          </div>
        </div>
        <StatusBadge status={net.state || 'active'} pulse={false} />
      </div>
      {net.name === 'default' && (
        <p className="text-xs text-on-surface-variant mb-3">{t('networks.defaultHint')}</p>
      )}
      <div className="text-sm mb-4">
        <p className="text-on-surface-variant">{t('networks.cidr')}</p>
        <p className="font-data-mono text-primary-fixed-dim">{net.cidr}</p>
      </div>
      <ResourceActions
        editLabel={t('common.edit')}
        deleteLabel={t('common.delete')}
        onEdit={() => onEdit(net)}
        onDelete={() => onDelete({ id: net.id, name: net.name })}
      />
    </ResourceGridCard>
  );
}

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
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant,
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

  const isolatedNetworks = (data?.networks ?? []).filter(isIsolatedNetwork);
  const vpcs = vpcData?.vpcs ?? [];
  const filtered = isolatedNetworks.filter((n) => n.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  if (error) {
    return (
      <div className="text-center py-12 space-y-3">
        <p className="text-error">{t('common.errorLoad')}: {(error as Error).message}</p>
        <button type="button" onClick={() => refetch()} className="btn-primary">{t('common.retry')}</button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('networks.title')}
        subtitle={`${filtered.length} ${t('networks.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary" disabled={vpcs.length === 0}>
              <Plus size={18} /> {t('networks.create')}
            </button>
          </>
        }
      />

      <InfoBanner>
        {t('networks.isolatedOnlyHint')}{' '}
        <Link to="/networks/public" className="font-medium underline">{t('nav.publicNetwork')}</Link>
      </InfoBanner>

      {vpcs.length === 0 && (
        <InfoBanner variant="warning">
          {t('networks.emptyHint')}{' '}
          <Link to="/vpcs" className="font-medium underline">{t('vpcs.create')}</Link>
        </InfoBanner>
      )}

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <EmptyState icon={<NetworkIcon size={48} />} title={t('networks.empty')} hint={t('networks.emptyHint')} />
          ) : (
            filtered.map((net) => (
              <NetworkCard
                key={net.id}
                net={net}
                vpcs={vpcs}
                onEdit={setEditNet}
                onDelete={setDeleteTarget}
                t={t}
              />
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
              className={formSelectClass}>
              <option value="">{t('networks.selectVpc')}</option>
              {vpcs.map((v) => (
                <option key={v.id} value={v.id}>{v.name} ({v.cidr})</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={formInputClass} placeholder="private-net" />
          </div>
          {form.vpc_id && (
            <>
              <SubnetPrefixSelect prefix={prefix} onChange={setPrefix} />
              <div>
                <label className="block text-sm font-medium mb-2">{t('networks.ipRange')}</label>
                {selectedVPC && (
                  <p className="text-xs text-on-surface-variant mb-2">
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
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
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
                className={formInputClass} />
            </div>
            <p className="text-sm text-on-surface-variant">{t('networks.cidr')}: {editNet.cidr}</p>
            {updateMutation.isError && (
              <p className="text-error text-sm">{(updateMutation.error as Error).message}</p>
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
