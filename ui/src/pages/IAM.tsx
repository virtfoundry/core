import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Copy, Check, AlertCircle } from 'lucide-react';
import {
  listIAMUsers, createIAMUser, deleteIAMUser,
  listIAMRoles,
  listIAMAPIKeys, createIAMAPIKey, revokeIAMAPIKey,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { OverflowMenu } from '../components/OverflowMenu';
import { ago } from '../lib/ago';
import { RefreshButton } from '../components/RefreshButton';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { useCopyToClipboard } from '../hooks/useCopyToClipboard';
import {
  PageHeader, SurfaceCard, TabBar, TenantRequiredNotice, InfoBanner, TableToolbar, TableFilter, TablePager, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

type Tab = 'users' | 'roles' | 'keys';

export function IAM() {
  const { t, dateLocale } = useI18n();
  const [search, setSearch] = useState('');
  const currentUser = authService.getUser();
  const canManageIAM = currentUser?.role === 'root' || currentUser?.role === 'tenant_admin';
  const [tab, setTab] = useState<Tab>(canManageIAM ? 'users' : 'keys');
  const [userModal, setUserModal] = useState(false);
  const [keyModal, setKeyModal] = useState(false);
  const [secretModal, setSecretModal] = useState<string | null>(null);
  const { state: copyState, copy: copyToClipboard, reset: resetCopy } = useCopyToClipboard();
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
    onSuccess: (res) => { inv(); setKeyModal(false); setSecretModal(res.secret); resetCopy(); },
  });
  const revokeKeyMut = useMutation({
    mutationFn: revokeIAMAPIKey,
    onSuccess: inv,
  });

  const handleCopySecret = () => {
    if (!secretModal) return;
    void copyToClipboard(secretModal);
  };

  if (needsTenant) {
    return <TenantRequiredNotice message={t('iam.selectTenant')} />;
  }

  const users = usersData?.users || [];
  const roles = rolesData?.roles || [];
  const keys = (keysData?.api_keys || []).filter((k) => !k.revoked_at);
  const q = search.toLowerCase();
  const fUsers = users.filter((u) => u.username?.toLowerCase().includes(q) || u.email?.toLowerCase().includes(q));
  const fRoles = roles.filter((r) => r.name?.toLowerCase().includes(q));
  const fKeys = keys.filter((k) => k.name?.toLowerCase().includes(q) || k.prefix?.toLowerCase().includes(q));

  const tabs: { id: Tab; label: string; count: number }[] = [
    ...(canManageIAM ? [
      { id: 'users' as Tab, label: t('iam.tabUsers'), count: users.length },
      { id: 'roles' as Tab, label: t('iam.tabRoles'), count: roles.length },
    ] : []),
    { id: 'keys', label: t('iam.tabKeys'), count: keys.length },
  ];

  const shown = tab === 'users' ? fUsers.length : tab === 'roles' ? fRoles.length : fKeys.length;
  const total = tab === 'users' ? users.length : tab === 'roles' ? roles.length : keys.length;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('iam.title')}
        subtitle={t('iam.subtitle')}
        actions={
          <>
            <RefreshButton onRefresh={() => { refetchUsers(); refetchKeys(); }} isFetching={uFetch} />
            {tab === 'users' && (
              <button type="button" onClick={() => setUserModal(true)} className="btn-primary">
                <Plus size={18} /> {t('iam.addUser')}
              </button>
            )}
            {tab === 'keys' && (
              <button type="button" onClick={() => setKeyModal(true)} className="btn-primary">
                <Plus size={18} /> {t('iam.createKey')}
              </button>
            )}
          </>
        }
      />

      <SurfaceCard padding="none">
        <TableToolbar tabs={<TabBar variant="inline" tabs={tabs} active={tab} onChange={setTab} />}>
          <TableFilter
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('table.filter')}
            aria-label={t('table.filter')}
          />
        </TableToolbar>

        {tab === 'users' && (
          <PageTable minWidth={640}>
            <PageTableHead>
              <PageTableTh>{t('login.username')}</PageTableTh>
              <PageTableTh>{t('iam.role')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {fUsers.length === 0 && <TableMessageRow cols={4}>{t('iam.empty')}</TableMessageRow>}
              {fUsers.map((u) => (
                <PageTableRow key={u.id}>
                  <PageTableTd>
                    <div className="font-semibold">{u.username}</div>
                    {u.email && <div className="font-mono text-xs text-on-surface-variant">{u.email}</div>}
                  </PageTableTd>
                  <PageTableTd>{u.role}</PageTableTd>
                  <PageTableTd>
                    <StatusBadge status={u.state || 'active'} pulse={false} />
                  </PageTableTd>
                  <PageTableTd className="text-right">
                    <OverflowMenu
                      variant="dots"
                      label={t('table.moreActions')}
                      items={u.role !== 'root' ? [{
                        id: 'delete',
                        label: t('common.delete'),
                        icon: <Trash2 size={14} />,
                        danger: true,
                        onSelect: () => setDeleteUser({ id: u.id, name: u.username }),
                      }] : []}
                    />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        )}

        {tab === 'roles' && (
          <PageTable minWidth={640}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('iam.permissions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {fRoles.length === 0 && <TableMessageRow cols={2}>{t('iam.empty')}</TableMessageRow>}
              {fRoles.map((r) => (
                <PageTableRow key={r.id}>
                  <PageTableTd>
                    <span className="font-semibold">{r.name}</span>
                    {r.is_system && (
                      <span className="ml-2 text-xs bg-off-bg text-off-fg px-2 py-0.5 rounded-full font-medium">
                        {t('iam.systemRole')}
                      </span>
                    )}
                  </PageTableTd>
                  <PageTableTd mono className="text-on-surface-variant text-xs">{(r.permissions || []).join(', ') || '—'}</PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        )}

        {tab === 'keys' && (
          <PageTable minWidth={640}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('iam.prefix')}</PageTableTh>
              <PageTableTh>{t('iam.created')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {fKeys.length === 0 && <TableMessageRow cols={4}>{t('iam.empty')}</TableMessageRow>}
              {fKeys.map((k) => (
                <PageTableRow key={k.id}>
                  <PageTableTd className="font-semibold">{k.name}</PageTableTd>
                  <PageTableTd mono>{k.prefix}</PageTableTd>
                  <PageTableTd className="text-on-surface-variant">{ago(k.created_at, dateLocale)}</PageTableTd>
                  <PageTableTd className="text-right">
                    <OverflowMenu
                      variant="dots"
                      label={t('table.moreActions')}
                      items={[{
                        id: 'revoke',
                        label: t('iam.revoke'),
                        icon: <Trash2 size={14} />,
                        danger: true,
                        onSelect: () => revokeKeyMut.mutate(k.id),
                      }]}
                    />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        )}

        <TablePager shown={shown} total={total} />
      </SurfaceCard>

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
