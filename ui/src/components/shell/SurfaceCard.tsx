import type { ReactNode } from 'react';
import clsx from 'clsx';

interface SurfaceCardProps {
  children: ReactNode;
  className?: string;
  title?: string;
  headerAction?: ReactNode;
  padding?: 'none' | 'md' | 'lg';
}

export function SurfaceCard({ children, className, title, headerAction, padding = 'lg' }: SurfaceCardProps) {
  const pad = padding === 'none' ? '' : padding === 'md' ? 'p-4 md:p-6' : 'p-6';

  return (
    <div className={clsx('vf-card overflow-hidden', className)}>
      {title && (
        <div className="vf-card-header flex justify-between items-center">
          <h3 className="font-headline text-headline-md font-semibold text-on-surface">{title}</h3>
          {headerAction}
        </div>
      )}
      <div className={pad}>{children}</div>
    </div>
  );
}
