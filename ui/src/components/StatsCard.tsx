import { LucideIcon } from 'lucide-react';
import clsx from 'clsx';

interface StatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: LucideIcon;
  trend?: { value: number; positive: boolean };
  variant?: 'default' | 'success' | 'warning' | 'danger';
}

const iconStyles = {
  default: 'bg-primary-container/10 text-primary',
  success: 'bg-success-muted text-success',
  warning: 'bg-warning-muted text-warning',
  danger: 'bg-error-container/30 text-error',
};

export function StatsCard({ title, value, subtitle, icon: Icon, trend, variant = 'default' }: StatsCardProps) {
  return (
    <div className="vf-card p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-body-sm text-on-surface-variant">{title}</p>
          <p className="mt-2 font-mono text-3xl font-medium tracking-tight text-on-surface">{value}</p>
          {subtitle && <p className="mt-1 text-body-sm text-on-surface-variant">{subtitle}</p>}
          {trend && (
            <p className={clsx('mt-2 text-sm font-medium', trend.positive ? 'text-success' : 'text-error')}>
              {trend.positive ? '+' : ''}{trend.value}%
            </p>
          )}
        </div>
        <div className={clsx('p-2.5 rounded-lg shrink-0', iconStyles[variant])}>
          <Icon size={20} />
        </div>
      </div>
    </div>
  );
}
