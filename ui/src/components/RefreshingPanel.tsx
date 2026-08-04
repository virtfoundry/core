import clsx from 'clsx';
import { ReactNode } from 'react';

interface RefreshingPanelProps {
  /** @deprecated Background refetch no longer dims the panel — kept for API compat */
  isFetching?: boolean;
  isLoading?: boolean;
  children: ReactNode;
  className?: string;
}

/** Layout wrapper — data updates happen in place without full-panel overlay. */
export function RefreshingPanel({
  children,
  className,
}: RefreshingPanelProps) {
  return <div className={clsx('relative', className)}>{children}</div>;
}
