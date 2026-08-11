import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Users, Copy, Check, Trash2 } from 'lucide-react';
import { listTenants, createTenant, deleteTenant } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
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

const DEFAULT_TENANT_SLUG = 'default';

function suggestSlug(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/_/g, '-')
    .replace(/[^a-z0-9-]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

export function Tenants() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [slugTouched, setSlugTouched] = useState(false);
  const [form, setForm] = useState({ name: '', slug: '', admin_password: '' });
  const [createdAdmin, setCreatedAdmin] = useState<{
    tenantName: string;
    slug: string;
    username: string;
  } | null>(null);
  const [copied, setCopied] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string; slug: string } | null>(null);
  const queryClient = useQueryClient();
  const isRoot = useAppSelector(selectIsRoot);

  const { data, isLoading, isRefetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: isRoot,
  });

  const createMutation = useMutation({
    mutationFn: createTenant,
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
      setCreateModal(false);
      setSlugTouched(false);
      setForm({ name: '', slug: '', admin_password: '' });
      const username = res.admin_user?.username || `${res.tenant.slug}-admin`;
      setCreatedAdmin({
        tenantName: res.tenant.name,
        slug: res.tenant.slug,
        username,
      });
      setCopied(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTenant(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
      setDeleteTarget(null);
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

  const handleCopyUsername = async () => {
    if (!createdAdmin) return;
    await navigator.clipboard.writeText(createdAdmin.username);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

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
            filtered.map((tenant) => {
              const canDelete = tenant.slug !== DEFAULT_TENANT_SLUG;
              return (
                <ResourceGridCard key={tenant.id}>
                  <div className="flex items-start justify-between gap-2 mb-3">
                    <h3 className="font-headline text-headline-md font-semibold text-on-surface">{tenant.name}</h3>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={tenant.state || 'active'} pulse={false} />
                      {canDelete && (
                        <button
                          type="button"
                          className="p-1.5 rounded-lg text-error hover:bg-error/10"
                          title={t('tenants.delete')}
                          aria-label={t('tenants.delete')}
                          onClick={() => setDeleteTarget({ id: tenant.id, name: tenant.name, slug: tenant.slug })}
                        >
                          <Trash2 size={16} />
                        </button>
                      )}
                    </div>
                  </div>
                  <p className="text-sm text-on-surface-variant mb-3">{tenant.slug}</p>
                  <div className="text-sm space-y-1">
                    <p>
                      <span className="text-on-surface-variant">{t('tenants.adminUser')}:</span>{' '}
                      <code className="text-on-surface">{tenant.slug}-admin</code>
                    </p>
                    {!canDelete && (
                      <p className="text-xs text-on-surface-variant">{t('tenants.defaultProtected')}</p>
                    )}
                  </div>
                </ResourceGridCard>
              );
            })
          )}
        </div>
      </RefreshingPanel>

      <Modal
        isOpen={createModal}
        onClose={() => {
          setCreateModal(false);
          setSlugTouched(false);
        }}
        title={t('tenants.modalTitle')}
      >
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
              onChange={(e) => {
                const name = e.target.value;
                setForm((prev) => ({
                  ...prev,
                  name,
                  slug: slugTouched ? prev.slug : suggestSlug(name),
                }));
              }}
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
              onChange={(e) => {
                setSlugTouched(true);
                setForm({ ...form, slug: e.target.value.toLowerCase() });
              }}
              className={formInputClass}
              placeholder="acme"
            />
            <p className="mt-1 text-xs text-on-surface-variant">
              {t('tenants.adminHint').replace('{username}', form.slug ? `${form.slug}-admin` : '…-admin')}
            </p>
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
            <button
              type="button"
              onClick={() => {
                setCreateModal(false);
                setSlugTouched(false);
              }}
              className="btn-secondary"
            >
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={!!createdAdmin}
        onClose={() => setCreatedAdmin(null)}
        title={t('tenants.createdTitle')}
      >
        {createdAdmin && (
          <div className="space-y-4">
            <p className="text-sm text-on-surface-variant">
              {t('tenants.createdBody').replace('{name}', createdAdmin.tenantName)}
            </p>
            <div className="rounded-lg border border-card-border bg-surface-container-low p-4 space-y-2">
              <p className="text-sm">
                <span className="text-on-surface-variant">{t('login.username')}:</span>{' '}
                <code className="font-semibold text-on-surface">{createdAdmin.username}</code>
              </p>
              <p className="text-xs text-on-surface-variant">{t('tenants.createdPasswordNote')}</p>
            </div>
            <div className="flex justify-end gap-3">
              <button type="button" onClick={handleCopyUsername} className="btn-secondary flex items-center gap-2">
                {copied ? <Check size={16} /> : <Copy size={16} />}
                {copied ? t('common.copied') : t('tenants.copyUsername')}
              </button>
              <button type="button" onClick={() => setCreatedAdmin(null)} className="btn-primary">
                {t('common.close')}
              </button>
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => {
          setDeleteTarget(null);
          deleteMutation.reset();
        }}
        onConfirm={() => {
          if (deleteTarget) deleteMutation.mutate(deleteTarget.id);
        }}
        title={t('tenants.deleteTitle')}
        message={t('tenants.deleteMessage')}
        resourceName={deleteTarget ? `${deleteTarget.name} (${deleteTarget.slug})` : undefined}
        loading={deleteMutation.isPending}
        error={deleteMutation.isError ? (deleteMutation.error as Error).message : undefined}
      />
    </div>
  );
}
