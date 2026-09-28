import clsx from 'clsx';

/** Marks UI-only / stub surfaces in the homelab preview. */
export function ComingSoonBadge({ className, label = 'preview / coming soon' }: { className?: string; label?: string }) {
  return (
    <span
      className={clsx(
        'inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono uppercase tracking-wide',
        'border border-outline-variant bg-surface-container-high text-on-surface-variant',
        className,
      )}
    >
      {label}
    </span>
  );
}
