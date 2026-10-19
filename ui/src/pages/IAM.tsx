import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Copy, Check, AlertCircle, Pencil } from 'lucide-react';
import {
  listTenants, listIAMUsers, createIAMUser, updateIAMUser, deleteIAMUser,
  listIAMRoles, updateIAMRole,
  listIAMAPIKeys, createIAMAPIKey, updateIAMAPIKey, revokeIAMAPIKey,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { OverflowMenu } from '../components/OverflowMenu';
import { TagsInput, TagPills, mergeTags } from '../components/TagsInput';
import { ago } from '../lib/ago';
import { RefreshButton } from '../components/RefreshButton';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { useCopyToClipboard } from '../hooks/useCopyToClipboard';
import {
  PageHeader, SurfaceCard, TabBar, TenantRequiredNotice, InfoBanner, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
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
  const [tagEdit, setTagEdit] = useState<{ kind: 'user' | 'role' | 'key'; id: string; label: string; tags: string[] } | null>(null);
  const [tagDraft, setTagDraft] = useState('');
  const [userForm, setUserForm] = useState({ username: '', password: '', email: '', role_name: 'tenant.operator', tags: [] as string[] });
  const [keyForm, setKeyForm] = useState({ name: '', expires_in_days: 90, tags: [] as string[] });
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

  const { data: tenantsData } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: !needsTenant && currentUser?.role === 'root',
  });
  const tenantName = (u: { role: string; tenant_id?: string }) =>
    u.role === 'root' ? t('iam.allTenants')
      : (tenantsData?.tenants.find((x) => x.id === u.tenant_id)?.name ?? '—');

  const inv = () => {
    qc.invalidateQueries({ queryKey: queryKeys.iamUsers });
    qc.invalidateQueries({ queryKey: queryKeys.iamKeys });
    qc.invalidateQueries({ queryKey: queryKeys.iamRoles });
  };

  const createUserMut = useMutation({
    mutationFn: createIAMUser,
    onSuccess: () => { inv(); setUserModal(false); setUserForm({ username: '', password: '', email: '', role_name: 'tenant.operator', tags: [] }); setTagDraft(''); },
  });
  const deleteUserMut = useMutation({
    mutationFn: deleteIAMUser,
    onSuccess: () => { inv(); setDeleteUser(null); },
  });
  const createKeyMut = useMutation({
    mutationFn: createIAMAPIKey,
    onSuccess: (res) => { inv(); setKeyModal(false); setSecretModal(res.secret); setKeyForm({ name: '', expires_in_days: 90, tags: [] }); setTagDraft(''); resetCopy(); },
  });
  const revokeKeyMut = useMutation({
    mutationFn: revokeIAMAPIKey,
    onSuccess: inv,
  });
  const updateTagsMut = useMutation({
    mutationFn: async ({ kind, id, tags }: { kind: 'user' | 'role' | 'key'; id: string; tags: string[] }) => {
      if (kind === 'user') { await updateIAMUser(id, { tags }); return; }
      if (kind === 'role') { await updateIAMRole(id, { tags }); return; }
      await updateIAMAPIKey(id, { tags });
    },
    onSuccess: () => { inv(); setTagEdit(null); },
  });

  const handleCopySecret = () => {
    if (!secretModal) return;
    void copyToClipboard(secretModal);
  };

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

  const pUsers = usePagination(fUsers, q);
  const pRoles = usePagination(fRoles, q);
  const pKeys = usePagination(fKeys, q);
  const pager = tab === 'users' ? pUsers.pager : tab === 'roles' ? pRoles.pager : pKeys.pager;

  if (needsTenant) {
    return <TenantRequiredNotice message={t('iam.selectTenant')} />;
  }

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
              <PageTableTh>{t('nav.tenants')}</PageTableTh>
              <PageTableTh>{t('tags.column')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {fUsers.length === 0 && <TableMessageRow cols={6}>{t('iam.empty')}</TableMessageRow>}
              {pUsers.rows.map((u) => (
                <PageTableRow key={u.id}>
                  <PageTableTd>
                    <div className="font-semibold">{u.username}</div>
                    {u.email && <div className="font-mono text-xs text-on-surface-variant">{u.email}</div>}
                  </PageTableTd>
                  <PageTableTd>{u.role}</PageTableTd>
                  <PageTableTd mono className="text-on-surface-variant">{tenantName(u)}</PageTableTd>
                  <PageTableTd><TagPills tags={u.tags} /></PageTableTd>
                  <PageTableTd>
                    <StatusBadge status={u.state || 'active'} pulse={false} />
                  </PageTableTd>
                  <PageTableTd className="text-right">
                    <OverflowMenu
                      variant="dots"
                      label={t('table.moreActions')}
                      items={u.role !== 'root' ? [
                        {
                          id: 'edit',
                          label: t('common.edit'),
                          icon: <Pencil size={14} />,
                          onSelect: () => { setTagDraft(''); setTagEdit({ kind: 'user', id: u.id, label: u.username, tags: u.tags ? [...u.tags] : [] }); },
                        },
                        {
                          id: 'delete',
                          label: t('common.delete'),
                          icon: <Trash2 size={14} />,
                          danger: true,
                          onSelect: () => setDeleteUser({ id: u.id, name: u.username }),
                        },
                      ] : []}
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
              <PageTableTh>{t('tags.column')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {fRoles.length === 0 && <TableMessageRow cols={4}>{t('iam.empty')}</TableMessageRow>}
              {pRoles.rows.map((r) => (
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
                  <PageTableTd><TagPills tags={r.tags} /></PageTableTd>
                  <PageTableTd className="text-right">
                    {!r.is_system && (
                      <OverflowMenu
                        variant="dots"
                        label={t('table.moreActions')}
                        items={[{
                          id: 'edit',
                          label: t('common.edit'),
                          icon: <Pencil size={14} />,
                          onSelect: () => { setTagDraft(''); setTagEdit({ kind: 'role', id: r.id, label: r.name, tags: r.tags ? [...r.tags] : [] }); },
                        }]}
                      />
                    )}
                  </PageTableTd>
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
              <PageTableTh>{t('tags.column')}</PageTableTh>
              <PageTableTh>{t('iam.created')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {fKeys.length === 0 && <TableMessageRow cols={5}>{t('iam.empty')}</TableMessageRow>}
              {pKeys.rows.map((k) => (
                <PageTableRow key={k.id}>
                  <PageTableTd className="font-semibold">{k.name}</PageTableTd>
                  <PageTableTd mono>{k.prefix}</PageTableTd>
                  <PageTableTd><TagPills tags={k.tags} /></PageTableTd>
                  <PageTableTd className="text-on-surface-variant">{ago(k.created_at, dateLocale)}</PageTableTd>
                  <PageTableTd className="text-right">
                    <OverflowMenu
                      variant="dots"
                      label={t('table.moreActions')}
                      items={[
                        {
                          id: 'edit',
                          label: t('common.edit'),
                          icon: <Pencil size={14} />,
                          onSelect: () => { setTagDraft(''); setTagEdit({ kind: 'key', id: k.id, label: k.name, tags: k.tags ? [...k.tags] : [] }); },
                        },
                        {
                          id: 'revoke',
                          label: t('iam.revoke'),
                          icon: <Trash2 size={14} />,
                          danger: true,
                          onSelect: () => revokeKeyMut.mutate(k.id),
                        },
                      ]}
                    />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        )}

        <TablePager pager={pager} />
      </SurfaceCard>

      <Modal isOpen={userModal} onClose={() => setUserModal(false)} title={t('iam.addUserTitle')}>
        <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); createUserMut.mutate({ ...userForm, tags: mergeTags(userForm.tags, tagDraft) }); }}>
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
          <div>
            <label className="block text-sm font-medium mb-1">{t('tags.column')}</label>
            <TagsInput
              tags={userForm.tags}
              onTagsChange={(tags) => setUserForm({ ...userForm, tags })}
              draft={tagDraft}
              onDraftChange={setTagDraft}
              placeholder={t('common.tagsPlaceholder')}
            />
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={() => setUserModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" className="btn-primary" disabled={createUserMut.isPending}>{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={keyModal} onClose={() => setKeyModal(false)} title={t('iam.createKeyTitle')}>
        <form className="space-y-4" onSubmit={(e) => { e.preventDefault(); createKeyMut.mutate({ ...keyForm, tags: mergeTags(keyForm.tags, tagDraft) }); }}>
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
          <div>
            <label className="block text-sm font-medium mb-1">{t('tags.column')}</label>
            <TagsInput
              tags={keyForm.tags}
              onTagsChange={(tags) => setKeyForm({ ...keyForm, tags })}
              draft={tagDraft}
              onDraftChange={setTagDraft}
              placeholder={t('common.tagsPlaceholder')}
            />
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

      <Modal isOpen={tagEdit !== null} onClose={() => setTagEdit(null)} title={t('common.edit')}>
        {tagEdit && (
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
              updateTagsMut.mutate({ kind: tagEdit.kind, id: tagEdit.id, tags: mergeTags(tagEdit.tags, tagDraft) });
            }}
          >
            <p className="text-sm text-on-surface-variant">{tagEdit.label}</p>
            <div>
              <label className="block text-sm font-medium mb-1">{t('tags.column')}</label>
              <TagsInput
                tags={tagEdit.tags}
                onTagsChange={(tags) => setTagEdit({ ...tagEdit, tags })}
                draft={tagDraft}
                onDraftChange={setTagDraft}
                placeholder={t('common.tagsPlaceholder')}
              />
            </div>
            {updateTagsMut.isError && (
              <p className="text-sm text-error">{(updateTagsMut.error as Error).message}</p>
            )}
            <div className="flex justify-end gap-3 pt-2">
              <button type="button" onClick={() => setTagEdit(null)} className="btn-secondary">{t('common.cancel')}</button>
              <button type="submit" className="btn-primary" disabled={updateTagsMut.isPending}>{t('common.save')}</button>
            </div>
          </form>
        )}
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
