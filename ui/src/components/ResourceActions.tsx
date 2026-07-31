import { Pencil, Trash2 } from 'lucide-react';

interface ResourceActionsProps {
  onEdit: () => void;
  onDelete: () => void;
  editLabel?: string;
  deleteLabel?: string;
}

export function ResourceActions({ onEdit, onDelete, editLabel = 'Editar', deleteLabel = 'Remover' }: ResourceActionsProps) {
  return (
    <div className="flex gap-1 mt-3 pt-3 border-t">
      <button type="button" onClick={onEdit} className="btn-ghost-muted flex items-center gap-1">
        <Pencil size={14} /> {editLabel}
      </button>
      <button type="button" onClick={onDelete} className="btn-ghost-muted flex items-center gap-1 text-red-600 hover:text-red-700">
        <Trash2 size={14} /> {deleteLabel}
      </button>
    </div>
  );
}
