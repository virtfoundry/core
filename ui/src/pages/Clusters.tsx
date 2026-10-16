import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Download } from 'lucide-react';
import {
  listVKSClusters, createVKSCluster, deleteVKSCluster, downloadVKSKubeconfig,
  listNetworks, listSSHKeys, listVMTemplates, listServiceOfferings,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { OverflowMenu } from '../components/OverflowMenu';
import { StatsCard } from '../components/StatsCard';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { StatusBadge } from '../components/StatusBadge';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  vksCanDownloadKubeconfig, vksNodesLabel, vksPhaseBadgeStatus,
} from '../lib/vks-display';
import { resolveVKSVersionOptions } from '../lib/vks-catalog';
import { findOfferingByName, offeringLabel } from '../lib/offerings';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass, InfoBanner,
} from '../components/shell';

const DEFAULT_NETWORK = 'default';
const DEFAULT_WORKERS = 1;

type CreateFormState = {
  name: string;
  kubernetes_version: string;
  template: string;
  offering: string;
  network: string;
  workers: number;
  port: string;
  sshKeyNames: string[];
};

function emptyCreateForm(): CreateFormState {
  return {
    name: '',
    kubernetes_version: '',
    template: '',
    offering: '',
    network: DEFAULT_NETWORK,
    workers: DEFAULT_WORKERS,
    port: '',
    sshKeyNames: [],
  };
}

export function Clusters() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState<CreateFormState>(emptyCreateForm);
  const [deleteTarget, setDeleteTarget] = useState<{ name: string } | null>(null);
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vksClusters,
    queryFn: listVKSClusters,
    enabled: !needsTenant,
    refetchInterval: 10_000,
  });

  const catalogEnabled = !needsTenant;
  const { data: networksData } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: catalogEnabled,
  });
  const { data: templatesData, isFetched: templatesFetched } = useQuery({
    queryKey: queryKeys.templates,
    queryFn: listVMTemplates,
    enabled: catalogEnabled,
  });
  const { data: offeringsData } = useQuery({
    queryKey: queryKeys.offerings,
    queryFn: listServiceOfferings,
    enabled: catalogEnabled,
  });
  const { data: sshData } = useQuery({
    queryKey: queryKeys.sshKeys,
    queryFn: listSSHKeys,
    enabled: catalogEnabled,
  });

  const networkList = networksData?.networks;
  const offeringList = offeringsData?.service_offerings;
  const networks = networkList ?? [];
  const offerings = offeringList ?? [];
  const sshKeys = sshData?.ssh_keys ?? [];
  const versionOptions = useMemo(
    () => resolveVKSVersionOptions(templatesData?.vm_templates ?? []),
    [templatesData?.vm_templates],
  );
  const canSubmitCreate =
    templatesFetched &&
    versionOptions.length > 0 &&
    !!form.kubernetes_version &&
    !!form.template &&
    !!form.offering;

  useEffect(() => {
    if (!createModal) return;
    setForm((f) => {
      const next = { ...f };

      if (versionOptions.length > 0) {
        const match = versionOptions.find((v) => v.kubernetes_version === f.kubernetes_version);
        if (!match) {
          next.kubernetes_version = versionOptions[0].kubernetes_version;
          next.template = versionOptions[0].template;
        } else if (f.template !== match.template) {
          next.template = match.template;
        }
      } else {
        next.kubernetes_version = '';
        next.template = '';
      }

      const nets = networkList ?? [];
      if (nets.length > 0) {
        if (!nets.some((n) => n.name === f.network)) {
          next.network = nets[0].name;
        }
      } else {
        next.network = DEFAULT_NETWORK;
      }

      const offs = offeringList ?? [];
      if (offs.length > 0 && !offs.some((o) => o.name === f.offering)) {
        const medium = findOfferingByName(offs, 'medium');
        next.offering = medium?.name ?? offs[0].name;
      }

      return next;
    });
  }, [createModal, versionOptions, networkList, offeringList]);

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.vksClusters });
  };

  const resetForm = () => setForm(emptyCreateForm());

  const createMutation = useMutation({
    mutationFn: createVKSCluster,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      resetForm();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteVKSCluster(name),
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const downloadMutation = useMutation({
    mutationFn: async (name: string) => {
      const blob = await downloadVKSKubeconfig(name);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${name}.kubeconfig`;
      a.click();
      URL.revokeObjectURL(url);
    },
  });

  const clusters = data?.clusters || [];
  const filtered = clusters.filter((c) => {
    const q = search.toLowerCase();
    return (
      c.name?.toLowerCase().includes(q) ||
      c.phase?.toLowerCase().includes(q) ||
      c.kubernetes_version?.toLowerCase().includes(q) ||
      c.workers?.network_ref?.name?.toLowerCase().includes(q)
    );
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmitCreate) return;
    const port = form.port.trim() ? Number(form.port) : undefined;
    createMutation.mutate({
      name: form.name,
      kubernetes_version: form.kubernetes_version,
      control_plane: port
        ? { service_type: 'NodePort', port }
        : { service_type: 'LoadBalancer' },
      workers: {
        count: form.workers,
        template_ref: { name: form.template },
        offering_ref: { name: form.offering },
        network_ref: { name: form.network },
        ...(form.sshKeyNames.length > 0
          ? { ssh_key_refs: form.sshKeyNames.map((name) => ({ name })) }
          : {}),
      },
    });
  };

  const { rows, pager } = usePagination(filtered, search);

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  const readyClusters = clusters.filter((c) => (c.phase || '').toLowerCase() === 'ready').length;
  const desiredWorkers = clusters.reduce((n, c) => n + (c.workers?.count ?? 0), 0);
  const readyWorkers = clusters.reduce((n, c) => n + (c.ready_workers ?? 0), 0);
  const pct = (a: number, b: number) => (b ? Math.round((a / b) * 100) : 0);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('vks.title')}
        subtitle={t('vks.subtitle')}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('vks.create')}
            </button>
          </>
        }
      />

      {clusters.length > 0 && (
        <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(220px,1fr))]">
          <StatsCard
            title={t('vks.stat.ready')}
            value={readyClusters}
            of={t('dashboard.of').replace('{n}', String(clusters.length))}
            progress={pct(readyClusters, clusters.length)}
            variant="success"
          />
          <StatsCard
            title={t('vks.col.workers')}
            value={readyWorkers}
            of={t('dashboard.of').replace('{n}', String(desiredWorkers))}
            progress={pct(readyWorkers, desiredWorkers)}
            variant="success"
          />
          <StatsCard
            title={t('vks.stat.vip')}
            value={clusters.filter((c) => c.control_plane_endpoint).length}
          />
        </section>
      )}

      <InfoBanner>{t('vks.info')}</InfoBanner>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <TableToolbar>
            <TableFilter
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('table.filter')}
              aria-label={t('table.filter')}
            />
          </TableToolbar>
          <PageTable minWidth={900}>
            <PageTableHead>
              <PageTableTh>{t('vks.col.name')}</PageTableTh>
              <PageTableTh>{t('vks.col.version')}</PageTableTh>
              <PageTableTh>{t('vks.col.status')}</PageTableTh>
              <PageTableTh>{t('vks.col.endpoint')}</PageTableTh>
              <PageTableTh>{t('vks.col.workers')}</PageTableTh>
              <PageTableTh>{t('vks.col.machineType')}</PageTableTh>
              <PageTableTh>{t('vks.col.network')}</PageTableTh>
              <PageTableTh className="text-right">{t('vks.col.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={8}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={8}>{t('vks.empty')}</TableMessageRow>
              ) : (
                rows.map((c) => (
                  <PageTableRow key={c.name}>
                    <PageTableTd>
                      <Link
                        to={`/clusters/${encodeURIComponent(c.name)}`}
                        className="font-semibold text-link hover:underline"
                      >
                        {c.name}
                      </Link>
                    </PageTableTd>
                    <PageTableTd mono>{c.kubernetes_version || '—'}</PageTableTd>
                    <PageTableTd>
                      <StatusBadge status={vksPhaseBadgeStatus(c.phase)} />
                    </PageTableTd>
                    <PageTableTd mono>{c.control_plane_endpoint || '—'}</PageTableTd>
                    <PageTableTd mono>{vksNodesLabel(c.ready_workers, c.workers?.count)}</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant">{c.workers?.offering_ref?.name || '—'}</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant">{c.workers?.network_ref?.name || '—'}</PageTableTd>
                    <PageTableTd className="text-right">
                      <OverflowMenu
                        variant="dots"
                        label={t('table.moreActions')}
                        items={[
                          {
                            id: 'kubeconfig',
                            label: t('vks.downloadKubeconfig'),
                            icon: <Download size={14} />,
                            disabled: !vksCanDownloadKubeconfig(c.phase) || downloadMutation.isPending,
                            onSelect: () => downloadMutation.mutate(c.name),
                          },
                          {
                            id: 'delete',
                            label: t('common.delete'),
                            icon: <Trash2 size={14} />,
                            danger: true,
                            onSelect: () => setDeleteTarget({ name: c.name }),
                          },
                        ]}
                      />
                    </PageTableTd>
                  </PageTableRow>
                ))
              )}
            </PageTableBody>
          </PageTable>
          <TablePager pager={pager} />
        </SurfaceCard>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('vks.create')}>
        <form className="space-y-3" onSubmit={handleSubmit}>
          <label className="block text-sm">
            {t('vks.form.name')}
            <input
              className={formInputClass}
              required
              value={form.name}
              onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.version')}
            <select
              className={formSelectClass}
              required
              disabled={!templatesFetched || versionOptions.length === 0}
              value={form.kubernetes_version}
              onChange={(e) => {
                const opt = versionOptions.find((v) => v.kubernetes_version === e.target.value);
                if (!opt) return;
                setForm((f) => ({
                  ...f,
                  kubernetes_version: opt.kubernetes_version,
                  template: opt.template,
                }));
              }}
            >
              {versionOptions.length === 0 ? (
                <option value="">{t('vks.form.selectVersion')}</option>
              ) : (
                versionOptions.map((v) => (
                  <option key={v.kubernetes_version} value={v.kubernetes_version}>
                    {v.kubernetes_version}
                  </option>
                ))
              )}
            </select>
            {templatesFetched && versionOptions.length === 0 && (
              <span className="mt-1 block text-xs text-error">{t('vks.form.noNodeImage')}</span>
            )}
            {form.template && versionOptions.length > 0 && (
              <span className="mt-1 block text-xs text-on-surface-variant font-mono">{form.template}</span>
            )}
          </label>
          <label className="block text-sm">
            {t('vks.form.workers')}
            <input
              type="number"
              min={1}
              max={3}
              className={formInputClass}
              required
              value={form.workers}
              onChange={(e) => setForm((f) => ({ ...f, workers: Number(e.target.value) }))}
            />
          </label>
          <label className="block text-sm">
            {t('vks.form.offering')}
            <select
              className={formSelectClass}
              required
              disabled={offerings.length === 0}
              value={form.offering}
              onChange={(e) => setForm((f) => ({ ...f, offering: e.target.value }))}
            >
              {offerings.length === 0 ? (
                <option value="">{t('vks.form.selectOffering')}</option>
              ) : (
                offerings.map((o) => (
                  <option key={o.id} value={o.name}>{offeringLabel(o)}</option>
                ))
              )}
            </select>
          </label>
          <label className="block text-sm">
            {t('vks.form.network')}
            <select
              className={formSelectClass}
              required
              value={form.network}
              onChange={(e) => setForm((f) => ({ ...f, network: e.target.value }))}
            >
              {networks.length === 0 ? (
                <option value={DEFAULT_NETWORK}>{DEFAULT_NETWORK}</option>
              ) : (
                networks.map((n) => (
                  <option key={n.id} value={n.name}>{n.name}</option>
                ))
              )}
            </select>
          </label>
          <fieldset className="block text-sm">
            <legend className="mb-1">{t('vks.form.sshKeys')}</legend>
            {sshKeys.length === 0 ? (
              <p className="text-xs text-on-surface-variant">{t('vks.form.sshKeysEmpty')}</p>
            ) : (
              <div className="max-h-32 space-y-1 overflow-y-auto rounded-lg border border-outline-variant p-2">
                {sshKeys.map((k) => (
                  <label key={k.id} className="flex cursor-pointer items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={form.sshKeyNames.includes(k.name)}
                      onChange={(e) => {
                        setForm((f) => ({
                          ...f,
                          sshKeyNames: e.target.checked
                            ? [...f.sshKeyNames, k.name]
                            : f.sshKeyNames.filter((n) => n !== k.name),
                        }));
                      }}
                    />
                    <span className="font-mono">{k.name}</span>
                  </label>
                ))}
              </div>
            )}
          </fieldset>
          <label className="block text-sm">
            {t('vks.form.nodePort')}
            <input
              type="number"
              min={30000}
              max={32767}
              className={formInputClass}
              placeholder={t('vks.form.nodePortHint')}
              value={form.port}
              onChange={(e) => setForm((f) => ({ ...f, port: e.target.value }))}
            />
            <span className="mt-1 block text-xs text-on-surface-variant">{t('vks.form.nodePortHelp')}</span>
          </label>
          {createMutation.error && (
            <p className="text-sm text-error">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" className="rounded-lg px-3 py-2 text-sm" onClick={() => setCreateModal(false)}>
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              disabled={createMutation.isPending || !canSubmitCreate}
              className="btn-primary disabled:opacity-50"
            >
              {t('vks.create')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.name)}
        title={t('vks.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
