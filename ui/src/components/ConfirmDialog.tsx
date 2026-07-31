import { AlertTriangle, Loader2 } from 'lucide-react';
import clsx from 'clsx';
import { useEffect, useRef } from 'react';

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  resourceName?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  loading?: boolean;
}

/**
 * Destructive-action confirmation (Material / Apple HIG pattern):
 * - modal, not native alert
 * - explicit cancel + destructive confirm
 * - resource name highlighted
 * - backdrop click does not dismiss (avoid accidental delete)
 */
export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  message,
  resourceName,
  confirmLabel = 'Remover',
  cancelLabel = 'Cancelar',
  loading = false,
}: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (open) cancelRef.current?.focus();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[60] overflow-y-auto" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
      <div className="flex min-h-full items-center justify-center p-4">
        <div className="fixed inset-0 bg-black/50" aria-hidden="true" />
        <div className="relative bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md">
          <div className="p-6">
            <div className="flex gap-4">
              <div className="shrink-0 w-10 h-10 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
                <AlertTriangle className="text-red-600" size={22} />
              </div>
              <div className="flex-1 min-w-0">
                <h3 id="confirm-title" className="text-lg font-semibold text-gray-900 dark:text-white">
                  {title}
                </h3>
                <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">{message}</p>
                {resourceName && (
                  <p className="mt-2 text-sm font-mono font-medium text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700/50 px-3 py-2 rounded-lg truncate">
                    {resourceName}
                  </p>
                )}
              </div>
            </div>
          </div>
          <div className="flex justify-end gap-3 px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 rounded-b-xl">
            <button
              ref={cancelRef}
              type="button"
              onClick={onClose}
              disabled={loading}
              className="btn-secondary"
            >
              {cancelLabel}
            </button>
            <button
              type="button"
              onClick={onConfirm}
              disabled={loading}
              className={clsx(
                'inline-flex items-center justify-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
                'bg-red-600 text-white hover:bg-red-700 disabled:opacity-50'
              )}
            >
              {loading && <Loader2 size={16} className="animate-spin" />}
              {confirmLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
