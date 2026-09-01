import { authService } from './auth';

export function consoleUrl(name: string): string {
  const params = new URLSearchParams({ name });
  return `/console?${params.toString()}`;
}

export function openConsole(name: string, _namespace?: string, newTab = true): void {
  const url = consoleUrl(name);
  if (newTab) {
    window.open(url, '_blank', 'noopener,noreferrer');
  } else {
    window.location.assign(url);
  }
}

export function consoleWsUrl(name: string): string {
  const params = new URLSearchParams({ name });
  const token = authService.getToken();
  if (token) params.set('token', token);
  const tenantId = localStorage.getItem('tenant_id');
  if (tenantId) params.set('tenant_id', tenantId);
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws/console?${params.toString()}`;
}
