import { Outlet, NavLink } from 'react-router-dom';
import { useState, useEffect } from 'react';
import type { LucideIcon } from 'lucide-react';
import {
  LayoutDashboard, Server, HardDrive, Network, Globe, Shield, Boxes,
  Users, LogOut, Menu, X, Camera, Key, Disc,
} from 'lucide-react';
import { VirtFoundryLogo } from './VirtFoundryLogo';
import { authService } from '../lib/auth';
import { listTenants } from '../lib/platform-api';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRealtimeEvents } from '../hooks/useRealtimeEvents';
import { queryKeys, isPlatformQueryKey } from '../lib/query-keys';
import { useI18n, LanguageToggle, type TranslationKey } from '../lib/i18n';
import { ThemeToggle } from '../lib/theme';

type SubMenuItem = { path: string; icon: LucideIcon; labelKey: TranslationKey; rootOnly?: boolean };
type MenuLinkItem = { group: string; icon: LucideIcon; path: string; labelKey: TranslationKey };
type MenuGroupItem = { group: string; items: SubMenuItem[] };
type MenuItem = MenuLinkItem | MenuGroupItem;

function isLinkItem(item: MenuItem): item is MenuLinkItem {
  return 'path' in item;
}

const menuItems: MenuItem[] = [
  { group: 'Dashboard', icon: LayoutDashboard, path: '/dashboard', labelKey: 'nav.dashboard' },
  {
    group: 'Compute',
    items: [
      { path: '/vms', icon: Server, labelKey: 'nav.vms' },
      { path: '/templates', icon: Disc, labelKey: 'nav.templates' },
      { path: '/ssh-keys', icon: Key, labelKey: 'nav.sshKeys' },
      { path: '/vm-snapshots', icon: Camera, labelKey: 'nav.vmSnapshots' },
    ],
  },
  {
    group: 'Storage',
    items: [
      { path: '/volumes', icon: HardDrive, labelKey: 'nav.volumes' },
      { path: '/snapshots', icon: HardDrive, labelKey: 'nav.volumeSnapshots' },
    ],
  },
  {
    group: 'Network',
    items: [
      { path: '/networks/public', icon: Globe, labelKey: 'nav.publicNetwork' },
      { path: '/networks', icon: Network, labelKey: 'nav.networks' },
      { path: '/vpcs', icon: Boxes, labelKey: 'nav.vpcs' },
      { path: '/security-groups', icon: Shield, labelKey: 'nav.securityGroups' },
    ],
  },
  {
    group: 'Platform',
    items: [
      { path: '/iam', icon: Key, labelKey: 'nav.iam' as TranslationKey },
      { path: '/tenants', icon: Users, labelKey: 'nav.tenants', rootOnly: true },
    ],
  },
];

export function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const user = authService.getUser();
  const isRoot = authService.isRoot();
  const defaultTenantId = user?.tenant_id || '';
  const [selectedTenant, setSelectedTenant] = useState(
    localStorage.getItem('tenant_id') || defaultTenantId,
  );
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t } = useI18n();

  useEffect(() => {
    if (!isRoot || !defaultTenantId) return;
    if (!localStorage.getItem('tenant_id')) {
      localStorage.setItem('tenant_id', defaultTenantId);
      setSelectedTenant(defaultTenantId);
    }
  }, [isRoot, defaultTenantId]);

  useRealtimeEvents();

  const { data: tenantsData } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: isRoot,
  });
  const tenants = tenantsData?.tenants || [];
  const impersonating = isRoot && selectedTenant !== '' && selectedTenant !== defaultTenantId;

  const handleTenantChange = (tenantId: string) => {
    setSelectedTenant(tenantId);
    if (tenantId) localStorage.setItem('tenant_id', tenantId);
    else localStorage.removeItem('tenant_id');
    queryClient.refetchQueries({
      predicate: (q) => isPlatformQueryKey(q.queryKey),
      type: 'active',
    });
  };

  const handleLogout = () => {
    authService.logout();
    navigate('/login');
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-dark-200 flex">
      <aside className={`${sidebarOpen ? 'w-64' : 'w-20'} bg-white dark:bg-dark-100 border-r border-gray-200 dark:border-gray-700 flex flex-col transition-all duration-300 fixed h-full z-40`}>
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <VirtFoundryLogo size={40} />
            {sidebarOpen && (
              <div>
                <h1 className="font-bold text-gray-900 dark:text-white">VirtFoundry</h1>
                <p className="text-xs text-gray-500">IaaS Platform</p>
              </div>
            )}
          </div>
        </div>

        <nav className="flex-1 overflow-y-auto p-3 space-y-4">
          {menuItems.map((item) => (
            <div key={item.group}>
              {isLinkItem(item) ? (
                <NavLink
                  to={item.path}
                  className={({ isActive }) =>
                    `flex items-center gap-3 px-3 py-2.5 rounded-lg transition ${
                      isActive ? 'bg-brand-500 text-white' : 'text-gray-600 hover:bg-gray-100'
                    }`
                  }
                >
                  <item.icon size={20} />
                  {sidebarOpen && <span className="font-medium">{t(item.labelKey)}</span>}
                </NavLink>
              ) : (
                <div>
                  {sidebarOpen && (
                    <p className="px-3 py-1 text-xs font-semibold text-gray-400 uppercase">{item.group}</p>
                  )}
                  <div className="space-y-1 mt-1">
                    {item.items.filter((sub) => !sub.rootOnly || isRoot).map((sub) => (
                      <NavLink
                        key={sub.path}
                        to={sub.path}
                        end={sub.path === '/networks'}
                        className={({ isActive }) =>
                          `flex items-center gap-3 px-3 py-2 rounded-lg transition ml-1 ${
                            isActive ? 'bg-brand-500 text-white' : 'text-gray-600 hover:bg-gray-100'
                          }`
                        }
                      >
                        <sub.icon size={18} />
                        {sidebarOpen && <span className="text-sm">{t(sub.labelKey)}</span>}
                      </NavLink>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </nav>

        <div className="p-4 border-t border-gray-200 dark:border-gray-700">
          {sidebarOpen && (
            <>
              <p className="font-medium truncate">{user?.username}</p>
              <p className="text-xs text-gray-500 mb-3">{user?.role}</p>
              <button onClick={handleLogout} className="btn-ghost flex items-center gap-2 text-sm text-gray-600 px-2 py-1 rounded-lg hover:bg-gray-100 dark:hover:bg-dark-200">
                <LogOut size={16} /> {t('nav.logout')}
              </button>
            </>
          )}
        </div>
      </aside>

      <main className={`flex-1 ${sidebarOpen ? 'ml-64' : 'ml-20'} transition-all duration-300`}>
        <header className="bg-white dark:bg-dark-100 border-b px-6 py-4 flex items-center justify-between sticky top-0 z-30">
          <button onClick={() => setSidebarOpen(!sidebarOpen)} className="btn-sidebar">
            {sidebarOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
          <div className="flex items-center gap-3">
            <ThemeToggle compact />
            <LanguageToggle />
            {isRoot && (
              <select
                value={selectedTenant}
                onChange={(e) => handleTenantChange(e.target.value)}
                className="text-sm border rounded-lg px-3 py-2"
              >
                <option value="">{t('nav.selectTenant')}</option>
                {tenants.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            )}
          </div>
        </header>
        {impersonating && (
          <div className="bg-amber-50 border-b border-amber-200 text-amber-900 px-6 py-2 text-sm">
            {t('nav.impersonatingTenant')}: {tenants.find((tn) => tn.id === selectedTenant)?.name || selectedTenant}
          </div>
        )}
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
