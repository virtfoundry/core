import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'virtfoundry_theme';

function systemPrefersDark() {
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

export function resolveTheme(theme: Theme): 'light' | 'dark' {
  if (theme === 'system') return systemPrefersDark() ? 'dark' : 'light';
  return theme;
}

export function applyThemeClass(theme: Theme) {
  document.documentElement.classList.toggle('dark', resolveTheme(theme) === 'dark');
}

function readStoredTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
  return 'system';
}

interface ThemeState {
  theme: Theme;
}

const initialTheme = readStoredTheme();
applyThemeClass(initialTheme);

// Follow system when preference is "system"
if (typeof window !== 'undefined') {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    const cur = localStorage.getItem(STORAGE_KEY);
    if (cur === 'system' || !cur) applyThemeClass('system');
  });
}

const themeSlice = createSlice({
  name: 'theme',
  initialState: { theme: initialTheme } satisfies ThemeState,
  reducers: {
    setTheme(state, action: PayloadAction<Theme>) {
      state.theme = action.payload;
      localStorage.setItem(STORAGE_KEY, action.payload);
      applyThemeClass(action.payload);
    },
    toggleTheme(state) {
      const resolved = resolveTheme(state.theme);
      const next: Theme = resolved === 'dark' ? 'light' : 'dark';
      state.theme = next;
      localStorage.setItem(STORAGE_KEY, next);
      applyThemeClass(next);
    },
  },
});

export const { setTheme, toggleTheme } = themeSlice.actions;
export default themeSlice.reducer;

export const selectTheme = (state: { theme: ThemeState }) => state.theme.theme;
export const selectIsDarkTheme = (state: { theme: ThemeState }) =>
  resolveTheme(state.theme.theme) === 'dark';
