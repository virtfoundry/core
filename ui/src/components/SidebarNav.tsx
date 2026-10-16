import { NavLink } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import {
  LayoutGrid as LayoutDashboardIcon,
  Server as ServerIcon,
  HardDrive as HardDriveIcon,
  Database as DatabaseIcon,
  Network as NetworkIcon,
  Globe as GlobeIcon,
  Shield as ShieldIcon,
  Boxes as BoxesIcon,
  Box as BoxIcon,
  Camera as CameraIcon,
  KeyRound as KeyRoundIcon,
  Key as KeyIcon,
  Layers as LayersIcon,
  Tag as TagIcon,
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
      { path: '/templates', icon: LayersIcon, labelKey: 'nav.templates' },
      { path: '/ssh-keys', icon: KeyRoundIcon, labelKey: 'nav.sshKeys' },
      { path: '/clusters', icon: BoxIcon, labelKey: 'nav.clusters' },
      { path: '/vm-snapshots', icon: CameraIcon, labelKey: 'nav.vmSnapshots' },
    ],
  },
  {
    kind: 'group',
    id: 'storage',
    icon: HardDriveIcon,
    labelKey: 'nav.storage',
    items: [
      { path: '/volumes', icon: DatabaseIcon, labelKey: 'nav.volumes' },
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
    icon: BoxIcon,
    labelKey: 'nav.platform',
    items: [
      { path: '/iam', icon: KeyIcon, labelKey: 'nav.iam' },
      { path: '/offerings', icon: TagIcon, labelKey: 'nav.offerings', rootOnly: true },
      { path: '/tenants', icon: BoxIcon, labelKey: 'nav.tenants', rootOnly: true },
    ],
  },
];

function navLinkClass(isActive: boolean, collapsed: boolean) {
  return clsx(
    'flex items-center gap-2.5 rounded-lg transition-colors text-sm font-sans h-9 px-2.5',
    collapsed && 'justify-center px-2',
    isActive
      ? 'bg-side-active-bg text-side-active-text'
      : 'text-side-text hover:bg-side-hover hover:text-side-active-text',
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
    <div className="flex flex-col gap-3.5">
      {menuItems.map((item) => {
        if (item.kind === 'link') {
          return (
            <div key={item.path} className="flex flex-col gap-0.5">
              <NavLink
                to={item.path}
                onClick={onNavigate}
                title={collapsed ? t(item.labelKey) : undefined}
                className={({ isActive }) => navLinkClass(isActive, collapsed)}
              >
                <item.icon size={18} strokeWidth={1.7} className="shrink-0" />
                {!collapsed && <span className="flex-grow">{t(item.labelKey)}</span>}
              </NavLink>
            </div>
          );
        }

        const visibleItems = item.items.filter((sub) => !sub.rootOnly || isRoot);
        if (visibleItems.length === 0) return null;

        return (
          <div key={item.id} className="flex flex-col gap-0.5">
            {!collapsed && (
              <div className="px-2.5 pb-1.5 text-[11px] font-medium uppercase tracking-[0.08em] text-side-label">
                {t(item.labelKey)}
              </div>
            )}
            {visibleItems.map((sub) => (
              <NavLink
                key={sub.path}
                to={sub.path}
                end={sub.path === '/networks'}
                title={collapsed ? t(sub.labelKey) : undefined}
                onClick={onNavigate}
                className={({ isActive }) => navLinkClass(isActive, collapsed)}
              >
                <sub.icon size={18} strokeWidth={1.7} className="shrink-0" />
                {!collapsed && <span className="flex-grow">{t(sub.labelKey)}</span>}
              </NavLink>
            ))}
          </div>
        );
      })}
    </div>
  );
}
