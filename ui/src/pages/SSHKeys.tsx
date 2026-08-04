import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Key, Upload, Copy, Check } from 'lucide-react';
import {
  listSSHKeys, createSSHKey, registerSSHKey, deleteSSHKey,
} from '../lib/platform-api';
import type { SSHKeyPair } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, EmptyState, ResourceGridCard, TenantRequiredNotice,
  formInputClass, formTextareaClass, InfoBanner,
} from '../components/shell';

type CreateMode = 'generate' | 'register';

export function SSHKeys() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [createMode, setCreateMode] = useState<CreateMode>('generate');
  const [form, setForm] = useState({ name: '', public_key: '' });
  const [privateKeyModal, setPrivateKeyModal] = useState<{ name: string; pem: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.sshKeys,
    queryFn: listSSHKeys,
    enabled: !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.sshKeys });

  const resetForm = () => {
    setForm({ name: '', public_key: '' });
    setCreateMode('generate');
  };

  const createMutation = useMutation({
    mutationFn: createSSHKey,
    onSuccess: (res) => {
      invalidate();
      setCreateModal(false);
      resetForm();
      setPrivateKeyModal({ name: res.key.name, pem: res.private_key_pem });
      setCopied(false);
    },
  });

  const registerMutation = useMutation({
    mutationFn: registerSSHKey,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      resetForm();
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

  const handleCopyPrivateKey = async () => {
    if (!privateKeyModal) return;
    await navigator.clipboard.writeText(privateKeyModal.pem);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (createMode === 'generate') {
      createMutation.mutate({ name: form.name });
    } else {
      registerMutation.mutate({ name: form.name, public_key: form.public_key });
    }
  };

  const pending = createMutation.isPending || registerMutation.isPending;
  const error = createMutation.error || registerMutation.error;

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
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

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <>
              <EmptyState icon={<Key size={48} />} title={t('ssh.empty')} />
              <p className="col-span-full text-center -mt-6">
                <Link to="/vms" className="text-primary-fixed-dim hover:underline text-sm">
                  → Deploy VM com chave SSH
                </Link>
              </p>
            </>
          ) : (
            filtered.map((key: SSHKeyPair) => (
              <ResourceGridCard key={key.id}>
                <div className="flex items-center gap-3 mb-3">
                  <div className="w-10 h-10 bg-primary-container/20 rounded-lg flex items-center justify-center shrink-0">
                    <Key size={20} className="text-primary-fixed-dim" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="font-headline text-headline-md font-semibold text-on-surface truncate">{key.name}</h3>
                    <p className="text-xs text-on-surface-variant font-data-mono truncate">{key.fingerprint}</p>
                  </div>
                </div>
                <p className="text-xs text-on-surface-variant font-data-mono break-all line-clamp-2 mb-3" title={key.public_key}>
                  {key.public_key}
                </p>
                <button
                  type="button"
                  onClick={() => setDeleteTarget({ id: key.id, name: key.name })}
                  className="btn-ghost-muted flex items-center gap-1 text-error hover:text-error w-full justify-end mt-3 pt-3 border-t border-outline-variant"
                >
                  {t('common.delete')}
                </button>
              </ResourceGridCard>
            ))
          )}
        </div>
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
              <button type="button" onClick={handleCopyPrivateKey} className="btn-primary">
                {copied ? <Check size={16} /> : <Copy size={16} />}
                {copied ? t('ssh.copied') : t('ssh.copy')}
              </button>
              <button type="button" onClick={() => setPrivateKeyModal(null)} className="btn-secondary">
                {t('common.cancel')}
              </button>
            </div>
          </div>
        )}
      </Modal>

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
