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

export type StatusTone = 'ok' | 'warn' | 'bad' | 'off' | 'info';

export const toneClass: Record<StatusTone, string> = {
  ok: 'bg-ok-bg text-ok-fg',
  warn: 'bg-warn-bg text-warn-fg',
  bad: 'bg-bad-bg text-bad-fg',
  off: 'bg-off-bg text-off-fg',
  info: 'bg-accent-soft text-link',
};

const statusTone: Record<Status, StatusTone> = {
  running: 'ok',
  stopped: 'off',
  starting: 'info',
  stopping: 'warn',
  creating: 'info',
  error: 'bad',
  enabled: 'ok',
  disabled: 'off',
  active: 'ok',
  inactive: 'off',
};

interface StatusBadgeProps {
  status: string;
  className?: string;
  pulse?: boolean;
  /** Override tone for statuses outside the known set (e.g. Ready, Provisioning). */
  tone?: StatusTone;
  /** Override the translated label. */
  label?: string;
}

export function StatusBadge({ status, className, pulse = true, tone, label: labelOverride }: StatusBadgeProps) {
  const { t } = useI18n();
  const normalizedStatus = status.toLowerCase() as Status;
  const known = normalizedStatus in statusTone;
  const style = toneClass[tone ?? (known ? statusTone[normalizedStatus] : 'off')];
  const label = labelOverride ?? (known ? t(`status.${normalizedStatus}` as TranslationKey) : status);
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
        'inline-flex items-center gap-1.5 whitespace-nowrap px-2.5 py-[3px] rounded-full text-[13px] font-sans font-medium',
        style,
        className,
      )}
    >
      <span className={clsx('w-1.5 h-1.5 rounded-full bg-current', showPulse && 'animate-vf-pulse')} aria-hidden />
      {label}
    </span>
  );
}
