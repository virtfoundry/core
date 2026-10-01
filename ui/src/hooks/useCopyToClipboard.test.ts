import { describe, expect, it, vi } from 'vitest';
import { copyTextWithFallback } from './useCopyToClipboard';

function makeClipboard(impl: (text: string) => Promise<void> | void) {
  return { writeText: impl };
}

function makeFakeDocument(overrides: Partial<{
  execCommandReturn: boolean;
  execCommandThrows: boolean;
}> = {}) {
  const elements: HTMLElement[] = [];
  const fakeTextarea = {
    value: '',
    style: {} as Record<string, string>,
    setAttribute: () => undefined,
    focus: () => undefined,
    select: () => undefined,
    setSelectionRange: () => undefined,
  };
  const body = {
    appendChild: (el: unknown) => {
      elements.push(el as HTMLElement);
      return el;
    },
    removeChild: (el: unknown) => {
      const i = elements.indexOf(el as HTMLElement);
      if (i >= 0) elements.splice(i, 1);
      return el;
    },
  };
  const document = {
    createElement: () => fakeTextarea,
    body,
    execCommand: (_cmd: string) => {
      if (overrides.execCommandThrows) throw new Error('boom');
      return overrides.execCommandReturn ?? true;
    },
  };
  return { document, fakeTextarea, elements };
}

describe('copyTextWithFallback', () => {
  it('resolves true via clipboard API when writeText succeeds', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    const ok = await copyTextWithFallback('hello', {
      clipboard: makeClipboard(writeText),
    });
    expect(ok).toBe(true);
    expect(writeText).toHaveBeenCalledWith('hello');
  });

  it('falls back to execCommand when clipboard.writeText rejects', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'));
    const { document, fakeTextarea } = makeFakeDocument({ execCommandReturn: true });
    const ok = await copyTextWithFallback('secret', {
      clipboard: makeClipboard(writeText),
      document: document as unknown as Document,
    });
    expect(ok).toBe(true);
    expect(writeText).toHaveBeenCalledTimes(1);
    expect(fakeTextarea.value).toBe('secret');
  });

  it('returns false when both clipboard and execCommand fail', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'));
    const { document } = makeFakeDocument({ execCommandReturn: false });
    const ok = await copyTextWithFallback('secret', {
      clipboard: makeClipboard(writeText),
      document: document as unknown as Document,
    });
    expect(ok).toBe(false);
  });

  it('returns false immediately for empty input', async () => {
    const writeText = vi.fn();
    const ok = await copyTextWithFallback('', {
      clipboard: makeClipboard(writeText),
    });
    expect(ok).toBe(false);
    expect(writeText).not.toHaveBeenCalled();
  });

  it('uses legacy path when no clipboard API is available', async () => {
    const { document, fakeTextarea } = makeFakeDocument({ execCommandReturn: true });
    const ok = await copyTextWithFallback('legacy-only', {
      clipboard: null,
      document: document as unknown as Document,
    });
    expect(ok).toBe(true);
    expect(fakeTextarea.value).toBe('legacy-only');
  });

  it('returns false when both clipboard and document are unavailable', async () => {
    const ok = await copyTextWithFallback('text', {
      clipboard: null,
      document: null,
    });
    expect(ok).toBe(false);
  });

  it('treats execCommand throwing as a failure', async () => {
    const { document } = makeFakeDocument({ execCommandThrows: true });
    const ok = await copyTextWithFallback('text', {
      clipboard: null,
      document: document as unknown as Document,
    });
    expect(ok).toBe(false);
  });
});
