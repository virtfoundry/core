import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { listVPCs, createVPC, updateVPC, deleteVPC, fetchVPCCIDRPlan } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { CIDRPicker } from '../components/CIDRPicker';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import type { VPC } from '../lib/platform-api';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice, formInputClass, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function VPCs() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editVpc, setEditVpc] = useState<VPC | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', cidr: '' });
  const [cidrMode, setCidrMode] = useState<'auto' | 'custom'>('auto');
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vpcs,
    queryFn: listVPCs,
    enabled: !needsTenant,
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
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('vpcs.title')}
        subtitle={`${vpcs.length} ${t('vpcs.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('vpcs.create')}
            </button>
          </>
        }
      />

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
      <SurfaceCard padding="none" className="overflow-hidden">
        <PageTable>
          <PageTableHead>
            <PageTableTh>{t('common.name')}</PageTableTh>
            <PageTableTh>{t('common.state')}</PageTableTh>
            <PageTableTh>{t('vpcs.privateNet')}</PageTableTh>
            <PageTableTh>CIDR</PageTableTh>
            <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
          </PageTableHead>
          <PageTableBody>
            {isLoading ? (
              <tr><td colSpan={5} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
            ) : filtered.length === 0 ? (
              <tr><td colSpan={5} className="text-center py-12 text-on-surface-variant">{t('vpcs.empty')}</td></tr>
            ) : (
              filtered.map((vpc) => (
                <PageTableRow key={vpc.id}>
                  <PageTableTd className="font-medium">{vpc.name}</PageTableTd>
                  <PageTableTd>
                    <StatusBadge status={vpc.state || 'active'} pulse={false} />
                  </PageTableTd>
                  <PageTableTd className="text-on-surface-variant text-sm">{t('vpcs.privateNet')}</PageTableTd>
                  <PageTableTd className="font-data-mono text-primary-fixed-dim">{vpc.cidr}</PageTableTd>
                  <PageTableTd>
                    <ResourceActions
                      variant="inline"
                      editLabel={t('common.edit')}
                      deleteLabel={t('common.delete')}
                      onEdit={() => setEditVpc(vpc)}
                      onDelete={() => setDeleteTarget({ id: vpc.id, name: vpc.name })}
                    />
                  </PageTableTd>
                </PageTableRow>
              ))
            )}
          </PageTableBody>
        </PageTable>
      </SurfaceCard>
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
              className={formInputClass} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t('vpcs.cidrLabel')}</label>
            <p className="text-xs text-on-surface-variant mb-2">{t('vpcs.cidrHint')}</p>
            <InfoBanner className="mb-3 text-xs">{t('vpcs.defaultSubnetInfo')}</InfoBanner>
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
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
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
                className={formInputClass} />
            </div>
            <p className="text-sm text-on-surface-variant font-data-mono">CIDR: {editVpc.cidr}</p>
            {updateMutation.isError && (
              <p className="text-error text-sm">{(updateMutation.error as Error).message}</p>
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
