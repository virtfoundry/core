import { Plus, Trash2 } from 'lucide-react';
import type { SecurityGroup } from '../lib/platform-api';
import { useI18n } from '../lib/i18n';

export type SGRule = SecurityGroup['rules'][number];

export const defaultSGRules = (): SGRule[] => [
  { direction: 'ingress', protocol: 'tcp', port_from: 80, cidr: '0.0.0.0/0' },
];

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

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <label className="block text-sm font-medium">{t('sg.rules')}</label>
        <button type="button" onClick={addRule} className="text-sm text-primary-600 hover:underline flex items-center gap-1">
          <Plus size={14} /> {t('sg.addRule')}
        </button>
      </div>
      {rules.length === 0 ? (
        <p className="text-sm text-gray-500">{t('sg.noRules')}</p>
      ) : (
        rules.map((rule, index) => (
          <div key={index} className="grid grid-cols-2 md:grid-cols-6 gap-2 items-end p-3 border rounded-lg bg-gray-50 dark:bg-dark-200">
            <div>
              <label className="block text-xs text-gray-500 mb-1">{t('sg.direction')}</label>
              <select
                value={rule.direction}
                onChange={(e) => updateRule(index, { direction: e.target.value })}
                className="w-full px-2 py-1.5 border rounded text-sm"
              >
                <option value="ingress">{t('sg.ingress')}</option>
                <option value="egress">{t('sg.egress')}</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">{t('sg.protocol')}</label>
              <select
                value={rule.protocol}
                onChange={(e) => updateRule(index, { protocol: e.target.value })}
                className="w-full px-2 py-1.5 border rounded text-sm"
              >
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
                <option value="icmp">ICMP</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">{t('sg.portFrom')}</label>
              <input
                type="number"
                min={1}
                max={65535}
                value={rule.port_from ?? ''}
                onChange={(e) => updateRule(index, { port_from: Number(e.target.value) || undefined })}
                className="w-full px-2 py-1.5 border rounded text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">{t('sg.portTo')}</label>
              <input
                type="number"
                min={1}
                max={65535}
                placeholder="—"
                value={rule.port_to ?? ''}
                onChange={(e) => updateRule(index, { port_to: e.target.value ? Number(e.target.value) : undefined })}
                className="w-full px-2 py-1.5 border rounded text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">CIDR</label>
              <input
                required
                value={rule.cidr}
                onChange={(e) => updateRule(index, { cidr: e.target.value })}
                className="w-full px-2 py-1.5 border rounded text-sm"
                placeholder="0.0.0.0/0"
              />
            </div>
            <div className="flex justify-end">
              <button
                type="button"
                onClick={() => removeRule(index)}
                className="p-2 text-red-500 hover:bg-red-50 rounded"
                title={t('sg.removeRule')}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        ))
      )}
    </div>
  );
}
