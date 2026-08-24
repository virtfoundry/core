import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { listTargetGroups, createTargetGroup, deleteTargetGroup } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass,
} from '../components/shell';

export function TargetGroups() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', port: 80 });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.targetGroups,
    queryFn: listTargetGroups,
    enabled: !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.targetGroups });

  const createMutation = useMutation({
    mutationFn: () => createTargetGroup({ name: form.name, port: Number(form.port), protocol: 'tcp' }),
    onSuccess: (res) => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', port: 80 });
      navigate(`/target-groups/${res.target_group.id}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteTargetGroup,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const rows = data?.target_groups || [];
  const filtered = rows.filter((g) => g.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('tg.title')}
        subtitle={`${rows.length} ${t('tg.subtitle')}`}
        actions={(
          <>
            <RefreshButton onRefresh={() => void refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" className="btn-primary" onClick={() => setCreateModal(true)}>
              <Plus size={18} /> {t('tg.create')}
            </button>
          </>
        )}
      />

      <InfoBanner>{t('tg.emptyHint')}</InfoBanner>

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('lb.protocol')}</PageTableTh>
              <PageTableTh>{t('tg.instancePort')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <tr><td colSpan={4} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={4} className="text-center py-12 text-on-surface-variant">{t('tg.empty')}</td></tr>
              ) : filtered.map((tg) => (
                <PageTableRow key={tg.id}>
                  <PageTableTd>
                    <Link to={`/target-groups/${tg.id}`} className="font-medium hover:underline text-on-surface">
                      {tg.name}
                    </Link>
                  </PageTableTd>
                  <PageTableTd className="uppercase font-data-mono text-sm">{tg.protocol}</PageTableTd>
                  <PageTableTd className="font-data-mono">{tg.port}</PageTableTd>
                  <PageTableTd>
                    <ResourceActions
                      variant="inline"
                      editLabel={t('tg.openDetail')}
                      deleteLabel={t('common.delete')}
                      onEdit={() => navigate(`/target-groups/${tg.id}`)}
                      onDelete={() => setDeleteTarget({ id: tg.id, name: tg.name })}
                    />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      </RefreshingPanel>

      <Modal open={createModal} onClose={() => !createMutation.isPending && setCreateModal(false)} title={t('tg.modalCreate')}>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate();
          }}
        >
          <label className="block space-y-1">
            <span className="text-sm">{t('common.name')}</span>
            <input
              className={formInputClass}
              required
              disabled={createMutation.isPending}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </label>
          <label className="block space-y-1">
            <span className="text-sm">{t('tg.instancePort')}</span>
            <input
              className={formInputClass}
              type="number"
              min={1}
              max={65535}
              required
              disabled={createMutation.isPending}
              value={form.port}
              onChange={(e) => setForm({ ...form, port: Number(e.target.value) })}
            />
          </label>
          {createMutation.error && (
            <p className="text-sm text-error">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-2">
            <button type="button" className="btn-secondary" disabled={createMutation.isPending} onClick={() => setCreateModal(false)}>
              {t('common.cancel')}
            </button>
            <button type="submit" className="btn-primary" disabled={createMutation.isPending}>
              {t('common.create')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('tg.deleteTitle')}
        message={t('common.confirmDeleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
      />
    </div>
  );
}
