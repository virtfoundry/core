import { useEffect, useMemo, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  LayoutDashboard as LayoutDashboardIcon,
  Server as ServerIcon,
  HardDrive as HardDriveIcon,
  Network as NetworkIcon,
  Globe as GlobeIcon,
  Shield as ShieldIcon,
  Boxes as BoxesIcon,
  Users as UsersIcon,
  Camera as CameraIcon,
  Key as KeyIcon,
  Disc as DiscIcon,
  Cpu as CpuIcon,
  ChevronDown,
} from 'lucide-react';
import clsx from 'clsx';
import { useI18n, type TranslationKey } from '../lib/i18n';

export type SubMenuItem = {
  path: string;
  icon: LucideIcon;
  labelKey: TranslationKey;
  rootOnly?: boolean;
};

export type MenuLinkItem = {
  kind: 'link';
  icon: LucideIcon;
  path: string;
  labelKey: TranslationKey;
};

export type MenuGroupItem = {
  kind: 'group';
  id: string;
  icon: LucideIcon;
  labelKey: TranslationKey;
  items: SubMenuItem[];
};

export type MenuItem = MenuLinkItem | MenuGroupItem;

export const menuItems: MenuItem[] = [
  { kind: 'link', icon: LayoutDashboardIcon, path: '/dashboard', labelKey: 'nav.dashboard' },
  {
    kind: 'group',
    id: 'compute',
    icon: ServerIcon,
    labelKey: 'nav.compute',
    items: [
      { path: '/vms', icon: ServerIcon, labelKey: 'nav.vms' },
      { path: '/templates', icon: DiscIcon, labelKey: 'nav.templates' },
      { path: '/ssh-keys', icon: KeyIcon, labelKey: 'nav.sshKeys' },
      { path: '/vm-snapshots', icon: CameraIcon, labelKey: 'nav.vmSnapshots' },
    ],
  },
  {
    kind: 'group',
    id: 'storage',
    icon: HardDriveIcon,
    labelKey: 'nav.storage',
    items: [
      { path: '/volumes', icon: HardDriveIcon, labelKey: 'nav.volumes' },
      { path: '/snapshots', icon: HardDriveIcon, labelKey: 'nav.volumeSnapshots' },
    ],
  },
  {
    kind: 'group',
    id: 'network',
    icon: NetworkIcon,
    labelKey: 'nav.network',
    items: [
      { path: '/networks/public', icon: GlobeIcon, labelKey: 'nav.publicNetwork' },
      { path: '/networks', icon: NetworkIcon, labelKey: 'nav.networks' },
      { path: '/vpcs', icon: BoxesIcon, labelKey: 'nav.vpcs' },
      { path: '/security-groups', icon: ShieldIcon, labelKey: 'nav.securityGroups' },
    ],
  },
  {
    kind: 'group',
    id: 'platform',
    icon: UsersIcon,
    labelKey: 'nav.platform',
    items: [
      { path: '/iam', icon: KeyIcon, labelKey: 'nav.iam' },
      { path: '/offerings', icon: CpuIcon, labelKey: 'nav.offerings', rootOnly: true },
      { path: '/tenants', icon: UsersIcon, labelKey: 'nav.tenants', rootOnly: true },
    ],
  },
];

function navLinkClass(isActive: boolean, collapsed: boolean, nested = false) {
  return clsx(
    'flex items-center gap-3 rounded-lg transition-colors text-label-md font-mono active:scale-95',
    nested ? 'px-3 py-2 text-sm' : 'px-4 py-2.5',
    collapsed && !nested && 'justify-center px-2',
    nested && 'ml-7 mr-1',
    isActive
      ? 'bg-primary-container text-on-primary-container font-semibold shadow-nav-active inner-glow'
      : 'text-on-surface-variant hover:bg-surface-variant hover:text-primary',
  );
}

function isPathActive(pathname: string, path: string) {
  if (path === '/networks') {
    return pathname === '/networks' || (pathname.startsWith('/networks/') && !pathname.startsWith('/networks/public'));
  }
  return pathname === path || pathname.startsWith(`${path}/`);
}

function groupHasActive(pathname: string, items: SubMenuItem[]) {
  return items.some((sub) => isPathActive(pathname, sub.path));
}

type SidebarNavProps = {
  collapsed: boolean;
  isRoot: boolean;
  onNavigate?: () => void;
};

export function SidebarNav({ collapsed, isRoot, onNavigate }: SidebarNavProps) {
  const { t } = useI18n();
  const { pathname } = useLocation();

  const activeGroupIds = useMemo(() => {
    const ids = new Set<string>();
    for (const item of menuItems) {
      if (item.kind !== 'group') continue;
      const visible = item.items.filter((sub) => !sub.rootOnly || isRoot);
      if (groupHasActive(pathname, visible)) ids.add(item.id);
    }
    return ids;
  }, [pathname, isRoot]);

  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(activeGroupIds));

  useEffect(() => {
    setExpanded((prev) => {
      const next = new Set(prev);
      activeGroupIds.forEach((id) => next.add(id));
      return next;
    });
  }, [activeGroupIds]);

  const toggleGroup = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <>
      {menuItems.map((item) => {
        if (item.kind === 'link') {
          return (
            <NavLink
              key={item.path}
              to={item.path}
              onClick={onNavigate}
              className={({ isActive }) => navLinkClass(isActive, collapsed)}
            >
              <item.icon size={20} className="shrink-0" />
              {!collapsed && <span>{t(item.labelKey)}</span>}
            </NavLink>
          );
        }

        const visibleItems = item.items.filter((sub) => !sub.rootOnly || isRoot);
        if (visibleItems.length === 0) return null;

        const isOpen = expanded.has(item.id);
        const isActiveGroup = groupHasActive(pathname, visibleItems);

        if (collapsed) {
          return (
            <div key={item.id} className="space-y-1">
              {visibleItems.map((sub) => (
                <NavLink
                  key={sub.path}
                  to={sub.path}
                  end={sub.path === '/networks'}
                  title={t(sub.labelKey)}
                  onClick={onNavigate}
                  className={({ isActive }) => navLinkClass(isActive, true)}
                >
                  <sub.icon size={18} className="shrink-0" />
                </NavLink>
              ))}
            </div>
          );
        }

        return (
          <div key={item.id} className="rounded-lg">
            <button
              type="button"
              onClick={() => toggleGroup(item.id)}
              aria-expanded={isOpen}
              className={clsx(
                'flex w-full items-center gap-3 px-4 py-2.5 rounded-lg transition-colors text-label-md font-mono',
                isActiveGroup
                  ? 'text-on-surface bg-surface-container-high/60'
                  : 'text-on-surface-variant hover:bg-surface-variant hover:text-on-surface',
              )}
            >
              <item.icon size={20} className="shrink-0" />
              <span className="flex-1 text-left">{t(item.labelKey)}</span>
              <ChevronDown
                size={16}
                className={clsx('shrink-0 transition-transform duration-200', isOpen && 'rotate-180')}
              />
            </button>

            <div
              className={clsx(
                'grid transition-[grid-template-rows] duration-200 ease-in-out',
                isOpen ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]',
              )}
            >
              <div className="overflow-hidden">
                <div className="space-y-0.5 pb-1 pt-0.5">
                  {visibleItems.map((sub) => (
                    <NavLink
                      key={sub.path}
                      to={sub.path}
                      end={sub.path === '/networks'}
                      onClick={onNavigate}
                      className={({ isActive }) => navLinkClass(isActive, false, true)}
                    >
                      <sub.icon size={16} className="shrink-0 opacity-80" />
                      <span>{t(sub.labelKey)}</span>
                    </NavLink>
                  ))}
                </div>
              </div>
            </div>
          </div>
        );
      })}
    </>
  );
}
