import { formatDistanceToNowStrict } from 'date-fns';
import type { Locale } from 'date-fns';

/** "3 days ago" / "há 3 dias"; em dash when missing or invalid. */
export function ago(iso: string | undefined, locale: Locale): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '—' : formatDistanceToNowStrict(d, { addSuffix: true, locale });
}
