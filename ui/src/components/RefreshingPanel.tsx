import clsx from 'clsx';
import { Loader2 } from 'lucide-react';
import { ReactNode } from 'react';
import { useI18n } from '../lib/i18n';

interface RefreshingPanelProps {
  isFetching: boolean;
  isLoading?: boolean;
  children: ReactNode;
  className?: string;
}

/** Subtle overlay while background refetch runs (keeps existing data visible). */
export function RefreshingPanel({
  isFetching,
  isLoading = false,
  children,
  className,
}: RefreshingPanelProps) {
  const { t } = useI18n();
  const showOverlay = isFetching && !isLoading;

  return (
    <div className={clsx('relative', className)}>
      {showOverlay && (
        <div
          className="absolute inset-0 z-10 rounded-xl pointer-events-none"
          aria-live="polite"
          aria-busy="true"
        >
          <div className="absolute top-3 right-3 flex items-center gap-2 bg-brand-500 text-white text-xs font-medium px-3 py-1.5 rounded-full shadow-lg animate-pulse">
            <Loader2 size={14} className="animate-spin" />
            {t('refresh.syncing')}
          </div>
          <div className="absolute inset-0 bg-white/30 dark:bg-dark-100/30 rounded-xl backdrop-blur-[1px]" />
        </div>
      )}
      <div
        className={clsx(
          'transition-opacity duration-300',
          showOverlay && 'opacity-75'
        )}
      >
        {children}
      </div>
    </div>
  );
}
