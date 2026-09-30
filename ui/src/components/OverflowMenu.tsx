import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { MoreVertical } from 'lucide-react';
import clsx from 'clsx';
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

type MenuCoords = {
  top: number;
  left: number;
  minWidth: number;
};

export function OverflowMenu({ items, label, align = 'right', className }: OverflowMenuProps) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [coords, setCoords] = useState<MenuCoords | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const menuId = useId();
  const close = useCallback(() => setOpen(false), []);

  const placeMenu = useCallback(() => {
    const trigger = triggerRef.current;
    const menu = menuRef.current;
    if (!trigger || !menu) return;

    const rect = trigger.getBoundingClientRect();
    const menuRect = menu.getBoundingClientRect();
    const gap = 4;
    const pad = 8;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    let top = rect.bottom + gap;
    if (top + menuRect.height > vh - pad && rect.top - gap - menuRect.height >= pad) {
      top = rect.top - gap - menuRect.height;
    } else {
      top = Math.min(top, Math.max(pad, vh - pad - menuRect.height));
    }

    let left = align === 'right' ? rect.right - menuRect.width : rect.left;
    if (left < pad) left = pad;
    if (left + menuRect.width > vw - pad) left = Math.max(pad, vw - pad - menuRect.width);

    setCoords({ top, left, minWidth: Math.max(176, rect.width) });
  }, [align]);

  useLayoutEffect(() => {
    if (!open) {
      setCoords(null);
      return;
    }
    placeMenu();
  }, [open, placeMenu, items.length]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        close();
      }
    };
    const onReposition = () => placeMenu();
    document.addEventListener('keydown', onKey);
    window.addEventListener('resize', onReposition);
    window.addEventListener('scroll', onReposition, true);
    return () => {
      document.removeEventListener('keydown', onKey);
      window.removeEventListener('resize', onReposition);
      window.removeEventListener('scroll', onReposition, true);
    };
  }, [open, close, placeMenu]);

  // Click-outside must also see the portaled menu.
  useEffect(() => {
    if (!open) return;
    const onPointer = (e: MouseEvent) => {
      const target = e.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if (menuRef.current?.contains(target)) return;
      close();
    };
    document.addEventListener('mousedown', onPointer);
    return () => document.removeEventListener('mousedown', onPointer);
  }, [open, close]);

  if (items.length === 0) return null;

  const menu = open
    ? createPortal(
        <div
          ref={menuRef}
          id={menuId}
          role="menu"
          style={
            coords
              ? { position: 'fixed', top: coords.top, left: coords.left, minWidth: coords.minWidth }
              : { position: 'fixed', top: 0, left: 0, visibility: 'hidden' }
          }
          className="z-[200] vf-card shadow-xl py-1"
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
        </div>,
        document.body,
      )
    : null;

  return (
    <div ref={rootRef} className={clsx('relative', className)}>
      <button
        ref={triggerRef}
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
      {menu}
    </div>
  );
}
