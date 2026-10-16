import { Link } from 'react-router-dom';
import { Loader2, X } from 'lucide-react';
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

  const nextKey = steps.find((s) => !s.done)?.key;

  return (
    <SurfaceCard
      title={t('onboarding.title')}
      headerAction={
        <>
          <span className="text-[13px] font-mono text-on-surface-variant">
            {loading ? t('common.loading') : `${doneCount} / ${steps.length}`}
          </span>
          <button
            type="button"
            onClick={dismiss}
            className="p-1 rounded text-on-surface-variant hover:bg-sunk"
            aria-label={t('common.cancel')}
            title={t('onboarding.dismiss')}
          >
            <X size={16} />
          </button>
        </>
      }
    >
      <p className="m-0 mb-3 text-on-surface-variant">{t('onboarding.subtitle')}</p>
      <ul className="m-0 p-0 list-none flex flex-col">
        {steps.map((s) => {
          const next = s.key === nextKey;
          return (
            <li key={s.key}>
              <Link
                to={s.to}
                className={clsx(
                  'flex items-center gap-3 rounded-lg px-2 py-2.5 transition-colors',
                  next ? 'bg-accent-soft' : 'hover:bg-sunk',
                )}
              >
                <span
                  className={clsx('w-2 h-2 rounded-full shrink-0', s.done ? 'bg-ok-fg' : next ? 'bg-accent' : 'bg-off-fg')}
                  aria-hidden
                />
                <span className="flex-1 min-w-0 font-medium">{s.label}</span>
                {s.pending ? (
                  <Loader2 size={14} className="text-on-surface-variant shrink-0 animate-spin" aria-hidden />
                ) : (
                  <span className="font-mono text-xs text-on-surface-variant">{s.done ? '✓' : next ? '→' : ''}</span>
                )}
              </Link>
            </li>
          );
        })}
      </ul>
    </SurfaceCard>
  );
}
