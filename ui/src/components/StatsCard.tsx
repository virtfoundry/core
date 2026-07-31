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

export function StatsCard({ title, value, subtitle, icon: Icon, trend, variant = 'default' }: StatsCardProps) {
  const variantStyles = {
    default: 'bg-white dark:bg-gray-800',
    success: 'bg-green-50 dark:bg-green-900/20',
    warning: 'bg-yellow-50 dark:bg-yellow-900/20',
    danger: 'bg-red-50 dark:bg-red-900/20',
  };

  const iconStyles = {
    default: 'text-primary-500',
    success: 'text-green-500',
    warning: 'text-yellow-500',
    danger: 'text-red-500',
  };

  return (
    <div className={clsx('rounded-xl p-6 border border-gray-200 dark:border-gray-700', variantStyles[variant])}>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</p>
          <p className="mt-2 text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
          {subtitle && (
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
          )}
          {trend && (
            <p className={clsx('mt-2 text-sm font-medium', trend.positive ? 'text-green-600' : 'text-red-600')}>
              {trend.positive ? '+' : ''}{trend.value}% vs último mês
            </p>
          )}
        </div>
        <div className={clsx('p-3 rounded-lg bg-white dark:bg-gray-800 shadow-sm', iconStyles[variant])}>
          <Icon size={24} />
        </div>
      </div>
    </div>
  );
}
