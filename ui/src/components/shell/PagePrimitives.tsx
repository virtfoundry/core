import { useState, type ReactNode, type InputHTMLAttributes, type TdHTMLAttributes } from 'react';
import { Search, type LucideIcon } from 'lucide-react';
import clsx from 'clsx';
import { useI18n } from '../../lib/i18n';

export const formInputClass =
  'w-full px-3.5 h-11 border border-outline-variant rounded-lg bg-card text-on-surface text-[15px] placeholder:text-on-surface-variant focus:border-accent focus:ring-1 focus:ring-accent outline-none transition-colors disabled:opacity-60';

export const formSelectClass = formInputClass;

export const formTextareaClass =
  'w-full px-3.5 py-2.5 border border-outline-variant rounded-lg bg-card text-on-surface text-[15px] placeholder:text-on-surface-variant focus:border-accent focus:ring-1 focus:ring-accent outline-none transition-colors';

/** Field label (weight 500, 14px). */
export const formLabelClass = 'block text-sm font-medium text-on-surface mb-1.5';

/** Outline 32px button used in table toolbars and pagination. */
export const toolbarBtnClass =
  'inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded-lg border border-outline-variant bg-card text-on-surface text-[13px] hover:bg-sunk disabled:opacity-40 disabled:pointer-events-none transition-colors';

interface SearchFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> {
  className?: string;
  containerClassName?: string;
}

export function SearchField({ className, containerClassName, ...props }: SearchFieldProps) {
  return (
    <div className={clsx('relative max-w-xl', containerClassName)}>
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none" size={16} />
      <input
        type="search"
        className={clsx(
          'w-full h-10 pl-9 pr-3 border border-outline-variant rounded-lg bg-card text-sm text-on-surface placeholder:text-on-surface-variant focus:border-accent focus:ring-1 focus:ring-accent outline-none',
          className,
        )}
        {...props}
      />
    </div>
  );
}

interface InfoBannerProps {
  children: ReactNode;
  variant?: 'info' | 'warning';
  className?: string;
}

export function InfoBanner({ children, variant = 'info', className }: InfoBannerProps) {
  return (
    <div
      className={clsx(
        'rounded-lg border px-4 py-3 text-sm',
        variant === 'warning'
          ? 'border-warn-fg/30 bg-warn-bg text-warn-fg'
          : 'border-accent/30 bg-accent-soft text-on-surface',
        className,
      )}
    >
      {children}
    </div>
  );
}

interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  hint?: string;
  action?: ReactNode;
}

export function EmptyState({ icon, title, hint, action }: EmptyStateProps) {
  return (
    <div className="col-span-full flex flex-col items-center gap-2.5 text-center py-6 px-2">
      <div className="w-12 h-12 rounded-xl bg-accent-soft text-accent flex items-center justify-center [&>svg]:w-6 [&>svg]:h-6 [&>svg]:stroke-[1.7]">
        {icon}
      </div>
      <strong className="text-base font-semibold text-on-surface">{title}</strong>
      {hint && <span className="text-on-surface-variant max-w-[360px] leading-[21px]">{hint}</span>}
      {action && <div className="mt-1 flex justify-center">{action}</div>}
    </div>
  );
}

interface ResourceGridCardProps {
  children: ReactNode;
  className?: string;
}

export function ResourceGridCard({ children, className }: ResourceGridCardProps) {
  return (
    <div className={clsx('vf-card p-5 hover:border-accent/40 transition-colors', className)}>
      {children}
    </div>
  );
}

interface TabBarProps<T extends string> {
  tabs: {
    id: T;
    label: string;
    icon?: LucideIcon;
    /** Mono counter shown after the label. */
    count?: number | string;
  }[];
  active: T;
  onChange: (id: T) => void;
  /** `bar` = page-level tabs with bottom divider (34px). `inline` = compact 32px, no divider (card toolbars). */
  variant?: 'bar' | 'inline';
  className?: string;
}

export function TabBar<T extends string>({ tabs, active, onChange, variant = 'bar', className }: TabBarProps<T>) {
  const inline = variant === 'inline';
  return (
    <div
      role="tablist"
      className={clsx('flex flex-wrap gap-1.5', !inline && 'border-b border-card-border pb-3', className)}
    >
      {tabs.map(({ id, label, icon: Icon, count }) => {
        const on = active === id;
        return (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onChange(id)}
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-lg text-[13px] font-medium font-sans whitespace-nowrap border transition-colors',
              inline ? 'h-8 px-3' : 'h-[34px] px-3.5',
              on
                ? 'border-accent bg-accent text-white'
                : 'border-outline-variant bg-card text-on-surface hover:bg-sunk',
            )}
          >
            {Icon && <Icon size={16} />}
            {label}
            {count !== undefined && <span className="font-mono opacity-80">{count}</span>}
          </button>
        );
      })}
    </div>
  );
}

export function TenantRequiredNotice({ message }: { message: string }) {
  return <div className="text-center py-16 text-warn-fg font-body-sm">{message}</div>;
}

/** Table inside a SurfaceCard/vf-card: scrolls horizontally, 13px text. */
export function PageTable({ children, minWidth }: { children: ReactNode; minWidth?: number }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-[13px] border-collapse" style={minWidth ? { minWidth } : undefined}>
        {children}
      </table>
    </div>
  );
}

export function PageTableHead({ children }: { children: ReactNode }) {
  return (
    <thead className="bg-sunk">
      <tr>{children}</tr>
    </thead>
  );
}

export function PageTableTh({ children, className }: { children?: ReactNode; className?: string }) {
  return (
    <th
      scope="col"
      className={clsx(
        'text-left px-3 py-2.5 first:pl-5 last:pr-5 font-sans text-[13px] font-medium normal-case tracking-normal text-on-surface-variant whitespace-nowrap',
        className,
      )}
    >
      {children}
    </th>
  );
}

export function PageTableBody({ children }: { children: ReactNode }) {
  return <tbody className="divide-y divide-line3 text-on-surface">{children}</tbody>;
}

export function PageTableRow({ children, className }: { children: ReactNode; className?: string }) {
  return <tr className={clsx('table-row-hover', className)}>{children}</tr>;
}

export function PageTableTd({ children, className, mono, ...rest }: TdHTMLAttributes<HTMLTableCellElement> & { mono?: boolean }) {
  return (
    <td className={clsx('px-3 py-3 first:pl-5 last:pr-5 text-sm', mono && 'font-mono text-[13px]', className)} {...rest}>
      {children}
    </td>
  );
}

interface TableToolbarProps {
  /** Left side, usually an inline TabBar. */
  tabs?: ReactNode;
  /** Right side: filter input, buttons. */
  children?: ReactNode;
}

/** Card toolbar above a table: tabs left, filter right, bottom divider. */
export function TableToolbar({ tabs, children }: TableToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-3 px-5 py-3.5 border-b border-card-border">
      {tabs}
      <span className="flex-grow" />
      {children}
    </div>
  );
}

/** 32px filter box with search icon, for TableToolbar. */
export function TableFilter({ className, ...props }: Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> & { className?: string }) {
  return (
    <label className="flex items-center gap-2 h-8 px-2.5 border border-outline-variant rounded-lg bg-card text-on-surface-variant">
      <Search size={14} aria-hidden />
      <input
        type="text"
        className={clsx('border-0 outline-none bg-transparent text-[13px] w-[170px] text-on-surface placeholder:text-on-surface-variant', className)}
        {...props}
      />
    </label>
  );
}

/** Pagination footer: muted 13px text left, actions (toolbarBtnClass buttons) right. */
export function TableFooter({ children, actions }: { children?: ReactNode; actions?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center gap-3 px-5 py-3 border-t border-card-border text-on-surface-variant text-[13px]">
      <span className="flex-grow">{children}</span>
      {actions}
    </div>
  );
}

export interface Pager {
  page: number;
  size: number;
  total: number;
  from: number;
  to: number;
  setPage: (p: number) => void;
}

/** Client-side pagination (lists are fetched whole). Page resets when `resetKey` changes (filter/tab). */
export function usePagination<T>(items: T[], resetKey = '', size = 10): { rows: T[]; pager: Pager } {
  const [state, setState] = useState({ page: 0, key: resetKey });
  const last = Math.max(0, Math.ceil(items.length / size) - 1);
  const page = Math.min(state.key === resetKey ? state.page : 0, last);
  const start = page * size;
  const rows = items.slice(start, start + size);
  return {
    rows,
    pager: {
      page, size, total: items.length,
      from: rows.length ? start + 1 : 0,
      to: start + rows.length,
      setPage: (p) => setState({ page: p, key: resetKey }),
    },
  };
}

/** Footer "Showing a–b of N" + prev/next; buttons hidden when everything fits one page. */
export function TablePager({ pager }: { pager: Pager }) {
  const { t } = useI18n();
  const paged = pager.total > pager.size;
  return (
    <TableFooter
      actions={
        paged && (
          <>
            <button type="button" disabled={pager.page === 0} onClick={() => pager.setPage(pager.page - 1)} className={toolbarBtnClass}>{t('table.prev')}</button>
            <button type="button" disabled={pager.to >= pager.total} onClick={() => pager.setPage(pager.page + 1)} className={toolbarBtnClass}>{t('table.next')}</button>
          </>
        )
      }
    >
      {paged
        ? t('table.range').replace('{from}', String(pager.from)).replace('{to}', String(pager.to)).replace('{total}', String(pager.total))
        : t('table.showing').replace('{n}', String(pager.total)).replace('{total}', String(pager.total))}
    </TableFooter>
  );
}

/** Full-width message row for loading / empty states inside a table body. */
export function TableMessageRow({ cols, children }: { cols: number; children: ReactNode }) {
  return (
    <tr>
      <td colSpan={cols} className="text-center py-12 text-on-surface-variant">{children}</td>
    </tr>
  );
}
