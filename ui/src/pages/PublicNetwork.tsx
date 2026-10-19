import { Globe, Network as NetworkIcon } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { listNetworks } from '../lib/platform-api';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { isPublicNetwork } from '../lib/networks';
import {
  PageHeader, SurfaceCard, EmptyState, TenantRequiredNotice, InfoBanner, TablePager, usePagination,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';
import { StatsCard } from '../components/StatsCard';
import { TagPills } from '../components/TagsInput';

export function PublicNetwork() {
  const { t } = useI18n();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant,
  });

  const publicNet = (data?.networks ?? []).find(isPublicNetwork);
  const { pager } = usePagination(publicNet ? [publicNet] : [], '');

  if (needsTenant) {
    return <TenantRequiredNotice message={t('common.selectTenant')} />;
  }

  if (error) {
    return (
      <div className="text-center py-12 space-y-3">
        <p className="text-error">{t('common.errorLoad')}: {(error as Error).message}</p>
        <button type="button" onClick={() => refetch()} className="btn-primary">{t('common.retry')}</button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={t('publicNetwork.title')}
        subtitle={t('publicNetwork.subtitle')}
        actions={
          <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
        }
      />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        {!publicNet ? (
          <SurfaceCard>
            <EmptyState
              icon={<Globe />}
              title={t('publicNetwork.notConfigured')}
              hint={t('publicNetwork.notConfiguredHint')}
            />
          </SurfaceCard>
        ) : (
          <div className="flex flex-col gap-6">
            <section className="grid gap-4 grid-cols-[repeat(auto-fit,minmax(220px,1fr))]">
              <StatsCard title={t('networks.cidr')} value={publicNet.cidr} />
              {publicNet.gateway && <StatsCard title={t('publicNetwork.gateway')} value={publicNet.gateway} />}
            </section>

            <SurfaceCard padding="none" className="overflow-hidden">
              <PageTable minWidth={800}>
                <PageTableHead>
                  <PageTableTh>{t('common.name')}</PageTableTh>
                  <PageTableTh>{t('networks.cidr')}</PageTableTh>
                  <PageTableTh>{t('publicNetwork.gateway')}</PageTableTh>
                  <PageTableTh>{t('publicNetwork.nad')}</PageTableTh>
                  <PageTableTh>{t('common.state')}</PageTableTh>
                  <PageTableTh>{t('tags.column')}</PageTableTh>
                </PageTableHead>
                <PageTableBody>
                  <PageTableRow>
                    <PageTableTd>
                      <span className="font-semibold">{t('publicNetwork.name')}</span>
                      <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-medium bg-accent-soft text-link">
                        {t('publicNetwork.badge')}
                      </span>
                      <div className="text-xs text-on-surface-variant">{t('publicNetwork.scope')}</div>
                    </PageTableTd>
                    <PageTableTd mono>{publicNet.cidr}</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant">{publicNet.gateway || '—'}</PageTableTd>
                    <PageTableTd mono className="text-on-surface-variant">
                      {publicNet.nad_namespace || publicNet.nad_name
                        ? `${publicNet.nad_namespace}/${publicNet.nad_name}`
                        : '—'}
                    </PageTableTd>
                    <PageTableTd>
                      <StatusBadge status={publicNet.state || 'active'} pulse={false} />
                    </PageTableTd>
                    <PageTableTd><TagPills tags={publicNet.tags} /></PageTableTd>
                  </PageTableRow>
                </PageTableBody>
              </PageTable>
              <TablePager pager={pager} />
            </SurfaceCard>

            <InfoBanner>
              <div className="flex gap-2">
                <NetworkIcon size={16} className="shrink-0 mt-0.5" />
                <p>{t('publicNetwork.description')} {t('publicNetwork.vmDefaultHint')}</p>
              </div>
            </InfoBanner>
          </div>
        )}
      </RefreshingPanel>
    </div>
  );
}
