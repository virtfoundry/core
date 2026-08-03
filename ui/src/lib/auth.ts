const API_BASE = import.meta.env.VITE_PLATFORM_API_URL || '/api/v1';

export interface LoginCredentials {
  username: string;
  password: string;
}

export interface AuthUser {
  id: string;
  username: string;
  email: string;
  role: string;
  tenant_id?: string;
}

class AuthService {
  async login(credentials: LoginCredentials): Promise<AuthUser> {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials),
    });
    const body = await res.json();
    if (!res.ok) {
      throw new Error(body.error || 'Credenciais inválidas');
    }

    localStorage.setItem('jwt_token', body.token);
    if (body.user?.tenant_id) {
      localStorage.setItem('tenant_id', body.user.tenant_id);
    }

    const user: AuthUser = {
      id: body.user.id,
      username: body.user.username,
      email: body.user.email || `${body.user.username}@virtfoundry.local`,
      role: body.user.role,
      tenant_id: body.user.tenant_id,
    };
    localStorage.setItem('user', JSON.stringify(user));
    return user;
  }

  logout() {
    localStorage.removeItem('user');
    localStorage.removeItem('jwt_token');
    localStorage.removeItem('tenant_id');
  }

  getUser(): AuthUser | null {
    const stored = localStorage.getItem('user');
    if (!stored) return null;
    return JSON.parse(stored);
  }

  isAuthenticated(): boolean {
    return localStorage.getItem('jwt_token') !== null;
  }

  getToken(): string {
    return localStorage.getItem('jwt_token') || '';
  }

  isRoot(): boolean {
    return this.getUser()?.role === 'root';
  }
}

export const authService = new AuthService();
