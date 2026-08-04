import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit';
import { getMe, login as apiLogin } from '../lib/platform-api';

export interface AuthUser {
  id: string;
  username: string;
  email: string;
  role: string;
  tenant_id?: string;
}

export type AuthStatus = 'idle' | 'loading' | 'authenticated' | 'unauthenticated';

interface AuthState {
  token: string | null;
  user: AuthUser | null;
  status: AuthStatus;
}

function readPersistedUser(): AuthUser | null {
  try {
    const raw = localStorage.getItem('user');
    return raw ? (JSON.parse(raw) as AuthUser) : null;
  } catch {
    return null;
  }
}

const initialState: AuthState = {
  token: localStorage.getItem('jwt_token'),
  user: readPersistedUser(),
  status: localStorage.getItem('jwt_token') ? 'idle' : 'unauthenticated',
};

function persistAuth(token: string | null, user: AuthUser | null, tenantId?: string | null) {
  if (token) localStorage.setItem('jwt_token', token);
  else localStorage.removeItem('jwt_token');

  if (user) localStorage.setItem('user', JSON.stringify(user));
  else localStorage.removeItem('user');

  if (tenantId) localStorage.setItem('tenant_id', tenantId);
  else if (tenantId === null) localStorage.removeItem('tenant_id');
}

export const loginThunk = createAsyncThunk<
  AuthUser,
  { username: string; password: string },
  { rejectValue: string }
>('auth/login', async (credentials, { rejectWithValue }) => {
  try {
    const body = await apiLogin(credentials.username, credentials.password);
    const user: AuthUser = {
      id: body.user.id,
      username: body.user.username,
      email: body.user.email || `${body.user.username}@virtfoundry.local`,
      role: body.user.role,
      tenant_id: body.user.tenant_id,
    };
    persistAuth(body.token, user, body.user.tenant_id ?? null);
    return user;
  } catch (e) {
    return rejectWithValue((e as Error).message);
  }
});

export const validateSessionThunk = createAsyncThunk<AuthUser | null>(
  'auth/validateSession',
  async (_, { dispatch }) => {
    const token = localStorage.getItem('jwt_token');
    if (!token) return null;
    try {
      const me = await getMe();
      const user: AuthUser = {
        id: me.id,
        username: me.username,
        email: `${me.username}@virtfoundry.local`,
        role: me.role,
        tenant_id: me.tenant_id,
      };
      persistAuth(token, user, me.tenant_id ?? null);
      return user;
    } catch {
      dispatch(logout());
      return null;
    }
  },
);

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    logout(state) {
      state.token = null;
      state.user = null;
      state.status = 'unauthenticated';
      persistAuth(null, null, null);
    },
    setUser(state, action: PayloadAction<AuthUser | null>) {
      state.user = action.payload;
      if (action.payload) {
        localStorage.setItem('user', JSON.stringify(action.payload));
      }
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(loginThunk.pending, (state) => {
        state.status = 'loading';
      })
      .addCase(loginThunk.fulfilled, (state, action) => {
        state.token = localStorage.getItem('jwt_token');
        state.user = action.payload;
        state.status = 'authenticated';
      })
      .addCase(loginThunk.rejected, (state) => {
        state.status = 'unauthenticated';
      })
      .addCase(validateSessionThunk.pending, (state) => {
        if (state.token) state.status = 'loading';
      })
      .addCase(validateSessionThunk.fulfilled, (state, action) => {
        if (action.payload) {
          state.token = localStorage.getItem('jwt_token');
          state.user = action.payload;
          state.status = 'authenticated';
        } else {
          state.token = null;
          state.user = null;
          state.status = 'unauthenticated';
        }
      })
      .addCase(validateSessionThunk.rejected, (state) => {
        state.token = null;
        state.user = null;
        state.status = 'unauthenticated';
      });
  },
});

export const { logout, setUser } = authSlice.actions;
export default authSlice.reducer;

export const selectAuth = (state: { auth: AuthState }) => state.auth;
export const selectUser = (state: { auth: AuthState }) => state.auth.user;
export const selectIsAuthenticated = (state: { auth: AuthState }) =>
  state.auth.status === 'authenticated';
export const selectIsRoot = (state: { auth: AuthState }) => state.auth.user?.role === 'root';
export const selectAuthStatus = (state: { auth: AuthState }) => state.auth.status;
