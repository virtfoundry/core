import { Outlet, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { Menu, X } from 'lucide-react';
import { VirtFoundryLogo } from './VirtFoundryLogo';
import { SidebarNav } from './SidebarNav';
import { SettingsMenu, UserMenu } from './HeaderMenus';
import { HeaderSearch, NotificationsMenu } from './HeaderToolbar';
import { authService } from '../lib/auth';
import { listTenants } from '../lib/platform-api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRealtimeEvents } from '../hooks/useRealtimeEvents';
import { queryKeys, isPlatformQueryKey } from '../lib/query-keys';
import { useI18n } from '../lib/i18n';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { selectIsRoot } from '../store/authSlice';
import { selectSidebarOpen, selectTenantId, setSidebarOpen, setTenantId } from '../store/uiSlice';
import clsx from 'clsx';

export function Layout() {
  const [notifOpen, setNotifOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [userOpen, setUserOpen] = useState(false);
  const isRoot = useAppSelector(selectIsRoot);
  const sidebarOpen = useAppSelector(selectSidebarOpen);
  const selectedTenant = useAppSelector(selectTenantId) ?? '';
  const dispatch = useAppDispatch();
  const defaultTenantId = useAppSelector((s) => s.auth.user?.tenant_id) || '';
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t } = useI18n();

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

  const { data: tenantsData } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: isRoot,
  });
  const tenants = tenantsData?.tenants || [];
  const impersonating = isRoot && selectedTenant !== '' && selectedTenant !== defaultTenantId;
  const sidebarWidth = sidebarOpen ? 'md:ml-sidebar-expanded' : 'md:ml-sidebar-collapsed';

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
      {/* Sidebar */}
      <aside
        className={clsx(
          'hidden md:flex flex-col fixed left-0 top-0 h-full z-40 border-r border-outline-variant inner-glow',
          'bg-surface-container transition-[width] duration-300 ease-in-out',
          sidebarOpen ? 'w-sidebar-expanded' : 'w-sidebar-collapsed',
        )}
      >
        <div className={clsx('border-b border-outline-variant', sidebarOpen ? 'px-5 py-5' : 'px-2 py-4')}>
          <div className={clsx('flex items-center', !sidebarOpen && 'justify-center')}>
            <VirtFoundryLogo fullWidth iconOnly={!sidebarOpen} height={36} />
          </div>
        </div>

        <nav className="flex-1 flex flex-col gap-1 px-2 py-4 overflow-y-auto">
          <SidebarNav collapsed={!sidebarOpen} isRoot={isRoot} />
        </nav>
      </aside>

      {/* Main */}
      <div className={clsx('flex-1 flex flex-col min-h-screen w-full pt-16', sidebarWidth, 'transition-[margin] duration-300 ease-in-out')}>
        <header
          className={clsx(
            'fixed top-0 right-0 z-50 h-16 bg-surface border-b border-outline-variant inner-glow',
            'flex items-center justify-between px-4 md:px-6 gap-4 left-0',
            sidebarOpen ? 'md:left-sidebar-expanded' : 'md:left-sidebar-collapsed',
          )}
        >
          <div className="flex items-center gap-3 min-w-0">
            <button
              type="button"
              onClick={() => dispatch(setSidebarOpen(!sidebarOpen))}
              className="hidden md:flex p-2 rounded-full text-on-surface-variant hover:bg-surface-container transition-colors"
            >
              {sidebarOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
            <button type="button" className="md:hidden p-2 rounded-full text-on-surface-variant" onClick={() => dispatch(setSidebarOpen(!sidebarOpen))}>
              <Menu size={20} />
            </button>
          </div>

          <div className="hidden md:flex flex-1 max-w-xs ml-2">
            <HeaderSearch />
          </div>

          <div className="flex items-center gap-1 shrink-0">
            {isRoot && (
              <select
                value={selectedTenant}
                onChange={(e) => handleTenantChange(e.target.value)}
                className="hidden lg:block text-sm border border-outline-variant rounded-lg px-3 py-2 bg-surface-container-high text-on-surface max-w-[180px]"
              >
                <option value="">{t('nav.selectTenant')}</option>
                {tenants.map((tn) => (
                  <option key={tn.id} value={tn.id}>{tn.name}</option>
                ))}
              </select>
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
          <div className="bg-error-container/30 border-b border-error-container text-on-error-container px-6 py-2 text-sm">
            {t('nav.impersonatingTenant')}: {tenants.find((tn) => tn.id === selectedTenant)?.name || selectedTenant}
          </div>
        )}

        <main className="flex-1 p-4 md:p-6 lg:p-margin-desktop max-w-content mx-auto w-full">
          <Outlet />
        </main>
      </div>

      {/* Mobile sidebar overlay */}
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
    </div>
  );
}
