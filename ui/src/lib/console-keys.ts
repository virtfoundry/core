import type RFB from '@novnc/novnc';

const KEYS = {
  Control_L: 0xffe3,
  Alt_L: 0xffe9,
  Delete: 0xffff,
  Insert: 0xff63,
  BackSpace: 0xff08,
  Escape: 0xff1b,
  Tab: 0xff09,
  Return: 0xff0d,
  F1: 0xffbe,
  F2: 0xffbf,
  F3: 0xffc0,
  F4: 0xffc1,
  F5: 0xffc2,
  F6: 0xffc3,
  F7: 0xffc4,
  F8: 0xffc5,
  F9: 0xffc6,
  F10: 0xffc7,
  F11: 0xffc8,
  F12: 0xffc9,
} as const;

const KEY_CODES: Record<string, string> = {
  ControlLeft: 'ControlLeft',
  AltLeft: 'AltLeft',
  Delete: 'Delete',
  Insert: 'Insert',
  Backspace: 'Backspace',
  Escape: 'Escape',
  Tab: 'Tab',
  Enter: 'Enter',
  F1: 'F1',
  F2: 'F2',
  F3: 'F3',
  F4: 'F4',
  F5: 'F5',
  F6: 'F6',
  F7: 'F7',
  F8: 'F8',
  F9: 'F9',
  F10: 'F10',
  F11: 'F11',
  F12: 'F12',
};

export interface ConsoleKeyCommand {
  id: string;
  label: string;
  group: 'system' | 'function' | 'input';
  run: (rfb: RFB) => void;
}

function pressKey(rfb: RFB, keysym: number, code: string) {
  rfb.sendKey(keysym, code, true);
  rfb.sendKey(keysym, code, false);
}

function withModifiers(
  rfb: RFB,
  keysym: number,
  code: string,
  modifiers: { ctrl?: boolean; alt?: boolean; shift?: boolean },
) {
  if (modifiers.ctrl) rfb.sendKey(KEYS.Control_L, KEY_CODES.ControlLeft, true);
  if (modifiers.alt) rfb.sendKey(KEYS.Alt_L, KEY_CODES.AltLeft, true);

  pressKey(rfb, keysym, code);

  if (modifiers.alt) rfb.sendKey(KEYS.Alt_L, KEY_CODES.AltLeft, false);
  if (modifiers.ctrl) rfb.sendKey(KEYS.Control_L, KEY_CODES.ControlLeft, false);
}

export const CONSOLE_KEY_COMMANDS: ConsoleKeyCommand[] = [
  {
    id: 'ctrl-alt-del',
    label: 'Ctrl+Alt+Del',
    group: 'system',
    run: (rfb) => rfb.sendCtrlAltDel(),
  },
  {
    id: 'ctrl-alt-ins',
    label: 'Ctrl+Alt+Ins',
    group: 'system',
    run: (rfb) => withModifiers(rfb, KEYS.Insert, KEY_CODES.Insert, { ctrl: true, alt: true }),
  },
  {
    id: 'ctrl-alt-backspace',
    label: 'Ctrl+Alt+Backspace',
    group: 'system',
    run: (rfb) => withModifiers(rfb, KEYS.BackSpace, KEY_CODES.Backspace, { ctrl: true, alt: true }),
  },
  {
    id: 'esc',
    label: 'Esc',
    group: 'input',
    run: (rfb) => pressKey(rfb, KEYS.Escape, KEY_CODES.Escape),
  },
  {
    id: 'tab',
    label: 'Tab',
    group: 'input',
    run: (rfb) => pressKey(rfb, KEYS.Tab, KEY_CODES.Tab),
  },
  {
    id: 'enter',
    label: 'Enter',
    group: 'input',
    run: (rfb) => pressKey(rfb, KEYS.Return, KEY_CODES.Enter),
  },
  ...([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12] as const).map((n) => ({
    id: `ctrl-alt-f${n}`,
    label: `Ctrl+Alt+F${n}`,
    group: 'function' as const,
    run: (rfb: RFB) =>
      withModifiers(rfb, KEYS[`F${n}` as keyof typeof KEYS], KEY_CODES[`F${n}`], {
        ctrl: true,
        alt: true,
      }),
  })),
];

export function sendConsoleText(rfb: RFB, text: string) {
  if (!text) return;
  rfb.clipboardPasteFrom(text);
}
