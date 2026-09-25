import { createConsoleTicket } from './platform-api';

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

/**
 * Builds the console socket URL from a freshly minted ticket. The JWT never
 * leaves the Authorization header, and the ticket in the URL is single-use,
 * expires in seconds, and is bound to this VM and tenant.
 */
export async function consoleWsUrl(name: string): Promise<string> {
  const { ticket } = await createConsoleTicket(name);
  const params = new URLSearchParams({ ticket });
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws/console?${params.toString()}`;
}
