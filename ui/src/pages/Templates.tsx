import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Disc } from 'lucide-react';
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
  PageHeader, SearchField, EmptyState, ResourceGridCard, TenantRequiredNotice,
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
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <EmptyState icon={<Disc size={48} />} title={t('templates.empty')} />
          ) : (
            filtered.map((tmpl) => (
              <ResourceGridCard key={tmpl.id}>
                <div className="flex items-center gap-3 mb-3">
                  <div className="w-10 h-10 bg-primary-container/20 rounded-lg flex items-center justify-center shrink-0">
                    <Disc size={20} className="text-primary-fixed-dim" />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-headline text-headline-md font-semibold text-on-surface">{tmpl.display_name}</h3>
                      {!tmpl.tenant_id && (
                        <span className="text-xs px-2 py-0.5 rounded-full bg-surface-container-high text-on-surface-variant">
                          {t('templates.platformBadge')}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-on-surface-variant">{tmpl.name}</p>
                  </div>
                </div>
                <p className="text-xs font-data-mono text-on-surface-variant truncate mb-2" title={tmpl.image}>{tmpl.image}</p>
                <div className="flex gap-2 text-xs text-on-surface-variant mb-3 flex-wrap">
                  <span>{tmpl.os_type || 'linux'}</span>
                  <span>·</span>
                  <span>{tmpl.source_type || 'container'}</span>
                  {tmpl.import_state && tmpl.import_state !== 'ready' && (
                    <>
                      <span>·</span>
                      <span className="text-warning">{t('templates.importState')}: {tmpl.import_state}</span>
                    </>
                  )}
                  {tmpl.cloud_init_user_data && tmpl.source_type !== 'iso' && (
                    <>
                      <span>·</span>
                      <span>{t('templates.cloudInit')}</span>
                    </>
                  )}
                </div>
                {tmpl.tenant_id ? (
                  <ResourceActions
                    editLabel={t('common.edit')}
                    deleteLabel={t('common.delete')}
                    onEdit={() => setEditTmpl({ ...tmpl })}
                    onDelete={() => setDeleteTarget({ id: tmpl.id, name: tmpl.display_name })}
                  />
                ) : (
                  <p className="text-xs text-on-surface-variant/70">{t('templates.platformReadOnly')}</p>
                )}
              </ResourceGridCard>
            ))
          )}
        </div>
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
        onChange={setForm}
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
        <TemplateFormModal
          isOpen
          onClose={() => setEditTmpl(null)}
          title={t('templates.modalEdit')}
          form={{
            name: editTmpl.name,
            display_name: editTmpl.display_name,
            description: editTmpl.description || '',
            image: editTmpl.image,
            source_type: editTmpl.source_type || 'container',
            os_type: editTmpl.os_type || 'linux',
            cloud_init_user_data: editTmpl.cloud_init_user_data || '',
            iso_size_gi: editTmpl.iso_size_gi || 8,
            boot_disk_size_gi: editTmpl.boot_disk_size_gi || 32,
            storage_class: editTmpl.storage_class || 'local-path',
          }}
          onChange={(patch) => setEditTmpl({ ...editTmpl, ...patch })}
          nameReadOnly
          onSubmit={(e) => {
            e.preventDefault();
            updateMutation.mutate({
              id: editTmpl.id,
              display_name: editTmpl.display_name,
              description: editTmpl.description,
              image: editTmpl.image,
              source_type: editTmpl.source_type,
              os_type: editTmpl.os_type,
              cloud_init_user_data: editTmpl.cloud_init_user_data,
            });
          }}
          pending={updateMutation.isPending}
          error={updateMutation.error as Error | null}
          submitLabel={t('common.save')}
          t={t}
        />
      )}
    </div>
  );
}

function TemplateFormModal({
  isOpen, onClose, title, form, onChange, onSubmit, pending, error, submitLabel, nameReadOnly, t,
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
  nameReadOnly?: boolean;
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
              readOnly={nameReadOnly}
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
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('templates.isoSizeGi')}</label>
              <input type="number" min={1} value={form.iso_size_gi} onChange={(e) => onChange({ iso_size_gi: Number(e.target.value) })} className={formInputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('templates.bootDiskSizeGi')}</label>
              <input type="number" min={1} value={form.boot_disk_size_gi} onChange={(e) => onChange({ boot_disk_size_gi: Number(e.target.value) })} className={formInputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('templates.storageClass')}</label>
              <input value={form.storage_class} onChange={(e) => onChange({ storage_class: e.target.value })} className={formInputClass} placeholder="local-path" />
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
