import clsx from 'clsx';
import { CheckCircle2, XCircle } from 'lucide-react';
import { useI18n } from '../lib/i18n';
import { formInputClass, formSelectClass } from './shell';

export interface CIDRBlock {
  cidr: string;
  label: string;
  available: boolean;
  reason?: string;
}

interface CIDRPickerProps {
  mode: 'auto' | 'custom';
  onModeChange: (mode: 'auto' | 'custom') => void;
  value: string;
  onChange: (cidr: string) => void;
  suggestions: CIDRBlock[];
  autoValue?: string;
  loading?: boolean;
  customPlaceholder?: string;
}

export function CIDRPicker({
  mode,
  onModeChange,
  value,
  onChange,
  suggestions,
  autoValue,
  loading,
  customPlaceholder = '10.0.0.0/16',
}: CIDRPickerProps) {
  const { t } = useI18n();
  const selectedAuto = autoValue || suggestions.find((s) => s.available)?.cidr || '';

  return (
    <div className="space-y-3">
      <div className="flex gap-4 text-sm text-on-surface">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            checked={mode === 'auto'}
            onChange={() => {
              onModeChange('auto');
              if (selectedAuto) onChange(selectedAuto);
            }}
          />
          {t('cidr.autoAllocation')}
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            checked={mode === 'custom'}
            onChange={() => onModeChange('custom')}
          />
          {t('cidr.customRange')}
        </label>
      </div>

      {mode === 'auto' ? (
        <div className="rounded-lg border border-outline-variant bg-surface-container-high px-4 py-3 inner-glow">
          {loading ? (
            <p className="text-sm text-on-surface-variant">{t('cidr.calculating')}</p>
          ) : selectedAuto ? (
            <p className="font-data-mono text-sm">
              <span className="text-on-surface-variant">{t('cidr.willUse')} </span>
              {selectedAuto}
            </p>
          ) : (
            <p className="text-sm text-warning">{t('cidr.noFreeBlocks')}</p>
          )}
        </div>
      ) : (
        <>
          <input
            required
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={customPlaceholder}
            className={clsx(formInputClass, 'font-data-mono')}
          />
          {suggestions.length > 0 && (
            <div className="space-y-2">
              <p className="font-label text-on-surface-variant normal-case">{t('cidr.suggestions')}</p>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-48 overflow-y-auto">
                {suggestions.map((block) => (
                  <button
                    key={block.cidr}
                    type="button"
                    disabled={!block.available}
                    onClick={() => onChange(block.cidr)}
                    className={clsx(
                      'flex items-start gap-2 text-left px-3 py-2 rounded-lg border text-sm transition inner-glow',
                      block.available
                        ? value === block.cidr
                          ? 'border-primary-container bg-primary-container/10'
                          : 'border-outline-variant hover:border-primary-container/40'
                        : 'opacity-50 cursor-not-allowed border-outline-variant',
                    )}
                  >
                    {block.available ? (
                      <CheckCircle2 size={16} className="text-success shrink-0 mt-0.5" />
                    ) : (
                      <XCircle size={16} className="text-on-surface-variant shrink-0 mt-0.5" />
                    )}
                    <span>
                      <span className="font-data-mono block">{block.cidr}</span>
                      <span className="text-xs text-on-surface-variant">
                        {block.available ? block.label : block.reason || t('common.unavailable')}
                      </span>
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

interface SubnetPrefixSelectProps {
  prefix: number;
  onChange: (prefix: number) => void;
}

const SUBNET_PREFIXES = [24, 25, 26, 27, 28];

export function SubnetPrefixSelect({ prefix, onChange }: SubnetPrefixSelectProps) {
  const { t } = useI18n();

  return (
    <div>
      <label className="block text-sm font-medium mb-1 text-on-surface">{t('cidr.subnetSize')}</label>
      <select value={prefix} onChange={(e) => onChange(Number(e.target.value))} className={formSelectClass}>
        {SUBNET_PREFIXES.map((p) => (
          <option key={p} value={p}>
            /{p} ({Math.pow(2, 32 - p) - 2} {t('cidr.usableAddresses')})
          </option>
        ))}
      </select>
    </div>
  );
}
