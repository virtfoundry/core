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
import {
  PageHeader, SurfaceCard, KeyValueList, InfoBanner, toolbarBtnClass, formTextareaClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

type Status = 'connecting' | 'connected' | 'error';

export function VMConsole() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const name = params.get('name') ?? '';

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

    const connect = async () => {
      let wsUrl: string;
      try {
        wsUrl = await consoleWsUrl(name);
      } catch (e) {
        if (cancelled) return;
        setStatus('error');
        setError(e instanceof Error ? e.message : t('console.ticketFailed'));
        return;
      }
      if (cancelled || !containerRef.current) return;

      const rfb = new RFB(containerRef.current, wsUrl);
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
    };

    const frame = requestAnimationFrame(() => {
      if (cancelled) return;
      void connect();
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
  }, [name, t]);

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

  const cmdClass = toolbarBtnClass + ' font-mono';
  const connected = status === 'connected';

  return (
    <div className="h-screen flex flex-col gap-4 bg-bg text-on-surface p-4 md:p-6 overflow-hidden">
      <PageHeader
        title={`Console · ${name}`}
        subtitle={t('console.subtitle')}
        status={
          status === 'connecting' ? (
            <StatusBadge status="connecting" tone="warn" label={t('console.connecting')} />
          ) : status === 'connected' ? (
            <StatusBadge status="connected" tone="ok" label={t('console.connected')} />
          ) : (
            <StatusBadge status="error" tone="bad" label={t('status.error')} />
          )
        }
        actions={
          <Link to={`/vms/${encodeURIComponent(name)}`} className="btn-secondary">
            <ArrowLeft size={16} /> {t('console.back')}
          </Link>
        }
      />

      {status === 'error' && error && <InfoBanner variant="warning">{error}</InfoBanner>}

      <div className="shrink-0 flex flex-wrap items-center gap-2">
        <span className="flex items-center gap-1.5 mr-1 text-[13px] font-medium text-on-surface-variant">
          <Keyboard size={14} /> {t('console.commands')}
        </span>

        {systemCommands.map((cmd) => (
          <button key={cmd.id} type="button" disabled={!connected} onClick={() => runCommand(cmd.id)} className={cmdClass}>
            {cmd.label}
          </button>
        ))}

        <span className="text-card-border select-none">|</span>

        {inputCommands.map((cmd) => (
          <button key={cmd.id} type="button" disabled={!connected} onClick={() => runCommand(cmd.id)} className={cmdClass}>
            {cmd.label}
          </button>
        ))}

        <button type="button" disabled={!connected} onClick={() => setPasteOpen((v) => !v)} className={cmdClass}>
          <ClipboardPaste size={14} /> {t('console.pasteText')}
        </button>

        <details className="relative">
          <summary className={cmdClass + ' cursor-pointer list-none'}>Ctrl+Alt+F1–F12</summary>
          <div className="absolute z-20 mt-1 p-2 bg-card border border-card-border rounded-xl shadow-lg grid grid-cols-4 gap-1.5 min-w-[240px]">
            {functionCommands.map((cmd, i) => (
              <button key={cmd.id} type="button" disabled={!connected} onClick={() => runCommand(cmd.id)} className={cmdClass}>
                F{i + 1}
              </button>
            ))}
          </div>
        </details>
      </div>

      {lowResWarning && connected && (
        <InfoBanner variant="warning" className="shrink-0 text-xs">
          {t('console.lowResWarning')
            .replace('{resolution}', resolution ?? '—')
            .replace('{width}', String(CONSOLE_HD.width))
            .replace('{height}', String(CONSOLE_HD.height))}
          {!supportsDesktopResize(rfbRef.current) && ` ${t('console.remoteResizeUnavailable')}`}
        </InfoBanner>
      )}

      {pasteOpen && (
        <div className="shrink-0 flex gap-2 items-start">
          <textarea
            value={pasteText}
            onChange={(e) => setPasteText(e.target.value)}
            placeholder={t('console.pastePlaceholder')}
            rows={2}
            className={formTextareaClass + ' flex-1'}
          />
          <button
            type="button"
            disabled={!connected || !pasteText.trim()}
            onClick={handlePaste}
            className="btn-primary"
          >
            {t('console.send')}
          </button>
        </div>
      )}

      <div className="flex-1 min-h-0 flex gap-4">
        <div className="relative flex-1 min-w-0 rounded-xl overflow-hidden border border-card-border bg-black">
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
        <SurfaceCard title={t('console.session')} className="hidden lg:block w-[260px] shrink-0 self-start">
          <KeyValueList
            rows={[
              { k: t('console.protocol'), v: 'VNC / WebSocket' },
              ...(resolution ? [{ k: t('console.resolution'), v: resolution, mono: true }] : []),
            ]}
          />
        </SurfaceCard>
      </div>
    </div>
  );
}
