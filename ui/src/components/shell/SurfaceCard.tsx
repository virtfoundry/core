import type { ReactNode } from 'react';
import clsx from 'clsx';

interface SurfaceCardProps {
  children: ReactNode;
  className?: string;
  title?: string;
  headerAction?: ReactNode;
  padding?: 'none' | 'md' | 'lg';
}

/**
 * Card block (12px radius, 20px padding). With `padding="none"` the title stays
 * padded and children bleed to the edges (tables, lists).
 */
export function SurfaceCard({ children, className, title, headerAction, padding = 'lg' }: SurfaceCardProps) {
  const pad = padding === 'none' ? '' : padding === 'md' ? 'px-4 pb-4 md:px-5 md:pb-5' : 'px-5 pb-5';
  const noTitlePad = padding === 'none' ? '' : padding === 'md' ? 'pt-4 md:pt-5' : 'pt-5';

  return (
    <div className={clsx('vf-card overflow-hidden', className)}>
      {title && (
        <div className="vf-card-header flex items-center gap-2">
          <h2 className="m-0 flex-grow font-headline text-base font-semibold text-on-surface">{title}</h2>
          {headerAction}
        </div>
      )}
      <div className={clsx(pad, !title && noTitlePad)}>{children}</div>
    </div>
  );
}
