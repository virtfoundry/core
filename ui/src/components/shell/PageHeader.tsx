import type { ReactNode } from 'react';
import clsx from 'clsx';

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  className?: string;
  hero?: boolean;
  breadcrumb?: string;
  /** Pill rendered next to the title (e.g. StatusBadge on detail pages). */
  status?: ReactNode;
}

export function PageHeader({ title, subtitle, actions, className, hero = false, breadcrumb, status }: PageHeaderProps) {
  return (
    <div className={clsx('flex flex-wrap items-end gap-4', className)}>
      <div className="flex-1 basis-80 min-w-0 flex flex-col gap-1.5">
        {breadcrumb && <p className="text-on-surface-variant text-[13px]">{breadcrumb}</p>}
        <div className="flex items-center gap-3 flex-wrap">
          <h1
            className={clsx(
              'font-headline text-on-surface m-0 tracking-[-0.02em]',
              hero ? 'text-[36px] leading-[42px]' : 'text-[28px] leading-[34px]',
            )}
          >
            {title}
          </h1>
          {status}
        </div>
        {subtitle && <p className="text-on-surface-variant m-0">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-3 shrink-0">{actions}</div>}
    </div>
  );
}
