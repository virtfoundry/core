import { Link } from 'react-router-dom';
import { CheckCircle2, Circle, Loader2, X } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { listNetworks, listSSHKeys, listServiceOfferings, listVMTemplates, listVMs } from '../lib/platform-api';
import { isIsolatedNetwork } from '../lib/networks';
import { queryKeys } from '../lib/query-keys';
import { getOnboarding, saveOnboarding } from '../lib/preview-prefs';
import { useI18n } from '../lib/i18n';
import { SurfaceCard } from './shell';
import { useState } from 'react';
import clsx from 'clsx';
import { useNeedsTenant } from '../store/hooks';

export function OnboardingChecklist() {
  const { t } = useI18n();
  const needsTenant = useNeedsTenant();
  const [state, setState] = useState(() => getOnboarding());

  const enabled = !needsTenant && !state.dismissed;

  const tmplQ = useQuery({ queryKey: queryKeys.templates, queryFn: listVMTemplates, enabled });
  const offsQ = useQuery({ queryKey: queryKeys.offerings, queryFn: listServiceOfferings, enabled });
  const keysQ = useQuery({ queryKey: queryKeys.sshKeys, queryFn: listSSHKeys, enabled });
  const netsQ = useQuery({ queryKey: queryKeys.networks, queryFn: listNetworks, enabled });
  const vmsQ = useQuery({ queryKey: queryKeys.vms, queryFn: listVMs, enabled });

  if (needsTenant || state.dismissed) return null;

  const loading =
    tmplQ.isLoading || offsQ.isLoading || keysQ.isLoading || netsQ.isLoading || vmsQ.isLoading;

  const hasTemplate = (tmplQ.data?.vm_templates?.length ?? 0) > 0;
  const hasOffering = (offsQ.data?.service_offerings?.length ?? 0) > 0;
  const hasSsh = (keysQ.data?.ssh_keys?.length ?? 0) > 0;
  // DeployVMWizard Multus path needs an isolated Network CR (VPC default subnet counts).
  const hasNetwork = (netsQ.data?.networks ?? []).some(isIsolatedNetwork);
  const hasVm = (vmsQ.data?.vms?.length ?? 0) > 0;

  // Hide after first VM exists (API-backed), not a local stub flag.
  if (!loading && hasVm) return null;

  const steps = [
    { key: 'template', done: hasTemplate, pending: tmplQ.isLoading, label: t('onboarding.template'), to: '/templates' },
    { key: 'offering', done: hasOffering, pending: offsQ.isLoading, label: t('onboarding.offering'), to: '/offerings' },
    { key: 'ssh', done: hasSsh, pending: keysQ.isLoading, label: t('onboarding.ssh'), to: '/ssh-keys' },
    // Link /vpcs: creating a VPC auto-provisions the default private subnet.
    { key: 'network', done: hasNetwork, pending: netsQ.isLoading, label: t('onboarding.network'), to: '/vpcs' },
    { key: 'deploy', done: hasVm, pending: vmsQ.isLoading, label: t('onboarding.deploy'), to: '/vms' },
  ];

  const doneCount = steps.filter((s) => s.done).length;

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
      <p className="text-sm text-on-surface-variant mt-1 mb-1">{t('onboarding.subtitle')}</p>
      <p className="text-xs font-mono text-on-surface-variant mb-4">
        {loading ? t('common.loading') : t('onboarding.progress').replace('{done}', String(doneCount)).replace('{total}', String(steps.length))}
      </p>
      <ul className="space-y-2">
        {steps.map((s) => (
          <li key={s.key}>
            <Link
              to={s.to}
              className={clsx(
                'flex items-center gap-3 rounded-lg border px-3 py-2.5 text-sm transition-colors',
                s.done
                  ? 'border-success/30 bg-success-muted/30 text-on-surface'
                  : 'border-outline-variant hover:border-primary-container/50 text-on-surface',
              )}
            >
              {s.pending ? (
                <Loader2 size={18} className="text-on-surface-variant shrink-0 animate-spin" aria-hidden />
              ) : s.done ? (
                <CheckCircle2 size={18} className="text-success shrink-0" />
              ) : (
                <Circle size={18} className="text-on-surface-variant shrink-0" />
              )}
              <span className={s.done ? 'line-through opacity-70' : ''}>{s.label}</span>
            </Link>
          </li>
        ))}
      </ul>
    </SurfaceCard>
  );
}
