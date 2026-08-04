import { type ReactNode, useEffect } from 'react';
import { Moon, Sun } from 'lucide-react';
import clsx from 'clsx';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { selectIsDarkTheme, selectTheme, setTheme, toggleTheme, type Theme } from '../store/themeSlice';

export type { Theme };

export function ThemeProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    for (const src of [
      '/virtfounfry-light.png',
      '/virtfounfry-dark.png',
      '/virtfounfry-icon-light.png',
      '/virtfounfry-icon-dark.png',
    ]) {
      const img = new Image();
      img.src = src;
    }
  }, []);

  return children;
}

export function useTheme() {
  const theme = useAppSelector(selectTheme);
  const dispatch = useAppDispatch();
  return {
    theme,
    setTheme: (next: Theme) => dispatch(setTheme(next)),
    toggleTheme: () => dispatch(toggleTheme()),
  };
}

interface ThemeToggleProps {
  onDark?: boolean;
  compact?: boolean;
}

export function ThemeToggle({ onDark = false, compact = false }: ThemeToggleProps) {
  const isDark = useAppSelector(selectIsDarkTheme);
  const dispatch = useAppDispatch();

  return (
    <button
      type="button"
      onClick={() => dispatch(toggleTheme())}
      title={isDark ? 'Light mode' : 'Dark mode'}
      aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
      className={clsx(
        'inline-flex items-center justify-center rounded-lg border transition-colors inner-glow',
        compact ? 'p-2' : 'px-3 py-1.5 gap-2 text-sm',
        onDark
          ? 'border-white/30 text-white hover:bg-white/10'
          : 'border-outline-variant text-on-surface-variant hover:bg-surface-variant hover:text-primary',
      )}
    >
      {isDark ? <Sun size={18} /> : <Moon size={18} />}
      {!compact && <span>{isDark ? 'Light' : 'Dark'}</span>}
    </button>
  );
}

/** Kept for index.html inline script parity */
export function initThemeFromStorage() {
  // themeSlice initializer already applied the class on import
}
