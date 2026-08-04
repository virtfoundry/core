import { useEffect, useRef, useState, type ReactNode, type RefObject } from 'react';
import { ExternalLink, HelpCircle, LogOut, Moon, Settings, Sun, User } from 'lucide-react';
import clsx from 'clsx';
import { useI18n } from '../lib/i18n';
import { useTheme } from '../lib/theme';
import { useAppSelector } from '../store/hooks';
import { selectUser } from '../store/authSlice';

function userInitials(username?: string) {
  if (!username) return '?';
  return username.slice(0, 2).toUpperCase();
}

function useClickOutside(ref: RefObject<HTMLElement | null>, open: boolean, onClose: () => void) {
  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, [open, onClose, ref]);
}

function HeaderPopover({ open, children }: { open: boolean; children: ReactNode }) {
  if (!open) return null;
  return (
    <div className="absolute z-50 right-0 top-full mt-2 w-64 vf-card shadow-xl py-1">
      {children}
    </div>
  );
}

type SettingsMenuProps = {
  open: boolean;
  onToggle: () => void;
  onClose: () => void;
};

export function SettingsMenu({ open, onToggle, onClose }: SettingsMenuProps) {
  const { t, locale, setLocale } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const rootRef = useRef<HTMLDivElement>(null);
  const isDark = theme === 'dark';

  useClickOutside(rootRef, open, onClose);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        title={t('sidebar.settings')}
        className="p-2 text-on-surface-variant hover:text-primary hover:bg-surface-variant rounded-full transition-colors"
      >
        <Settings size={20} />
      </button>

      <HeaderPopover open={open}>
        <div className="px-4 py-3 border-b border-card-border flex items-center gap-2 text-on-surface-variant">
          <Settings size={16} />
          <span className="font-label text-sm">{t('sidebar.settings')}</span>
        </div>

        <div className="px-4 py-3 border-b border-card-border">
          <p className="text-xs font-label text-on-surface-variant mb-2">{t('sidebar.theme')}</p>
          <button
            type="button"
            onClick={toggleTheme}
            className="flex w-full items-center gap-3 px-3 py-2 rounded-lg text-sm text-on-surface hover:bg-surface-variant transition-colors"
          >
            {isDark ? <Sun size={16} /> : <Moon size={16} />}
            {isDark ? t('sidebar.themeLight') : t('sidebar.themeDark')}
          </button>
        </div>

        <div className="px-4 py-3 border-b border-card-border">
          <p className="text-xs font-label text-on-surface-variant mb-2">{t('sidebar.language')}</p>
          <div className="flex rounded-lg border border-outline-variant overflow-hidden text-sm font-mono">
            <button
              type="button"
              onClick={() => setLocale('pt')}
              className={clsx(
                'flex-1 px-3 py-1.5 transition-colors',
                locale === 'pt'
                  ? 'bg-primary-container text-on-primary-container'
                  : 'text-on-surface-variant hover:bg-surface-variant',
              )}
            >
              {t('lang.pt')}
            </button>
            <button
              type="button"
              onClick={() => setLocale('en')}
              className={clsx(
                'flex-1 px-3 py-1.5 transition-colors',
                locale === 'en'
                  ? 'bg-primary-container text-on-primary-container'
                  : 'text-on-surface-variant hover:bg-surface-variant',
              )}
            >
              {t('lang.en')}
            </button>
          </div>
        </div>

        <a
          href="https://github.com/virtfoundry/core/tree/main/core/ui/docs"
          target="_blank"
          rel="noopener noreferrer"
          onClick={onClose}
          className="flex items-center gap-3 px-4 py-2.5 text-sm text-on-surface hover:bg-surface-variant transition-colors"
        >
          <HelpCircle size={16} />
          <span className="flex-1">{t('header.help')}</span>
          <ExternalLink size={14} className="text-on-surface-variant" />
        </a>

        <div className="px-4 py-3 border-t border-card-border flex items-center gap-2 text-xs text-on-surface-variant">
          <User size={14} />
          {t('sidebar.about')}
        </div>
      </HeaderPopover>
    </div>
  );
}

type UserMenuProps = {
  open: boolean;
  onToggle: () => void;
  onClose: () => void;
  onLogout: () => void;
};

export function UserMenu({ open, onToggle, onClose, onLogout }: UserMenuProps) {
  const { t } = useI18n();
  const user = useAppSelector(selectUser);
  const rootRef = useRef<HTMLDivElement>(null);

  useClickOutside(rootRef, open, onClose);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        title={user?.username}
        className="w-8 h-8 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center border border-outline-variant font-semibold text-xs hover:ring-2 hover:ring-primary-container/50 transition-shadow"
      >
        {userInitials(user?.username)}
      </button>

      <HeaderPopover open={open}>
        <div className="px-4 py-3 border-b border-card-border">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-primary-container text-on-primary-container flex items-center justify-center shrink-0 border border-outline-variant font-semibold text-sm">
              {userInitials(user?.username)}
            </div>
            <div className="min-w-0">
              <p className="font-medium text-sm text-on-surface truncate">{user?.username}</p>
              {user?.email && <p className="text-xs text-on-surface-variant truncate">{user.email}</p>}
              <p className="text-xs text-on-surface-variant capitalize mt-0.5">{user?.role}</p>
            </div>
          </div>
        </div>
        <button
          type="button"
          onClick={() => {
            onClose();
            onLogout();
          }}
          className="flex w-full items-center gap-3 px-4 py-2.5 text-sm text-error hover:bg-surface-variant transition-colors"
        >
          <LogOut size={16} />
          {t('nav.logout')}
        </button>
      </HeaderPopover>
    </div>
  );
}
