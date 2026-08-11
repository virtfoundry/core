import { AlertTriangle, Loader2 } from 'lucide-react';
import clsx from 'clsx';
import { useEffect, useRef } from 'react';
import { useI18n } from '../lib/i18n';

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
  error?: string;
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  message,
  resourceName,
  confirmLabel,
  cancelLabel,
  loading = false,
  error,
}: ConfirmDialogProps) {
  const { t } = useI18n();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const resolvedConfirmLabel = confirmLabel ?? t('common.delete');
  const resolvedCancelLabel = cancelLabel ?? t('common.cancel');

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
        <div className="fixed inset-0 bg-black/50 backdrop-blur-[1px]" aria-hidden="true" />
        <div className="relative vf-card shadow-xl w-full max-w-md">
          <div className="p-6">
            <div className="flex gap-4">
              <div
                className="shrink-0 w-10 h-10 rounded-full flex items-center justify-center border border-error/30"
                style={{ backgroundColor: 'color-mix(in srgb, var(--vf-error-container) 25%, transparent)' }}
              >
                <AlertTriangle className="text-error" size={22} />
              </div>
              <div className="flex-1 min-w-0">
                <h3 id="confirm-title" className="font-headline text-headline-md font-semibold text-on-surface">
                  {title}
                </h3>
                <p className="mt-2 text-sm text-on-surface-variant">{message}</p>
                {resourceName && (
                  <p className="mt-2 text-sm font-data-mono font-medium text-on-surface bg-surface-container-high px-3 py-2 rounded-lg truncate">
                    {resourceName}
                  </p>
                )}
                {error && <p className="mt-2 text-sm text-error">{error}</p>}
              </div>
            </div>
          </div>
          <div className="flex justify-end gap-3 px-6 py-4 border-t border-card-border bg-surface-container-low rounded-b-xl">
            <button
              ref={cancelRef}
              type="button"
              onClick={onClose}
              disabled={loading}
              className="btn-secondary"
            >
              {resolvedCancelLabel}
            </button>
            <button
              type="button"
              onClick={onConfirm}
              disabled={loading}
              className={clsx(
                'inline-flex items-center justify-center gap-2 px-4 py-2 h-10 rounded-lg text-label-md font-mono font-medium transition-all',
                'bg-error-container text-on-error-container hover:opacity-90 disabled:opacity-50 inner-glow',
              )}
            >
              {loading && <Loader2 size={16} className="animate-spin" />}
              {resolvedConfirmLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
