import { Pencil, Trash2 } from 'lucide-react';
import clsx from 'clsx';
import { useI18n } from '../lib/i18n';

interface ResourceActionsProps {
  onEdit?: () => void;
  onDelete?: () => void;
  editLabel?: string;
  deleteLabel?: string;
  /** card = bordered footer (grid); inline = compact row actions (table) */
  variant?: 'card' | 'inline';
}

export function ResourceActions({
  onEdit,
  onDelete,
  editLabel,
  deleteLabel,
  variant = 'card',
}: ResourceActionsProps) {
  const { t } = useI18n();
  const inline = variant === 'inline';

  return (
    <div
      className={clsx(
        'flex gap-1',
        inline ? 'justify-end' : 'mt-3 pt-3 border-t border-card-border',
      )}
    >
      {onEdit && (
        <button
          type="button"
          onClick={onEdit}
          className={inline ? 'btn-icon-neutral' : 'btn-ghost-muted flex items-center gap-1'}
          title={editLabel ?? t('common.edit')}
        >
          <Pencil size={inline ? 16 : 14} />
          {!inline && (editLabel ?? t('common.edit'))}
        </button>
      )}
      {onDelete && (
        <button
          type="button"
          onClick={onDelete}
          className={
            inline
              ? 'btn-icon-danger'
              : 'btn-ghost-muted flex items-center gap-1 text-error hover:text-error'
          }
          title={deleteLabel ?? t('common.delete')}
        >
          <Trash2 size={inline ? 16 : 14} />
          {!inline && (deleteLabel ?? t('common.delete'))}
        </button>
      )}
    </div>
  );
}
