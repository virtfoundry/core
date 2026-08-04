import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Users } from 'lucide-react';
import { listTenants, createTenant } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { useAppSelector } from '../store/hooks';
import { selectIsRoot } from '../store/authSlice';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SearchField, EmptyState, ResourceGridCard,
  formInputClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function Tenants() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ name: '', slug: '', admin_password: '' });
  const queryClient = useQueryClient();
  const isRoot = useAppSelector(selectIsRoot);

  const { data, isLoading, isFetching, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: isRoot,
  });

  const createMutation = useMutation({
    mutationFn: createTenant,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
      setCreateModal(false);
      setForm({ name: '', slug: '', admin_password: '' });
    },
  });

  if (!isRoot) {
    return (
      <div className="text-center py-16 text-on-surface-variant">
        {t('tenants.rootOnly')}
      </div>
    );
  }

  const tenants = data?.tenants || [];
  const filtered = tenants.filter((tenant) =>
    tenant.name.toLowerCase().includes(search.toLowerCase()) ||
    tenant.slug.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('nav.tenants')}
        subtitle={`${tenants.length} ${t('tenants.subtitle')}`}
        actions={
          <>
            <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
            <button type="button" onClick={() => setCreateModal(true)} className="btn-primary">
              <Plus size={18} /> {t('tenants.create')}
            </button>
          </>
        }
      />

      <SearchField value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t('tenants.searchPlaceholder')} />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-gutter">
          {isLoading ? (
            <div className="col-span-full text-center py-12 text-on-surface-variant">{t('common.loading')}</div>
          ) : filtered.length === 0 ? (
            <EmptyState icon={<Users size={48} />} title={t('tenants.empty')} />
          ) : (
            filtered.map((tenant) => (
              <ResourceGridCard key={tenant.id}>
                <div className="flex items-start justify-between gap-2 mb-3">
                  <h3 className="font-headline text-headline-md font-semibold text-on-surface">{tenant.name}</h3>
                  <StatusBadge status={tenant.state || 'active'} pulse={false} />
                </div>
                <p className="text-sm text-on-surface-variant mb-3">{tenant.slug}</p>
                <div className="text-sm space-y-1">
                  <p><span className="text-on-surface-variant">{t('common.region')}:</span> {tenant.slug}</p>
                </div>
              </ResourceGridCard>
            ))
          )}
        </div>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title={t('tenants.modalTitle')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input
              required
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className={formInputClass}
              placeholder="Acme Corp"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Slug</label>
            <input
              required
              pattern="[-a-z0-9]+"
              value={form.slug}
              onChange={(e) => setForm({ ...form, slug: e.target.value.toLowerCase() })}
              className={formInputClass}
              placeholder="acme"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('tenants.adminPassword')}</label>
            <input
              required
              type="password"
              value={form.admin_password}
              onChange={(e) => setForm({ ...form, admin_password: e.target.value })}
              className={formInputClass}
            />
          </div>
          {createMutation.isError && (
            <p className="text-error text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
