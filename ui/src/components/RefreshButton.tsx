import clsx from 'clsx';
import { RefreshCw } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { ptBR } from 'date-fns/locale';

interface RefreshButtonProps {
  onRefresh: () => void;
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
  label = 'Atualizar',
  compact = false,
  className,
}: RefreshButtonProps) {
  const lastUpdate =
    dataUpdatedAt && dataUpdatedAt > 0
      ? formatDistanceToNow(dataUpdatedAt, { addSuffix: true, locale: ptBR })
      : null;

  return (
    <div className={clsx('flex items-center gap-2', className)}>
      {!compact && lastUpdate && !isFetching && (
        <span className="text-xs text-gray-400 hidden sm:inline">
          {lastUpdate}
        </span>
      )}
      <button
        type="button"
        onClick={onRefresh}
        disabled={isFetching}
        title={isFetching ? 'Atualizando...' : 'Atualizar dados'}
        className={clsx(
          'flex items-center gap-2 border rounded-lg transition-all duration-200',
          compact ? 'p-2' : 'px-4 py-2',
          isFetching
            ? 'opacity-80 cursor-wait border-nimbus-300 bg-nimbus-50 dark:bg-nimbus-900/20 text-nimbus-700'
            : 'hover:bg-gray-50 dark:hover:bg-dark-200 active:scale-95'
        )}
      >
        <RefreshCw
          size={18}
          className={clsx(
            'transition-transform',
            isFetching && 'animate-spin text-nimbus-600'
          )}
        />
        {!compact && (
          <span className="text-sm font-medium">
            {isFetching ? 'Atualizando...' : label}
          </span>
        )}
      </button>
    </div>
  );
}
