import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Network as NetworkIcon } from 'lucide-react';
import { listVPCs, listNetworks, createVPC, updateVPC, deleteVPC, fetchVPCCIDRPlan } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { CIDRPicker } from '../components/CIDRPicker';
import { ResourceActions } from '../components/ResourceActions';
import { TagsInput, TagPills, mergeTags } from '../components/TagsInput';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import type { VPC } from '../lib/platform-api';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice, formInputClass, formLabelClass, InfoBanner, EmptyState,
} from '../components/shell';
import { StatsCard } from '../components/StatsCard';

export function VPCs() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editVpc, setEditVpc] = useState<VPC | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', cidr: '', tags: [] as string[] });
  const [tagDraft, setTagDraft] = useState('');
  const [cidrMode, setCidrMode] = useState<'auto' | 'custom'>('auto');
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vpcs,
    queryFn: listVPCs,
    enabled: !needsTenant,
  });

  const { data: netData } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
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
      setForm({ name: '', cidr: '', tags: [] });
      setTagDraft('');
      setCidrMode('auto');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, name, tags }: { id: string; name: string; tags: string[] }) => updateVPC(id, { name, tags }),
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
  const subnetsOf = (id: string) => (netData?.networks || []).filter((n) => n.vpc_id === id);
  const subnetCount = vpcs.reduce((acc, v) => acc + subnetsOf(v.id).length, 0);

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

      <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(200px,1fr))]">
        <StatsCard title="VPCs" value={vpcs.length} />
        {netData && <StatsCard title={t('vpcs.subnets')} value={subnetCount} />}
      </section>

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        {isLoading ? (
          <p className="text-center py-12 text-on-surface-variant">{t('common.loading')}</p>
        ) : filtered.length === 0 ? (
          <SurfaceCard>
            <EmptyState icon={<NetworkIcon />} title={t('vpcs.empty')} />
          </SurfaceCard>
        ) : (
          <div className="flex flex-wrap gap-4 items-stretch">
            {filtered.map((vpc) => {
              const subnets = subnetsOf(vpc.id);
              return (
                <SurfaceCard
                  key={vpc.id}
                  className="flex-1 basis-[420px] min-w-0"
                  title={vpc.name}
                  headerAction={
                    <>
                      <span className="font-mono text-[13px] text-on-surface-variant">{vpc.cidr}</span>
                      <ResourceActions
                        variant="inline"
                        editLabel={t('common.edit')}
                        deleteLabel={t('common.delete')}
                        onEdit={() => { setTagDraft(''); setEditVpc(vpc); }}
                        onDelete={() => setDeleteTarget({ id: vpc.id, name: vpc.name })}
                      />
                    </>
                  }
                >
                  {vpc.tags && vpc.tags.length > 0 && (
                    <div className="mb-3"><TagPills tags={vpc.tags} /></div>
                  )}
                  {subnets.length === 0 ? (
                    <p className="m-0 text-sm text-on-surface-variant">{t('vpcs.noSubnets')}</p>
                  ) : (
                    <ul className="m-0 p-0 list-none flex flex-col">
                      {subnets.map((n) => (
                        <li key={n.id} className="flex items-center gap-3 px-2 py-2.5 rounded-lg">
                          <span
                            className={`w-2 h-2 rounded-full shrink-0 ${n.state === 'active' || n.state === 'Ready' ? 'bg-ok-fg' : 'bg-off-fg'}`}
                            aria-hidden
                          />
                          <span className="flex-1 min-w-0 flex flex-col">
                            <span className="font-medium">{n.name}</span>
                            <span className="font-mono text-xs text-on-surface-variant">{n.cidr}</span>
                          </span>
                          <span className="font-mono text-xs text-on-surface-variant">NAD</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </SurfaceCard>
              );
            })}
          </div>
        )}
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
              tags: mergeTags(form.tags, tagDraft),
              ...(cidrMode === 'custom' ? { cidr: form.cidr } : {}),
            });
          }}
          className="space-y-4"
        >
          <div>
            <label className={formLabelClass}>{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={formInputClass} />
          </div>
          <div>
            <label className={formLabelClass}>{t('vpcs.cidrLabel')}</label>
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
          <div>
            <label className={formLabelClass}>{t('tags.column')}</label>
            <TagsInput
              tags={form.tags}
              onTagsChange={(tags) => setForm({ ...form, tags })}
              draft={tagDraft}
              onDraftChange={setTagDraft}
              placeholder={t('common.tagsPlaceholder')}
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
              updateMutation.mutate({ id: editVpc.id, name: editVpc.name, tags: mergeTags(editVpc.tags || [], tagDraft) });
            }}
            className="space-y-4"
          >
            <div>
              <label className={formLabelClass}>{t('common.name')}</label>
              <input required value={editVpc.name} onChange={(e) => setEditVpc({ ...editVpc, name: e.target.value })}
                className={formInputClass} />
            </div>
            <p className="text-sm text-on-surface-variant font-data-mono">CIDR: {editVpc.cidr}</p>
            <div>
              <label className={formLabelClass}>{t('tags.column')}</label>
              <TagsInput
                tags={editVpc.tags || []}
                onTagsChange={(tags) => setEditVpc({ ...editVpc, tags })}
                draft={tagDraft}
                onDraftChange={setTagDraft}
                placeholder={t('common.tagsPlaceholder')}
              />
            </div>
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
