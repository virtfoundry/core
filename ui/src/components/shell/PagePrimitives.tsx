import type { ReactNode, InputHTMLAttributes, ComponentType } from 'react';
import { Search } from 'lucide-react';
import clsx from 'clsx';

export const formInputClass =
  'w-full px-4 py-2 h-10 border border-outline-variant rounded-lg bg-surface-container-high text-on-surface text-body-sm focus:border-primary-container focus:ring-1 focus:ring-primary-container outline-none transition-colors';

export const formSelectClass = formInputClass;

export const formTextareaClass =
  'w-full px-4 py-2 border border-outline-variant rounded-lg bg-surface-container-high text-on-surface text-body-sm focus:border-primary-container focus:ring-1 focus:ring-primary-container outline-none transition-colors';

interface SearchFieldProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> {
  className?: string;
  containerClassName?: string;
}

export function SearchField({ className, containerClassName, ...props }: SearchFieldProps) {
  return (
    <div className={clsx('relative max-w-xl', containerClassName)}>
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none" size={20} />
      <input
        type="search"
        className={clsx(
          'w-full h-10 pl-10 pr-4 border border-outline-variant rounded-lg bg-surface-container-high text-body-sm text-on-surface focus:border-primary-container focus:ring-1 focus:ring-primary-container outline-none',
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
        'rounded-lg border px-4 py-3 text-sm inner-glow',
        variant === 'warning'
          ? 'border-warning/30 bg-warning-muted text-on-surface'
          : 'border-primary-container/30 bg-primary-container/10 text-on-surface',
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
}

export function EmptyState({ icon, title, hint }: EmptyStateProps) {
  return (
    <div className="col-span-full text-center py-12">
      <div className="mx-auto mb-4 text-on-surface-variant opacity-40">{icon}</div>
      <p className="text-on-surface-variant">{title}</p>
      {hint && <p className="text-sm text-on-surface-variant/70 mt-2 max-w-md mx-auto">{hint}</p>}
    </div>
  );
}

interface ResourceGridCardProps {
  children: ReactNode;
  className?: string;
}

export function ResourceGridCard({ children, className }: ResourceGridCardProps) {
  return (
    <div className={clsx('vf-card p-5 hover:border-primary-container/40 transition-colors', className)}>
      {children}
    </div>
  );
}

interface TabBarProps<T extends string> {
  tabs: { id: T; label: string; icon?: ComponentType<{ className?: string; size?: number }> }[];
  active: T;
  onChange: (id: T) => void;
}

export function TabBar<T extends string>({ tabs, active, onChange }: TabBarProps<T>) {
  return (
    <div className="flex gap-1 border-b border-outline-variant overflow-x-auto">
      {tabs.map(({ id, label, icon: Icon }) => (
        <button
          key={id}
          type="button"
          onClick={() => onChange(id)}
          className={clsx(
            'flex items-center gap-2 px-4 py-2.5 text-label-md font-mono whitespace-nowrap border-b-2 -mb-px transition-colors',
            active === id
              ? 'border-primary-container text-primary shadow-nav-active'
              : 'border-transparent text-on-surface-variant hover:text-primary hover:bg-surface-variant',
          )}
        >
          {Icon && <Icon size={16} />}
          {label}
        </button>
      ))}
    </div>
  );
}

export function TenantRequiredNotice({ message }: { message: string }) {
  return <div className="text-center py-16 text-warning font-body-sm">{message}</div>;
}

export function PageTable({ children }: { children: React.ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm border-collapse">{children}</table>
    </div>
  );
}

export function PageTableHead({ children }: { children: React.ReactNode }) {
  return (
    <thead className="bg-surface-container-high border-b border-card-border">
      <tr>{children}</tr>
    </thead>
  );
}

export function PageTableTh({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <th className={clsx('text-left px-4 py-3 font-label text-on-surface-variant', className)}>
      {children}
    </th>
  );
}

export function PageTableBody({ children }: { children: React.ReactNode }) {
  return <tbody className="divide-y divide-card-border text-body-sm text-on-surface">{children}</tbody>;
}

export function PageTableRow({ children, className }: { children: React.ReactNode; className?: string }) {
  return <tr className={clsx('table-row-hover', className)}>{children}</tr>;
}

export function PageTableTd({ children, className }: { children: React.ReactNode; className?: string }) {
  return <td className={clsx('px-4 py-3', className)}>{children}</td>;
}
