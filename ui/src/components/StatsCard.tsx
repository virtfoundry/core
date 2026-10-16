import type { LucideIcon } from 'lucide-react';
import clsx from 'clsx';

type Tone = 'default' | 'success' | 'warning' | 'danger';

interface StatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  /** Optional small icon at the top-right (the redesign omits it). */
  icon?: LucideIcon;
  trend?: { value: number; positive: boolean };
  variant?: Tone;
  /** Muted text after the value ("of 24", "/ 128"). */
  of?: string;
  /** Progress bar 0-100; hidden when undefined. Color follows `variant`. */
  progress?: number;
}

const iconStyles: Record<Tone, string> = {
  default: 'bg-accent-soft text-accent',
  success: 'bg-ok-bg text-ok-fg',
  warning: 'bg-warn-bg text-warn-fg',
  danger: 'bg-bad-bg text-bad-fg',
};

const barStyles: Record<Tone, string> = {
  default: 'bg-accent',
  success: 'bg-ok-fg',
  warning: 'bg-warn-fg',
  danger: 'bg-bad-fg',
};

export function StatsCard({ title, value, subtitle, icon: Icon, trend, variant = 'default', of, progress }: StatsCardProps) {
  return (
    <div className="vf-card px-5 py-[18px] flex flex-col gap-2.5">
      <div className="flex items-start justify-between gap-3">
        <p className="m-0 text-[13px] text-on-surface-variant">{title}</p>
        {Icon && (
          <span className={clsx('p-1.5 rounded-lg shrink-0', iconStyles[variant])}>
            <Icon size={16} strokeWidth={1.7} />
          </span>
        )}
      </div>
      <p className="m-0 flex items-baseline gap-2">
        <span className="font-sans tabular-nums text-[28px] leading-8 font-semibold tracking-[-0.02em] text-on-surface">{value}</span>
        {of && <span className="text-[13px] text-on-surface-variant">{of}</span>}
      </p>
      {subtitle && <p className="m-0 text-[13px] text-on-surface-variant">{subtitle}</p>}
      {trend && (
        <p className={clsx('m-0 text-sm font-medium', trend.positive ? 'text-ok-fg' : 'text-bad-fg')}>
          {trend.positive ? '+' : ''}{trend.value}%
        </p>
      )}
      {progress !== undefined && (
        <div className="h-1.5 rounded-full bg-track" role="progressbar" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100}>
          <div
            className={clsx('h-1.5 rounded-full transition-all duration-500', barStyles[variant])}
            style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
          />
        </div>
      )}
    </div>
  );
}
