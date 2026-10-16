import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2 } from 'lucide-react';
import {
  listSecurityGroups, createSecurityGroup, updateSecurityGroup, deleteSecurityGroup,
} from '../lib/platform-api';
import type { SecurityGroup } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { OverflowMenu } from '../components/OverflowMenu';
import { SGRulesEditor, defaultSGRules, type SGRule } from '../components/SGRulesEditor';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { ago } from '../lib/ago';
import {
  PageHeader, SurfaceCard, TenantRequiredNotice, TableToolbar, TableFilter, TablePager, usePagination, TableMessageRow,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formTextareaClass,
} from '../components/shell';

const emptyForm = () => ({ name: '', description: '', rules: defaultSGRules() });

export function SecurityGroups() {
  const { t, dateLocale } = useI18n();
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

  const { rows, pager } = usePagination(filtered, search);

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="flex flex-col gap-6">
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
          <PageTable minWidth={600}>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('sg.rulesInOut')}</PageTableTh>
              <PageTableTh>{t('common.created')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <TableMessageRow cols={4}>{t('common.loading')}</TableMessageRow>
              ) : filtered.length === 0 ? (
                <TableMessageRow cols={4}>{t('sg.empty')}</TableMessageRow>
              ) : (
                rows.map((sg) => {
                  const rules = sg.rules ?? [];
                  const ingress = rules.filter((r) => r.direction?.toLowerCase().startsWith('in')).length;
                  return (
                    <PageTableRow key={sg.id}>
                      <PageTableTd>
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold">{sg.name}</span>
                          {sg.name === 'default' && (
                            <span className="text-xs px-2 py-0.5 rounded-full bg-accent-soft text-link font-medium">
                              {t('sg.defaultBadge')}
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-on-surface-variant max-w-md truncate">
                          {sg.description || t('sg.noDescription')}
                        </div>
                      </PageTableTd>
                      <PageTableTd mono>{ingress} / {rules.length - ingress}</PageTableTd>
                      <PageTableTd className="text-on-surface-variant whitespace-nowrap">{ago(sg.created_at, dateLocale)}</PageTableTd>
                      <PageTableTd className="text-right">
                        <OverflowMenu
                          variant="dots"
                          label={t('table.moreActions')}
                          items={[
                            {
                              id: 'edit',
                              label: t('common.edit'),
                              icon: <Pencil size={14} />,
                              onSelect: () => setEditSg({ ...sg, rules: sg.rules ? [...sg.rules] : [] }),
                            },
                            {
                              id: 'delete',
                              label: t('common.delete'),
                              icon: <Trash2 size={14} />,
                              danger: true,
                              onSelect: () => setDeleteTarget({ id: sg.id, name: sg.name }),
                            },
                          ]}
                        />
                      </PageTableTd>
                    </PageTableRow>
                  );
                })
              )}
            </PageTableBody>
          </PageTable>
          <TablePager pager={pager} />
        </SurfaceCard>
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
