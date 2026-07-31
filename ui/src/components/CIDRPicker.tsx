import { CheckCircle2, XCircle } from 'lucide-react';

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
  const selectedAuto = autoValue || suggestions.find((s) => s.available)?.cidr || '';

  return (
    <div className="space-y-3">
      <div className="flex gap-4 text-sm">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            checked={mode === 'auto'}
            onChange={() => {
              onModeChange('auto');
              if (selectedAuto) onChange(selectedAuto);
            }}
          />
          Alocação automática (recomendado)
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="radio"
            checked={mode === 'custom'}
            onChange={() => onModeChange('custom')}
          />
          Intervalo personalizado
        </label>
      </div>

      {mode === 'auto' ? (
        <div className="rounded-lg border bg-gray-50 dark:bg-dark-200 px-4 py-3">
          {loading ? (
            <p className="text-sm text-gray-500">Calculando intervalo livre...</p>
          ) : selectedAuto ? (
            <p className="font-mono text-sm">
              <span className="text-gray-500">Será usado: </span>
              {selectedAuto}
            </p>
          ) : (
            <p className="text-sm text-amber-600">Nenhum bloco livre encontrado — use personalizado.</p>
          )}
        </div>
      ) : (
        <>
          <input
            required
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={customPlaceholder}
            className="w-full px-4 py-2 border rounded-lg font-mono text-sm"
          />
          {suggestions.length > 0 && (
            <div className="space-y-2">
              <p className="text-xs text-gray-500 uppercase tracking-wide">Sugestões disponíveis</p>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-48 overflow-y-auto">
                {suggestions.map((block) => (
                  <button
                    key={block.cidr}
                    type="button"
                    disabled={!block.available}
                    onClick={() => onChange(block.cidr)}
                    className={`flex items-start gap-2 text-left px-3 py-2 rounded-lg border text-sm transition ${
                      block.available
                        ? value === block.cidr
                          ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                          : 'hover:border-blue-300'
                        : 'opacity-50 cursor-not-allowed'
                    }`}
                  >
                    {block.available ? (
                      <CheckCircle2 size={16} className="text-green-500 shrink-0 mt-0.5" />
                    ) : (
                      <XCircle size={16} className="text-gray-400 shrink-0 mt-0.5" />
                    )}
                    <span>
                      <span className="font-mono block">{block.cidr}</span>
                      <span className="text-xs text-gray-500">
                        {block.available ? block.label : block.reason || 'indisponível'}
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
  return (
    <div>
      <label className="block text-sm font-medium mb-1">Tamanho da sub-rede</label>
      <select
        value={prefix}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full px-4 py-2 border rounded-lg"
      >
        {SUBNET_PREFIXES.map((p) => (
          <option key={p} value={p}>
            /{p} ({Math.pow(2, 32 - p) - 2} endereços utilizáveis)
          </option>
        ))}
      </select>
    </div>
  );
}
