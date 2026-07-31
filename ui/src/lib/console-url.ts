export function consoleUrl(name: string, namespace?: string): string {
  const params = new URLSearchParams({ name });
  if (namespace) params.set('namespace', namespace);
  return `/console?${params.toString()}`;
}

export function openConsole(name: string, namespace?: string, newTab = true): void {
  const url = consoleUrl(name, namespace);
  if (newTab) {
    window.open(url, '_blank', 'noopener,noreferrer');
  } else {
    window.location.assign(url);
  }
}

export function consoleWsUrl(name: string, namespace?: string): string {
  const params = new URLSearchParams({ name });
  if (namespace) params.set('namespace', namespace);
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws/console?${params.toString()}`;
}
