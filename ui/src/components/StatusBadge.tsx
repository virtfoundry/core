import clsx from 'clsx';

type Status = 'running' | 'stopped' | 'starting' | 'stopping' | 'error' | 'enabled' | 'disabled' | 'active' | 'inactive';

const statusStyles: Record<Status, string> = {
  running: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  stopped: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
  starting: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  stopping: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  error: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  enabled: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  disabled: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
  active: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  inactive: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
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
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const normalizedStatus = status.toLowerCase() as Status;
  const style = statusStyles[normalizedStatus] || statusStyles.inactive;
  const label = statusLabels[normalizedStatus] || status;

  return (
    <span className={clsx('inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium', style, className)}>
      {label}
    </span>
  );
}
