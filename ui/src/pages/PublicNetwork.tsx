import { Globe, Network as NetworkIcon } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { listNetworks } from '../lib/platform-api';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';
import { isPublicNetwork } from '../lib/networks';

export function PublicNetwork() {
  const { t } = useI18n();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant,
    refetchInterval: 12_000,
  });

  const publicNet = (data?.networks ?? []).find(isPublicNetwork);

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('common.selectTenant')}</div>;
  }

  if (error) {
    return (
      <div className="text-center py-12 space-y-3">
        <p className="text-red-600">{t('common.errorLoad')}: {(error as Error).message}</p>
        <button type="button" onClick={() => refetch()} className="btn-primary">{t('common.retry')}</button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('publicNetwork.title')}</h1>
          <p className="text-gray-500">{t('publicNetwork.subtitle')}</p>
        </div>
        <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
        {!publicNet ? (
          <div className="text-center py-12">
            <Globe size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">{t('publicNetwork.notConfigured')}</p>
            <p className="text-sm text-gray-400 mt-2 max-w-lg mx-auto">{t('publicNetwork.notConfiguredHint')}</p>
          </div>
        ) : (
          <div className="max-w-2xl">
            <div className="bg-white dark:bg-dark-100 rounded-xl border border-purple-200 dark:border-purple-900/40 p-6">
              <div className="flex items-start gap-4 mb-6">
                <div className="w-12 h-12 bg-purple-100 dark:bg-purple-900/30 rounded-xl flex items-center justify-center shrink-0">
                  <Globe size={24} className="text-purple-600" />
                </div>
                <div>
                  <div className="flex items-center gap-2 flex-wrap">
                    <h2 className="text-xl font-semibold">{t('publicNetwork.name')}</h2>
                    <span className="px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300">
                      {t('publicNetwork.badge')}
                    </span>
                    <span className="px-2 py-1 rounded text-xs font-medium bg-green-100 text-green-700">{publicNet.state}</span>
                  </div>
                  <p className="text-sm text-gray-500 mt-1">{t('publicNetwork.scope')}</p>
                </div>
              </div>

              <p className="text-sm text-gray-600 dark:text-gray-400 mb-6">{t('publicNetwork.description')}</p>

              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                <div>
                  <dt className="text-gray-500">{t('networks.cidr')}</dt>
                  <dd className="font-mono font-medium">{publicNet.cidr}</dd>
                </div>
                {publicNet.gateway && (
                  <div>
                    <dt className="text-gray-500">{t('publicNetwork.gateway')}</dt>
                    <dd className="font-mono font-medium">{publicNet.gateway}</dd>
                  </div>
                )}
                {(publicNet.nad_namespace || publicNet.nad_name) && (
                  <div className="sm:col-span-2">
                    <dt className="text-gray-500">{t('publicNetwork.nad')}</dt>
                    <dd className="font-mono font-medium">
                      {publicNet.nad_namespace}/{publicNet.nad_name}
                    </dd>
                  </div>
                )}
              </dl>

              <div className="mt-6 rounded-lg bg-purple-50 dark:bg-purple-900/20 border border-purple-100 dark:border-purple-900/40 px-4 py-3 text-sm text-purple-900 dark:text-purple-200">
                <div className="flex gap-2">
                  <NetworkIcon size={16} className="shrink-0 mt-0.5" />
                  <p>{t('publicNetwork.vmDefaultHint')}</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </RefreshingPanel>
    </div>
  );
}
