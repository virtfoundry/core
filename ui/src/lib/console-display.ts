import RFB from '@novnc/novnc';

/** Target remote desktop resolution (Full HD). */
export const CONSOLE_HD = { width: 1920, height: 1080 } as const;

type RFBInternal = RFB & {
  _rfbConnectionState?: string;
  _supportsSetDesktopSize?: boolean;
  _sock?: unknown;
  _screenID?: number;
  _screenFlags?: number;
  _pendingRemoteResize?: boolean;
  _lastResize?: number;
  _fbWidth?: number;
  _fbHeight?: number;
};

type RFBClass = typeof RFB & {
  messages?: {
    setDesktopSize: (
      sock: unknown,
      width: number,
      height: number,
      screenId: number,
      screenFlags: number,
    ) => void;
  };
};

export function supportsDesktopResize(rfb: RFB | null): boolean {
  return !!(rfb as RFBInternal)?._supportsSetDesktopSize;
}

/** Request a fixed remote desktop size (requires ExtendedDesktopSize from server). */
export function requestFixedDesktopSize(
  rfb: RFB,
  width = CONSOLE_HD.width,
  height = CONSOLE_HD.height,
): boolean {
  const r = rfb as RFBInternal;
  if (r._rfbConnectionState !== 'connected' || !r._supportsSetDesktopSize || !r._sock) {
    return false;
  }
  if (r._fbWidth === width && r._fbHeight === height) {
    return true;
  }

  const messages = (RFB as RFBClass).messages;
  if (!messages?.setDesktopSize) return false;

  messages.setDesktopSize(r._sock, width, height, r._screenID ?? 0, r._screenFlags ?? 0);
  r._pendingRemoteResize = true;
  r._lastResize = Date.now();
  return true;
}

/** Try HD resize only when the server advertises support — never on fixed VGA/Cirros. */
export function scheduleHdDesktopSize(
  rfb: RFB,
  onUpdate?: (w: number, h: number) => void,
) {
  if (!supportsDesktopResize(rfb)) {
    const fb = readFramebufferSize(rfb);
    if (fb) onUpdate?.(fb.width, fb.height);
    return;
  }

  const apply = () => {
    requestFixedDesktopSize(rfb);
    const fb = readFramebufferSize(rfb);
    if (fb) onUpdate?.(fb.width, fb.height);
    return !!fb;
  };

  apply();
  for (const ms of [50, 200, 500, 1000, 2000]) {
    window.setTimeout(apply, ms);
  }
}

export function readFramebufferSize(rfb: RFB | null): { width: number; height: number } | null {
  const r = rfb as RFBInternal;
  if (!r?._fbWidth || !r?._fbHeight) return null;
  return { width: r._fbWidth, height: r._fbHeight };
}

export function configureConsoleRfb(rfb: RFB) {
  rfb.viewOnly = false;
  rfb.focusOnClick = true;
  rfb.scaleViewport = true;
  rfb.resizeSession = false;
  rfb.clipViewport = false;
  rfb.showDotCursor = true;
  rfb.compressionLevel = 0;
  rfb.qualityLevel = 9;
  rfb.background = '#000000';
}

export function enableRemoteResizeIfSupported(rfb: RFB) {
  if (supportsDesktopResize(rfb)) {
    rfb.resizeSession = true;
  }
}

export function isHdResolution(width: number, height: number): boolean {
  return width >= 1280 && height >= 720;
}
