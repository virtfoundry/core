import pkg from '../../package.json';

/** SemVer synced with core/CHANGELOG on release (ui/package.json). */
export const APP_VERSION = pkg.version;

export function appVersionLabel(prefix = 'VirtFoundry'): string {
  return `${prefix} v${APP_VERSION}`;
}
