import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Loader2 } from 'lucide-react';
import {
  listVMTemplates, createVMTemplate, updateVMTemplate, deleteVMTemplate,
} from '../lib/platform-api';
import type { VMTemplate } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass, formTextareaClass,
} from '../components/shell';

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
  const [createModal, setCreateModal] = useState(false);
  const [editTmpl, setEditTmpl] = useState<VMTemplate | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState(emptyForm());
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.templates,
    queryFn: listVMTemplates,
    enabled: !needsTenant,
    refetchInterval: (q) => {
      const templates = q.state.data?.vm_templates || [];
      if (templates.some((tmpl) => isTemplateImporting(tmpl.import_state))) return 5_000;
      return false;
    },
  });

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
  const filtered = templates.filter((tmpl) =>
    tmpl.display_name?.toLowerCase().includes(search.toLowerCase()) ||
    tmpl.name?.toLowerCase().includes(search.toLowerCase()) ||
    tmpl.image?.toLowerCase().includes(search.toLowerCase())
  );

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('templates.title')}
        subtitle={`${templates.length} ${t('templates.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('templates.create')}
            </button>
          </>
        }
      />

      <p className="text-sm text-on-surface-variant">{t('templates.hint')}</p>

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('templates.displayName')}</PageTableTh>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('templates.imageUrl')}</PageTableTh>
              <PageTableTh>OS</PageTableTh>
              <PageTableTh>{t('templates.sourceType')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <tr><td colSpan={7} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={7} className="text-center py-12 text-on-surface-variant">{t('templates.empty')}</td></tr>
              ) : (
                filtered.map((tmpl) => (
                  <PageTableRow key={tmpl.id}>
                    <PageTableTd>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-medium">{tmpl.display_name}</span>
                        {!tmpl.tenant_id && (
                          <span className="text-xs px-2 py-0.5 rounded-full bg-surface-container-high text-on-surface-variant">
                            {t('templates.platformBadge')}
                          </span>
                        )}
                      </div>
                    </PageTableTd>
                    <PageTableTd className="text-on-surface-variant text-xs font-data-mono">{tmpl.name}</PageTableTd>
                    <PageTableTd className="font-data-mono text-xs text-on-surface-variant max-w-xs truncate" title={tmpl.image}>
                      {tmpl.image}
                    </PageTableTd>
                    <PageTableTd>{tmpl.os_type || 'linux'}</PageTableTd>
                    <PageTableTd>{tmpl.source_type || 'container'}</PageTableTd>
                    <PageTableTd>
                      {isTemplateImporting(tmpl.import_state) ? (
                        <span className="inline-flex items-center gap-1 text-xs text-warning">
                          <Loader2 size={12} className="animate-spin" />
                          {tmpl.import_state}
                        </span>
                      ) : tmpl.import_state === 'failed' ? (
                        <span className="text-xs text-error">{tmpl.import_state}</span>
                      ) : (
                        <span className="text-xs text-on-surface-variant">
                          {tmpl.cloud_init_user_data && tmpl.source_type !== 'iso' ? t('templates.cloudInit') : '—'}
                        </span>
                      )}
                    </PageTableTd>
                    <PageTableTd>
                      {tmpl.tenant_id ? (
                        <ResourceActions
                          variant="inline"
                          editLabel={t('common.edit')}
                          deleteLabel={t('common.delete')}
                          onEdit={() => setEditTmpl({ ...tmpl })}
                          onDelete={() => setDeleteTarget({ id: tmpl.id, name: tmpl.display_name })}
                        />
                      ) : (
                        <span className="text-xs text-on-surface-variant/70">{t('templates.platformReadOnly')}</span>
                      )}
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
  t: (key: string) => string;
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
