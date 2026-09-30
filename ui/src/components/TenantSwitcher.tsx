import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { Building2, Check, ChevronDown } from 'lucide-react';
import clsx from 'clsx';
import type { Tenant } from '../lib/platform-api';
import { useClickOutside } from '../hooks/useClickOutside';
import { useI18n } from '../lib/i18n';

type TenantSwitcherProps = {
  tenants: Tenant[];
  selectedTenantId: string;
  /** Root user's own tenant (exit impersonation target). */
  defaultTenantId?: string;
  onChange: (tenantId: string) => void;
  loading?: boolean;
  className?: string;
};

export function TenantSwitcher({
  tenants,
  selectedTenantId,
  defaultTenantId = '',
  onChange,
  loading = false,
  className,
}: TenantSwitcherProps) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const listId = useId();
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

  const selected = tenants.find((tn) => tn.id === selectedTenantId);
  const label = selected?.name
    || (selectedTenantId ? selectedTenantId : t('nav.selectTenant'));
  const impersonating =
    !!selectedTenantId && !!defaultTenantId && selectedTenantId !== defaultTenantId;

  return (
    <div ref={rootRef} className={clsx('relative', className)}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-controls={open ? listId : undefined}
        aria-label={t('header.tenant')}
        title={t('header.tenantHint')}
        disabled={loading}
        className={clsx(
          'flex items-center gap-2 h-9 max-w-[220px] px-2.5 rounded-lg text-sm border transition-colors',
          'bg-surface-container-high text-on-surface focus-visible:ring-2 focus-visible:ring-primary',
          impersonating
            ? 'border-error ring-2 ring-error/50'
            : 'border-outline-variant hover:border-primary-container/50',
          loading && 'opacity-60 cursor-wait',
        )}
      >
        <Building2 size={16} className="text-on-surface-variant shrink-0" aria-hidden />
        <span className="truncate min-w-0 flex-1 text-left">{loading ? t('common.loading') : label}</span>
        <ChevronDown size={14} className="text-on-surface-variant shrink-0" aria-hidden />
      </button>

      {open && (
        <div
          id={listId}
          role="listbox"
          aria-label={t('header.tenant')}
          className="absolute z-50 right-0 top-full mt-2 w-64 max-h-72 overflow-y-auto vf-card shadow-xl py-1"
        >
          <div className="px-3 py-2 border-b border-card-border">
            <p className="text-xs font-label text-on-surface-variant">{t('header.tenant')}</p>
            <p className="text-[11px] text-on-surface-variant mt-0.5">{t('header.tenantHint')}</p>
          </div>

          <button
            type="button"
            role="option"
            aria-selected={!selectedTenantId}
            onClick={() => {
              onChange('');
              close();
            }}
            className={clsx(
              'flex w-full items-center gap-2 px-3 py-2 text-sm text-left hover:bg-surface-variant transition-colors',
              !selectedTenantId ? 'text-primary' : 'text-on-surface-variant',
            )}
          >
            <span className="flex-1 truncate">{t('nav.selectTenant')}</span>
            {!selectedTenantId && <Check size={14} className="shrink-0" aria-hidden />}
          </button>

          {tenants.length === 0 && !loading ? (
            <p className="px-3 py-3 text-sm text-on-surface-variant">{t('header.tenantEmpty')}</p>
          ) : (
            tenants.map((tn) => {
              const active = tn.id === selectedTenantId;
              return (
                <button
                  key={tn.id}
                  type="button"
                  role="option"
                  aria-selected={active}
                  onClick={() => {
                    onChange(tn.id);
                    close();
                  }}
                  className={clsx(
                    'flex w-full items-center gap-2 px-3 py-2 text-sm text-left hover:bg-surface-variant transition-colors',
                    active ? 'text-primary bg-primary-container/10' : 'text-on-surface',
                  )}
                >
                  <span className="flex-1 min-w-0">
                    <span className="block truncate font-medium">{tn.name}</span>
                    <span className="block truncate text-[11px] text-on-surface-variant font-data-mono">
                      {tn.slug}
                    </span>
                  </span>
                  {active && <Check size={14} className="shrink-0" aria-hidden />}
                </button>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}
