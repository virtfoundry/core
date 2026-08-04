import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'virtfoundry_theme';

export function applyThemeClass(theme: Theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark');
}

function readStoredTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'light' || stored === 'dark') return stored;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

interface ThemeState {
  theme: Theme;
}

const initialTheme = readStoredTheme();
applyThemeClass(initialTheme);

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
      const next: Theme = state.theme === 'dark' ? 'light' : 'dark';
      state.theme = next;
      localStorage.setItem(STORAGE_KEY, next);
      applyThemeClass(next);
    },
  },
});

export const { setTheme, toggleTheme } = themeSlice.actions;
export default themeSlice.reducer;

export const selectTheme = (state: { theme: ThemeState }) => state.theme.theme;
export const selectIsDarkTheme = (state: { theme: ThemeState }) => state.theme.theme === 'dark';
