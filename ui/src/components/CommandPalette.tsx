import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Search, Command } from 'lucide-react';
import { globalSearch, type SearchHit } from '../lib/platform-api';
import { queryKeys } from '../lib/query-keys';
import { useAppSelector } from '../store/hooks';
import { selectTenantId } from '../store/uiSlice';
import { useI18n } from '../lib/i18n';
import { getRecentActions } from '../lib/preview-prefs';
import clsx from 'clsx';

const typeLabels: Record<string, string> = {
  vm: 'VM',
  volume: 'Volume',
  vpc: 'VPC',
  network: 'Network',
  security_group: 'SG',
  template: 'Template',
  tenant: 'Tenant',
};

type Props = {
  open: boolean;
  onClose: () => void;
};

export function CommandPalette({ open, onClose }: Props) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const tenantId = useAppSelector(selectTenantId);
  const hasTenant = !!tenantId;
  const recent = useMemo(() => getRecentActions(), [open]);

  const { data, isFetching } = useQuery({
    queryKey: queryKeys.search(query.trim()),
    queryFn: () => globalSearch(query.trim()),
    enabled: open && query.trim().length >= 2 && hasTenant,
    staleTime: 10_000,
  });

  const results = data?.results ?? [];

  useEffect(() => {
    if (!open) {
      setQuery('');
      return;
    }
    const tmr = window.setTimeout(() => inputRef.current?.focus(), 30);
    return () => window.clearTimeout(tmr);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  const pick = (path: string) => {
    onClose();
    navigate(path);
  };

  const pickHit = (hit: SearchHit) => pick(hit.path);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[80] flex items-start justify-center pt-[12vh] px-4">
      <button type="button" className="absolute inset-0 bg-black/50" aria-label="Close" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-label={t('cmdk.title')}
        className="relative w-full max-w-xl vf-card shadow-2xl overflow-hidden"
      >
        <div className="flex items-center gap-3 px-4 border-b border-outline-variant">
          <Search size={18} className="text-on-surface-variant shrink-0" />
          <input
            ref={inputRef}
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('cmdk.placeholder')}
            className="flex-1 h-12 bg-transparent outline-none text-on-surface text-sm"
          />
          <kbd className="hidden sm:inline-flex items-center gap-1 text-[10px] font-mono text-on-surface-variant border border-outline-variant rounded px-1.5 py-0.5">
            <Command size={10} /> K
          </kbd>
        </div>

        <div className="max-h-80 overflow-y-auto py-2">
          {!hasTenant && (
            <p className="px-4 py-3 text-sm text-on-surface-variant">{t('common.selectTenant')}</p>
          )}

          {hasTenant && query.trim().length < 2 && (
            <>
              <p className="px-4 py-1 text-[10px] font-label uppercase text-on-surface-variant">{t('cmdk.recent')}</p>
              {recent.length === 0 ? (
                <p className="px-4 py-3 text-sm text-on-surface-variant">{t('cmdk.hint')}</p>
              ) : (
                recent.map((r) => (
                  <button
                    key={r.id}
                    type="button"
                    onClick={() => pick(r.path)}
                    className="w-full text-left px-4 py-2.5 hover:bg-surface-variant text-sm text-on-surface"
                  >
                    {r.label}
                  </button>
                ))
              )}
              <p className="px-4 py-1 mt-2 text-[10px] font-label uppercase text-on-surface-variant">{t('cmdk.shortcuts')}</p>
              {[
                { label: t('nav.vms'), path: '/vms' },
                { label: t('nav.templates'), path: '/templates' },
                { label: t('nav.volumes'), path: '/volumes' },
                { label: t('nav.dashboard'), path: '/dashboard' },
              ].map((s) => (
                <button
                  key={s.path}
                  type="button"
                  onClick={() => pick(s.path)}
                  className="w-full text-left px-4 py-2.5 hover:bg-surface-variant text-sm text-on-surface"
                >
                  {s.label}
                </button>
              ))}
            </>
          )}

          {hasTenant && query.trim().length >= 2 && (
            <>
              {isFetching && (
                <p className="px-4 py-3 text-sm text-on-surface-variant">{t('common.loading')}</p>
              )}
              {!isFetching && results.length === 0 && (
                <p className="px-4 py-3 text-sm text-on-surface-variant">{t('header.searchEmpty')}</p>
              )}
              {results.map((hit) => (
                <button
                  key={`${hit.type}-${hit.id}`}
                  type="button"
                  onClick={() => pickHit(hit)}
                  className={clsx(
                    'w-full text-left px-4 py-2.5 hover:bg-surface-variant transition-colors',
                    'border-b border-outline-variant/40 last:border-0',
                  )}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium text-on-surface truncate text-sm">{hit.name}</span>
                    <span className="font-label text-[10px] text-primary-fixed-dim shrink-0">
                      {typeLabels[hit.type] ?? hit.type}
                    </span>
                  </div>
                  {hit.subtitle && (
                    <p className="text-xs text-on-surface-variant font-data-mono truncate mt-0.5">{hit.subtitle}</p>
                  )}
                </button>
              ))}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

/** Global ⌘K / Ctrl+K listener — mount once in Layout. */
export function useCommandPaletteHotkey(onOpen: () => void) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        onOpen();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onOpen]);
}
