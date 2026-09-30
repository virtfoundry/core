import { Outlet, useNavigate } from 'react-router-dom';
import { useState, useEffect, useCallback } from 'react';
import { Menu, X } from 'lucide-react';
import { VirtFoundryLogo } from './VirtFoundryLogo';
import { SidebarNav } from './SidebarNav';
import { SettingsMenu, UserMenu } from './HeaderMenus';
import { HeaderSearch, NotificationsMenu } from './HeaderToolbar';
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
import { roleBadgeLabel } from '../lib/vm-display';
import clsx from 'clsx';

export function Layout() {
  const [notifOpen, setNotifOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
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
  const { t } = useI18n();

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

  return (
    <div className="min-h-screen bg-background flex">
      <aside
        className={clsx(
          'hidden md:flex flex-col fixed left-0 top-0 h-full z-40 border-r border-outline-variant inner-glow',
          'bg-surface-container transition-[width] duration-300 ease-in-out',
          sidebarOpen ? 'w-sidebar-expanded' : 'w-sidebar-collapsed',
        )}
      >
        <div className="flex h-16 items-center border-b border-outline-variant">
          <div className={clsx('flex h-full min-w-0 items-center', sidebarOpen ? 'w-full justify-center px-5' : 'w-full justify-center px-2')}>
            <VirtFoundryLogo iconOnly={!sidebarOpen} height={56} />
          </div>
        </div>

        <nav className="flex-1 flex flex-col gap-1 px-2 py-4 overflow-y-auto">
          <SidebarNav collapsed={!sidebarOpen} isRoot={isRoot} />
        </nav>
      </aside>

      <div className={clsx('flex-1 flex flex-col min-h-screen w-full pt-16', sidebarWidth, 'transition-[margin] duration-300 ease-in-out')}>
        <header
          className={clsx(
            'fixed top-0 right-0 z-50 h-16 bg-surface border-b border-outline-variant inner-glow',
            'flex items-center justify-between px-4 md:px-6 gap-4 left-0 transition-[left] duration-300 ease-in-out',
            sidebarOpen ? 'md:left-sidebar-expanded' : 'md:left-sidebar-collapsed',
          )}
        >
          <div className="flex items-center gap-3 min-w-0">
            <button
              type="button"
              onClick={() => dispatch(setSidebarOpen(!sidebarOpen))}
              className="hidden md:flex p-2 rounded-full text-on-surface-variant hover:bg-surface-container transition-colors focus-visible:ring-2 focus-visible:ring-primary"
            >
              {sidebarOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
            <button type="button" className="md:hidden p-2 rounded-full text-on-surface-variant" onClick={() => dispatch(setSidebarOpen(!sidebarOpen))}>
              <Menu size={20} />
            </button>
          </div>

          <div className="hidden md:flex flex-1 max-w-xs ml-2">
            <button
              type="button"
              onClick={openCmd}
              className="w-full text-left h-10 pl-3 pr-3 bg-surface-container-high border border-outline-variant rounded-lg text-body-sm text-on-surface-variant hover:border-primary-container transition-colors flex items-center justify-between gap-2"
            >
              <span className="truncate">{t('cmdk.placeholder')}</span>
              <kbd className="text-[10px] font-mono border border-outline-variant rounded px-1.5 py-0.5 shrink-0">⌘K</kbd>
            </button>
            <div className="sr-only" aria-hidden>
              <HeaderSearch />
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {user?.role && (
              <span
                className="hidden sm:inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-mono border border-outline-variant bg-surface-container-high text-on-surface"
                title={t('header.role')}
              >
                {roleBadgeLabel(user.role)}
              </span>
            )}
            {isRoot && (
              <TenantSwitcher
                tenants={tenants}
                selectedTenantId={selectedTenant}
                defaultTenantId={defaultTenantId}
                onChange={handleTenantChange}
                loading={tenantsLoading}
              />
            )}
            <NotificationsMenu
              open={notifOpen}
              onToggle={() => {
                setSettingsOpen(false);
                setUserOpen(false);
                setNotifOpen((v) => !v);
              }}
              onClose={() => setNotifOpen(false)}
            />
            <SettingsMenu
              open={settingsOpen}
              onToggle={() => {
                setNotifOpen(false);
                setUserOpen(false);
                setSettingsOpen((v) => !v);
              }}
              onClose={() => setSettingsOpen(false)}
            />
            <UserMenu
              open={userOpen}
              onToggle={() => {
                setNotifOpen(false);
                setSettingsOpen(false);
                setUserOpen((v) => !v);
              }}
              onClose={() => setUserOpen(false)}
              onLogout={handleLogout}
            />
          </div>
        </header>

        {impersonating && (
          <div className="bg-error text-white px-4 md:px-6 py-2.5 text-sm flex flex-wrap items-center justify-between gap-2 shadow-md">
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

        <main className="flex-1 p-4 md:p-6 lg:p-margin-desktop max-w-content mx-auto w-full">
          <Outlet />
        </main>
      </div>

      {sidebarOpen && (
        <div className="md:hidden fixed inset-0 z-50 flex">
          <button type="button" className="absolute inset-0 bg-black/40" aria-label="Close menu" onClick={closeMobileSidebar} />
          <aside className="relative w-sidebar-expanded max-w-[85vw] h-full bg-surface-container border-r border-outline-variant inner-glow flex flex-col">
            <div className="p-4 border-b border-outline-variant flex justify-between items-center">
              <VirtFoundryLogo fullWidth />
              <button type="button" onClick={closeMobileSidebar} className="p-2 rounded-lg hover:bg-surface-container-high">
                <X size={20} />
              </button>
            </div>
            <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
              <SidebarNav collapsed={false} isRoot={isRoot} onNavigate={closeMobileSidebar} />
            </nav>
          </aside>
        </div>
      )}

      <CommandPalette open={cmdOpen} onClose={() => setCmdOpen(false)} />
    </div>
  );
}
