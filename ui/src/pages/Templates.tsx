import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, LayoutGrid, List, Rocket, Pencil, Trash2 } from 'lucide-react';
import {
  listVMTemplates, listVMs, createVMTemplate, updateVMTemplate, deleteVMTemplate,
} from '../lib/platform-api';
import type { VMTemplate } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { DeployVMWizard } from '../components/DeployVMWizard';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { OverflowMenu } from '../components/OverflowMenu';
import { StatusBadge } from '../components/StatusBadge';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import type { TranslationKey } from '../lib/i18n';
import {
  POLL_WS_HEALTHY_SLOW_MS,
  realtimePollInterval,
  useRealtimeConnected,
} from '../hooks/useRealtimeEvents';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TabBar, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow, toolbarBtnClass,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass, formTextareaClass,
} from '../components/shell';
import clsx from 'clsx';

type Scope = 'all' | 'system' | 'custom';

type TemplateForm = {
  name: string;
  display_name: string;
  description: string;
  image: string;
  source_type: string;
  os_type: string;
  cloud_init_user_data: string;
  iso_size_gi: number;
  boot_disk_size_gi: number;
  storage_class: string;
};

function isTemplateImporting(state?: string) {
  return state === 'pending' || state === 'importing';
}

const emptyForm = (): TemplateForm => ({
  name: '',
  display_name: '',
  description: '',
  image: '',
  source_type: 'container',
  os_type: 'linux',
  cloud_init_user_data: '',
  iso_size_gi: 8,
  boot_disk_size_gi: 32,
  storage_class: 'local-path',
});

export function Templates() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [view, setView] = useState<'gallery' | 'list'>('list');
  const [scope, setScope] = useState<Scope>('all');
  const [createModal, setCreateModal] = useState(false);
  const [editTmpl, setEditTmpl] = useState<VMTemplate | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState(emptyForm());
  const [deployOpen, setDeployOpen] = useState(false);
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();
  const wsConnected = useRealtimeConnected();

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.templates,
    queryFn: listVMTemplates,
    enabled: !needsTenant,
    refetchInterval: (q) => {
      const templates = q.state.data?.vm_templates || [];
      const importing = templates.some((tmpl) => isTemplateImporting(tmpl.import_state));
      // Backend may not emit template.* yet — keep a slow safety poll while importing
      return realtimePollInterval(wsConnected, importing, {
        downMs: 5_000,
        healthyMs: POLL_WS_HEALTHY_SLOW_MS,
      });
    },
  });

  const { data: vmData } = useQuery({ queryKey: queryKeys.vms, queryFn: listVMs, enabled: !needsTenant });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.templates });

  const createMutation = useMutation({
    mutationFn: createVMTemplate,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm(emptyForm());
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & Parameters<typeof updateVMTemplate>[1]) =>
      updateVMTemplate(id, payload),
    onSuccess: () => {
      invalidate();
      setEditTmpl(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteVMTemplate,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const templates = data?.vm_templates || [];
  const q = search.toLowerCase();
  const byScope = templates.filter((tmpl) =>
    scope === 'all' ? true : scope === 'system' ? !tmpl.tenant_id : !!tmpl.tenant_id,
  );
  const filtered = byScope.filter((tmpl) =>
    tmpl.display_name?.toLowerCase().includes(q) ||
    tmpl.name?.toLowerCase().includes(q) ||
    tmpl.image?.toLowerCase().includes(q)
  );
  const { rows, pager } = usePagination(filtered, `${search}|${scope}`);
  const inUse = (tmpl: VMTemplate) =>
    (vmData?.vms ?? []).filter((vm) => vm.template_ref === tmpl.name || vm.template === tmpl.display_name).length;
  const scopeTabs = [
    { id: 'all' as Scope, label: t('templates.scopeAll'), count: templates.length },
    { id: 'system' as Scope, label: t('templates.scopeSystem'), count: templates.filter((x) => !x.tenant_id).length },
    { id: 'custom' as Scope, label: t('templates.scopeCustom'), count: templates.filter((x) => !!x.tenant_id).length },
  ];

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  const viewBtn = (id: 'gallery' | 'list', Icon: typeof List, label: string) => (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={view === id}
      onClick={() => setView(id)}
      className={clsx(toolbarBtnClass, '!px-2', view === id && '!bg-accent-soft !text-link !border-accent/40')}
    >
      <Icon size={14} />
    </button>
  );

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('templates.title')}
        subtitle={t('templates.hint')}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('templates.create')}
            </button>
          </>
        }
      />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <TableToolbar tabs={<TabBar variant="inline" tabs={scopeTabs} active={scope} onChange={setScope} />}>
            <TableFilter
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('table.filter')}
              aria-label={t('table.filter')}
            />
            <div className="inline-flex gap-1.5">
              {viewBtn('list', List, t('templates.list'))}
              {viewBtn('gallery', LayoutGrid, t('templates.gallery'))}
            </div>
          </TableToolbar>
          {view === 'gallery' ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 p-5">
              {isLoading && <p className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</p>}
              {!isLoading && filtered.length === 0 && (
                <p className="col-span-full text-center py-12 text-on-surface-variant">{t('templates.empty')}</p>
              )}
              {filtered.map((tmpl) => (
                <div key={tmpl.id} className="flex flex-col rounded-xl border border-outline-variant bg-card p-4">
                  <div className="flex items-start gap-3 mb-3">
                    <div className="w-12 h-12 rounded-xl border border-outline-variant bg-sunk flex items-center justify-center font-mono text-xs text-link shrink-0">
                      {(tmpl.os_type || 'linux').slice(0, 3).toUpperCase()}
                    </div>
                    <div className="min-w-0">
                      <h3 className="font-semibold text-on-surface truncate m-0">{tmpl.display_name || tmpl.name}</h3>
                      <p className="text-xs font-mono text-on-surface-variant truncate m-0">{tmpl.name}</p>
                    </div>
                  </div>
                  <p className="text-xs text-on-surface-variant line-clamp-2 mb-2">{tmpl.description || tmpl.image}</p>
                  <p className="text-[11px] text-on-surface-variant mb-3">
                    {t('templates.minOffering')}: {tmpl.boot_disk_size_gi ? `${tmpl.boot_disk_size_gi} Gi` : 'small+'}
                  </p>
                  <button
                    type="button"
                    className="btn-primary mt-auto justify-center text-sm"
                    onClick={() => setDeployOpen(true)}
                  >
                    <Rocket size={14} /> {t('templates.deploy')}
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <PageTable minWidth={820}>
              <PageTableHead>
                <PageTableTh>{t('common.name')}</PageTableTh>
                <PageTableTh>{t('templates.sourceType')}</PageTableTh>
                <PageTableTh>{t('templates.defaultDisk')}</PageTableTh>
                <PageTableTh>{t('common.state')}</PageTableTh>
                <PageTableTh>{t('templates.inUse')}</PageTableTh>
                <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
              </PageTableHead>
              <PageTableBody>
                {isLoading ? (
                  <TableMessageRow cols={6}>{t('common.loading')}</TableMessageRow>
                ) : filtered.length === 0 ? (
                  <TableMessageRow cols={6}>{t('templates.empty')}</TableMessageRow>
                ) : (
                  rows.map((tmpl) => (
                    <PageTableRow key={tmpl.id}>
                      <PageTableTd>
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold">{tmpl.display_name || tmpl.name}</span>
                          {!tmpl.tenant_id && (
                            <span className="text-xs px-2 py-0.5 rounded-full bg-off-bg text-off-fg font-medium">
                              {t('templates.platformBadge')}
                            </span>
                          )}
                        </div>
                        <div className="font-mono text-xs text-on-surface-variant max-w-xs truncate" title={tmpl.image}>
                          {tmpl.name}
                        </div>
                      </PageTableTd>
                      <PageTableTd>{tmpl.source_type || 'container'}</PageTableTd>
                      <PageTableTd mono>{tmpl.boot_disk_size_gi ? `${tmpl.boot_disk_size_gi} GiB` : '—'}</PageTableTd>
                      <PageTableTd>
                        {isTemplateImporting(tmpl.import_state) ? (
                          <StatusBadge status="creating" tone="warn" label={tmpl.import_state} />
                        ) : tmpl.import_state === 'failed' ? (
                          <StatusBadge status="error" label={tmpl.import_state} pulse={false} />
                        ) : (
                          <StatusBadge status="active" tone="ok" label={t('templates.ready')} pulse={false} />
                        )}
                      </PageTableTd>
                      <PageTableTd mono>{t('common.vmCount').replace('{n}', String(inUse(tmpl)))}</PageTableTd>
                      <PageTableTd className="text-right">
                        <OverflowMenu
                          variant="dots"
                          label={t('table.moreActions')}
                          items={[
                            { id: 'deploy', label: t('templates.deploy'), icon: <Rocket size={14} />, onSelect: () => setDeployOpen(true) },
                            ...(tmpl.tenant_id ? [
                              { id: 'edit', label: t('common.edit'), icon: <Pencil size={14} />, onSelect: () => setEditTmpl({ ...tmpl }) },
                              {
                                id: 'delete',
                                label: t('common.delete'),
                                icon: <Trash2 size={14} />,
                                danger: true,
                                onSelect: () => setDeleteTarget({ id: tmpl.id, name: tmpl.display_name }),
                              },
                            ] : []),
                          ]}
                        />
                      </PageTableTd>
                    </PageTableRow>
                  ))
                )}
              </PageTableBody>
            </PageTable>
          )}
          {view === 'list' && <TablePager pager={pager} />}
        </SurfaceCard>
      </RefreshingPanel>

      <DeployVMWizard open={deployOpen} onClose={() => setDeployOpen(false)} />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('templates.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        loading={deleteMutation.isPending}
      />

      <TemplateFormModal
        isOpen={createModal}
        onClose={() => setCreateModal(false)}
        title={t('templates.modalCreate')}
        form={form}
        onChange={(next) => setForm((prev) => ({ ...prev, ...next }))}
        onSubmit={(e) => {
          e.preventDefault();
          createMutation.mutate(form);
        }}
        pending={createMutation.isPending}
        error={createMutation.error as Error | null}
        submitLabel={t('common.create')}
        t={t}
      />

      {editTmpl && (
        <Modal isOpen onClose={() => setEditTmpl(null)} title={t('templates.modalEdit')} size="md">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              updateMutation.mutate({
                id: editTmpl.id,
                display_name: editTmpl.display_name,
              });
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
              <input readOnly value={editTmpl.name} className={formInputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('templates.displayName')}</label>
              <input
                required
                value={editTmpl.display_name}
                onChange={(e) => setEditTmpl({ ...editTmpl, display_name: e.target.value })}
                className={formInputClass}
                placeholder="Ubuntu 22.04"
              />
            </div>
            {updateMutation.error && (
              <p className="text-error text-sm">{(updateMutation.error as Error).message}</p>
            )}
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={() => setEditTmpl(null)} className="btn-secondary">
                {t('common.cancel')}
              </button>
              <button type="submit" disabled={updateMutation.isPending} className="btn-primary">
                {t('common.save')}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}

function TemplateFormModal({
  isOpen, onClose, title, form, onChange, onSubmit, pending, error, submitLabel, t,
}: {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  form: TemplateForm;
  onChange: (next: Partial<TemplateForm>) => void;
  onSubmit: (e: React.FormEvent) => void;
  pending: boolean;
  error: Error | null;
  submitLabel: string;
  t: (key: TranslationKey) => string;
}) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} size="lg">
      <form onSubmit={onSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input
              required
              value={form.name}
              onChange={(e) => onChange({ name: e.target.value.toLowerCase() })}
              className={formInputClass}
              placeholder="ubuntu-2204"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('templates.displayName')}</label>
            <input
              required
              value={form.display_name}
              onChange={(e) => onChange({ display_name: e.target.value })}
              className={formInputClass}
              placeholder="Ubuntu 22.04"
            />
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">
            {form.source_type === 'iso' ? t('templates.isoUrl') : t('templates.imageUrl')}
          </label>
          <input
            required={form.source_type !== 'iso' || !form.image}
            value={form.image}
            onChange={(e) => onChange({ image: e.target.value })}
            className={`${formInputClass} font-data-mono text-sm`}
            placeholder={form.source_type === 'iso' ? 'https://go.microsoft.com/fwlink/?linkid=2195280' : 'quay.io/containerdisks/ubuntu:22.04'}
          />
          <p className="text-xs text-on-surface-variant mt-1">
            {form.source_type === 'iso' ? t('templates.isoHint') : t('templates.imageHint')}
          </p>
        </div>
        {form.source_type === 'iso' && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('templates.isoSizeGi')}</label>
              <input type="number" min={1} value={form.iso_size_gi} onChange={(e) => onChange({ iso_size_gi: Number(e.target.value) })} className={formInputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('templates.bootDiskSizeGi')}</label>
              <input type="number" min={1} value={form.boot_disk_size_gi} onChange={(e) => onChange({ boot_disk_size_gi: Number(e.target.value) })} className={formInputClass} />
            </div>
          </div>
        )}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('templates.sourceType')}</label>
            <select value={form.source_type} onChange={(e) => onChange({ source_type: e.target.value, os_type: e.target.value === 'iso' ? 'windows' : form.os_type })} className={formSelectClass}>
              <option value="container">{t('templates.sourceContainer')}</option>
              <option value="iso">{t('templates.sourceIso')}</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('templates.osType')}</label>
            <select value={form.os_type} onChange={(e) => onChange({ os_type: e.target.value })} className={formSelectClass}>
              <option value="linux">Linux</option>
              <option value="windows">Windows</option>
              <option value="other">{t('templates.osOther')}</option>
            </select>
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t('templates.description')}</label>
          <input value={form.description} onChange={(e) => onChange({ description: e.target.value })} className={formInputClass} />
        </div>
        {form.source_type !== 'iso' && (
        <div>
          <label className="block text-sm font-medium mb-1">{t('templates.cloudInitExtra')}</label>
          <textarea
            value={form.cloud_init_user_data}
            onChange={(e) => onChange({ cloud_init_user_data: e.target.value })}
            className={`${formTextareaClass} font-data-mono text-sm`}
            rows={5}
            placeholder="#cloud-config\npackages:\n  - nginx"
          />
          <p className="text-xs text-on-surface-variant mt-1">{t('templates.cloudInitHint')}</p>
        </div>
        )}
        {error && <p className="text-error text-sm">{error.message}</p>}
        <div className="flex justify-end gap-3 pt-4">
          <button type="button" onClick={onClose} className="btn-secondary">{t('common.cancel')}</button>
          <button type="submit" disabled={pending} className="btn-primary">{submitLabel}</button>
        </div>
      </form>
    </Modal>
  );
}
