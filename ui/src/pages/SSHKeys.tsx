import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Key, Upload, Copy, Check } from 'lucide-react';
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
import { useI18n } from '../lib/i18n';

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
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.sshKeys,
    queryFn: listSSHKeys,
    enabled: !needsTenant,
    refetchInterval: 12_000,
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
    return <div className="text-center py-12 text-amber-600">{t('common.selectTenant')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('ssh.title')}</h1>
          <p className="text-gray-500">{keys.length} {t('ssh.subtitle')}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button
            onClick={() => { resetForm(); setCreateMode('register'); setCreateModal(true); }}
            className="btn-secondary"
          >
            <Upload size={18} /> {t('ssh.register')}
          </button>
          <button
            onClick={() => { resetForm(); setCreateMode('generate'); setCreateModal(true); }}
            className="btn-primary"
          >
            <Plus size={18} /> {t('ssh.create')}
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={`${t('common.search')}...`}
          className="w-full pl-10 pr-4 py-3 border rounded-lg"
        />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {isLoading ? (
            <div className="col-span-full text-center py-12">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <div className="col-span-full text-center py-12">
              <Key size={48} className="mx-auto text-gray-300 mb-4" />
              <p className="text-gray-500 mb-4">{t('ssh.empty')}</p>
              <Link to="/vms" className="text-brand-600 hover:underline text-sm">
                → Deploy VM com chave SSH
              </Link>
            </div>
          ) : (
            filtered.map((key: SSHKeyPair) => (
              <div key={key.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5">
                <div className="flex items-center gap-3 mb-3">
                  <div className="w-10 h-10 bg-brand-100 dark:bg-brand-900/30 rounded-lg flex items-center justify-center">
                    <Key size={20} className="text-brand-600" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="font-semibold truncate">{key.name}</h3>
                    <p className="text-xs text-gray-500 font-mono truncate">{key.fingerprint}</p>
                  </div>
                </div>
                <p className="text-xs text-gray-500 font-mono break-all line-clamp-2 mb-3" title={key.public_key}>
                  {key.public_key}
                </p>
                <button
                  type="button"
                  onClick={() => setDeleteTarget({ id: key.id, name: key.name })}
                  className="btn-ghost-muted flex items-center gap-1 text-red-600 hover:text-red-700 mt-3 pt-3 border-t w-full justify-end"
                >
                  {t('common.delete')}
                </button>
              </div>
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
              className="w-full px-4 py-2 border rounded-lg"
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
                className="w-full px-4 py-2 border rounded-lg font-mono text-xs"
                placeholder="ssh-ed25519 AAAA... user@host"
              />
              <p className="text-xs text-gray-500 mt-1">{t('ssh.registerHint')}</p>
            </div>
          )}
          {createMode === 'generate' && (
            <p className="text-sm text-gray-500 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
              {t('ssh.privateKeyWarning')}
            </p>
          )}
          {error && (
            <p className="text-red-500 text-sm">{(error as Error).message}</p>
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
            <p className="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
              {t('ssh.privateKeyWarning')}
            </p>
            <p className="text-sm text-gray-600">
              <strong>{privateKeyModal.name}</strong>
            </p>
            <pre className="text-xs font-mono bg-gray-900 text-green-400 p-4 rounded-lg overflow-x-auto whitespace-pre-wrap break-all">
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
