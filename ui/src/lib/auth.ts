import { store } from '../store';
import {
  loginThunk,
  validateSessionThunk,
  logout,
  selectUser,
  selectIsRoot,
  selectIsAuthenticated,
  type AuthUser,
} from '../store/authSlice';
import { setTenantId } from '../store/uiSlice';

export type { AuthUser };

export const authService = {
  async login(credentials: { username: string; password: string }): Promise<AuthUser> {
    const user = await store.dispatch(loginThunk(credentials)).unwrap();
    store.dispatch(setTenantId(user.tenant_id ?? null));
    return user;
  },

  async validateSession(): Promise<AuthUser | null> {
    const user = await store.dispatch(validateSessionThunk()).unwrap();
    if (user?.tenant_id) {
      store.dispatch(setTenantId(user.tenant_id));
    }
    return user;
  },

  logout() {
    store.dispatch(logout());
    store.dispatch(setTenantId(null));
  },

  getUser(): AuthUser | null {
    return selectUser(store.getState());
  },

  isAuthenticated(): boolean {
    return selectIsAuthenticated(store.getState());
  },

  getToken(): string {
    return store.getState().auth.token || localStorage.getItem('jwt_token') || '';
  },

  isRoot(): boolean {
    return selectIsRoot(store.getState());
  },
};
