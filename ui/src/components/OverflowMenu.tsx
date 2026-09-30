import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from 'react';
import { MoreVertical } from 'lucide-react';
import clsx from 'clsx';
import { useClickOutside } from '../hooks/useClickOutside';
import { useI18n } from '../lib/i18n';

export type OverflowMenuItem = {
  id: string;
  label: string;
  icon?: ReactNode;
  onSelect: () => void;
  disabled?: boolean;
  danger?: boolean;
};

type OverflowMenuProps = {
  items: OverflowMenuItem[];
  /** Accessible name for the trigger (defaults to common.moreActions). */
  label?: string;
  align?: 'left' | 'right';
  className?: string;
};

export function OverflowMenu({ items, label, align = 'right', className }: OverflowMenuProps) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const menuId = useId();
  const close = useCallback(() => setOpen(false), []);

  useClickOutside(rootRef, open, close);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        close();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, close]);

  if (items.length === 0) return null;

  return (
    <div ref={rootRef} className={clsx('relative', className)}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-haspopup="menu"
        aria-controls={open ? menuId : undefined}
        title={label ?? t('common.moreActions')}
        aria-label={label ?? t('common.moreActions')}
        className="btn-icon-neutral focus-visible:ring-2 focus-visible:ring-primary"
      >
        <MoreVertical size={16} />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          className={clsx(
            'absolute z-50 top-full mt-1 min-w-[11rem] vf-card shadow-xl py-1',
            align === 'right' ? 'right-0' : 'left-0',
          )}
        >
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              role="menuitem"
              disabled={item.disabled}
              onClick={() => {
                if (item.disabled) return;
                close();
                item.onSelect();
              }}
              className={clsx(
                'flex w-full items-center gap-2 px-3 py-2 text-sm text-left transition-colors',
                item.disabled
                  ? 'opacity-40 cursor-not-allowed text-on-surface-variant'
                  : item.danger
                    ? 'text-error hover:bg-surface-variant'
                    : 'text-on-surface hover:bg-surface-variant',
              )}
            >
              {item.icon}
              <span className="truncate">{item.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
