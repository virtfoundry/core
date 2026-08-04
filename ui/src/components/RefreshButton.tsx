import clsx from 'clsx';
import { RefreshCw } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { useI18n } from '../lib/i18n';

interface RefreshButtonProps {
  onRefresh: () => void;
  /** Prefer isRefetching from useQuery — avoids spin on unrelated background fetches */
  isFetching: boolean;
  dataUpdatedAt?: number;
  label?: string;
  compact?: boolean;
  className?: string;
}

export function RefreshButton({
  onRefresh,
  isFetching,
  dataUpdatedAt,
  label,
  compact = false,
  className,
}: RefreshButtonProps) {
  const { t, dateLocale } = useI18n();
  const refreshLabel = label ?? t('refresh.label');

  const lastUpdate =
    dataUpdatedAt && dataUpdatedAt > 0
      ? formatDistanceToNow(dataUpdatedAt, { addSuffix: true, locale: dateLocale })
      : null;

  return (
    <div className={clsx('flex items-center gap-2', className)}>
      {!compact && lastUpdate && !isFetching && (
        <span className="text-xs text-on-surface-variant hidden sm:inline font-data-mono">
          {lastUpdate}
        </span>
      )}
      <button
        type="button"
        onClick={onRefresh}
        disabled={isFetching}
        title={isFetching ? t('refresh.fetching') : t('refresh.title')}
        className={clsx(
          compact ? 'btn-secondary p-2' : 'btn-secondary',
          isFetching && 'opacity-80 cursor-wait border-primary-container/40 bg-primary-container/10',
        )}
      >
        <RefreshCw
          size={18}
          className={clsx('transition-transform', isFetching && 'animate-spin text-primary-fixed-dim')}
        />
        {!compact && (
          <span>{isFetching ? t('refresh.fetching') : refreshLabel}</span>
        )}
      </button>
    </div>
  );
}
