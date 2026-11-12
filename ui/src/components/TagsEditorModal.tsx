import { useState } from 'react';
import { Modal } from './Modal';
import { TagsInput, mergeTags } from './TagsInput';
import { useI18n } from '../lib/i18n';

type TagsEditorModalProps = {
  isOpen: boolean;
  onClose: () => void;
  initialTags: string[];
  onSubmit: (tags: string[]) => void;
  loading?: boolean;
  error?: string | null;
};

/**
 * Modal that edits the tag list of a single resource. Mount it with a `key`
 * tied to the resource id so switching rows resets the editor state.
 */
export function TagsEditorModal({ isOpen, onClose, initialTags, onSubmit, loading, error }: TagsEditorModalProps) {
  const { t } = useI18n();
  const [tags, setTags] = useState<string[]>(initialTags);
  const [draft, setDraft] = useState('');

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t('tags.editTitle')}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit(mergeTags(tags, draft));
        }}
        className="space-y-4"
      >
        <TagsInput
          tags={tags}
          onTagsChange={setTags}
          draft={draft}
          onDraftChange={setDraft}
          placeholder={t('common.tagsPlaceholder')}
        />
        <p className="text-xs text-on-surface-variant">{t('tags.editHint')}</p>
        {error && <p className="text-error text-sm">{error}</p>}
        <div className="flex justify-end gap-3 pt-2">
          <button type="button" onClick={onClose} className="btn-secondary">
            {t('common.cancel')}
          </button>
          <button type="submit" disabled={loading} className="btn-primary">
            {loading ? t('common.loading') : t('common.save')}
          </button>
        </div>
      </form>
    </Modal>
  );
}
