import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { listLoadBalancers, createLoadBalancer, deleteLoadBalancer } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { ResourceActions } from '../components/ResourceActions';
import { StatusBadge } from '../components/StatusBadge';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, SurfaceCard, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formTextareaClass,
} from '../components/shell';

export function LoadBalancers() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [form, setForm] = useState({ name: '', description: '' });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.loadBalancers,
    queryFn: listLoadBalancers,
    enabled: !needsTenant,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.loadBalancers });

  const createMutation = useMutation({
    mutationFn: createLoadBalancer,
    onSuccess: (res) => {
      invalidate();
      setCreateModal(false);
      setForm({ name: '', description: '' });
      navigate(`/load-balancers/${res.load_balancer.id}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteLoadBalancer,
    onSuccess: () => {
      invalidate();
      setDeleteTarget(null);
    },
  });

  const rows = data?.load_balancers || [];
  const filtered = rows.filter((g) => g.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('lb.title')}
        subtitle={`${rows.length} ${t('lb.subtitle')}`}
        actions={(
          <>
            <RefreshButton onRefresh={() => void refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" className="btn-primary" onClick={() => setCreateModal(true)}>
              <Plus size={18} /> {t('lb.create')}
            </button>
          </>
        )}
      />

      <InfoBanner>{t('lb.emptyHint')}</InfoBanner>

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={`${t('common.search')}...`} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <SurfaceCard padding="none" className="overflow-hidden">
          <PageTable>
            <PageTableHead>
              <PageTableTh>{t('common.name')}</PageTableTh>
              <PageTableTh>{t('lb.vip')}</PageTableTh>
              <PageTableTh>{t('common.state')}</PageTableTh>
              <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {isLoading ? (
                <tr><td colSpan={4} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={4} className="text-center py-12 text-on-surface-variant">{t('lb.empty')}</td></tr>
              ) : filtered.map((lb) => (
                <PageTableRow key={lb.id}>
                  <PageTableTd>
                    <Link to={`/load-balancers/${lb.id}`} className="font-medium hover:underline text-on-surface">
                      {lb.name}
                    </Link>
                    {lb.description ? (
                      <p className="text-xs text-on-surface-variant mt-0.5 truncate max-w-xs">{lb.description}</p>
                    ) : null}
                  </PageTableTd>
                  <PageTableTd className="font-data-mono text-sm">{lb.vip || '—'}</PageTableTd>
                  <PageTableTd><StatusBadge status={lb.state} /></PageTableTd>
                  <PageTableTd>
                    <ResourceActions
                      variant="inline"
                      editLabel={t('lb.openDetail')}
                      deleteLabel={t('common.delete')}
                      onEdit={() => navigate(`/load-balancers/${lb.id}`)}
                      onDelete={() => setDeleteTarget({ id: lb.id, name: lb.name })}
                    />
                  </PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
      </RefreshingPanel>

      <Modal open={createModal} onClose={() => !createMutation.isPending && setCreateModal(false)} title={t('lb.modalCreate')}>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
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
            <span className="text-sm">{t('lb.description')}</span>
            <textarea
              className={formTextareaClass}
              disabled={createMutation.isPending}
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
          </label>
          {createMutation.isPending && (
            <p className="text-sm text-on-surface-variant">{t('lb.creatingVip')}</p>
          )}
          {createMutation.error && (
            <p className="text-sm text-error">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-2">
            <button type="button" className="btn-secondary" disabled={createMutation.isPending} onClick={() => setCreateModal(false)}>
              {t('common.cancel')}
            </button>
            <button type="submit" className="btn-primary" disabled={createMutation.isPending}>
              {createMutation.isPending ? t('common.loading') : t('common.create')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title={t('lb.deleteTitle')}
        message={t('lb.deleteMessage')}
        resourceName={deleteTarget?.name}
        confirmLabel={t('common.delete')}
        cancelLabel={t('common.cancel')}
      />
    </div>
  );
}
