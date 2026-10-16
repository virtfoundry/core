import clsx from 'clsx';
import { useI18n, type TranslationKey } from '../lib/i18n';

type Status =
  | 'running'
  | 'stopped'
  | 'starting'
  | 'stopping'
  | 'creating'
  | 'error'
  | 'enabled'
  | 'disabled'
  | 'active'
  | 'inactive';

const statusStyles: Record<Status, string> = {
  running: 'bg-success-muted text-success',
  stopped: 'bg-surface-container text-on-surface-variant',
  starting: 'bg-primary-container/20 text-primary-fixed-dim',
  stopping: 'bg-warning-muted text-warning',
  creating: 'bg-primary-container/20 text-primary-fixed-dim',
  error: 'bg-error-container/30 text-error font-medium',
  enabled: 'bg-success-muted text-success',
  disabled: 'bg-surface-container text-on-surface-variant',
  active: 'bg-success-muted text-success',
  inactive: 'bg-surface-container text-on-surface-variant',
};

interface StatusBadgeProps {
  status: string;
  className?: string;
  pulse?: boolean;
}

export function StatusBadge({ status, className, pulse = true }: StatusBadgeProps) {
  const { t } = useI18n();
  const normalizedStatus = status.toLowerCase() as Status;
  const style = statusStyles[normalizedStatus] || statusStyles.inactive;
  const label = normalizedStatus in statusStyles ? t(`status.${normalizedStatus}` as TranslationKey) : status;
  const showPulse = pulse && (
    normalizedStatus === 'running'
    || normalizedStatus === 'starting'
    || normalizedStatus === 'creating'
    || normalizedStatus === 'active'
    || normalizedStatus === 'error'
  );

  return (
    <span
      role="status"
      aria-label={label}
      className={clsx(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-sans font-medium',
        style,
        className,
      )}
    >
      <span className={clsx('w-1.5 h-1.5 rounded-full bg-current', showPulse && 'animate-vf-pulse')} aria-hidden />
      {label}
    </span>
  );
}
