import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Shield } from 'lucide-react';
import {
  listSecurityGroups, createSecurityGroup, updateSecurityGroup, deleteSecurityGroup,
} from '../lib/platform-api';
import type { SecurityGroup } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { SGRulesEditor, defaultSGRules, type SGRule } from '../components/SGRulesEditor';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, EmptyState, ResourceGridCard, TenantRequiredNotice,
  formInputClass, formTextareaClass,
} from '../components/shell';

const emptyForm = () => ({ name: '', description: '', rules: defaultSGRules() });

export function SecurityGroups() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editSg, setEditSg] = useState<SecurityGroup | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState(emptyForm());
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.securityGroups,
    queryFn: listSecurityGroups,
    enabled: !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.securityGroups });

  const createMutation = useMutation({
    mutationFn: createSecurityGroup,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm(emptyForm());
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, name, description, rules }: {
      id: string; name: string; description: string; rules: SGRule[];
    }) => updateSecurityGroup(id, { name, description, rules }),
    onSuccess: () => {
      invalidate();
      setEditSg(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteSecurityGroup,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const groups = data?.security_groups || [];
  const filtered = groups.filter((g) => g.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('sg.title')}
        subtitle={`${groups.length} ${t('sg.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('sg.create')}
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
            <EmptyState icon={<Shield size={48} />} title={t('sg.empty')} />
          ) : (
            filtered.map((sg) => (
              <ResourceGridCard key={sg.id}>
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 bg-primary-container/20 rounded-lg flex items-center justify-center shrink-0">
                    <Shield size={20} className="text-primary-fixed-dim" />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-headline text-headline-md font-semibold text-on-surface">{sg.name}</h3>
                      {sg.name === 'default' && (
                        <span className="text-xs px-2 py-0.5 rounded-full bg-primary-container/20 text-primary-fixed-dim">
                          {t('sg.defaultBadge')}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-on-surface-variant">{sg.description || t('sg.noDescription')}</p>
                  </div>
                </div>
                <div className="text-sm text-on-surface-variant mb-3">
                  <p>{t('sg.rulesCount')}: {sg.rules?.length || 0}</p>
                  {sg.rules?.slice(0, 2).map((r, i) => (
                    <p key={i} className="text-xs font-data-mono truncate">
                      {r.direction} {r.protocol} {r.port_from}{r.port_to ? `-${r.port_to}` : ''} {r.cidr}
                    </p>
                  ))}
                </div>
                <ResourceActions
                  editLabel={t('common.edit')}
                  deleteLabel={t('common.delete')}
                  onEdit={() => setEditSg({ ...sg, rules: sg.rules ? [...sg.rules] : [] })}
                  onDelete={() => setDeleteTarget({ id: sg.id, name: sg.name })}
                />
              </ResourceGridCard>
            ))
          )}
        </div>
      </RefreshingPanel>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('sg.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
        loading={deleteMutation.isPending}
      />

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('sg.modalCreate')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate({
              name: form.name,
              description: form.description,
              rules: form.rules,
            });
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={formInputClass} placeholder="web-servers" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('sg.description')}</label>
            <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
              className={formTextareaClass} rows={2} />
          </div>
          <SGRulesEditor rules={form.rules} onChange={(rules) => setForm({ ...form, rules })} />
          {createMutation.isError && (
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">{t('common.create')}</button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={!!editSg} onClose={() => setEditSg(null)} title={t('sg.modalEdit')}>
        {editSg && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              updateMutation.mutate({
                id: editSg.id,
                name: editSg.name,
                description: editSg.description || '',
                rules: editSg.rules || [],
              });
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
              <input required value={editSg.name} onChange={(e) => setEditSg({ ...editSg, name: e.target.value })}
                className={formInputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('sg.description')}</label>
              <textarea value={editSg.description || ''} onChange={(e) => setEditSg({ ...editSg, description: e.target.value })}
                className={formTextareaClass} rows={2} />
            </div>
            <SGRulesEditor
              rules={editSg.rules || []}
              onChange={(rules) => setEditSg({ ...editSg, rules })}
            />
            {updateMutation.isError && (
              <p className="text-error text-sm">{(updateMutation.error as Error).message}</p>
            )}
            <div className="flex justify-end gap-3 pt-4">
              <button type="button" onClick={() => setEditSg(null)} className="btn-secondary">{t('common.cancel')}</button>
              <button type="submit" disabled={updateMutation.isPending} className="btn-primary">{t('common.save')}</button>
            </div>
          </form>
        )}
      </Modal>
    </div>
  );
}
