import { Link } from 'react-router-dom';
import { CheckCircle2, Circle, X } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { listSSHKeys, listServiceOfferings, listVMTemplates, listVMs } from '../lib/platform-api';
import { queryKeys } from '../lib/query-keys';
import { getOnboarding, saveOnboarding } from '../lib/preview-prefs';
import { useI18n } from '../lib/i18n';
import { SurfaceCard } from './shell';
import { useState } from 'react';
import clsx from 'clsx';

export function OnboardingChecklist() {
  const { t } = useI18n();
  const [state, setState] = useState(() => getOnboarding());

  const { data: tmpl } = useQuery({ queryKey: queryKeys.templates, queryFn: listVMTemplates });
  const { data: offs } = useQuery({ queryKey: queryKeys.offerings, queryFn: listServiceOfferings });
  const { data: keys } = useQuery({ queryKey: queryKeys.sshKeys, queryFn: listSSHKeys });
  const { data: vms } = useQuery({ queryKey: queryKeys.vms, queryFn: listVMs });

  const hasTemplate = (tmpl?.vm_templates?.length ?? 0) > 0;
  const hasOffering = (offs?.service_offerings?.length ?? 0) > 0;
  const hasSsh = (keys?.ssh_keys?.length ?? 0) > 0;
  const hasVm = (vms?.vms?.length ?? 0) > 0;

  if (state.dismissed || hasVm) return null;

  const steps = [
    { done: hasTemplate, label: t('onboarding.template'), to: '/templates' },
    { done: hasOffering, label: t('onboarding.offering'), to: '/offerings' },
    { done: hasSsh, label: t('onboarding.ssh'), to: '/ssh-keys' },
    { done: hasVm, label: t('onboarding.deploy'), to: '/vms' },
  ];

  const dismiss = () => {
    const next = saveOnboarding({ dismissed: true });
    setState(next);
  };

  return (
    <SurfaceCard className="relative border-primary-container/40" padding="md">
      <button
        type="button"
        onClick={dismiss}
        className="absolute top-3 right-3 p-1 rounded text-on-surface-variant hover:bg-surface-variant"
        aria-label={t('common.cancel')}
        title={t('onboarding.dismiss')}
      >
        <X size={16} />
      </button>
      <h2 className="font-headline text-title-md font-semibold text-on-surface pr-8">{t('onboarding.title')}</h2>
      <p className="text-sm text-on-surface-variant mt-1 mb-4">{t('onboarding.subtitle')}</p>
      <ul className="space-y-2">
        {steps.map((s) => (
          <li key={s.to}>
            <Link
              to={s.to}
              className={clsx(
                'flex items-center gap-3 rounded-lg border px-3 py-2.5 text-sm transition-colors',
                s.done
                  ? 'border-success/30 bg-success-muted/30 text-on-surface'
                  : 'border-outline-variant hover:border-primary-container/50 text-on-surface',
              )}
            >
              {s.done ? <CheckCircle2 size={18} className="text-success shrink-0" /> : <Circle size={18} className="text-on-surface-variant shrink-0" />}
              <span className={s.done ? 'line-through opacity-70' : ''}>{s.label}</span>
            </Link>
          </li>
        ))}
      </ul>
    </SurfaceCard>
  );
}
