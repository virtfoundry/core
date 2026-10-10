import { X } from 'lucide-react';
import { useI18n } from '../lib/i18n';

type TagsInputProps = {
  tags: string[];
  onTagsChange: (tags: string[]) => void;
  draft: string;
  onDraftChange: (draft: string) => void;
  placeholder?: string;
};

/**
 * Controlled tag editor: committed tags render as pills with an x to remove
 * them, while `draft` holds the text being typed. Enter, comma or blur commits
 * the draft; Backspace on an empty draft removes the last tag. Parents keep the
 * draft so a submit handler can flush pending input via `mergeTags`.
 */
export function TagsInput({ tags, onTagsChange, draft, onDraftChange, placeholder }: TagsInputProps) {
  const { t } = useI18n();

  const commit = () => {
    const parts = draft.split(/[,;]/).map((s) => s.trim()).filter(Boolean);
    if (parts.length === 0) {
      if (draft) onDraftChange('');
      return;
    }
    const next = [...tags];
    for (const part of parts) if (!next.includes(part)) next.push(part);
    onTagsChange(next);
    onDraftChange('');
  };

  const remove = (tag: string) => onTagsChange(tags.filter((x) => x !== tag));

  return (
    <div className="w-full min-h-10 px-2 py-1.5 flex flex-wrap items-center gap-1.5 border border-outline-variant rounded-lg bg-surface-container-high text-on-surface text-body-sm focus-within:border-primary-container focus-within:ring-1 focus-within:ring-primary-container transition-colors">
      {tags.map((tag) => (
        <span
          key={tag}
          className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded border border-outline-variant bg-surface-container text-on-surface"
        >
          {tag}
          <button
            type="button"
            onClick={() => remove(tag)}
            className="text-on-surface-variant hover:text-error rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            aria-label={`${t('common.removeTag')}: ${tag}`}
            title={t('common.removeTag')}
          >
            <X size={12} />
          </button>
        </span>
      ))}
      <input
        type="text"
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ',') {
            e.preventDefault();
            commit();
          } else if (e.key === 'Backspace' && draft === '' && tags.length > 0) {
            remove(tags[tags.length - 1]);
          }
        }}
        onBlur={commit}
        className="flex-1 min-w-[8rem] bg-transparent outline-none text-sm py-0.5 text-on-surface placeholder:text-on-surface-variant"
        placeholder={tags.length === 0 ? placeholder : ''}
      />
    </div>
  );
}

/** Full tag list including any text still held in the draft input. */
export function mergeTags(tags: string[], draft: string): string[] {
  const extra = draft.split(/[,;]/).map((s) => s.trim()).filter(Boolean);
  return Array.from(new Set([...tags, ...extra]));
}
