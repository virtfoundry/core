import { useCallback, useEffect, useRef, useState } from 'react';

export type CopyState = 'idle' | 'copied' | 'error';

type ClipboardWriter = {
  writeText?: (text: string) => Promise<void> | void;
};

/**
 * Pure copy helper — exported for testing. Tries the async Clipboard API and
 * falls back to a hidden-textarea + `execCommand('copy')` when it rejects.
 */
export async function copyTextWithFallback(
  text: string,
  deps: {
    clipboard?: ClipboardWriter | null;
    document?: Document | null;
  } = {},
): Promise<boolean> {
  if (!text) return false;

  const clipboard = deps.clipboard ?? (typeof navigator !== 'undefined' ? navigator.clipboard : null);
  const doc = deps.document ?? (typeof document !== 'undefined' ? document : null);

  if (clipboard?.writeText) {
    try {
      await clipboard.writeText(text);
      return true;
    } catch {
      /* fall through to legacy */
    }
  }

  if (!doc) return false;
  return legacyCopy(doc, text);
}

function legacyCopy(doc: Document, text: string): boolean {
  const textarea = doc.createElement('textarea');
  textarea.value = text;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.top = '0';
  textarea.style.left = '0';
  textarea.style.width = '1px';
  textarea.style.height = '1px';
  textarea.style.opacity = '0';
  textarea.style.pointerEvents = 'none';
  doc.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  textarea.setSelectionRange(0, textarea.value.length);
  let ok = false;
  try {
    ok = doc.execCommand('copy');
  } catch {
    ok = false;
  }
  doc.body.removeChild(textarea);
  return ok;
}

/**
 * Copy a string to the system clipboard with a legacy `execCommand` fallback
 * for browsers/contexts where `navigator.clipboard.writeText` rejects
 * (non-secure context, no user-gesture focus, permission denied, etc.).
 *
 * The returned `state` resets to `'idle'` automatically after `resetMs`.
 */
export function useCopyToClipboard(resetMs = 2000) {
  const [state, setState] = useState<CopyState>('idle');
  const timerRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, []);

  const scheduleReset = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
    }
    timerRef.current = window.setTimeout(() => {
      setState('idle');
      timerRef.current = null;
    }, resetMs);
  }, [resetMs]);

  const copy = useCallback(
    async (text: string): Promise<boolean> => {
      const ok = await copyTextWithFallback(text);
      setState(ok ? 'copied' : 'error');
      scheduleReset();
      return ok;
    },
    [scheduleReset],
  );

  const reset = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    setState('idle');
  }, []);

  return { state, copy, reset } as const;
}
