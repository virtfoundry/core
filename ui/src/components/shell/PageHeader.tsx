import type { ReactNode } from 'react';
import clsx from 'clsx';

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  className?: string;
  hero?: boolean;
  breadcrumb?: string;
}

export function PageHeader({ title, subtitle, actions, className, hero = false, breadcrumb }: PageHeaderProps) {
  return (
    <div className={clsx('flex flex-col md:flex-row justify-between items-start md:items-end gap-4', className)}>
      <div>
        {breadcrumb && (
          <p className="font-label text-on-surface-variant mb-2 normal-case tracking-normal">{breadcrumb}</p>
        )}
        <h1
          className={clsx(
            'font-headline text-on-surface tracking-tight',
            hero ? 'text-headline-xl md:text-headline-xl' : 'text-headline-lg-mobile md:text-headline-lg',
          )}
        >
          {title}
        </h1>
        {subtitle && (
          <p className="text-body-sm md:text-body-md text-on-surface-variant mt-1">{subtitle}</p>
        )}
      </div>
      {actions && <div className="flex flex-wrap gap-3 shrink-0">{actions}</div>}
    </div>
  );
}
