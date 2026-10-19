import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Upload, Copy, Check, AlertCircle, Trash2, Download, Pencil } from 'lucide-react';
import {
  listSSHKeys, listVMs, createSSHKey, registerSSHKey, deleteSSHKey, updateSSHKey,
} from '../lib/platform-api';
import type { SSHKeyPair } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { TagsInput, TagPills, mergeTags } from '../components/TagsInput';
import { TagsEditorModal } from '../components/TagsEditorModal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { useCopyToClipboard } from '../hooks/useCopyToClipboard';
import { ago } from '../lib/ago';
import { OverflowMenu } from '../components/OverflowMenu';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formTextareaClass, InfoBanner,
} from '../components/shell';

type CreateMode = 'generate' | 'register';

/** "ssh-ed25519 AAAA…" -> "ed25519" */
const keyType = (pub?: string) => (pub?.split(' ')[0] ?? '').replace(/^ssh-/, '') || '—';

export function SSHKeys() {
  const { t, dateLocale } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [createMode, setCreateMode] = useState<CreateMode>('generate');
  const [form, setForm] = useState<{ name: string; public_key: string; tags: string[] }>({ name: '', public_key: '', tags: [] });
  const [privateKeyModal, setPrivateKeyModal] = useState<{ name: string; pem: string } | null>(null);
  const { state: copyState, copy: copyToClipboard, reset: resetCopy } = useCopyToClipboard();
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.sshKeys,
    queryFn: listSSHKeys,
    enabled: !needsTenant,
  });

  const { data: vmData } = useQuery({ queryKey: queryKeys.vms, queryFn: listVMs, enabled: !needsTenant });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.sshKeys });

  const [editKey, setEditKey] = useState<SSHKeyPair | null>(null);
  const [tagDraft, setTagDraft] = useState('');

  const resetForm = () => {
    setForm({ name: '', public_key: '', tags: [] });
    setTagDraft('');
    setCreateMode('generate');
  };

  const createMutation = useMutation({
    mutationFn: ({ name, tags }: { name: string; tags: string[] }) => createSSHKey({ name, tags }),
    onSuccess: (res) => {
      invalidate();
      setCreateModal(false);
      resetForm();
      setPrivateKeyModal({ name: res.key.name, pem: res.private_key_pem });
      resetCopy();
    },
  });

  const registerMutation = useMutation({
    mutationFn: ({ name, public_key, tags }: { name: string; public_key: string; tags: string[] }) => registerSSHKey({ name, public_key, tags }),
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      resetForm();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, tags }: { id: string; tags: string[] }) => updateSSHKey(id, { tags }),
    onSuccess: () => {
      invalidate();
      setEditKey(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSSHKey,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const keys = data?.ssh_keys || [];
  const filtered = keys.filter((k) =>
    k.name?.toLowerCase().includes(search.toLowerCase()) ||
    k.fingerprint?.toLowerCase().includes(search.toLowerCase())
  );

  const { rows, pager } = usePagination(filtered, search);
  const usedBy = (name: string) => (vmData?.vms ?? []).filter((vm) => vm.ssh_key_refs?.includes(name)).length;

  const handleCopyPrivateKey = () => {
    if (!privateKeyModal) return;
    void copyToClipboard(privateKeyModal.pem);
  };

  const handleDownloadPrivateKey = () => {
    if (!privateKeyModal) return;
    const blob = new Blob([privateKeyModal.pem], { type: 'application/x-pem-file' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${privateKeyModal.name}.pem`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const tags = mergeTags(form.tags, tagDraft);
    if (createMode === 'generate') {
      createMutation.mutate({ name: form.name, tags });
    } else {
      registerMutation.mutate({ name: form.name, public_key: form.public_key, tags });
    }
  };

  const pending = createMutation.isPending || registerMutation.isPending;
  const error = createMutation.error || registerMutation.error;

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('ssh.title')}
        subtitle={`${keys.length} ${t('ssh.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button
              type="button"
              onClick={() => { resetForm(); setCreateMode('register'); setCreateModal(true); }}
              className="btn-secondary"
            >
              <Upload size={18} /> {t('ssh.register')}
            </button>
            <button
              type="button"
              onClick={() => { resetForm(); setCreateMode('generate'); setCreateModal(true); }}
              className="btn-primary"
            >
              <Plus size={18} /> {t('ssh.create')}
            </button>
          </>
        }
      />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <TableToolbar>
            <TableFilter
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('table.filter')}
              aria-label={t('table.filter')}
            />
          </TableToolbar>
          <PageTable minWidth={680}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('ssh.fingerprint')}</PageTableTh>
              <PageTableTh>{t('ssh.type')}</PageTableTh>
              <PageTableTh>{t('common.created')}</PageTableTh>
              <PageTableTh>{t('ssh.usedBy')}</PageTableTh>
              <PageTableTh>{t('tags.column')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={7}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={7}>
                  <p>{t('ssh.empty')}</p>
                  <p className="mt-2">
                    <Link to="/vms" className="text-link hover:underline text-sm">
                      {t('ssh.deployLink')}
                    </Link>
                  </p>
                </TableMessageRow>
              ) : (
                rows.map((key: SSHKeyPair) => (
                  <PageTableRow key={key.id}>
                    <PageTableTd className="font-semibold">{key.name}</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant max-w-xs truncate" title={key.fingerprint}>
                      {key.fingerprint}
                    </PageTableTd>
                    <PageTableTd mono>{keyType(key.public_key)}</PageTableTd>
                    <PageTableTd className="text-on-surface-variant">{ago(key.created_at, dateLocale)}</PageTableTd>
                    <PageTableTd mono>{t('common.vmCount').replace('{n}', String(usedBy(key.name)))}</PageTableTd>
                    <PageTableTd><TagPills tags={key.tags} /></PageTableTd>
                    <PageTableTd className="text-right">
                      <OverflowMenu
                        variant="dots"
                        label={t('table.moreActions')}
                        items={[
                          {
                            id: 'edit-tags',
                            label: t('tags.editTitle'),
                            icon: <Pencil size={14} />,
                            onSelect: () => setEditKey(key),
                          },
                          {
                            id: 'delete',
                            label: t('common.delete'),
                            icon: <Trash2 size={14} />,
                            danger: true,
                            onSelect: () => setDeleteTarget({ id: key.id, name: key.name }),
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

      <Modal
        isOpen={createModal}
        onClose={() => { setCreateModal(false); resetForm(); }}
        title={createMode === 'generate' ? t('ssh.create') : t('ssh.register')}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input
              required
              pattern="[-a-z0-9]+"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
              className={formInputClass}
              placeholder={t('ssh.namePlaceholder')}
            />
          </div>
          {createMode === 'register' && (
            <div>
              <label className="block text-sm font-medium mb-1">{t('ssh.publicKey')}</label>
              <textarea
                required
                rows={5}
                value={form.public_key}
                onChange={(e) => setForm({ ...form, public_key: e.target.value })}
                className={formTextareaClass}
                placeholder="ssh-ed25519 AAAA... user@host"
              />
              <p className="text-xs text-on-surface-variant mt-1">{t('ssh.registerHint')}</p>
            </div>
          )}
          <div>
            <label className="block text-sm font-medium mb-1">{t('tags.column')}</label>
            <TagsInput
              tags={form.tags}
              onTagsChange={(tags) => setForm({ ...form, tags })}
              draft={tagDraft}
              onDraftChange={setTagDraft}
              placeholder={t('common.tagsPlaceholder')}
            />
          </div>
          {createMode === 'generate' && (
            <InfoBanner variant="warning">{t('ssh.privateKeyWarning')}</InfoBanner>
          )}
          {error && (
            <p className="text-error text-sm">{(error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={pending} className="btn-primary">
              {pending ? t('common.loading') : t('common.create')}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={privateKeyModal !== null}
        onClose={() => setPrivateKeyModal(null)}
        title={t('ssh.privateKeyTitle')}
        size="lg"
      >
        {privateKeyModal && (
          <div className="space-y-4">
            <InfoBanner variant="warning">{t('ssh.privateKeyWarning')}</InfoBanner>
            <p className="text-sm text-on-surface">
              <strong>{privateKeyModal.name}</strong>
            </p>
            <pre className="text-xs font-data-mono bg-surface-container-high text-success p-4 rounded-lg overflow-x-auto whitespace-pre-wrap break-all">
              {privateKeyModal.pem}
            </pre>
            <div className="flex justify-end gap-3">
              <button type="button" onClick={handleDownloadPrivateKey} className="btn-secondary">
                <Download size={16} />
                {t('ssh.download')}
              </button>
              <button type="button" onClick={handleCopyPrivateKey} className="btn-primary">
                {copyState === 'copied' ? (
                  <Check size={16} />
                ) : copyState === 'error' ? (
                  <AlertCircle size={16} />
                ) : (
                  <Copy size={16} />
                )}
                {copyState === 'copied'
                  ? t('ssh.copied')
                  : copyState === 'error'
                    ? t('common.copyError')
                    : t('ssh.copy')}
              </button>
              <button type="button" onClick={() => setPrivateKeyModal(null)} className="btn-secondary">
                {t('ssh.done')}
              </button>
            </div>
          </div>
        )}
      </Modal>

      <TagsEditorModal
        key={editKey?.id ?? 'none'}
        isOpen={!!editKey}
        onClose={() => setEditKey(null)}
        initialTags={editKey?.tags ?? []}
        onSubmit={(tags) => editKey && updateMutation.mutate({ id: editKey.id, tags })}
        loading={updateMutation.isPending}
        error={updateMutation.isError ? (updateMutation.error as Error).message : null}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('ssh.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
