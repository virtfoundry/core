import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Shield } from 'lucide-react';
import {
  listSecurityGroups, createSecurityGroup, updateSecurityGroup, deleteSecurityGroup,
} from '../lib/platform-api';
import type { SecurityGroup } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';

export function SecurityGroups() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [editSg, setEditSg] = useState<SecurityGroup | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', description: '' });
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.securityGroups,
    queryFn: listSecurityGroups,
    enabled: !needsTenant,
    refetchInterval: 12_000,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.securityGroups });

  const createMutation = useMutation({
    mutationFn: createSecurityGroup,
    onSuccess: () => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', description: '' });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, name, description, rules }: {
      id: string; name: string; description: string; rules: SecurityGroup['rules'];
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
    return <div className="text-center py-12 text-amber-600">{t('common.selectTenant')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('sg.title')}</h1>
          <p className="text-gray-500">{groups.length} {t('sg.subtitle')}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary">
            <Plus size={18} /> {t('sg.create')}
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`}
          className="w-full pl-10 pr-4 py-3 border rounded-lg" />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full text-center py-12">{t('common.loading')}</div>
        ) : filtered.length === 0 ? (
          <div className="col-span-full text-center py-12">
            <Shield size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">{t('sg.empty')}</p>
          </div>
        ) : (
          filtered.map((sg) => (
            <div key={sg.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-10 h-10 bg-purple-100 dark:bg-purple-900/30 rounded-lg flex items-center justify-center">
                  <Shield size={20} className="text-purple-500" />
                </div>
                <div>
                  <h3 className="font-semibold">{sg.name}</h3>
                  <p className="text-sm text-gray-500">{sg.description || t('sg.noDescription')}</p>
                </div>
              </div>
              <div className="text-sm text-gray-500">
                <p>{t('sg.rulesCount')}: {sg.rules?.length || 0}</p>
              </div>
              <ResourceActions
                editLabel={t('common.edit')}
                deleteLabel={t('common.delete')}
                onEdit={() => setEditSg(sg)}
                onDelete={() => setDeleteTarget({ id: sg.id, name: sg.name })}
              />
            </div>
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
              rules: [{ direction: 'ingress', protocol: 'tcp', port_from: 22, cidr: '0.0.0.0/0' }],
            });
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg" placeholder="web-servers" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('sg.description')}</label>
            <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg" rows={3} />
          </div>
          {createMutation.isError && (
            <p className="text-red-500 text-sm">{(createMutation.error as Error).message}</p>
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
                className="w-full px-4 py-2 border rounded-lg" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">{t('sg.description')}</label>
              <textarea value={editSg.description || ''} onChange={(e) => setEditSg({ ...editSg, description: e.target.value })}
                className="w-full px-4 py-2 border rounded-lg" rows={3} />
            </div>
            <p className="text-sm text-gray-500">{t('sg.rulesCount')}: {editSg.rules?.length || 0}</p>
            {updateMutation.isError && (
              <p className="text-red-500 text-sm">{(updateMutation.error as Error).message}</p>
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
