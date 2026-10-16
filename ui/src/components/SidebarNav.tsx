import { NavLink } from 'react-router-dom';
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
  Container as ContainerIcon,
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
      { path: '/clusters', icon: ContainerIcon, labelKey: 'nav.clusters' },
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
    'flex items-center gap-3 rounded-lg transition-colors text-body-sm font-sans h-9',
    nested ? 'px-3' : 'px-3',
    collapsed && !nested && 'justify-center px-2',
    isActive
      ? 'bg-primary-container/10 text-primary font-semibold'
      : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface',
  );
}

type SidebarNavProps = {
  collapsed: boolean;
  isRoot: boolean;
  onNavigate?: () => void;
};

export function SidebarNav({ collapsed, isRoot, onNavigate }: SidebarNavProps) {
  const { t } = useI18n();

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
              <item.icon size={18} className="shrink-0" />
              {!collapsed && <span>{t(item.labelKey)}</span>}
            </NavLink>
          );
        }

        const visibleItems = item.items.filter((sub) => !sub.rootOnly || isRoot);
        if (visibleItems.length === 0) return null;

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
          <div key={item.id} className="flex flex-col gap-0.5 pt-3">
            <div className="px-3 pb-1.5 text-[11px] font-medium uppercase tracking-[0.08em] text-on-surface-variant/80">
              {t(item.labelKey)}
            </div>
            {visibleItems.map((sub) => (
              <NavLink
                key={sub.path}
                to={sub.path}
                end={sub.path === '/networks'}
                onClick={onNavigate}
                className={({ isActive }) => navLinkClass(isActive, false, true)}
              >
                <sub.icon size={18} className="shrink-0" />
                <span>{t(sub.labelKey)}</span>
              </NavLink>
            ))}
          </div>
        );
      })}
    </>
  );
}
