import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import clsx from 'clsx';
import { Bell, Search } from 'lucide-react';
import { globalSearch, listNotifications, type SearchHit } from '../lib/platform-api';
import { queryKeys } from '../lib/query-keys';
import { useI18n } from '../lib/i18n';
import { useAppSelector } from '../store/hooks';
import { selectTenantId } from '../store/uiSlice';

const typeLabels: Record<string, string> = {
  vm: 'VM',
  volume: 'Volume',
  vpc: 'VPC',
  network: 'Network',
  security_group: 'SG',
};

export function HeaderSearch() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const tenantId = useAppSelector(selectTenantId);
  const hasTenant = !!tenantId;

  const { data, isFetching } = useQuery({
    queryKey: queryKeys.search(query.trim()),
    queryFn: () => globalSearch(query.trim()),
    enabled: open && query.trim().length >= 2 && hasTenant,
    staleTime: 10_000,
  });

  const results = data?.results ?? [];

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, []);

  const pick = (hit: SearchHit) => {
    setOpen(false);
    setQuery('');
    navigate(hit.path);
  };

  return (
    <div ref={rootRef} className="relative w-full">
      <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant pointer-events-none" />
      <input
        type="search"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        placeholder={t('header.searchPlaceholder')}
        disabled={!hasTenant}
        className="w-full h-10 pl-10 pr-4 bg-surface-container-high border border-outline-variant rounded-lg text-body-sm text-on-surface focus:border-primary-container focus:ring-1 focus:ring-primary-container outline-none transition-colors disabled:opacity-50"
      />
      {open && query.trim().length >= 2 && hasTenant && (
        <div className="absolute z-50 top-full mt-2 w-full min-w-[280px] vf-card shadow-xl max-h-80 overflow-y-auto">
          {isFetching ? (
            <p className="px-4 py-3 text-sm text-on-surface-variant">{t('common.loading')}</p>
          ) : results.length === 0 ? (
            <p className="px-4 py-3 text-sm text-on-surface-variant">{t('header.searchEmpty')}</p>
          ) : (
            results.map((hit) => (
              <button
                key={`${hit.type}-${hit.id}`}
                type="button"
                onClick={() => pick(hit)}
                className="w-full text-left px-4 py-3 hover:bg-surface-variant transition-colors border-b border-card-border last:border-0"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="font-medium text-on-surface truncate">{hit.name}</span>
                  <span className="font-label text-[10px] text-primary-fixed-dim shrink-0">
                    {typeLabels[hit.type] ?? hit.type}
                  </span>
                </div>
                {hit.subtitle && (
                  <p className="text-xs text-on-surface-variant font-data-mono truncate mt-0.5">{hit.subtitle}</p>
                )}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}

interface NotificationsMenuProps {
  open: boolean;
  onToggle: () => void;
  onClose: () => void;
}

export function NotificationsMenu({ open, onToggle, onClose }: NotificationsMenuProps) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const rootRef = useRef<HTMLDivElement>(null);
  const tenantId = useAppSelector(selectTenantId);
  const hasTenant = !!tenantId;

  const { data, refetch } = useQuery({
    queryKey: queryKeys.notifications,
    queryFn: listNotifications,
    enabled: hasTenant,
    staleTime: 60_000,
  });

  const items = data?.notifications ?? [];

  useEffect(() => {
    if (open && hasTenant) refetch();
  }, [open, hasTenant, refetch]);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, [open, onClose]);

  const levelClass = (level: string) =>
    clsx(
      'w-2 h-2 rounded-full shrink-0 mt-1.5',
      level === 'error' && 'bg-error animate-vf-pulse',
      level === 'warning' && 'bg-warning',
      level !== 'error' && level !== 'warning' && 'bg-primary-container',
    );

  if (!hasTenant) return null;

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={onToggle}
        className="relative p-2 text-on-surface-variant hover:text-primary hover:bg-surface-variant rounded-full transition-colors hidden sm:block"
        title={t('header.notifications')}
      >
        <Bell size={20} />
        {items.length > 0 && (
          <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-error border border-surface" />
        )}
      </button>
      {open && (
        <div className="absolute z-50 right-0 top-full mt-2 w-80 vf-card shadow-xl max-h-96 overflow-y-auto">
          <div className="px-4 py-3 border-b border-card-border font-label text-on-surface-variant">
            {t('header.notifications')}
          </div>
          {items.length === 0 ? (
            <p className="px-4 py-6 text-sm text-on-surface-variant text-center">{t('header.notificationsEmpty')}</p>
          ) : (
            items.map((n) => (
              <button
                key={n.id}
                type="button"
                onClick={() => {
                  onClose();
                  if (n.path) navigate(n.path);
                }}
                className="w-full text-left px-4 py-3 hover:bg-surface-variant transition-colors border-b border-card-border last:border-0 flex gap-2"
              >
                <span className={levelClass(n.level)} />
                <div className="min-w-0">
                  <p className="font-medium text-sm text-on-surface truncate">{n.title}</p>
                  <p className="text-xs text-on-surface-variant line-clamp-2">{n.message}</p>
                </div>
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
