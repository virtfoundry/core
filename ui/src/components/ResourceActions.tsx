import { Pencil, Trash2 } from 'lucide-react';
import { useI18n } from '../lib/i18n';

interface ResourceActionsProps {
  onEdit: () => void;
  onDelete: () => void;
  editLabel?: string;
  deleteLabel?: string;
}

export function ResourceActions({ onEdit, onDelete, editLabel, deleteLabel }: ResourceActionsProps) {
  const { t } = useI18n();

  return (
    <div className="flex gap-1 mt-3 pt-3 border-t">
      <button type="button" onClick={onEdit} className="btn-ghost-muted flex items-center gap-1">
        <Pencil size={14} /> {editLabel ?? t('common.edit')}
      </button>
      <button type="button" onClick={onDelete} className="btn-ghost-muted flex items-center gap-1 text-red-600 hover:text-red-700">
        <Trash2 size={14} /> {deleteLabel ?? t('common.delete')}
      </button>
    </div>
  );
}
