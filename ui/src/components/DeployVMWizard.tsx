import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Shield } from 'lucide-react';
import clsx from 'clsx';
import {
  deployVM,
  listNetworks,
  listSSHKeys,
  listVolumes,
  listSecurityGroups,
  createSecurityGroup,
  listVMTemplates,
  listServiceOfferings,
  listVMs,
  type PlatformVM,
} from '../lib/platform-api';
import {
  isWindowsTemplate,
  isDeployableImage,
  offeringsForTemplate,
  offeringLabel,
  findOfferingByName,
} from '../lib/offerings';
import { Modal } from './Modal';
import { SGRulesEditor, defaultSGRules } from './SGRulesEditor';
import { queryKeys } from '../lib/query-keys';
import { useI18n } from '../lib/i18n';
import { isIsolatedNetwork } from '../lib/networks';
import {
  formInputClass,
  formSelectClass,
  formTextareaClass,
  InfoBanner,
} from './shell';
import { StatusBadge } from './StatusBadge';
import { ComingSoonBadge } from './ComingSoonBadge';
import { CloudInitEditor } from './CloudInitEditor';
import {
  DEPLOY_PHASE_ORDER,
  deployPhaseFromVm,
  type DeployPhase,
} from '../lib/vm-display';
import { maskCloudInitSecrets } from '../lib/error-catalog';

function optionCardClass(selected: boolean) {
  return clsx(
    'flex-1 border rounded-lg p-3 cursor-pointer transition-colors inner-glow',
    selected ? 'border-primary-container bg-primary-container/10' : 'border-outline-variant hover:border-primary-container/40',
  );
}

type Step = 'compute' | 'disk' | 'network' | 'access' | 'review';
const STEPS: Step[] = ['compute', 'disk', 'network', 'access', 'review'];

type NetworkBackend = 'pod' | 'multus';

type FormState = {
  name: string;
  template_id: string;
  offering: string;
  dedicated_cpu: boolean;
  network_mode: 'private' | 'public';
  network_backend: NetworkBackend;
  network_ids: string[];
  security_group_ids: string[];
  ssh_key_id: string;
  data_volume_id: string;
};

const emptyForm = (): FormState => ({
  name: '',
  template_id: '',
  offering: '',
  dedicated_cpu: false,
  network_mode: 'private',
  network_backend: 'multus',
  network_ids: [],
  security_group_ids: [],
  ssh_key_id: '',
  data_volume_id: '',
});

type Props = {
  open: boolean;
  onClose: () => void;
  /** Prefill from an existing VM ("Deploy like this"). */
  cloneFrom?: PlatformVM | null;
};

export function DeployVMWizard({ open, onClose, cloneFrom = null }: Props) {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [step, setStep] = useState<Step>('compute');
  const [form, setForm] = useState<FormState>(emptyForm);
  const [createSgModal, setCreateSgModal] = useState(false);
  const [sgForm, setSgForm] = useState({ name: '', description: '', rules: defaultSGRules() });
  const [trackingName, setTrackingName] = useState<string | null>(null);
  const [cloudInitDraft, setCloudInitDraft] = useState('#cloud-config\n');
  const [showAdvanced, setShowAdvanced] = useState(false);

  const { data: netData } = useQuery({ queryKey: queryKeys.networks, queryFn: listNetworks, enabled: open });
  const { data: sshData } = useQuery({ queryKey: queryKeys.sshKeys, queryFn: listSSHKeys, enabled: open });
  const { data: volData } = useQuery({ queryKey: queryKeys.volumes, queryFn: listVolumes, enabled: open });
  const { data: sgData } = useQuery({ queryKey: queryKeys.securityGroups, queryFn: listSecurityGroups, enabled: open });
  const { data: tmplData } = useQuery({ queryKey: queryKeys.templates, queryFn: listVMTemplates, enabled: open });
  const { data: offeringsData } = useQuery({ queryKey: queryKeys.offerings, queryFn: listServiceOfferings, enabled: open });

  const templates = (tmplData?.vm_templates || []).filter(isDeployableImage);
  const selectedTemplate = templates.find((tmpl) => tmpl.id === form.template_id) || null;
  const linuxTemplates = templates.filter((tmpl) => !isWindowsTemplate(tmpl));
  const windowsTemplates = templates.filter((tmpl) => isWindowsTemplate(tmpl));
  const networks = netData?.networks || [];
  const privateNetworks = networks.filter(isIsolatedNetwork);
  const securityGroups = sgData?.security_groups || [];
  const defaultSg = securityGroups.find((sg) => sg.name === 'default');
  const offerings = offeringsData?.service_offerings || [];
  const templateOfferings = offeringsForTemplate(offerings, selectedTemplate);
  const sshKeys = sshData?.ssh_keys || [];
  const volumes = (volData?.volumes || []).filter((v) => !v.vm_id);
  const linux = !isWindowsTemplate(selectedTemplate);
  const isPublic = form.network_mode === 'public';

  const { data: trackData } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !!trackingName,
    refetchInterval: trackingName ? 2_000 : false,
  });
  const trackedVm: PlatformVM | undefined = (trackData?.vms || []).find((vm) => vm.name === trackingName);
  const phase = deployPhaseFromVm(trackedVm);

  useEffect(() => {
    if (!open) {
      setStep('compute');
      setForm(emptyForm());
      setTrackingName(null);
      setCloudInitDraft('#cloud-config\n');
      setShowAdvanced(false);
    }
  }, [open]);

  useEffect(() => {
    if (!open || !cloneFrom || templates.length === 0) return;
    const byName = templates.find((tmpl) => tmpl.name === cloneFrom.template);
    const byId = templates.find((tmpl) => tmpl.id === cloneFrom.template);
    const tmpl = byName || byId;
    const baseName = `${cloneFrom.name}-clone`.replace(/[^a-z0-9-]/g, '-').slice(0, 48);
    setForm((f) => ({
      ...f,
      name: baseName,
      template_id: tmpl?.id || f.template_id,
      offering: cloneFrom.service_offering_id || f.offering,
    }));
    setStep('compute');
  }, [open, cloneFrom, templates]);

  useEffect(() => {
    if (!open || form.network_mode !== 'public' || form.security_group_ids.length > 0) return;
    if (defaultSg) setForm((f) => ({ ...f, security_group_ids: [defaultSg.id] }));
  }, [open, form.network_mode, form.security_group_ids.length, defaultSg?.id]);

  useEffect(() => {
    if (!open || offerings.length === 0) return;
    if (cloneFrom?.service_offering_id && offerings.some((o) => o.id === cloneFrom.service_offering_id)) {
      return;
    }
    const available = offeringsForTemplate(offerings, selectedTemplate);
    if (available.length === 0) return;
    if (!available.some((o) => o.id === form.offering)) {
      const preferred = findOfferingByName(available, isWindowsTemplate(selectedTemplate) ? 'windows-large' : 'small');
      const pick = preferred || available[0];
      setForm((f) => ({ ...f, offering: pick.id, dedicated_cpu: !!pick.dedicated_cpu }));
    }
  }, [open, offerings, selectedTemplate, form.offering, cloneFrom?.service_offering_id]);

  useEffect(() => {
    if (!open || form.template_id || templates.length === 0 || cloneFrom) return;
    const preferred = templates.find((tmpl) => tmpl.name === 'ubuntu-2204') || templates.find((tmpl) => !isWindowsTemplate(tmpl));
    if (preferred) setForm((f) => ({ ...f, template_id: preferred.id }));
  }, [open, form.template_id, templates, cloneFrom]);

  const costEstimate = useMemo(() => {
    const off = templateOfferings.find((o) => o.id === form.offering) || templateOfferings[0];
    if (!off) return null;
    const disk = selectedTemplate?.boot_disk_size_gi ?? 20;
    return `${off.cpu} vCPU · ${(off.memory_mi / 1024).toFixed(off.memory_mi % 1024 === 0 ? 0 : 1)} GiB · disk ${disk} GiB`;
  }, [form.offering, templateOfferings, selectedTemplate]);

  const prechecks = useMemo(() => {
    const issues: string[] = [];
    if (!form.name.trim()) issues.push(t('common.name'));
    if (!form.template_id) issues.push(t('vms.precheckMissingTemplate'));
    if (!form.offering) issues.push(t('vms.precheckMissingOffering'));
    if (linux && !form.ssh_key_id) issues.push(t('vms.precheckMissingSsh'));
    if (isPublic && form.security_group_ids.length === 0) issues.push(t('vms.precheckMissingSg'));
    if (!isPublic && form.network_backend === 'multus' && form.network_ids.length === 0 && privateNetworks.length > 0) {
      issues.push(t('vms.precheckMissingMultus'));
    }
    if (!isPublic && form.network_backend === 'multus' && privateNetworks.length === 0) {
      issues.push(t('vms.noPrivateSubnets'));
    }
    return issues;
  }, [form, linux, isPublic, privateNetworks.length, t]);

  const createSgMutation = useMutation({
    mutationFn: createSecurityGroup,
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.securityGroups });
      setForm((f) => ({ ...f, security_group_ids: [...f.security_group_ids, res.security_group.id] }));
      setCreateSgModal(false);
      setSgForm({ name: '', description: '', rules: defaultSGRules() });
    },
  });

  const deployMutation = useMutation({
    mutationFn: deployVM,
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.vms });
      setTrackingName(res.vm.name);
    },
  });

  const stepIndex = STEPS.indexOf(step);
  const stepLabel = (s: Step) => {
    switch (s) {
      case 'compute': return t('vms.stepCompute');
      case 'disk': return t('vms.stepDisk');
      case 'network': return t('vms.stepNetwork');
      case 'access': return t('vms.stepAccess');
      case 'review': return t('vms.stepReview');
      default: {
        const _exhaustive: never = s;
        return _exhaustive;
      }
    }
  };

  const phaseLabel = (p: DeployPhase) => {
    switch (p) {
      case 'creating': return t('vms.phaseCreating');
      case 'scheduling': return t('vms.phaseScheduling');
      case 'networking': return t('vms.phaseNetworking');
      case 'running': return t('vms.phaseRunning');
      case 'error': return 'Error';
      case 'unknown': return t('common.state');
      default: {
        const _exhaustive: never = p;
        return _exhaustive;
      }
    }
  };

  const canGoNext = () => {
    if (step === 'compute') return !!form.name && !!form.template_id && !!form.offering;
    if (step === 'access') return !linux || !!form.ssh_key_id;
    if (step === 'network') {
      if (isPublic) return form.security_group_ids.length > 0;
      if (form.network_backend === 'multus') return form.network_ids.length > 0 || privateNetworks.length === 0;
      return true;
    }
    return true;
  };

  const handleDeploy = () => {
    if (prechecks.length > 0) return;
    const offering = offerings.find((o) => o.id === form.offering) || templateOfferings[0];
    if (!offering) return;

    const networkIds =
      isPublic || form.network_backend === 'multus'
        ? form.network_ids
        : [];

    deployMutation.mutate({
      name: form.name,
      template_id: form.template_id,
      service_offering_id: offering.id,
      cpu: offering.cpu,
      memory_mi: offering.memory_mi,
      dedicated_cpu: form.dedicated_cpu || !!offering.dedicated_cpu,
      ...(networkIds.length ? { network_ids: networkIds } : {}),
      ...(isPublic ? { public_ip: true, security_group_ids: form.security_group_ids } : {}),
      ...(linux ? { ssh_key_id: form.ssh_key_id } : {}),
      ...(linux && form.data_volume_id ? { data_volume_id: form.data_volume_id } : {}),
    });
  };

  const closeAll = () => {
    onClose();
  };

  return (
    <>
      <Modal
        isOpen={open}
        onClose={closeAll}
        title={cloneFrom ? `${t('vms.clone')} — ${cloneFrom.display_name || cloneFrom.name}` : t('vms.deployModalTitle')}
        size="lg"
      >
        {trackingName ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="font-medium text-on-surface">{t('vms.deployProgress')}</p>
                <p className="text-sm text-on-surface-variant font-data-mono mt-1">{trackingName}</p>
              </div>
              {trackedVm && <StatusBadge status={trackedVm.state || 'creating'} />}
            </div>

            <ol className="space-y-2">
              {DEPLOY_PHASE_ORDER.map((p) => {
                const currentIdx = DEPLOY_PHASE_ORDER.indexOf(phase === 'error' || phase === 'unknown' ? 'creating' : phase);
                const idx = DEPLOY_PHASE_ORDER.indexOf(p);
                const done = phase === 'running' || (phase !== 'error' && idx < currentIdx);
                const active = phase === p || (phase === 'error' && p === 'creating');
                return (
                  <li
                    key={p}
                    className={clsx(
                      'flex items-center gap-3 rounded-lg border px-3 py-2 text-sm',
                      done && 'border-success/30 bg-success-muted/40 text-on-surface',
                      active && !done && 'border-primary-container bg-primary-container/10 text-on-surface',
                      !done && !active && 'border-outline-variant text-on-surface-variant',
                    )}
                  >
                    <span className={clsx('w-5 h-5 rounded-full flex items-center justify-center text-xs', done ? 'bg-success text-on-success' : 'bg-surface-container-high')}>
                      {done ? <Check size={12} /> : idx + 1}
                    </span>
                    {phaseLabel(p)}
                  </li>
                );
              })}
            </ol>

            {trackedVm?.error_message && (
              <InfoBanner variant="warning">{trackedVm.error_message}</InfoBanner>
            )}
            {phase === 'error' && !trackedVm?.error_message && (
              <InfoBanner variant="warning">Deploy falhou — abra a VM para ver detalhes.</InfoBanner>
            )}

            <div className="flex justify-end gap-3 pt-2">
              <button type="button" onClick={closeAll} className="btn-secondary">{t('common.cancel')}</button>
              <Link to={`/vms/${trackingName}`} onClick={closeAll} className="btn-primary">
                {t('vms.viewVm')}
              </Link>
            </div>
          </div>
        ) : (
          <div className="space-y-5">
            <div className="flex flex-wrap gap-2">
              {STEPS.map((s, i) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => i <= stepIndex && setStep(s)}
                  className={clsx(
                    'px-3 py-1.5 rounded-lg text-xs font-mono border transition-colors',
                    s === step
                      ? 'border-primary-container bg-primary-container/15 text-on-surface'
                      : i < stepIndex
                        ? 'border-outline-variant text-on-surface hover:bg-surface-variant'
                        : 'border-outline-variant/60 text-on-surface-variant',
                  )}
                >
                  {i + 1}. {stepLabel(s)}
                </button>
              ))}
            </div>

            {step === 'compute' && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
                  <input
                    type="text"
                    required
                    pattern="[-a-z0-9]+"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
                    className={formInputClass}
                    placeholder="web-server-01"
                  />
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('common.image')}</label>
                    <select
                      required
                      value={form.template_id}
                      onChange={(e) => {
                        const template_id = e.target.value;
                        const tmpl = templates.find((x) => x.id === template_id) || null;
                        const available = offeringsForTemplate(offerings, tmpl);
                        const preferred = findOfferingByName(
                          available,
                          isWindowsTemplate(tmpl) ? 'windows-large' : 'small',
                        );
                        setForm({
                          ...form,
                          template_id,
                          offering: preferred?.id || available[0]?.id || '',
                        });
                      }}
                      className={formSelectClass}
                    >
                      <option value="">{t('vms.selectTemplate')}</option>
                      {linuxTemplates.length > 0 && (
                        <optgroup label="Linux">
                          {linuxTemplates.map((tmpl) => (
                            <option key={tmpl.id} value={tmpl.id}>{tmpl.display_name}</option>
                          ))}
                        </optgroup>
                      )}
                      {windowsTemplates.length > 0 && (
                        <optgroup label="Windows">
                          {windowsTemplates.map((tmpl) => (
                            <option key={tmpl.id} value={tmpl.id}>{tmpl.display_name}</option>
                          ))}
                        </optgroup>
                      )}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Offering</label>
                    <select
                      value={form.offering}
                      onChange={(e) => {
                        const offering = offerings.find((o) => o.id === e.target.value);
                        setForm({
                          ...form,
                          offering: e.target.value,
                          dedicated_cpu: !!offering?.dedicated_cpu,
                        });
                      }}
                      className={formSelectClass}
                    >
                      {templateOfferings.map((o) => <option key={o.id} value={o.id}>{offeringLabel(o)}</option>)}
                    </select>
                    <label className="mt-3 flex items-start gap-2 text-sm cursor-pointer">
                      <input
                        type="checkbox"
                        className="mt-1"
                        checked={form.dedicated_cpu}
                        onChange={(e) => setForm({ ...form, dedicated_cpu: e.target.checked })}
                      />
                      <span>
                        <span className="font-medium">{t('vms.dedicatedCpu')}</span>
                        <p className="text-xs text-on-surface-variant mt-0.5">{t('vms.dedicatedCpuHint')}</p>
                      </span>
                    </label>
                  </div>
                </div>
              </div>
            )}

            {step === 'disk' && (
              <div className="space-y-3">
                {linux ? (
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('vms.dataVolumeOptional')}</label>
                    <select
                      value={form.data_volume_id}
                      onChange={(e) => setForm({ ...form, data_volume_id: e.target.value })}
                      className={formSelectClass}
                    >
                      <option value="">{t('common.none')}</option>
                      {volumes.map((v) => (
                        <option key={v.id} value={v.id}>{v.name} ({v.size_gi} Gi)</option>
                      ))}
                    </select>
                    <p className="text-xs text-on-surface-variant mt-1">{t('ssh.dataVolumeHint')}</p>
                  </div>
                ) : (
                  <InfoBanner>Windows: disco de dados adicional não suportado neste fluxo.</InfoBanner>
                )}
              </div>
            )}

            {step === 'network' && (
              <div className="space-y-4">
                <div className="space-y-3 rounded-lg border border-outline-variant p-4 inner-glow">
                  <label className="block text-sm font-medium">{t('vms.networkMode')}</label>
                  <div className="flex flex-col sm:flex-row gap-3">
                    <label className={optionCardClass(form.network_mode === 'private')}>
                      <input
                        type="radio"
                        name="network_mode"
                        className="mr-2"
                        checked={form.network_mode === 'private'}
                        onChange={() => setForm({ ...form, network_mode: 'private', security_group_ids: [] })}
                      />
                      <span className="font-medium">{t('vms.networkModePrivate')}</span>
                      <p className="text-xs text-on-surface-variant mt-1 ml-5">{t('vms.networkModePrivateHint')}</p>
                    </label>
                    <label className={optionCardClass(form.network_mode === 'public')}>
                      <input
                        type="radio"
                        name="network_mode"
                        className="mr-2"
                        checked={form.network_mode === 'public'}
                        onChange={() => setForm({ ...form, network_mode: 'public', network_backend: 'multus' })}
                      />
                      <span className="font-medium">{t('vms.networkModePublic')}</span>
                      <p className="text-xs text-on-surface-variant mt-1 ml-5">{t('vms.networkModePublicHint')}</p>
                    </label>
                  </div>
                </div>

                {form.network_mode === 'private' && (
                  <div className="space-y-3 rounded-lg border border-outline-variant p-4 inner-glow">
                    <label className="block text-sm font-medium">{t('vms.networkBackend')}</label>
                    <div className="flex flex-col sm:flex-row gap-3">
                      <label className={optionCardClass(form.network_backend === 'pod')}>
                        <input
                          type="radio"
                          name="network_backend"
                          className="mr-2"
                          checked={form.network_backend === 'pod'}
                          onChange={() => setForm({ ...form, network_backend: 'pod', network_ids: [] })}
                        />
                        <span className="font-medium">{t('vms.networkPod')}</span>
                        <p className="text-xs text-on-surface-variant mt-1 ml-5">{t('vms.networkPodHint')}</p>
                      </label>
                      <label className={optionCardClass(form.network_backend === 'multus')}>
                        <input
                          type="radio"
                          name="network_backend"
                          className="mr-2"
                          checked={form.network_backend === 'multus'}
                          onChange={() => setForm({ ...form, network_backend: 'multus' })}
                        />
                        <span className="font-medium">{t('vms.networkMultus')}</span>
                        <p className="text-xs text-on-surface-variant mt-1 ml-5">{t('vms.networkMultusHint')}</p>
                      </label>
                    </div>
                    {form.network_backend === 'multus' && (
                      <div>
                        {privateNetworks.length === 0 ? (
                          <InfoBanner variant="warning">{t('vms.noPrivateSubnets')}</InfoBanner>
                        ) : (
                          <>
                            <label className="block text-sm font-medium mb-1">{t('vms.privateSubnetRequired')}</label>
                            <select
                              multiple
                              value={form.network_ids}
                              onChange={(e) => {
                                const selected = Array.from(e.target.selectedOptions, (o) => o.value);
                                setForm({ ...form, network_ids: selected });
                              }}
                              className={clsx(formSelectClass, 'min-h-[88px] !h-auto')}
                            >
                              {privateNetworks.map((n) => (
                                <option key={n.id} value={n.id}>{n.name} ({n.cidr})</option>
                              ))}
                            </select>
                            <p className="text-xs text-on-surface-variant mt-1">{t('vms.multiSelectHint')}</p>
                          </>
                        )}
                      </div>
                    )}
                  </div>
                )}

                {form.network_mode === 'public' && (
                  <div className="space-y-3">
                    <div className="flex items-center justify-between mb-1">
                      <label className="block text-sm font-medium">{t('vms.securityGroupRequired')}</label>
                      <button
                        type="button"
                        onClick={() => setCreateSgModal(true)}
                        className="btn-ghost-brand flex items-center gap-1"
                      >
                        <Shield size={14} /> {t('vms.createSecurityGroup')}
                      </button>
                    </div>
                    <select
                      multiple
                      required
                      value={form.security_group_ids}
                      onChange={(e) => {
                        const selected = Array.from(e.target.selectedOptions, (o) => o.value);
                        setForm({ ...form, security_group_ids: selected });
                      }}
                      className={clsx(formSelectClass, 'min-h-[88px] !h-auto')}
                    >
                      {securityGroups.map((sg) => (
                        <option key={sg.id} value={sg.id}>
                          {sg.name}{sg.name === 'default' ? ` (${t('sg.defaultBadge')})` : ''}
                        </option>
                      ))}
                    </select>
                    {privateNetworks.length > 0 && (
                      <div>
                        <label className="block text-sm font-medium mb-1">{t('vms.privateSubnetsOptional')}</label>
                        <select
                          multiple
                          value={form.network_ids}
                          onChange={(e) => {
                            const selected = Array.from(e.target.selectedOptions, (o) => o.value);
                            setForm({ ...form, network_ids: selected });
                          }}
                          className={clsx(formSelectClass, 'min-h-[72px] !h-auto')}
                        >
                          {privateNetworks.map((n) => (
                            <option key={n.id} value={n.id}>{n.name} ({n.cidr})</option>
                          ))}
                        </select>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}

            {step === 'access' && (
              <div className="space-y-3">
                {linux ? (
                  <div>
                    <label className="block text-sm font-medium mb-1">{t('vms.sshKeyRequired')}</label>
                    <select
                      required
                      value={form.ssh_key_id}
                      onChange={(e) => setForm({ ...form, ssh_key_id: e.target.value })}
                      className={formSelectClass}
                    >
                      <option value="">{t('common.noneFem')}</option>
                      {sshKeys.map((k) => (
                        <option key={k.id} value={k.id}>
                          {k.fingerprint ? `${k.name} (${k.fingerprint})` : k.name}
                        </option>
                      ))}
                    </select>
                    <p className="text-xs text-on-surface-variant mt-1">
                      {sshKeys.length === 0 ? t('vms.sshKeyMissing') : t('ssh.deployHint')}{' '}
                      <Link to="/ssh-keys" className="text-primary hover:underline">{t('vms.manageKeys')}</Link>
                    </p>
                  </div>
                ) : (
                  <InfoBanner>Windows: acesso via console VNC após o boot.</InfoBanner>
                )}
              </div>
            )}

            {step === 'review' && (
              <div className="space-y-4">
                {cloneFrom && (
                  <InfoBanner>
                    {t('vms.cloneFromHint').replace('{name}', cloneFrom.display_name || cloneFrom.name)}
                  </InfoBanner>
                )}
                <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                  <div><dt className="text-on-surface-variant">{t('common.name')}</dt><dd className="font-data-mono">{form.name || '—'}</dd></div>
                  <div><dt className="text-on-surface-variant">{t('common.image')}</dt><dd>{selectedTemplate?.display_name || '—'}</dd></div>
                  <div><dt className="text-on-surface-variant">Offering</dt><dd>{(() => {
                    const off = templateOfferings.find((o) => o.id === form.offering) || templateOfferings[0];
                    return off ? offeringLabel(off) : '—';
                  })()}</dd></div>
                  <div>
                    <dt className="text-on-surface-variant">{t('vms.networkMode')}</dt>
                    <dd>
                      {isPublic ? t('vms.networkModePublic') : form.network_backend === 'pod' ? t('vms.networkPod') : t('vms.networkMultus')}
                    </dd>
                  </div>
                  {linux && (
                    <div>
                      <dt className="text-on-surface-variant">SSH</dt>
                      <dd>{sshKeys.find((k) => k.id === form.ssh_key_id)?.name || '—'}</dd>
                    </div>
                  )}
                  {costEstimate && (
                    <div className="sm:col-span-2">
                      <dt className="text-on-surface-variant">{t('vms.costEstimate')}</dt>
                      <dd className="font-data-mono text-primary">{costEstimate}</dd>
                    </div>
                  )}
                </dl>
                <div className="rounded-lg border border-outline-variant p-3 space-y-2">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium">{t('cloudinit.reviewTitle')}</p>
                    <ComingSoonBadge label={t('preview.localOnly')} />
                  </div>
                  <pre className="text-[11px] font-data-mono whitespace-pre-wrap max-h-28 overflow-y-auto text-on-surface-variant">
                    {maskCloudInitSecrets(cloudInitDraft) || '—'}
                  </pre>
                </div>
                <button
                  type="button"
                  className="btn-ghost-muted text-sm"
                  onClick={() => setShowAdvanced((v) => !v)}
                >
                  {showAdvanced ? t('vms.hideAdvanced') : t('vms.showAdvanced')}
                </button>
                {showAdvanced && (
                  <div className="space-y-3 rounded-lg border border-dashed border-outline-variant p-3">
                    <CloudInitEditor
                      vmName={form.name || 'draft'}
                      reviewOnly
                      initialValue={cloudInitDraft}
                      onChange={setCloudInitDraft}
                    />
                    <div className="flex flex-wrap gap-2 text-xs text-on-surface-variant">
                      <span className="inline-flex items-center gap-1 border border-outline-variant rounded px-2 py-1">
                        {t('vms.affinity')} <ComingSoonBadge />
                      </span>
                      <span className="inline-flex items-center gap-1 border border-outline-variant rounded px-2 py-1">
                        GPU / host-device <ComingSoonBadge />
                      </span>
                    </div>
                  </div>
                )}
                <div className="rounded-lg border border-outline-variant p-3">
                  <p className="text-sm font-medium mb-2">{t('vms.precheckTitle')}</p>
                  {prechecks.length === 0 ? (
                    <p className="text-sm text-success">{t('vms.precheckOk')}</p>
                  ) : (
                    <ul className="text-sm text-error list-disc pl-5 space-y-1">
                      {prechecks.map((issue) => <li key={issue}>{issue}</li>)}
                    </ul>
                  )}
                </div>
                {deployMutation.isError && (
                  <p className="text-error text-sm">{(deployMutation.error as Error)?.message}</p>
                )}
              </div>
            )}

            <div className="flex justify-between gap-3 pt-2 border-t border-outline-variant">
              <button
                type="button"
                className="btn-secondary"
                disabled={stepIndex === 0}
                onClick={() => setStep(STEPS[Math.max(0, stepIndex - 1)])}
              >
                {t('vms.back')}
              </button>
              <div className="flex gap-2">
                <button type="button" onClick={closeAll} className="btn-ghost-muted">{t('common.cancel')}</button>
                {step !== 'review' ? (
                  <button
                    type="button"
                    className="btn-primary"
                    disabled={!canGoNext()}
                    onClick={() => setStep(STEPS[Math.min(STEPS.length - 1, stepIndex + 1)])}
                  >
                    {t('vms.next')}
                  </button>
                ) : (
                  <button
                    type="button"
                    className="btn-primary"
                    disabled={deployMutation.isPending || prechecks.length > 0}
                    onClick={handleDeploy}
                  >
                    {deployMutation.isPending ? t('common.deploying') : 'Deploy'}
                  </button>
                )}
              </div>
            </div>
          </div>
        )}
      </Modal>

      <Modal isOpen={createSgModal} onClose={() => setCreateSgModal(false)} title={t('vms.createSecurityGroup')}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createSgMutation.mutate({
              name: sgForm.name,
              description: sgForm.description,
              rules: sgForm.rules,
            });
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">{t('common.name')}</label>
            <input
              required
              value={sgForm.name}
              onChange={(e) => setSgForm({ ...sgForm, name: e.target.value })}
              className={formInputClass}
              placeholder="web-servers"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">{t('sg.description')}</label>
            <textarea
              value={sgForm.description}
              onChange={(e) => setSgForm({ ...sgForm, description: e.target.value })}
              className={formTextareaClass}
              rows={2}
            />
          </div>
          <SGRulesEditor rules={sgForm.rules} onChange={(rules) => setSgForm({ ...sgForm, rules })} />
          {createSgMutation.isError && (
            <p className="text-error text-sm">{(createSgMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateSgModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={createSgMutation.isPending} className="btn-primary">{t('common.create')}</button>
          </div>
        </form>
      </Modal>
    </>
  );
}
