import { Outlet, useNavigate } from 'react-router-dom';
import { useState, useEffect, useCallback } from 'react';
import { Menu, Moon, Search, Sun, X } from 'lucide-react';
import { VirtFoundryLogo } from './VirtFoundryLogo';
import { SidebarNav } from './SidebarNav';
import { UserMenu } from './HeaderMenus';
import { NotificationsMenu } from './HeaderToolbar';
import { TenantSwitcher } from './TenantSwitcher';
import { CommandPalette, useCommandPaletteHotkey } from './CommandPalette';
import { authService } from '../lib/auth';
import { listTenants } from '../lib/platform-api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRealtimeEvents } from '../hooks/useRealtimeEvents';
import { queryKeys, isPlatformQueryKey } from '../lib/query-keys';
import { useI18n } from '../lib/i18n';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { selectIsRoot, selectUser } from '../store/authSlice';
import { selectSidebarOpen, selectTenantId, setSidebarOpen, setTenantId } from '../store/uiSlice';
import { useTheme } from '../lib/theme';
import clsx from 'clsx';

export function Layout() {
  const [notifOpen, setNotifOpen] = useState(false);
  const [userOpen, setUserOpen] = useState(false);
  const [cmdOpen, setCmdOpen] = useState(false);
  const isRoot = useAppSelector(selectIsRoot);
  const user = useAppSelector(selectUser);
  const sidebarOpen = useAppSelector(selectSidebarOpen);
  const selectedTenant = useAppSelector(selectTenantId) ?? '';
  const dispatch = useAppDispatch();
  const defaultTenantId = useAppSelector((s) => s.auth.user?.tenant_id) || '';
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t, locale, setLocale } = useI18n();
  const { isDark, toggleTheme } = useTheme();

  const openCmd = useCallback(() => setCmdOpen(true), []);
  useCommandPaletteHotkey(openCmd);

  useEffect(() => {
    if (!isRoot || !defaultTenantId || selectedTenant) return;
    dispatch(setTenantId(defaultTenantId));
  }, [isRoot, defaultTenantId, selectedTenant, dispatch]);

  useEffect(() => {
    if (window.matchMedia('(max-width: 767px)').matches) {
      dispatch(setSidebarOpen(false));
    }
  }, [dispatch]);

  useRealtimeEvents();

  const { data: tenantsData, isLoading: tenantsLoading } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: isRoot,
  });
  const tenants = tenantsData?.tenants || [];
  const impersonating = isRoot && selectedTenant !== '' && selectedTenant !== defaultTenantId;
  const sidebarWidth = sidebarOpen ? 'md:ml-sidebar-expanded' : 'md:ml-sidebar-collapsed';
  const tenantChipLabel = user?.tenant_id || '';
  const impersonatedName = tenants.find((tn) => tn.id === selectedTenant)?.name || selectedTenant;

  const handleTenantChange = (tenantId: string) => {
    dispatch(setTenantId(tenantId || null));
    queryClient.invalidateQueries({
      predicate: (q) => isPlatformQueryKey(q.queryKey),
    });
  };

  const handleLogout = () => {
    authService.logout();
    navigate('/login');
  };

  const closeMobileSidebar = () => dispatch(setSidebarOpen(false));

  const iconBtn =
    'w-9 h-9 flex items-center justify-center border border-outline-variant rounded-lg bg-card text-on-surface hover:bg-sunk transition-colors focus-visible:ring-2 focus-visible:ring-accent';

  const userChip = (compact: boolean, onNav?: () => void) => (
    <UserMenu
      variant="chip"
      compact={compact}
      open={userOpen}
      onToggle={() => {
        setNotifOpen(false);
        setUserOpen((v) => !v);
      }}
      onClose={() => setUserOpen(false)}
      onLogout={() => {
        onNav?.();
        handleLogout();
      }}
    />
  );

  return (
    <div className="min-h-screen bg-background flex">
      <aside
        className={clsx(
          'hidden md:flex flex-col gap-[22px] fixed left-0 top-0 h-full z-40 border-r border-side-line',
          'bg-side-bg py-5 px-3 transition-[width] duration-300 ease-in-out',
          sidebarOpen ? 'w-sidebar-expanded' : 'w-sidebar-collapsed',
        )}
      >
        <div className={clsx('flex items-center', sidebarOpen ? 'px-2' : 'justify-center')}>
          {sidebarOpen ? (
            <div className="w-[196px]"><VirtFoundryLogo fullWidth /></div>
          ) : (
            <VirtFoundryLogo iconOnly height={40} />
          )}
        </div>

        <nav aria-label={t('topbar.menu')} className="flex-1 overflow-y-auto">
          <SidebarNav collapsed={!sidebarOpen} isRoot={isRoot} />
        </nav>

        {userChip(!sidebarOpen)}
      </aside>

      <div className={clsx('flex-1 flex flex-col min-h-screen min-w-0', sidebarWidth, 'transition-[margin] duration-300 ease-in-out')}>
        <header className="sticky top-0 z-40 flex flex-wrap items-center gap-3 min-h-16 box-border px-4 md:px-8 py-3 bg-card border-b border-card-border">
          <button
            type="button"
            onClick={() => dispatch(setSidebarOpen(!sidebarOpen))}
            aria-label={sidebarOpen ? t('topbar.collapse') : t('topbar.expand')}
            className={iconBtn}
          >
            {sidebarOpen ? <X size={16} className="hidden md:block" /> : <Menu size={16} className="hidden md:block" />}
            <Menu size={16} className="md:hidden" />
          </button>

          {isRoot ? (
            <TenantSwitcher
              tenants={tenants}
              selectedTenantId={selectedTenant}
              defaultTenantId={defaultTenantId}
              onChange={handleTenantChange}
              loading={tenantsLoading}
            />
          ) : (
            tenantChipLabel && (
              <span
                className="flex items-center gap-2 h-9 max-w-[220px] px-3 rounded-lg border border-outline-variant bg-card text-sm"
                title={t('header.tenantHint')}
              >
                <span className="w-2 h-2 rounded-full bg-ok-fg shrink-0" aria-hidden />
                <span className="truncate font-mono text-[13px]">{tenantChipLabel}</span>
              </span>
            )
          )}

          <button
            type="button"
            onClick={openCmd}
            className="order-last basis-full md:order-none md:flex-1 md:basis-[260px] md:max-w-[460px] h-9 px-3 flex items-center gap-2.5 box-border border border-outline-variant rounded-lg bg-sunk text-on-surface-variant text-sm text-left hover:border-accent/50 transition-colors focus-visible:ring-2 focus-visible:ring-accent"
          >
            <Search size={16} strokeWidth={2} aria-hidden />
            <span className="flex-grow truncate">{t('topbar.search')}</span>
            <kbd className="font-mono text-[11px] border border-outline-variant rounded px-1.5 py-px bg-card shrink-0">⌘K</kbd>
          </button>

          <span className="flex-grow" />

          <button
            type="button"
            onClick={() => setLocale(locale === 'pt' ? 'en' : 'pt')}
            aria-label={t('topbar.langAria')}
            className={clsx(iconBtn, 'w-auto px-3 text-[13px] font-semibold font-mono')}
          >
            {locale.toUpperCase()}
          </button>
          <button type="button" onClick={toggleTheme} aria-label={t('topbar.themeAria')} className={iconBtn}>
            {isDark ? <Sun size={16} strokeWidth={1.8} /> : <Moon size={16} strokeWidth={1.8} />}
          </button>
          <NotificationsMenu
            open={notifOpen}
            onToggle={() => {
              setUserOpen(false);
              setNotifOpen((v) => !v);
            }}
            onClose={() => setNotifOpen(false)}
          />
        </header>

        {impersonating && (
          <div className="bg-bad-fg text-white px-4 md:px-8 py-2.5 text-sm flex flex-wrap items-center justify-between gap-2">
            <span className="font-medium">
              {t('impersonate.banner')}: <strong className="font-data-mono">{impersonatedName}</strong>
            </span>
            <button
              type="button"
              className="text-sm px-3 py-1 rounded-lg border border-white/40 hover:bg-white/10"
              onClick={() => handleTenantChange(defaultTenantId || '')}
            >
              {t('impersonate.exit')}
            </button>
          </div>
        )}

        <main className="flex-1 px-4 pt-7 pb-10 md:px-8 max-w-content mx-auto w-full">
          <Outlet />
        </main>
      </div>

      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-50 flex">
          <button type="button" className="absolute inset-0 bg-black/40" aria-label="Close menu" onClick={closeMobileSidebar} />
          <aside className="relative w-sidebar-expanded max-w-[85vw] h-full bg-side-bg border-r border-side-line flex flex-col gap-[22px] py-5 px-3">
            <div className="flex items-center justify-between px-2">
              <div className="w-[170px]"><VirtFoundryLogo fullWidth /></div>
              <button type="button" onClick={closeMobileSidebar} aria-label="Close menu" className="p-2 rounded-lg hover:bg-side-hover text-side-text">
                <X size={20} />
              </button>
            </div>
            <nav aria-label={t('topbar.menu')} className="flex-1 overflow-y-auto">
              <SidebarNav collapsed={false} isRoot={isRoot} onNavigate={closeMobileSidebar} />
            </nav>
            {userChip(false, closeMobileSidebar)}
          </aside>
        </div>
      )}

      <CommandPalette open={cmdOpen} onClose={() => setCmdOpen(false)} />
    </div>
  );
}
