import type { ReactNode } from 'react';
import clsx from 'clsx';

export interface KVRow {
  k: ReactNode;
  v: ReactNode;
  mono?: boolean;
}

/** Key/value list inside a SurfaceCard: muted key left, value right, hairline between rows. */
export function KeyValueList({ rows, className }: { rows: KVRow[]; className?: string }) {
  return (
    <dl className={clsx('m-0 flex flex-col', className)}>
      {rows.map((r, i) => (
        <div key={i} className="flex justify-between gap-4 py-[9px] border-t border-line3 first:border-t-0 first:pt-0 text-sm">
          <dt className="text-on-surface-variant">{r.k}</dt>
          <dd className={clsx('m-0 text-right text-on-surface min-w-0 break-words', r.mono && 'font-mono text-[13px]')}>{r.v}</dd>
        </div>
      ))}
    </dl>
  );
}

/** Dark terminal-style block (same look in both themes). */
export function TerminalBlock({ lines, className }: { lines: { text: string; tone?: 'ok' | 'muted' | 'plain' }[]; className?: string }) {
  return (
    <div className={clsx('rounded-[10px] bg-[#0A0F1C] text-[#D6DEF5] px-[18px] py-4 font-mono text-[13px] leading-[22px] overflow-x-auto', className)}>
      {lines.map((l, i) => (
        <div
          key={i}
          className={clsx('whitespace-pre min-h-[22px]', l.tone === 'ok' && 'text-[#62DF7D]', l.tone === 'muted' && 'text-[#8E9BC0]')}
        >
          {l.text}
        </div>
      ))}
    </div>
  );
}
