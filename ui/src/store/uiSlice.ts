import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface UiState {
  sidebarOpen: boolean;
  tenantId: string | null;
}

const initialState: UiState = {
  sidebarOpen: true,
  tenantId: localStorage.getItem('tenant_id'),
};

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    setSidebarOpen(state, action: PayloadAction<boolean>) {
      state.sidebarOpen = action.payload;
    },
    toggleSidebar(state) {
      state.sidebarOpen = !state.sidebarOpen;
    },
    setTenantId(state, action: PayloadAction<string | null>) {
      state.tenantId = action.payload;
      if (action.payload) localStorage.setItem('tenant_id', action.payload);
      else localStorage.removeItem('tenant_id');
    },
  },
});

export const { setSidebarOpen, toggleSidebar, setTenantId } = uiSlice.actions;
export default uiSlice.reducer;

export const selectSidebarOpen = (state: { ui: UiState }) => state.ui.sidebarOpen;
export const selectTenantId = (state: { ui: UiState }) => state.ui.tenantId;
export const selectNeedsTenant = (state: { auth: { user: { role: string } | null }; ui: UiState }) =>
  state.auth.user?.role === 'root' && !state.ui.tenantId;
