import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Play, Trash2, Plus, Power, Monitor, Camera, Shield } from 'lucide-react';
import clsx from 'clsx';
import {
  listVMs, startVM, stopVM, deleteVM, deployVM, createVMSnapshot, listNetworks,
  listSSHKeys, listVolumes, listSecurityGroups, createSecurityGroup, listVMTemplates,
  listServiceOfferings, PlatformVM, VMTemplate,
} from '../lib/platform-api';
import {
  isWindowsTemplate, offeringsForTemplate, offeringLabel, findOfferingByName,
} from '../lib/offerings';
import { Modal } from '../components/Modal';
import { SGRulesEditor, defaultSGRules } from '../components/SGRulesEditor';
import { openConsole } from '../lib/console-url';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional } from '../hooks/useRealtimeEvents';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import { isIsolatedNetwork } from '../lib/networks';
import { PageHeader, SurfaceCard, SearchField, TenantRequiredNotice, formInputClass, formSelectClass, formTextareaClass } from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

function optionCardClass(selected: boolean) {
  return clsx(
    'flex-1 border rounded-lg p-3 cursor-pointer transition-colors inner-glow',
    selected ? 'border-primary-container bg-primary-container/10' : 'border-outline-variant hover:border-primary-container/40',
  );
}

function fmtMem(mi: number) {
  if (mi >= 1024) return `${(mi / 1024).toFixed(1)} GB`;
  return `${mi} MiB`;
}

export function VMs() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [deployModal, setDeployModal] = useState(false);
  const [createSgModal, setCreateSgModal] = useState(false);
  const [sgForm, setSgForm] = useState({ name: '', description: '', rules: defaultSGRules() });
  const [snapshotModal, setSnapshotModal] = useState<{ vmName: string } | null>(null);
  const [snapshotForm, setSnapshotForm] = useState({ name: '' });

  const [form, setForm] = useState({
    name: '',
    template_id: '',
    offering: '',
    network_mode: 'private' as 'private' | 'public',
    network_ids: [] as string[],
    security_group_ids: [] as string[],
    ssh_key_id: '',
    data_volume_id: '',
  });
  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();
  const { data: netData } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant && deployModal,
  });
  const { data: sshData } = useQuery({
    queryKey: queryKeys.sshKeys,
    queryFn: listSSHKeys,
    enabled: !needsTenant && deployModal,
  });
  const { data: volData } = useQuery({
    queryKey: queryKeys.volumes,
    queryFn: listVolumes,
    enabled: !needsTenant && deployModal,
  });
  const { data: sgData } = useQuery({
    queryKey: queryKeys.securityGroups,
    queryFn: listSecurityGroups,
    enabled: !needsTenant && deployModal,
  });
  const { data: tmplData } = useQuery({
    queryKey: queryKeys.templates,
    queryFn: listVMTemplates,
    enabled: !needsTenant && deployModal,
  });
  const templates = tmplData?.vm_templates || [];
  const selectedTemplate = templates.find((tmpl) => tmpl.id === form.template_id) || null;
  const linuxTemplates = templates.filter((tmpl) => !isWindowsTemplate(tmpl));
  const windowsTemplates = templates.filter((tmpl) => isWindowsTemplate(tmpl));
  const networks = netData?.networks || [];
  const privateNetworks = networks.filter(isIsolatedNetwork);
  const securityGroups = sgData?.security_groups || [];
  const defaultSg = securityGroups.find((sg) => sg.name === 'default');

  useEffect(() => {
    if (!deployModal || form.network_mode !== 'public' || form.security_group_ids.length > 0) return;
    if (defaultSg) {
      setForm((f) => ({ ...f, security_group_ids: [defaultSg.id] }));
    }
  }, [deployModal, form.network_mode, form.security_group_ids.length, defaultSg?.id]);

  const { data: offeringsData } = useQuery({
    queryKey: queryKeys.offerings,
    queryFn: listServiceOfferings,
    enabled: !needsTenant && deployModal,
  });
  const offerings = offeringsData?.service_offerings || [];
  const templateOfferings = offeringsForTemplate(offerings, selectedTemplate);

  useEffect(() => {
    if (!deployModal || offerings.length === 0) return;
    const available = offeringsForTemplate(offerings, selectedTemplate);
    if (available.length === 0) return;
    if (!available.some((o) => o.id === form.offering)) {
      const preferred = findOfferingByName(available, isWindowsTemplate(selectedTemplate) ? 'windows-large' : 'small');
      setForm((f) => ({ ...f, offering: preferred?.id || available[0].id }));
    }
  }, [deployModal, offerings, selectedTemplate, form.offering]);

  useEffect(() => {
    if (!deployModal || form.template_id || templates.length === 0) return;
    const preferred = templates.find((tmpl) => tmpl.name === 'ubuntu-2204') || templates.find((tmpl) => !isWindowsTemplate(tmpl));
    if (preferred) {
      setForm((f) => ({ ...f, template_id: preferred.id }));
    }
  }, [deployModal, form.template_id, templates]);

  const sshKeys = sshData?.ssh_keys || [];
  const volumes = volData?.volumes || [];

  const { data, isLoading, isFetching, isRefetching, refetch, error, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
    refetchInterval: (q) => {
      const vms = q.state.data?.vms || [];
      if (vms.some((vm) => isVMTransitional(vm.state))) return 3_000;
      return false;
    },
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.vms });

  const startMutation = useMutation({ mutationFn: startVM, onSuccess: invalidate });
  const stopMutation = useMutation({ mutationFn: stopVM, onSuccess: invalidate });
  const destroyMutation = useMutation({ mutationFn: deleteVM, onSuccess: invalidate });
  const snapshotMutation = useMutation({
    mutationFn: createVMSnapshot,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.vmSnapshots });
      setSnapshotModal(null);
      setSnapshotForm({ name: '' });
    },
  });
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
    onSuccess: () => {
      invalidate();
      setDeployModal(false);
      setForm({
        name: '', template_id: '', offering: '',
        network_mode: 'private', network_ids: [], security_group_ids: [],
        ssh_key_id: '', data_volume_id: '',
      });
    },
  });

  const vms = data?.vms || [];
  const filteredVMs = vms.filter((vm: PlatformVM) =>
    vm.name?.toLowerCase().includes(search.toLowerCase()) ||
    vm.display_name?.toLowerCase().includes(search.toLowerCase()) ||
    vm.ip?.includes(search)
  );

  const handleDeploy = (e: React.FormEvent) => {
    e.preventDefault();
    const offering = offerings.find((o) => o.id === form.offering) || templateOfferings[0];
    const linux = !isWindowsTemplate(selectedTemplate);
    const isPublic = form.network_mode === 'public';

    if (!form.template_id || !offering) {
      return;
    }
    if (isPublic && form.security_group_ids.length === 0) {
      return;
    }

    deployMutation.mutate({
      name: form.name,
      template_id: form.template_id,
      service_offering_id: offering.id,
      cpu: offering.cpu,
      memory_mi: offering.memory_mi,
      ...(form.network_ids.length ? { network_ids: form.network_ids } : {}),
      ...(isPublic ? { public_ip: true, security_group_ids: form.security_group_ids } : {}),
      ...(linux && form.ssh_key_id ? { ssh_key_id: form.ssh_key_id } : {}),
      ...(linux && form.data_volume_id ? { data_volume_id: form.data_volume_id } : {}),
    });
  };

  if (needsTenant) {
    return <TenantRequiredNotice message={t('vms.selectTenant')} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('nav.vms')}
        subtitle={`${vms.length} ${t('vms.subtitle')}`}
        actions={
          <>
            <RefreshButton
              onRefresh={() => refetch()}
              isFetching={isRefetching}
              dataUpdatedAt={dataUpdatedAt}
            />
            <button type="button" onClick={() => setDeployModal(true)} className="btn-primary">
              <Plus size={18} /> Deploy VM
            </button>
          </>
        }
      />

      {error && (
        <div className="p-3 bg-error-container/30 border border-error-container rounded-lg text-on-error-container text-sm">
          {(error as Error).message}
        </div>
      )}

      <SearchField
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder={t('vms.searchPlaceholder')}
      />

      <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
      <SurfaceCard padding="none" className="overflow-hidden">
        <div className="overflow-x-auto">
        <table className="w-full text-sm border-collapse">
          <thead className="bg-surface-container-high border-b border-card-border">
            <tr>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">{t('common.name')}</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">{t('vms.col.displayName')}</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">{t('common.state')}</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">IP</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">Zone</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">Host</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">Offering</th>
              <th className="text-left px-4 py-3 font-label text-on-surface-variant">Template</th>
              <th className="text-right px-4 py-3 font-label text-on-surface-variant">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-card-border">
            {isLoading ? (
              <tr><td colSpan={9} className="text-center py-12 text-on-surface-variant">{t('common.loading')}</td></tr>
            ) : filteredVMs.length === 0 ? (
              <tr><td colSpan={9} className="text-center py-12 text-on-surface-variant">{t('vms.empty')}</td></tr>
            ) : (
              filteredVMs.map((vm: PlatformVM) => (
                <tr key={vm.id || vm.name} className="table-row-hover">
                  <td className="px-4 py-3">
                    <Link to={`/vms/${vm.name}`} className="font-medium text-primary hover:text-primary-fixed-dim hover:underline">
                      {vm.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3">{vm.display_name || vm.name}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={vm.state || 'inactive'} />
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{vm.ip || '—'}</td>
                  <td className="px-4 py-3">{vm.zone || '—'}</td>
                  <td className="px-4 py-3 text-xs">{vm.host_name || '—'}</td>
                  <td className="px-4 py-3">{vm.cpu} vCPU / {fmtMem(vm.memory_mi)}</td>
                  <td className="px-4 py-3">{vm.template || '—'}</td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1">
                      {vm.state?.toLowerCase() === 'running' ? (
                        <button onClick={() => stopMutation.mutate(vm.name)} className="btn-icon-warning" title={t('vms.stop')}>
                          <Power size={16} />
                        </button>
                      ) : (
                        <button onClick={() => startMutation.mutate(vm.name)} className="btn-icon-success" title={t('vms.start')}>
                          <Play size={16} />
                        </button>
                      )}
                      <button
                        onClick={() => openConsole(vm.name, vm.namespace)}
                        disabled={vm.state?.toLowerCase() !== 'running'}
                        className="btn-icon-neutral"
                        title="Console"
                      >
                        <Monitor size={16} />
                      </button>
                      <button
                        onClick={() => { setSnapshotForm({ name: `${vm.name}-snap` }); setSnapshotModal({ vmName: vm.name }); }}
                        disabled={vm.state?.toLowerCase() !== 'running'}
                        className="btn-icon-neutral"
                        title="Snapshot"
                      >
                        <Camera size={16} />
                      </button>
                      <button onClick={() => destroyMutation.mutate(vm.name)} className="btn-icon-danger" title={t('vms.destroy')}>
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
        </div>
      </SurfaceCard>
      </RefreshingPanel>

      <Modal isOpen={deployModal} onClose={() => setDeployModal(false)} title={t('vms.deployModalTitle')} size="lg">
        <form onSubmit={handleDeploy} className="space-y-4">
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
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('common.image')}</label>
              <select
                required
                value={form.template_id}
                onChange={(e) => {
                  const template_id = e.target.value;
                  const tmpl = templates.find((t) => t.id === template_id) || null;
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
              <p className="text-xs text-on-surface-variant mt-1">
                <Link to="/templates" className="text-primary hover:text-primary-fixed-dim hover:underline">{t('templates.manageLink')}</Link>
              </p>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Offering</label>
              <select value={form.offering} onChange={(e) => setForm({ ...form, offering: e.target.value })} className={formSelectClass}>
                {templateOfferings.map((o) => <option key={o.id} value={o.id}>{offeringLabel(o)}</option>)}
              </select>
            </div>
          </div>
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
                  onChange={() => setForm({ ...form, network_mode: 'public' })}
                />
                <span className="font-medium">{t('vms.networkModePublic')}</span>
                <p className="text-xs text-on-surface-variant mt-1 ml-5">{t('vms.networkModePublicHint')}</p>
              </label>
            </div>

            {form.network_mode === 'private' && (
              <div>
                <p className="text-sm text-on-surface-variant mb-2">{t('vms.defaultVpcHint')}</p>
                {privateNetworks.length > 0 && (
                  <>
                    <label className="block text-sm font-medium mb-1">{t('vms.privateSubnetsOptional')}</label>
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

            {form.network_mode === 'public' && (
              <>
                <div>
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
                  <p className="text-xs text-on-surface-variant mt-1">{t('vms.multiSgHint')}</p>
                </div>
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
              </>
            )}
          </div>
          {!isWindowsTemplate(selectedTemplate) && (
            <>
              <div>
                <label className="block text-sm font-medium mb-1">{t('vms.sshKeyOptional')}</label>
                <select
                  value={form.ssh_key_id}
                  onChange={(e) => setForm({ ...form, ssh_key_id: e.target.value })}
                  className={formSelectClass}
                >
                  <option value="">{t('common.noneFem')}</option>
                  {sshKeys.map((k) => (
                    <option key={k.id} value={k.id}>{k.name} ({k.fingerprint})</option>
                  ))}
                </select>
                <p className="text-xs text-on-surface-variant mt-1">
                  {t('ssh.deployHint')}{' '}
                  <Link to="/ssh-keys" className="text-primary hover:text-primary-fixed-dim hover:underline">{t('vms.manageKeys')}</Link>
                </p>
              </div>
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
            </>
          )}
          {deployMutation.isError && (
            <p className="text-error text-sm">{(deployMutation.error as Error)?.message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setDeployModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button
              type="submit"
              disabled={
                deployMutation.isPending ||
                !form.template_id ||
                (form.network_mode === 'public' && form.security_group_ids.length === 0)
              }
              className="btn-primary"
            >
              {deployMutation.isPending ? t('common.deploying') : 'Deploy'}
            </button>
          </div>
        </form>
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

      <Modal isOpen={snapshotModal !== null} onClose={() => setSnapshotModal(null)} title={`Snapshot — ${snapshotModal?.vmName ?? ''}`}>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (!snapshotModal) return;
            snapshotMutation.mutate({ vm_name: snapshotModal.vmName, name: snapshotForm.name });
          }}
          className="space-y-4"
        >
          <input
            type="text"
            required
            pattern="[-a-z0-9]+"
            value={snapshotForm.name}
            onChange={(e) => setSnapshotForm({ name: e.target.value.toLowerCase() })}
            className={formInputClass}
          />
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setSnapshotModal(null)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={snapshotMutation.isPending} className="btn-primary">{t('common.create')}</button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
