import clsx from 'clsx';

type Status = 'running' | 'stopped' | 'starting' | 'stopping' | 'error' | 'enabled' | 'disabled' | 'active' | 'inactive';

const statusStyles: Record<Status, string> = {
  running: 'bg-success-muted text-success border border-success/20',
  stopped: 'bg-surface-container border border-outline-variant text-on-surface-variant',
  starting: 'bg-primary-container/20 text-primary-fixed-dim border border-primary-container/30',
  stopping: 'bg-warning-muted text-warning border border-warning/20',
  error: 'bg-error-container/20 text-error border border-error/30',
  enabled: 'bg-success-muted text-success border border-success/20',
  disabled: 'bg-surface-container border border-outline-variant text-on-surface-variant',
  active: 'bg-success-muted text-success border border-success/20',
  inactive: 'bg-surface-container border border-outline-variant text-on-surface-variant',
};

const statusLabels: Record<Status, string> = {
  running: 'Running',
  stopped: 'Stopped',
  starting: 'Starting',
  stopping: 'Stopping',
  error: 'Error',
  enabled: 'Enabled',
  disabled: 'Disabled',
  active: 'Active',
  inactive: 'Inactive',
};

interface StatusBadgeProps {
  status: string;
  className?: string;
  pulse?: boolean;
}

export function StatusBadge({ status, className, pulse = true }: StatusBadgeProps) {
  const normalizedStatus = status.toLowerCase() as Status;
  const style = statusStyles[normalizedStatus] || statusStyles.inactive;
  const label = statusLabels[normalizedStatus] || status;
  const showPulse = pulse && (normalizedStatus === 'running' || normalizedStatus === 'starting' || normalizedStatus === 'active' || normalizedStatus === 'error');

  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-label-sm font-mono',
        style,
        className,
      )}
    >
      {showPulse && <span className="w-1.5 h-1.5 rounded-full bg-current animate-vf-pulse" />}
      {label}
    </span>
  );
}
