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
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TabBar, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

type Tab = 'users' | 'roles' | 'keys';

export function IAM() {
  const { t } = useI18n();
  const currentUser = authService.getUser();
  const canManageIAM = currentUser?.role === 'root' || currentUser?.role === 'tenant_admin';
  const [tab, setTab] = useState<Tab>(canManageIAM ? 'users' : 'keys');
  const [userModal, setUserModal] = useState(false);
  const [keyModal, setKeyModal] = useState(false);
  const [secretModal, setSecretModal] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [deleteUser, setDeleteUser] = useState<{ id: string; name: string } | null>(null);
  const [userForm, setUserForm] = useState({ username: '', password: '', email: '', role_name: 'tenant.operator' });
  const [keyForm, setKeyForm] = useState({ name: '', expires_in_days: 90 });
  const qc = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data: usersData, isFetching: uFetch, refetch: refetchUsers } = useQuery({
    queryKey: queryKeys.iamUsers,
    queryFn: listIAMUsers,
    enabled: !needsTenant && canManageIAM,
  });
  const { data: rolesData } = useQuery({
    queryKey: queryKeys.iamRoles,
    queryFn: listIAMRoles,
    enabled: !needsTenant && canManageIAM,
  });
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
    return <TenantRequiredNotice message={t('iam.selectTenant')} />;
  }

  const tabs: { id: Tab; label: string; icon: typeof Users }[] = [
    ...(canManageIAM ? [
      { id: 'users' as Tab, label: t('iam.tabUsers'), icon: Users },
      { id: 'roles' as Tab, label: t('iam.tabRoles'), icon: Shield },
    ] : []),
    { id: 'keys', label: t('iam.tabKeys'), icon: Key },
  ];

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('iam.title')}
        subtitle={t('iam.subtitle')}
        actions={
          <RefreshButton onRefresh={() => { refetchUsers(); refetchKeys(); }} isFetching={uFetch} />
        }
      />

      <TabBar tabs={tabs} active={tab} onChange={setTab} />

      {tab === 'users' && (
        <SurfaceCard padding="none">
          <div className="flex justify-between items-center p-4 border-b border-card-border">
            <h2 className="font-headline text-headline-md font-semibold text-on-surface">{t('iam.tenantUsers')}</h2>
            <button type="button" onClick={() => setUserModal(true)} className="btn-primary flex items-center gap-2 text-sm">
              <Plus className="w-4 h-4" /> {t('iam.addUser')}
            </button>
          </div>
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('login.username')}</PageTableTh>
              <PageTableTh>{t('iam.role')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh className="text-right" />
            </PageTableHead>
            <PageTableBody>
              {(usersData?.users || []).map((u) => (
                <PageTableRow key={u.id}>
                  <PageTableTd className="font-medium">{u.username}</PageTableTd>
                  <PageTableTd>{u.role}</PageTableTd>
                  <PageTableTd>
                    <StatusBadge status={u.state || 'active'} pulse={false} />
                  </PageTableTd>
                  <PageTableTd className="text-right">
                    {u.role !== 'root' && (
                      <button type="button" className="text-error" onClick={() => setDeleteUser({ id: u.id, name: u.username })}>
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      )}

      {tab === 'roles' && (
        <SurfaceCard>
          <h2 className="font-headline text-headline-md font-semibold text-on-surface mb-4">{t('iam.tabRoles')}</h2>
          <ul className="space-y-2 text-sm">
            {(rolesData?.roles || []).map((r) => (
              <li key={r.id} className="flex justify-between border border-outline-variant rounded-lg p-3">
                <div>
                  <span className="font-medium text-on-surface">{r.name}</span>
                  {r.is_system && (
                    <span className="ml-2 text-xs bg-surface-container-high text-on-surface-variant px-2 py-0.5 rounded">
                      {t('iam.systemRole')}
                    </span>
                  )}
                  <p className="text-on-surface-variant text-xs mt-1">{(r.permissions || []).join(', ')}</p>
                </div>
              </li>
            ))}
          </ul>
        </SurfaceCard>
      )}

      {tab === 'keys' && (
        <SurfaceCard padding="none">
          <div className="flex justify-between items-center p-4 border-b border-card-border">
            <h2 className="font-headline text-headline-md font-semibold text-on-surface">{t('iam.tabKeys')}</h2>
            <button type="button" onClick={() => setKeyModal(true)} className="btn-primary flex items-center gap-2 text-sm">
              <Plus className="w-4 h-4" /> {t('iam.createKey')}
            </button>
          </div>
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('iam.prefix')}</PageTableTh>
              <PageTableTh>{t('iam.created')}</PageTableTh>
              <PageTableTh className="text-right" />
            </PageTableHead>
            <PageTableBody>
              {(keysData?.api_keys || []).filter((k) => !k.revoked_at).map((k) => (
                <PageTableRow key={k.id}>
                  <PageTableTd>{k.name}</PageTableTd>
                  <PageTableTd className="font-data-mono text-xs">{k.prefix}</PageTableTd>
                  <PageTableTd>{k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}</PageTableTd>
                  <PageTableTd className="text-right">
                    <button type="button" className="text-error text-xs" onClick={() => revokeKeyMut.mutate(k.id)}>{t('iam.revoke')}</button>
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      )}

      <Modal isOpen={userModal} onClose={() => setUserModal(false)} title={t('iam.addUserTitle')}>
        <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); createUserMut.mutate(userForm); }}>
          <div>
            <label className="block text-sm font-medium mb-1">{t('login.username')}</label>
            <input
              className={formInputClass}
              value={userForm.username}
              onChange={(e) => setUserForm({ ...userForm, username: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('login.password')}</label>
            <input
              className={formInputClass}
              type="password"
              value={userForm.password}
              onChange={(e) => setUserForm({ ...userForm, password: e.target.value })}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Email</label>
            <input
              className={formInputClass}
              type="email"
              value={userForm.email}
              onChange={(e) => setUserForm({ ...userForm, email: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('iam.role')}</label>
            <select
              className={formSelectClass}
              value={userForm.role_name}
              onChange={(e) => setUserForm({ ...userForm, role_name: e.target.value })}
            >
              <option value="tenant.admin">{t('iam.roleAdmin')}</option>
              <option value="tenant.operator">{t('iam.roleOperator')}</option>
              <option value="tenant.viewer">{t('iam.roleViewer')}</option>
            </select>
            <p className="text-xs text-on-surface-variant mt-1">{t('iam.roleHint')}</p>
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
              className={formInputClass}
              value={keyForm.name}
              onChange={(e) => setKeyForm({ ...keyForm, name: e.target.value })}
              placeholder={t('iam.keyNamePlaceholder')}
              required
            />
            <p className="text-xs text-on-surface-variant mt-1">{t('iam.keyNameHint')}</p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('iam.expiresDays')}</label>
            <input
              className={formInputClass}
              type="number"
              min={1}
              max={365}
              value={keyForm.expires_in_days}
              onChange={(e) => setKeyForm({ ...keyForm, expires_in_days: Number(e.target.value) })}
            />
            <p className="text-xs text-on-surface-variant mt-1">{t('iam.expiresHint')}</p>
          </div>
          <InfoBanner variant="warning">{t('iam.secretWarning')}</InfoBanner>
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={() => setKeyModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" className="btn-primary" disabled={createKeyMut.isPending}>{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={!!secretModal} onClose={() => setSecretModal(null)} title={t('iam.secretTitle')} size="lg">
        <div className="space-y-4">
          <InfoBanner variant="warning">{t('iam.secretWarning')}</InfoBanner>
          <pre className="text-xs font-data-mono bg-surface-container-high text-success p-4 rounded-lg overflow-x-auto whitespace-pre-wrap break-all">
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
