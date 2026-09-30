import { useMemo, useState } from 'react';
import { formTextareaClass, InfoBanner } from './shell';
import { ComingSoonBadge } from './ComingSoonBadge';
import { maskCloudInitSecrets, validateCloudInitYaml } from '../lib/error-catalog';
import { getCloudInitDraft, setCloudInitDraft } from '../lib/preview-prefs';
import { useI18n } from '../lib/i18n';

type Props = {
  vmName: string;
  /** When true, show masked preview for deploy review (no local persist button). */
  reviewOnly?: boolean;
  /** When true, userdata is sent on deployVM (hide local-only badge/hint). */
  applyOnDeploy?: boolean;
  initialValue?: string;
  onChange?: (value: string) => void;
};

export function CloudInitEditor({ vmName, reviewOnly = false, applyOnDeploy = false, initialValue, onChange }: Props) {
  const { t } = useI18n();
  const [value, setValue] = useState(() => initialValue ?? (getCloudInitDraft(vmName) || '#cloud-config\nusers:\n  - name: ubuntu\n    sudo: ALL=(ALL) NOPASSWD:ALL\n'));
  const [saved, setSaved] = useState(false);

  const validation = useMemo(() => validateCloudInitYaml(value), [value]);
  const masked = useMemo(() => maskCloudInitSecrets(value), [value]);

  const handleChange = (next: string) => {
    setValue(next);
    setSaved(false);
    onChange?.(next);
  };

  const handleSaveLocal = () => {
    setCloudInitDraft(vmName, value);
    setSaved(true);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 flex-wrap">
        <h3 className="text-sm font-medium text-on-surface">{t('cloudinit.title')}</h3>
        {!applyOnDeploy && <ComingSoonBadge label={t('preview.localOnly')} />}
      </div>
      <p className="text-xs text-on-surface-variant">
        {applyOnDeploy ? t('cloudinit.deployHint') : t('cloudinit.hint')}
      </p>
      <textarea
        value={value}
        onChange={(e) => handleChange(e.target.value)}
        rows={12}
        spellCheck={false}
        className={`${formTextareaClass} font-data-mono text-xs`}
        aria-invalid={!validation.ok}
      />
      {!validation.ok && (
        <InfoBanner variant="warning">{validation.issue}</InfoBanner>
      )}
      {validation.ok && value.trim() && (
        <InfoBanner>
          <p className="text-xs font-medium mb-1">{t('cloudinit.maskedPreview')}</p>
          <pre className="text-[11px] font-data-mono whitespace-pre-wrap max-h-32 overflow-y-auto opacity-90">{masked}</pre>
        </InfoBanner>
      )}
      {!reviewOnly && (
        <div className="flex items-center gap-2">
          <button type="button" className="btn-outline-sm" disabled={!validation.ok} onClick={handleSaveLocal}>
            {t('cloudinit.saveLocal')}
          </button>
          {saved && <span className="text-xs text-success">{t('cloudinit.saved')}</span>}
        </div>
      )}
    </div>
  );
}
