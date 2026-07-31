declare module '@novnc/novnc' {
  export default class RFB {
    constructor(target: HTMLElement, url: string, options?: Record<string, unknown>);
    scaleViewport: boolean;
    resizeSession: boolean;
    clipViewport: boolean;
    showDotCursor: boolean;
    viewOnly: boolean;
    focusOnClick: boolean;
    compressionLevel: number;
    qualityLevel: number;
    background: string;
    focus(options?: FocusOptions): void;
    disconnect(): void;
    sendCtrlAltDel(): void;
    sendKey(keysym: number, code: string, down?: boolean): void;
    clipboardPasteFrom(text: string): void;
    addEventListener(type: string, listener: (e: CustomEvent) => void): void;
  }
}
