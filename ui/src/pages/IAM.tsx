import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Key, Users, Shield, Copy, Check } from 'lucide-react';
import {
  listIAMUsers, createIAMUser, deleteIAMUser,
  listIAMRoles,
  listIAMAPIKeys, createIAMAPIKey, revokeIAMAPIKey,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';

type Tab = 'users' | 'roles' | 'keys';

const inputClass = 'w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-dark-200 text-gray-900 dark:text-gray-100';

export function IAM() {
  const { t } = useI18n();
  const [tab, setTab] = useState<Tab>('users');
  const [userModal, setUserModal] = useState(false);
  const [keyModal, setKeyModal] = useState(false);
  const [secretModal, setSecretModal] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [deleteUser, setDeleteUser] = useState<{ id: string; name: string } | null>(null);
  const [userForm, setUserForm] = useState({ username: '', password: '', email: '', role_name: 'tenant.operator' });
  const [keyForm, setKeyForm] = useState({ name: '', expires_in_days: 90 });
  const qc = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data: usersData, isFetching: uFetch, refetch: refetchUsers } = useQuery({
    queryKey: queryKeys.iamUsers,
    queryFn: listIAMUsers,
    enabled: !needsTenant,
  });
  const { data: rolesData } = useQuery({ queryKey: queryKeys.iamRoles, queryFn: listIAMRoles, enabled: !needsTenant });
  const { data: keysData, refetch: refetchKeys } = useQuery({
    queryKey: queryKeys.iamKeys,
    queryFn: listIAMAPIKeys,
    enabled: !needsTenant,
  });

  const inv = () => {
    qc.invalidateQueries({ queryKey: queryKeys.iamUsers });
    qc.invalidateQueries({ queryKey: queryKeys.iamKeys });
    qc.invalidateQueries({ queryKey: queryKeys.iamRoles });
  };

  const createUserMut = useMutation({
    mutationFn: createIAMUser,
    onSuccess: () => { inv(); setUserModal(false); setUserForm({ username: '', password: '', email: '', role_name: 'tenant.operator' }); },
  });
  const deleteUserMut = useMutation({
    mutationFn: deleteIAMUser,
    onSuccess: () => { inv(); setDeleteUser(null); },
  });
  const createKeyMut = useMutation({
    mutationFn: createIAMAPIKey,
    onSuccess: (res) => { inv(); setKeyModal(false); setSecretModal(res.secret); setCopied(false); },
  });
  const revokeKeyMut = useMutation({
    mutationFn: revokeIAMAPIKey,
    onSuccess: inv,
  });

  const handleCopySecret = async () => {
    if (!secretModal) return;
    await navigator.clipboard.writeText(secretModal);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('iam.selectTenant')}</div>;
  }

  const tabs: { id: Tab; labelKey: 'iam.tabUsers' | 'iam.tabRoles' | 'iam.tabKeys'; icon: typeof Users }[] = [
    { id: 'users', labelKey: 'iam.tabUsers', icon: Users },
    { id: 'roles', labelKey: 'iam.tabRoles', icon: Shield },
    { id: 'keys', labelKey: 'iam.tabKeys', icon: Key },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('iam.title')}</h1>
          <p className="text-sm text-gray-500">{t('iam.subtitle')}</p>
        </div>
        <RefreshButton onRefresh={() => { refetchUsers(); refetchKeys(); }} isFetching={uFetch} />
      </div>

      <div className="flex gap-2 border-b border-gray-200 dark:border-gray-700">
        {tabs.map(({ id, labelKey, icon: Icon }) => (
          <button
            key={id}
            type="button"
            onClick={() => setTab(id)}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 -mb-px ${tab === id ? 'border-indigo-600 text-indigo-600' : 'border-transparent text-gray-500'}`}
          >
            <Icon className="w-4 h-4" /> {t(labelKey)}
          </button>
        ))}
      </div>

      {tab === 'users' && (
        <section className="bg-white dark:bg-dark-100 rounded-lg shadow border border-gray-200 dark:border-gray-700">
          <div className="flex justify-between items-center p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="font-semibold">{t('iam.tenantUsers')}</h2>
            <button type="button" onClick={() => setUserModal(true)} className="btn-primary flex items-center gap-2 text-sm">
              <Plus className="w-4 h-4" /> {t('iam.addUser')}
            </button>
          </div>
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-dark-200 text-left text-gray-600 dark:text-gray-400">
              <tr><th className="p-3">{t('login.username')}</th><th>{t('iam.role')}</th><th>State</th><th /></tr>
            </thead>
            <tbody>
              {(usersData?.users || []).map((u) => (
                <tr key={u.id} className="border-t border-gray-200 dark:border-gray-700">
                  <td className="p-3 font-medium">{u.username}</td>
                  <td>{u.role}</td>
                  <td>{u.state || 'active'}</td>
                  <td className="p-3 text-right">
                    {u.role !== 'root' && (
                      <button type="button" className="text-red-600" onClick={() => setDeleteUser({ id: u.id, name: u.username })}>
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {tab === 'roles' && (
        <section className="bg-white dark:bg-dark-100 rounded-lg shadow border border-gray-200 dark:border-gray-700 p-4">
          <h2 className="font-semibold mb-4">{t('iam.tabRoles')}</h2>
          <ul className="space-y-2 text-sm">
            {(rolesData?.roles || []).map((r) => (
              <li key={r.id} className="flex justify-between border border-gray-200 dark:border-gray-700 rounded p-3">
                <div>
                  <span className="font-medium">{r.name}</span>
                  {r.is_system && <span className="ml-2 text-xs bg-gray-100 dark:bg-dark-200 px-2 py-0.5 rounded">{t('iam.systemRole')}</span>}
                  <p className="text-gray-500 text-xs mt-1">{(r.permissions || []).join(', ')}</p>
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      {tab === 'keys' && (
        <section className="bg-white dark:bg-dark-100 rounded-lg shadow border border-gray-200 dark:border-gray-700">
          <div className="flex justify-between items-center p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="font-semibold">{t('iam.tabKeys')}</h2>
            <button type="button" onClick={() => setKeyModal(true)} className="btn-primary flex items-center gap-2 text-sm">
              <Plus className="w-4 h-4" /> {t('iam.createKey')}
            </button>
          </div>
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-dark-200 text-left text-gray-600 dark:text-gray-400">
              <tr><th className="p-3">{t('common.name')}</th><th>{t('iam.prefix')}</th><th>{t('iam.created')}</th><th /></tr>
            </thead>
            <tbody>
              {(keysData?.api_keys || []).filter((k) => !k.revoked_at).map((k) => (
                <tr key={k.id} className="border-t border-gray-200 dark:border-gray-700">
                  <td className="p-3">{k.name}</td>
                  <td className="font-mono text-xs">{k.prefix}</td>
                  <td>{k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}</td>
                  <td className="p-3 text-right">
                    <button type="button" className="text-red-600 text-xs" onClick={() => revokeKeyMut.mutate(k.id)}>{t('iam.revoke')}</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      <Modal isOpen={userModal} onClose={() => setUserModal(false)} title={t('iam.addUserTitle')}>
        <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); createUserMut.mutate(userForm); }}>
          <div>
            <label className="block text-sm font-medium mb-1">{t('login.username')}</label>
            <input
              className={inputClass}
              value={userForm.username}
              onChange={(e) => setUserForm({ ...userForm, username: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('login.password')}</label>
            <input
              className={inputClass}
              type="password"
              value={userForm.password}
              onChange={(e) => setUserForm({ ...userForm, password: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Email</label>
            <input
              className={inputClass}
              type="email"
              value={userForm.email}
              onChange={(e) => setUserForm({ ...userForm, email: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('iam.role')}</label>
            <select
              className={inputClass}
              value={userForm.role_name}
              onChange={(e) => setUserForm({ ...userForm, role_name: e.target.value })}
            >
              <option value="tenant.admin">{t('iam.roleAdmin')}</option>
              <option value="tenant.operator">{t('iam.roleOperator')}</option>
              <option value="tenant.viewer">{t('iam.roleViewer')}</option>
            </select>
            <p className="text-xs text-gray-500 mt-1">{t('iam.roleHint')}</p>
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={() => setUserModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" className="btn-primary" disabled={createUserMut.isPending}>{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={keyModal} onClose={() => setKeyModal(false)} title={t('iam.createKeyTitle')}>
        <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); createKeyMut.mutate(keyForm); }}>
          <div>
            <label className="block text-sm font-medium mb-1">{t('iam.keyName')}</label>
            <input
              className={inputClass}
              value={keyForm.name}
              onChange={(e) => setKeyForm({ ...keyForm, name: e.target.value })}
              placeholder={t('iam.keyNamePlaceholder')}
              required
            />
            <p className="text-xs text-gray-500 mt-1">{t('iam.keyNameHint')}</p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('iam.expiresDays')}</label>
            <input
              className={inputClass}
              type="number"
              min={1}
              max={365}
              value={keyForm.expires_in_days}
              onChange={(e) => setKeyForm({ ...keyForm, expires_in_days: Number(e.target.value) })}
            />
            <p className="text-xs text-gray-500 mt-1">{t('iam.expiresHint')}</p>
          </div>
          <p className="text-sm text-amber-700 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
            {t('iam.secretWarning')}
          </p>
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={() => setKeyModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" className="btn-primary" disabled={createKeyMut.isPending}>{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={!!secretModal} onClose={() => setSecretModal(null)} title={t('iam.secretTitle')} size="lg">
        <div className="space-y-4">
          <p className="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-3">
            {t('iam.secretWarning')}
          </p>
          <pre className="text-xs font-mono bg-gray-900 text-green-400 p-4 rounded-lg overflow-x-auto whitespace-pre-wrap break-all">
            {secretModal}
          </pre>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={handleCopySecret} className="btn-primary">
              {copied ? <Check size={16} /> : <Copy size={16} />}
              {copied ? t('ssh.copied') : t('ssh.copy')}
            </button>
            <button type="button" onClick={() => setSecretModal(null)} className="btn-secondary">
              {t('common.cancel')}
            </button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={!!deleteUser}
        title={t('iam.deleteUserTitle')}
        message={t('iam.deleteUserMessage')}
        resourceName={deleteUser?.name}
        onConfirm={() => deleteUser && deleteUserMut.mutate(deleteUser.id)}
        onClose={() => setDeleteUser(null)}
      />
    </div>
  );
}
