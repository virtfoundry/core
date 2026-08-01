import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, Navigate, useSearchParams } from 'react-router-dom';
import RFB from '@novnc/novnc';
import { ArrowLeft, ClipboardPaste, Keyboard } from 'lucide-react';
import { consoleWsUrl } from '../lib/console-url';
import { CONSOLE_KEY_COMMANDS, sendConsoleText } from '../lib/console-keys';
import {
  CONSOLE_HD,
  configureConsoleRfb,
  enableRemoteResizeIfSupported,
  isHdResolution,
  readFramebufferSize,
  scheduleHdDesktopSize,
  supportsDesktopResize,
} from '../lib/console-display';
import { useI18n } from '../lib/i18n';

type Status = 'connecting' | 'connected' | 'error';

export function VMConsole() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const name = params.get('name') ?? '';
  const namespace = params.get('namespace') ?? undefined;

  const containerRef = useRef<HTMLDivElement>(null);
  const rfbRef = useRef<RFB | null>(null);
  const [status, setStatus] = useState<Status>('connecting');
  const [error, setError] = useState<string | null>(null);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteText, setPasteText] = useState('');
  const [resolution, setResolution] = useState<string | null>(null);
  const [lowResWarning, setLowResWarning] = useState(false);

  const systemCommands = useMemo(
    () => CONSOLE_KEY_COMMANDS.filter((c) => c.group === 'system'),
    [],
  );
  const inputCommands = useMemo(
    () => CONSOLE_KEY_COMMANDS.filter((c) => c.group === 'input'),
    [],
  );
  const functionCommands = useMemo(
    () => CONSOLE_KEY_COMMANDS.filter((c) => c.group === 'function'),
    [],
  );

  useEffect(() => {
    if (!name || !containerRef.current) return;

    let cancelled = false;
    setStatus('connecting');
    setError(null);
    setResolution(null);
    setLowResWarning(false);

    const updateResolution = () => {
      const fb = readFramebufferSize(rfbRef.current);
      if (!fb) return;
      setResolution(`${fb.width}×${fb.height}`);
      setLowResWarning(!isHdResolution(fb.width, fb.height));
    };

    const frame = requestAnimationFrame(() => {
      if (cancelled || !containerRef.current) return;

      const rfb = new RFB(containerRef.current, consoleWsUrl(name, namespace));
      configureConsoleRfb(rfb);
      rfbRef.current = rfb;

      const applyHd = () => {
        if (cancelled) return;
        enableRemoteResizeIfSupported(rfb);
        scheduleHdDesktopSize(rfb, updateResolution);
        updateResolution();
      };

      rfb.addEventListener('connect', () => {
        if (cancelled) return;
        setStatus('connected');
        applyHd();
        rfb.focus();
      });
      rfb.addEventListener('capabilities', applyHd);
      rfb.addEventListener('desktopname', applyHd);
      rfb.addEventListener('disconnect', (e: CustomEvent) => {
        if (cancelled || e.detail?.clean) return;
        setStatus('error');
        setError(e.detail?.reason || t('console.vncDisconnected'));
      });
      rfb.addEventListener('securityfailure', (e: CustomEvent) => {
        if (cancelled) return;
        setStatus('error');
        setError(e.detail?.reason || t('console.vncAuthFailed'));
      });
    });

    const pollFb = window.setInterval(updateResolution, 1500);

    const ro = new ResizeObserver(() => {
      if (rfbRef.current) {
        scheduleHdDesktopSize(rfbRef.current, updateResolution);
      }
    });
    if (containerRef.current) ro.observe(containerRef.current);

    const lowResTimer = window.setTimeout(() => {
      const fb = readFramebufferSize(rfbRef.current);
      if (fb && !isHdResolution(fb.width, fb.height)) {
        setLowResWarning(true);
      }
    }, 6000);

    return () => {
      cancelled = true;
      cancelAnimationFrame(frame);
      window.clearInterval(pollFb);
      window.clearTimeout(lowResTimer);
      ro.disconnect();
      rfbRef.current?.disconnect();
      rfbRef.current = null;
    };
  }, [name, namespace, t]);

  if (!name) {
    return <Navigate to="/vms" replace />;
  }

  const runCommand = (id: string) => {
    const cmd = CONSOLE_KEY_COMMANDS.find((c) => c.id === id);
    const rfb = rfbRef.current;
    if (!cmd || !rfb || status !== 'connected') return;
    cmd.run(rfb);
    rfb.focus();
  };

  const handlePaste = () => {
    const rfb = rfbRef.current;
    if (!rfb || status !== 'connected' || !pasteText.trim()) return;
    sendConsoleText(rfb, pasteText);
    setPasteOpen(false);
    setPasteText('');
    rfb.focus();
  };

  return (
    <div className="h-screen flex flex-col bg-black text-slate-100">
      <header className="shrink-0 border-b border-slate-600/80 bg-slate-800 shadow-lg shadow-black/40">
        <div className="flex items-center justify-between gap-4 px-4 py-2.5">
          <div className="flex items-center gap-3 min-w-0">
            <Link
              to={`/vms/${encodeURIComponent(name)}`}
              className="btn-console-back shrink-0"
            >
              <ArrowLeft size={16} /> {t('console.back')}
            </Link>
            <div className="min-w-0 border-l border-slate-600 pl-3">
              <h1 className="font-semibold truncate text-white">Console — {name}</h1>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {resolution && (
              <span className="text-xs text-slate-300 bg-slate-700/80 border border-slate-500/50 px-2 py-1 rounded-full font-mono">
                {resolution}
              </span>
            )}
            {status === 'connecting' && (
              <span className="inline-flex items-center gap-1.5 text-sm text-amber-300 bg-amber-950/60 border border-amber-700/50 px-2.5 py-1 rounded-full">
                <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
                {t('console.connecting')}
              </span>
            )}
            {status === 'connected' && (
              <span className="inline-flex items-center gap-1.5 text-sm text-green-300 bg-green-950/60 border border-green-700/50 px-2.5 py-1 rounded-full">
                <span className="w-2 h-2 rounded-full bg-green-400" />
                {t('console.connected')}
              </span>
            )}
            {status === 'error' && (
              <span className="text-sm text-red-300 bg-red-950/60 border border-red-700/50 px-2.5 py-1 rounded-full">
                {error}
              </span>
            )}
          </div>
        </div>

        <div className="px-4 pb-2.5 flex flex-wrap items-center gap-2 bg-slate-700/30">
          <span className="text-[11px] uppercase tracking-wider text-slate-300 font-semibold flex items-center gap-1.5 mr-1 px-2 py-1 rounded-md bg-slate-700/80 border border-slate-500/50">
            <Keyboard size={13} /> {t('console.commands')}
          </span>

          {systemCommands.map((cmd) => (
            <button
              key={cmd.id}
              type="button"
              disabled={status !== 'connected'}
              onClick={() => runCommand(cmd.id)}
              className="btn-console"
            >
              {cmd.label}
            </button>
          ))}

          <span className="text-slate-500 select-none">|</span>

          {inputCommands.map((cmd) => (
            <button
              key={cmd.id}
              type="button"
              disabled={status !== 'connected'}
              onClick={() => runCommand(cmd.id)}
              className="btn-console"
            >
              {cmd.label}
            </button>
          ))}

          <button
            type="button"
            disabled={status !== 'connected'}
            onClick={() => setPasteOpen((v) => !v)}
            className="btn-console"
          >
            <ClipboardPaste size={14} /> {t('console.pasteText')}
          </button>

          <details className="relative">
            <summary className="btn-console cursor-pointer list-none">
              Ctrl+Alt+F1–F12
            </summary>
            <div className="absolute z-20 mt-1 p-2 bg-slate-700 border border-slate-500 rounded-lg shadow-2xl grid grid-cols-4 gap-1.5 min-w-[240px]">
              {functionCommands.map((cmd, i) => (
                <button
                  key={cmd.id}
                  type="button"
                  disabled={status !== 'connected'}
                  onClick={() => runCommand(cmd.id)}
                  className="btn-console"
                >
                  F{i + 1}
                </button>
              ))}
            </div>
          </details>
        </div>

        {lowResWarning && status === 'connected' && (
          <div className="mx-4 mb-2 px-3 py-2 text-xs text-amber-200/90 bg-amber-950/50 border border-amber-700/40 rounded-lg">
            {t('console.lowResWarning')
              .replace('{resolution}', resolution ?? '—')
              .replace('{width}', String(CONSOLE_HD.width))
              .replace('{height}', String(CONSOLE_HD.height))}
            {!supportsDesktopResize(rfbRef.current) && ` ${t('console.remoteResizeUnavailable')}`}
          </div>
        )}

        {pasteOpen && (
          <div className="px-4 pb-3 flex gap-2 items-start bg-slate-800/80 border-t border-slate-600/50 pt-3">
            <textarea
              value={pasteText}
              onChange={(e) => setPasteText(e.target.value)}
              placeholder={t('console.pastePlaceholder')}
              rows={2}
              className="flex-1 px-3 py-2 text-sm bg-slate-700 border border-slate-500 rounded-lg text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-500/50"
            />
            <button
              type="button"
              disabled={status !== 'connected' || !pasteText.trim()}
              onClick={handlePaste}
              className="btn-console-action"
            >
              {t('console.send')}
            </button>
          </div>
        )}
      </header>

      <div className="relative flex-1 min-h-0">
        <div ref={containerRef} className="console-viewport absolute inset-0" />
        {status === 'connecting' && (
          <div className="absolute inset-0 flex flex-col items-center justify-center bg-black/80 text-slate-300 text-sm pointer-events-none gap-2">
            <span>{t('console.waiting')}</span>
            <span className="text-xs text-slate-500">
              {t('console.targetResolution')}: {CONSOLE_HD.width}×{CONSOLE_HD.height}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
