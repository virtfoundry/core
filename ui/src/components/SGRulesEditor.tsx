import { Plus, Trash2 } from 'lucide-react';
import type { SecurityGroup } from '../lib/platform-api';
import { useI18n } from '../lib/i18n';
import { formInputClass, formSelectClass, formLabelClass } from './shell';

export type SGRule = SecurityGroup['rules'][number];

export const defaultSGRules = (): SGRule[] => [
  { direction: 'ingress', protocol: 'tcp', port_from: 80, cidr: '0.0.0.0/0' },
];

const compactInputClass = `${formInputClass} !h-9 !px-2 !py-1.5 text-sm`;
const compactSelectClass = `${formSelectClass} !h-9 !px-2 !py-1.5 text-sm`;

export function SGRulesEditor({
  rules,
  onChange,
}: {
  rules: SGRule[];
  onChange: (rules: SGRule[]) => void;
}) {
  const { t } = useI18n();

  const updateRule = (index: number, patch: Partial<SGRule>) => {
    onChange(rules.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  };

  const removeRule = (index: number) => {
    onChange(rules.filter((_, i) => i !== index));
  };

  const addRule = () => {
    onChange([...rules, { direction: 'ingress', protocol: 'tcp', port_from: 443, cidr: '0.0.0.0/0' }]);
  };

  const cols = 'md:grid-cols-[1fr_1fr_0.8fr_0.8fr_1.6fr_auto]';
  const mobileLabel = 'md:hidden block text-xs text-on-surface-variant mb-1';

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <label className={formLabelClass + ' !mb-0'}>{t('sg.rules')}</label>
        <button type="button" onClick={addRule} className="btn-ghost-brand flex items-center gap-1">
          <Plus size={14} /> {t('sg.addRule')}
        </button>
      </div>
      {rules.length === 0 ? (
        <p className="text-sm text-on-surface-variant">{t('sg.noRules')}</p>
      ) : (
        <div className="rounded-xl border border-card-border overflow-hidden">
          <div className={`hidden md:grid ${cols} gap-2 px-3 py-2.5 bg-sunk text-[13px] font-medium text-on-surface-variant`}>
            <span>{t('sg.direction')}</span>
            <span>{t('sg.protocol')}</span>
            <span>{t('sg.portFrom')}</span>
            <span>{t('sg.portTo')}</span>
            <span>CIDR</span>
            <span className="w-9" />
          </div>
          {rules.map((rule, index) => (
            <div
              key={index}
              className={`grid grid-cols-2 ${cols} gap-2 items-end md:items-center px-3 py-2.5 border-t border-line3 first:border-t-0 md:first:border-t`}
            >
              <div>
                <label className={mobileLabel}>{t('sg.direction')}</label>
                <select
                  value={rule.direction}
                  onChange={(e) => updateRule(index, { direction: e.target.value })}
                  className={compactSelectClass}
                >
                  <option value="ingress">{t('sg.ingress')}</option>
                  <option value="egress">{t('sg.egress')}</option>
                </select>
              </div>
              <div>
                <label className={mobileLabel}>{t('sg.protocol')}</label>
                <select
                  value={rule.protocol}
                  onChange={(e) => updateRule(index, { protocol: e.target.value })}
                  className={compactSelectClass}
                >
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="icmp">ICMP</option>
                </select>
              </div>
              <div>
                <label className={mobileLabel}>{t('sg.portFrom')}</label>
                <input
                  type="number"
                  min={1}
                  max={65535}
                  value={rule.port_from ?? ''}
                  onChange={(e) => updateRule(index, { port_from: Number(e.target.value) || undefined })}
                  className={`${compactInputClass} font-mono`}
                />
              </div>
              <div>
                <label className={mobileLabel}>{t('sg.portTo')}</label>
                <input
                  type="number"
                  min={1}
                  max={65535}
                  placeholder="—"
                  value={rule.port_to ?? ''}
                  onChange={(e) => updateRule(index, { port_to: e.target.value ? Number(e.target.value) : undefined })}
                  className={`${compactInputClass} font-mono`}
                />
              </div>
              <div>
                <label className={mobileLabel}>CIDR</label>
                <input
                  required
                  value={rule.cidr}
                  onChange={(e) => updateRule(index, { cidr: e.target.value })}
                  className={`${compactInputClass} font-mono`}
                  placeholder="0.0.0.0/0"
                />
              </div>
              <div className="flex justify-end">
                <button
                  type="button"
                  onClick={() => removeRule(index)}
                  className="btn-icon-danger p-2"
                  title={t('sg.removeRule')}
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
