import { Globe, Network as NetworkIcon } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { listNetworks } from '../lib/platform-api';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { isPublicNetwork } from '../lib/networks';
import {
  PageHeader, SurfaceCard, EmptyState, TenantRequiredNotice, InfoBanner,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

export function PublicNetwork() {
  const { t } = useI18n();
  const needsTenant = useNeedsTenant();

  const { data, isLoading, isFetching, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant,
  });

  const publicNet = (data?.networks ?? []).find(isPublicNetwork);

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
    <div className="space-y-6">
      <PageHeader
        title={t('publicNetwork.title')}
        subtitle={t('publicNetwork.subtitle')}
        actions={
          <RefreshButton onRefresh={() => refetch()} isFetching={isRefetching} dataUpdatedAt={dataUpdatedAt} />
        }
      />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
        {!publicNet ? (
          <EmptyState
            icon={<Globe size={48} />}
            title={t('publicNetwork.notConfigured')}
            hint={t('publicNetwork.notConfiguredHint')}
          />
        ) : (
          <div className="max-w-2xl">
            <SurfaceCard>
              <div className="flex items-start gap-4 mb-6">
                <div className="w-12 h-12 bg-primary-container/20 rounded-xl flex items-center justify-center shrink-0">
                  <Globe size={24} className="text-primary-fixed-dim" />
                </div>
                <div>
                  <div className="flex items-center gap-2 flex-wrap">
                    <h2 className="font-headline text-headline-md font-semibold text-on-surface">{t('publicNetwork.name')}</h2>
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-primary-container/20 text-primary-fixed-dim">
                      {t('publicNetwork.badge')}
                    </span>
                    <StatusBadge status={publicNet.state || 'active'} pulse={false} />
                  </div>
                  <p className="text-sm text-on-surface-variant mt-1">{t('publicNetwork.scope')}</p>
                </div>
              </div>

              <p className="text-sm text-on-surface-variant mb-6">{t('publicNetwork.description')}</p>

              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                <div>
                  <dt className="text-on-surface-variant">{t('networks.cidr')}</dt>
                  <dd className="font-data-mono font-medium text-on-surface">{publicNet.cidr}</dd>
                </div>
                {publicNet.gateway && (
                  <div>
                    <dt className="text-on-surface-variant">{t('publicNetwork.gateway')}</dt>
                    <dd className="font-data-mono font-medium text-on-surface">{publicNet.gateway}</dd>
                  </div>
                )}
                {(publicNet.nad_namespace || publicNet.nad_name) && (
                  <div className="sm:col-span-2">
                    <dt className="text-on-surface-variant">{t('publicNetwork.nad')}</dt>
                    <dd className="font-data-mono font-medium text-on-surface">
                      {publicNet.nad_namespace}/{publicNet.nad_name}
                    </dd>
                  </div>
                )}
              </dl>

              <InfoBanner className="mt-6">
                <div className="flex gap-2">
                  <NetworkIcon size={16} className="shrink-0 mt-0.5" />
                  <p>{t('publicNetwork.vmDefaultHint')}</p>
                </div>
              </InfoBanner>
            </SurfaceCard>
          </div>
        )}
      </RefreshingPanel>
    </div>
  );
}
