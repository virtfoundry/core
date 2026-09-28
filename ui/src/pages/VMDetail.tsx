import { Link } from 'react-router-dom';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft, Play, Power, Trash2, Monitor, Camera, Save, HardDrive, Unlink, Plus,
  AlertCircle, RefreshCw, Activity, Copy, CopyPlus, Terminal,
} from 'lucide-react';
import {
  getVM, updateVM, startVM, stopVM, deleteVM, createVMSnapshot,
  listVMSnapshots, fetchVMLogs, listServiceOfferings,
  listVMVolumes, listVolumes, attachVolumeToVM, detachVolumeFromVM,
  type PlatformVM,
} from '../lib/platform-api';
import {
  offeringLabel, findOfferingBySpec, findOfferingByName,
} from '../lib/offerings';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { DeployVMWizard } from '../components/DeployVMWizard';
import { ComingSoonBadge } from '../components/ComingSoonBadge';
import { CloudInitEditor } from '../components/CloudInitEditor';
import { openConsole } from '../lib/console-url';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional } from '../hooks/useRealtimeEvents';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TabBar, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';
import {
  DEPLOY_PHASE_ORDER,
  deployPhaseFromVm,
  formatVmOffering,
  fmtMem,
  isVmError,
  isVmRunning,
  isVmStopped,
} from '../lib/vm-display';
import { matchErrorCatalog } from '../lib/error-catalog';
import { pushRecentAction } from '../lib/preview-prefs';

type Tab = 'overview' | 'activity' | 'networking' | 'storage' | 'cloudinit' | 'logs' | 'snapshots';

export function VMDetail() {
  const { name = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t, formatDate, locale } = useI18n();
  const [tab, setTab] = useState<Tab>('overview');
  const [snapshotModal, setSnapshotModal] = useState(false);
  const [snapshotName, setSnapshotName] = useState('');
  const [editMode, setEditMode] = useState(false);
  const [editForm, setEditForm] = useState({ display_name: '', offering: '' });
  const [logText, setLogText] = useState<string | null>(null);
  const [logError, setLogError] = useState<string | null>(null);
  const [attachVolumeId, setAttachVolumeId] = useState('');
  const [logLoading, setLogLoading] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [cloneOpen, setCloneOpen] = useState(false);
  const [sshCopied, setSshCopied] = useState(false);

  const needsTenant = useNeedsTenant();

  const loadLogs = async () => {
    setLogLoading(true);
    setLogError(null);
    try {
      setLogText(await fetchVMLogs(name, 300));
    } catch (e) {
      setLogError((e as Error).message);
      setLogText(null);
    } finally {
      setLogLoading(false);
    }
  };

  const { data: offeringsData } = useQuery({
    queryKey: queryKeys.offerings,
    queryFn: listServiceOfferings,
    enabled: !needsTenant,
  });
  const offerings = offeringsData?.service_offerings || [];

  const { data, isLoading, isFetching, isRefetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vm(name),
    queryFn: () => getVM(name),
    enabled: !needsTenant && !!name,
    refetchInterval: (q) => {
      const current = q.state.data?.vm;
      if (current && isVMTransitional(current.state)) return 3_000;
      if (current && isVmError(current.state)) return 5_000;
      return false;
    },
  });

  const { data: snapData } = useQuery({
    queryKey: queryKeys.vmSnapshots,
    queryFn: listVMSnapshots,
    enabled: !needsTenant && tab === 'snapshots',
  });

  const { data: vmVolData } = useQuery({
    queryKey: ['platform-vm-volumes', name],
    queryFn: () => listVMVolumes(name),
    enabled: !needsTenant && !!name && tab === 'storage',
  });

  const { data: allVolData } = useQuery({
    queryKey: queryKeys.volumes,
    queryFn: listVolumes,
    enabled: !needsTenant && tab === 'storage',
  });

  const vm = data?.vm;
  const velasUrl = data?.velas_url;
  const vmSnaps = (snapData?.vm_snapshots || []).filter((s) => s.vm_name === name);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.vm(name) });
    queryClient.invalidateQueries({ queryKey: queryKeys.vms });
  };

  const startMutation = useMutation({ mutationFn: () => startVM(name), onSuccess: invalidate });
  const stopMutation = useMutation({ mutationFn: () => stopVM(name), onSuccess: invalidate });
  const deleteMutation = useMutation({
    mutationFn: () => deleteVM(name),
    onSuccess: () => navigate('/vms'),
  });
  const updateMutation = useMutation({
    mutationFn: () => {
      const payload: { display_name?: string; service_offering_id?: string } = {
        display_name: editForm.display_name,
      };
      // Offering resize only when stopped; display_name always allowed.
      if (isVmStopped(data?.vm?.state) && editForm.offering) {
        payload.service_offering_id = editForm.offering;
      }
      return updateVM(name, payload);
    },
    onSuccess: () => {
      invalidate();
      setEditMode(false);
      pushRecentAction({ label: `Rename ${name}`, path: `/vms/${name}` });
    },
  });
  const snapshotMutation = useMutation({
    mutationFn: () => createVMSnapshot({ vm_name: name, name: snapshotName }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.vmSnapshots });
      setSnapshotModal(false);
      setSnapshotName('');
    },
  });

  const invalidateStorage = () => {
    queryClient.invalidateQueries({ queryKey: ['platform-vm-volumes', name] });
    queryClient.invalidateQueries({ queryKey: queryKeys.volumes });
  };

  const attachMutation = useMutation({
    mutationFn: () => attachVolumeToVM(name, attachVolumeId),
    onSuccess: () => {
      invalidateStorage();
      setAttachVolumeId('');
    },
  });

  const detachMutation = useMutation({
    mutationFn: (volumeId: string) => detachVolumeFromVM(name, volumeId),
    onSuccess: invalidateStorage,
  });

  if (needsTenant) {
    return <TenantRequiredNotice message={t('vmDetail.selectTenant')} />;
  }

  if (isLoading) return <div className="text-center py-12 text-on-surface-variant">{t('vmDetail.loading')}</div>;
  if (error || !vm) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center px-4">
        <div className="w-full max-w-md text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full border border-error/30 bg-error-container/30 text-error">
            <AlertCircle size={24} />
          </div>
          <h1 className="font-headline text-title-lg font-semibold text-on-surface">
            {t('vmDetail.notFoundTitle')}
          </h1>
          <p className="mt-2 text-sm text-on-surface-variant">
            {(error as Error)?.message || t('vmDetail.notFoundMessage')}
          </p>
          <div className="mt-5 flex justify-center gap-2">
            <button type="button" onClick={() => refetch()} className="btn-outline-sm">
              {t('common.retry')}
            </button>
            <Link to="/vms" className="btn-outline-sm">
              <ArrowLeft size={16} /> {t('nav.vms')}
            </Link>
          </div>
        </div>
      </div>
    );
  }

  const isWindowsVM = (v: PlatformVM) =>
    v.name.startsWith('win-') ||
    v.template?.toLowerCase().includes('windows') ||
    (v.image?.toLowerCase().includes('windows') ?? false);

  const sizesForVM = (v: PlatformVM) => {
    const windows = isWindowsVM(v);
    return windows
      ? offerings.filter((o) => o.name === 'windows-large')
      : offerings.filter((o) => o.name !== 'windows-large');
  };

  const resolveOfferingId = (v: PlatformVM) => {
    if (v.service_offering_id) {
      const byId = offerings.find((o) => o.id === v.service_offering_id);
      if (byId) return byId.id;
    }
    const matched = findOfferingBySpec(offerings, v.cpu, v.memory_mi);
    if (matched) return matched.id;
    if (isWindowsVM(v)) return findOfferingByName(offerings, 'windows-large')?.id || '';
    return findOfferingByName(offerings, 'small')?.id || offerings[0]?.id || '';
  };

  const resolveOfferingLabel = (v: PlatformVM) => {
    if (v.service_offering_id) {
      const byId = offerings.find((o) => o.id === v.service_offering_id);
      if (byId) return offeringLabel(byId);
    }
    const matched = findOfferingBySpec(offerings, v.cpu, v.memory_mi);
    if (matched) return offeringLabel(matched);
    return formatVmOffering(v);
  };

  const stopped = isVmStopped(vm.state);
  const running = isVmRunning(vm.state);
  const errored = isVmError(vm.state);
  const phase = deployPhaseFromVm(vm);

  const vmVolumes = vmVolData?.volumes || [];
  const availableVolumes = (allVolData?.volumes || []).filter((v) => !v.vm_id);

  const tabs: { id: Tab; label: string }[] = [
    { id: 'overview', label: t('vmDetail.overview') },
    { id: 'activity', label: t('vmDetail.activity') },
    { id: 'networking', label: t('vmDetail.networking') },
    { id: 'storage', label: t('vmDetail.storage') },
    { id: 'cloudinit', label: 'Cloud-init' },
    { id: 'logs', label: 'Logs' },
    { id: 'snapshots', label: 'Snapshots' },
  ];

  return (
    <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
    <div className="space-y-6">
      <div>
        <Link to="/vms" className="inline-flex items-center gap-2 text-sm text-primary-fixed-dim mb-2">
          <ArrowLeft size={16} /> {t('nav.vms')}
        </Link>
        <PageHeader
          title={vm.display_name || vm.name}
          subtitle={`${vm.name}${vm.zone ? ` · ${vm.zone}` : ''}${vm.ip ? ` · ${vm.ip}` : ''}`}
          actions={
            <>
              <StatusBadge status={vm.state} />
              <RefreshButton
                compact
                onRefresh={() => refetch()}
                isFetching={isRefetching}
                dataUpdatedAt={dataUpdatedAt}
              />
              {errored && (
                <button
                  type="button"
                  onClick={() => startMutation.mutate()}
                  disabled={startMutation.isPending}
                  className="btn-primary"
                  title={t('vmDetail.retryHint')}
                >
                  <RefreshCw size={16} /> {t('vmDetail.retry')}
                </button>
              )}
              {running ? (
                <button type="button" onClick={() => stopMutation.mutate()} className="btn-danger-soft">
                  <Power size={16} /> {t('vms.stop')}
                </button>
              ) : !errored ? (
                <button type="button" onClick={() => startMutation.mutate()} className="btn-success-soft">
                  <Play size={16} /> {t('vms.start')}
                </button>
              ) : null}
              <button
                type="button"
                onClick={() => openConsole(name!, vm.namespace)}
                disabled={!running}
                className={running ? 'btn-primary' : 'btn-outline-sm'}
                title={t('vmDetail.consoleHint')}
              >
                <Monitor size={16} /> {t('vmDetail.openConsole')}
              </button>
              <button
                type="button"
                onClick={() => { setSnapshotName(`${vm.name}-snap`); setSnapshotModal(true); }}
                disabled={!running}
                className="btn-outline-sm"
              >
                <Camera size={16} /> Snapshot
              </button>
              <button
                type="button"
                onClick={() => setCloneOpen(true)}
                className="btn-outline-sm"
                title={t('vmDetail.clone')}
              >
                <CopyPlus size={16} /> {t('vmDetail.clone')}
              </button>
              <button
                type="button"
                onClick={() => setDeleteOpen(true)}
                className="btn-danger-outline"
              >
                <Trash2 size={16} /> {t('vms.destroy')}
              </button>
            </>
          }
        />
      </div>

      {vm.error_message && (
        <InfoBanner variant="warning">
          <div className="flex flex-col gap-3">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
              <span>{vm.error_message}</span>
              <div className="flex gap-2 shrink-0">
                <button type="button" className="btn-outline-sm" onClick={() => setTab('activity')}>
                  <Activity size={14} /> {t('vmDetail.activity')}
                </button>
                {errored && (
                  <button type="button" className="btn-primary text-sm" onClick={() => startMutation.mutate()}>
                    {t('vmDetail.retry')}
                  </button>
                )}
              </div>
            </div>
            {(() => {
              const catalog = matchErrorCatalog(vm.error_message, locale);
              if (!catalog) return null;
              return (
                <div className="rounded-lg border border-outline-variant/60 bg-surface/40 p-3 text-sm">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium">{t('errorCatalog.title')}: {catalog.title}</span>
                    <ComingSoonBadge />
                  </div>
                  <p className="text-on-surface-variant">{t('errorCatalog.fix')}: {catalog.fix}</p>
                  <p className="text-xs text-on-surface-variant mt-1 font-data-mono">{catalog.docsHint}</p>
                </div>
              );
            })()}
          </div>
        </InfoBanner>
      )}

      {running && (
        <InfoBanner>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <span>{t('vmDetail.consoleHint')}</span>
            <div className="flex flex-wrap gap-2">
              <button type="button" className="btn-primary text-sm" onClick={() => openConsole(name!, vm.namespace)}>
                <Monitor size={14} /> {t('vmDetail.openConsole')}
              </button>
              {vm.ip && (
                <button
                  type="button"
                  className="btn-outline-sm"
                  onClick={async () => {
                    await navigator.clipboard.writeText(`ssh ubuntu@${vm.ip}`);
                    setSshCopied(true);
                    window.setTimeout(() => setSshCopied(false), 2000);
                  }}
                >
                  <Copy size={14} /> {sshCopied ? t('vms.copied') : t('vmDetail.copySsh')}
                </button>
              )}
              <button type="button" className="btn-outline-sm opacity-70" disabled title={t('preview.uiOnly')}>
                <Terminal size={14} /> {t('vmDetail.serialConsole')} <ComingSoonBadge className="ml-1" />
              </button>
            </div>
          </div>
        </InfoBanner>
      )}

      <TabBar tabs={tabs} active={tab} onChange={setTab} />

      {tab === 'overview' && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <SurfaceCard className="lg:col-span-2">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-headline text-headline-md font-semibold text-on-surface">{t('vmDetail.detailsTitle')}</h2>
              {!editMode ? (
                <button
                  onClick={() => {
                    setEditForm({
                      display_name: vm.display_name || vm.name,
                      offering: resolveOfferingId(vm),
                    });
                    setEditMode(true);
                  }}
                  className="btn-ghost-brand"
                >
                  {t('common.edit')}
                </button>
              ) : (
                <div className="flex gap-2">
                  <button onClick={() => setEditMode(false)} className="btn-ghost-muted">{t('common.cancel')}</button>
                  <button
                    onClick={() => updateMutation.mutate()}
                    disabled={updateMutation.isPending}
                    className="btn-ghost-brand flex items-center gap-1 disabled:opacity-40"
                    title={t('vmDetail.renameHint')}
                  >
                    <Save size={14} /> {t('common.save')}
                  </button>
                </div>
              )}
            </div>
            {editMode ? (
              <div className="space-y-4 max-w-md">
                <div>
                  <label className="block text-sm font-medium mb-1">Display name</label>
                  <input
                    value={editForm.display_name}
                    onChange={(e) => setEditForm({ ...editForm, display_name: e.target.value })}
                    className={formInputClass}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Service offering</label>
                  <select
                    value={editForm.offering}
                    onChange={(e) => setEditForm({ ...editForm, offering: e.target.value })}
                    disabled={!stopped}
                    className={`${formSelectClass} disabled:opacity-50`}
                  >
                    {sizesForVM(vm).map((o) => (
                      <option key={o.id} value={o.id}>{offeringLabel(o)}</option>
                    ))}
                  </select>
                  {!stopped && (
                    <p className="text-xs text-warning mt-1">{t('vmDetail.resizeRequiresStopped')}</p>
                  )}
                  <p className="text-xs text-on-surface-variant mt-2">{t('vmDetail.renameHint')}</p>
                </div>
              </div>
            ) : (
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-4 text-sm">
                <div><dt className="text-on-surface-variant">ID</dt><dd className="font-data-mono text-xs break-all text-on-surface">{vm.id}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.internalName')}</dt><dd className="text-on-surface">{vm.name}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vms.col.displayName')}</dt><dd className="text-on-surface">{vm.display_name || vm.name}</dd></div>
                <div><dt className="text-on-surface-variant">{t('common.state')}</dt><dd className="text-on-surface">{vm.state}</dd></div>
                <div><dt className="text-on-surface-variant">{t('common.region')}</dt><dd className="text-on-surface">{vm.zone || '—'}</dd></div>
                <div><dt className="text-on-surface-variant">{t('common.host')}</dt><dd className="text-on-surface">{vm.host_name || '—'}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.platform')}</dt><dd className="text-on-surface">VirtFoundry Compute</dd></div>
                <div><dt className="text-on-surface-variant">Template</dt><dd className="text-on-surface">{vm.template || '—'}</dd></div>
                <div><dt className="text-on-surface-variant">{t('common.image')}</dt><dd className="font-data-mono text-xs break-all text-on-surface">{vm.image || '—'}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.serviceOffering')}</dt><dd className="text-on-surface">{resolveOfferingLabel(vm)}</dd></div>
                <div><dt className="text-on-surface-variant">vCPUs</dt><dd className="text-on-surface">{vm.cpu > 0 ? vm.cpu : '—'}</dd></div>
                <div><dt className="text-on-surface-variant">RAM</dt><dd className="text-on-surface">{fmtMem(vm.memory_mi) || '—'}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.primaryIp')}</dt><dd className="font-data-mono text-on-surface">{vm.ip || '—'}</dd></div>
                <div>
                  <dt className="text-on-surface-variant flex items-center gap-2">
                    {t('vmDetail.guestAgent')} <ComingSoonBadge />
                  </dt>
                  <dd className="text-on-surface">
                    {running ? t('vmDetail.guestAgentOk') : t('vmDetail.guestAgentMissing')}
                  </dd>
                </div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.createdAt')}</dt><dd className="text-on-surface">{formatDate(vm.created_at)}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.updatedAt')}</dt><dd className="text-on-surface">{formatDate(vm.updated_at)}</dd></div>
              </dl>
            )}
          </SurfaceCard>
          <SurfaceCard>
            <h2 className="font-headline text-headline-md font-semibold text-on-surface mb-4">{t('vmDetail.quickActions')}</h2>
            <p className="text-sm text-on-surface-variant mb-4">
              {t('vmDetail.syncHint')}
            </p>
            <button
              type="button"
              disabled={!running}
              onClick={() => openConsole(name!, vm.namespace)}
              className="btn-primary w-full justify-center mb-2"
            >
              <Monitor size={16} /> {t('vmDetail.openConsole')}
            </button>
            {vm.ip && (
              <button
                type="button"
                className="btn-outline-sm w-full justify-center mb-2"
                onClick={async () => {
                  await navigator.clipboard.writeText(`ssh ubuntu@${vm.ip}`);
                  setSshCopied(true);
                  window.setTimeout(() => setSshCopied(false), 2000);
                }}
              >
                <Copy size={14} /> {sshCopied ? t('vms.copied') : t('vmDetail.copySsh')}
              </button>
            )}
            <button type="button" className="btn-outline-sm w-full justify-center mb-2" onClick={() => setCloneOpen(true)}>
              <CopyPlus size={14} /> {t('vmDetail.clone')}
            </button>
            {errored && (
              <button
                type="button"
                onClick={() => startMutation.mutate()}
                className="btn-outline-sm w-full justify-center mb-2"
              >
                <RefreshCw size={14} /> {t('vmDetail.retry')}
              </button>
            )}
            <div className="mt-4 space-y-2 border-t border-outline-variant pt-3">
              {[
                t('vmDetail.migrate'),
                t('vmDetail.passwordRotate'),
                t('vmDetail.yamlDiff'),
                t('vmDetail.isoBoot'),
              ].map((label) => (
                <button
                  key={label}
                  type="button"
                  disabled
                  className="btn-ghost-muted w-full justify-between text-sm opacity-70"
                >
                  {label} <ComingSoonBadge />
                </button>
              ))}
            </div>
          </SurfaceCard>
        </div>
      )}

      {tab === 'activity' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <SurfaceCard>
            <h2 className="font-headline text-headline-md font-semibold mb-2">{t('vms.deployProgress')}</h2>
            <p className="text-sm text-on-surface-variant mb-4">{t('vmDetail.activityHint')}</p>
            <ol className="space-y-2">
              {DEPLOY_PHASE_ORDER.map((p, idx) => {
                const currentIdx = DEPLOY_PHASE_ORDER.indexOf(
                  phase === 'error' || phase === 'unknown' ? 'creating' : phase,
                );
                const done = phase === 'running' || (phase !== 'error' && idx < currentIdx);
                const active = phase === p;
                return (
                  <li
                    key={p}
                    className={`flex items-center gap-3 rounded-lg border px-3 py-2 text-sm ${
                      done
                        ? 'border-success/30 bg-success-muted/40'
                        : active
                          ? 'border-primary-container bg-primary-container/10'
                          : 'border-outline-variant text-on-surface-variant'
                    }`}
                  >
                    <span className="font-data-mono text-xs w-5">{idx + 1}</span>
                    {p === 'creating' && t('vms.phaseCreating')}
                    {p === 'scheduling' && t('vms.phaseScheduling')}
                    {p === 'networking' && t('vms.phaseNetworking')}
                    {p === 'running' && t('vms.phaseRunning')}
                  </li>
                );
              })}
            </ol>
          </SurfaceCard>
          <SurfaceCard>
            <h2 className="font-headline text-headline-md font-semibold mb-2">{t('vmDetail.conditions')}</h2>
            {vm.error_message ? (
              <div className="rounded-lg border border-error/30 bg-error-container/15 p-3 text-sm text-error whitespace-pre-wrap">
                {vm.error_message}
              </div>
            ) : (
              <p className="text-sm text-on-surface-variant">
                {t('vmDetail.noActivity')} <StatusBadge status={vm.state} />
              </p>
            )}
            <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
              <div>
                <dt className="text-on-surface-variant">{t('vmDetail.createdAt')}</dt>
                <dd>{formatDate(vm.created_at)}</dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('vmDetail.updatedAt')}</dt>
                <dd>{formatDate(vm.updated_at)}</dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">{t('common.host')}</dt>
                <dd>{vm.host_name || '—'}</dd>
              </div>
              <div>
                <dt className="text-on-surface-variant">IP</dt>
                <dd className="font-data-mono">{vm.ip || '—'}</dd>
              </div>
            </dl>
          </SurfaceCard>
        </div>
      )}

      {tab === 'networking' && (
        <>
        <SurfaceCard padding="none">
          <PageTable>
            <PageTableHead>
              <PageTableTh>NIC</PageTableTh>
              <PageTableTh>Tipo</PageTableTh>
              <PageTableTh>IP</PageTableTh>
              <PageTableTh>MAC</PageTableTh>
            </PageTableHead>
            <PageTableBody>
              {(vm.nics?.length ? vm.nics : [{ name: 'default', ip: vm.ip, type: 'default' }]).map((nic) => (
                <PageTableRow key={nic.name}>
                  <PageTableTd>{nic.name}</PageTableTd>
                  <PageTableTd>{nic.type || '—'}</PageTableTd>
                  <PageTableTd className="font-data-mono">{nic.ip || '—'}</PageTableTd>
                  <PageTableTd className="font-data-mono text-xs">{nic.mac || '—'}</PageTableTd>
                </PageTableRow>
              ))}
            </PageTableBody>
          </PageTable>
        </SurfaceCard>
        <SurfaceCard className="mt-4" padding="md">
          <div className="flex items-center gap-2 mb-2">
            <h3 className="text-sm font-medium">{t('vmDetail.networkMap')}</h3>
            <ComingSoonBadge />
          </div>
          <div className="flex flex-wrap items-center gap-2 text-xs font-data-mono text-on-surface-variant">
            <span className="px-2 py-1 rounded border border-outline-variant bg-surface-container">{vm.name}</span>
            <span>→</span>
            {(vm.nics?.length ? vm.nics : [{ name: 'default', ip: vm.ip }]).map((nic) => (
              <span key={nic.name} className="px-2 py-1 rounded border border-outline-variant bg-surface-container">
                {nic.name}{nic.ip ? ` (${nic.ip})` : ''}
              </span>
            ))}
            <span>→</span>
            <span className="px-2 py-1 rounded border border-dashed border-outline-variant">VPC / subnet</span>
          </div>
        </SurfaceCard>
        </>
      )}

      {tab === 'storage' && (
        <SurfaceCard>
          <div className="space-y-4">
            <p className="text-sm text-on-surface-variant">{t('vmDetail.storageHint')}</p>
            <button type="button" disabled className="btn-outline-sm opacity-70">
              {t('vmDetail.expandVolume')} <ComingSoonBadge className="ml-2" />
            </button>
            <div className="flex flex-wrap gap-2 items-end">
              <div className="flex-1 min-w-[200px]">
                <label className="block text-sm font-medium mb-1">{t('vmDetail.attachVolume')}</label>
                <select
                  value={attachVolumeId}
                  onChange={(e) => setAttachVolumeId(e.target.value)}
                  className={formSelectClass}
                >
                  <option value="">{t('vmDetail.selectVolume')}</option>
                  {availableVolumes.map((v) => (
                    <option key={v.id} value={v.id}>{v.name} ({v.size_gi} Gi)</option>
                  ))}
                </select>
              </div>
              <button
                type="button"
                onClick={() => attachMutation.mutate()}
                disabled={!attachVolumeId || attachMutation.isPending}
                className="btn-primary flex items-center gap-1"
              >
                <Plus size={16} /> {attachMutation.isPending ? 'Attaching…' : t('vmDetail.attachVolume')}
              </button>
            </div>
            {attachMutation.isError && (
              <p className="text-error text-sm">{(attachMutation.error as Error).message}</p>
            )}
            {vmVolumes.length === 0 ? (
              <p className="text-on-surface-variant text-sm">{t('vmDetail.noVolumes')}</p>
            ) : (
              <PageTable>
                <PageTableHead>
                  <PageTableTh>{t('volumes.col.volume')}</PageTableTh>
                  <PageTableTh>{t('volumes.size')}</PageTableTh>
                  <PageTableTh>{t('common.state')}</PageTableTh>
                  <PageTableTh>PVC</PageTableTh>
                  <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
                </PageTableHead>
                <PageTableBody>
                  {vmVolumes.map((vol) => (
                    <PageTableRow key={vol.id}>
                      <PageTableTd>
                        <div className="flex items-center gap-3">
                          <HardDrive size={16} className="text-primary-fixed-dim" />
                          <span className="font-medium">{vol.name}</span>
                        </div>
                      </PageTableTd>
                      <PageTableTd>{vol.size_gi} Gi</PageTableTd>
                      <PageTableTd>
                        <StatusBadge status={vol.state || 'active'} pulse={false} />
                      </PageTableTd>
                      <PageTableTd className="font-data-mono text-xs text-on-surface-variant">{vol.pvc_name}</PageTableTd>
                      <PageTableTd className="text-right">
                        <button
                          type="button"
                          onClick={() => detachMutation.mutate(vol.id)}
                          disabled={detachMutation.isPending}
                          className="btn-outline-sm flex items-center gap-1 ml-auto"
                        >
                          <Unlink size={14} /> {detachMutation.isPending ? 'Detaching…' : t('vmDetail.detachVolume')}
                        </button>
                      </PageTableTd>
                    </PageTableRow>
                  ))}
                </PageTableBody>
              </PageTable>
            )}
            {detachMutation.isError && (
              <p className="text-error text-sm">{(detachMutation.error as Error).message}</p>
            )}
          </div>
        </SurfaceCard>
      )}

      {tab === 'cloudinit' && (
        <SurfaceCard padding="md">
          <CloudInitEditor vmName={name} />
        </SurfaceCard>
      )}

      {tab === 'logs' && (
        <SurfaceCard>
          <div className="space-y-4">
          <div className="flex flex-wrap gap-2 items-center justify-between">
            <p className="text-sm text-on-surface-variant">{t('logs.title')} · {t('logs.integration')}</p>
            <div className="flex gap-2">
              <button type="button" onClick={loadLogs} disabled={logLoading || !running} className="btn-outline-sm">
                {logLoading ? t('logs.loading') : t('logs.refresh')}
              </button>
              {velasUrl && (
                <a href={velasUrl} target="_blank" rel="noreferrer" className="btn-primary text-sm">
                  {t('logs.openVelas')}
                </a>
              )}
            </div>
          </div>
          {!running && (
            <p className="text-warning text-sm">{t('logs.startVm')}</p>
          )}
          {logError && <p className="text-error text-sm">{logError}</p>}
          <pre className="text-xs font-data-mono bg-surface-container-high text-success rounded-lg p-4 overflow-auto max-h-[480px] whitespace-pre-wrap">
            {logText ?? (running ? t('logs.clickRefresh') : '—')}
          </pre>
          </div>
        </SurfaceCard>
      )}

      {tab === 'snapshots' && (
        <SurfaceCard padding="none">
          {vmSnaps.length === 0 ? (
            <p className="p-6 text-on-surface-variant text-sm">{t('vmDetail.noSnapshots')}</p>
          ) : (
            <PageTable>
              <PageTableHead>
                <PageTableTh>{t('common.name')}</PageTableTh>
                <PageTableTh>{t('common.phase')}</PageTableTh>
                <PageTableTh>{t('common.created')}</PageTableTh>
              </PageTableHead>
              <PageTableBody>
                {vmSnaps.map((s) => (
                  <PageTableRow key={s.id}>
                    <PageTableTd>{s.name}</PageTableTd>
                    <PageTableTd>{s.phase}</PageTableTd>
                    <PageTableTd>{formatDate(s.created_at)}</PageTableTd>
                  </PageTableRow>
                ))}
              </PageTableBody>
            </PageTable>
          )}
        </SurfaceCard>
      )}

      <ConfirmDialog
        open={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate()}
        title={t('vms.destroyTitle')}
        message={t('vms.destroyMessage')}
        resourceName={vm.name}
        requireTypedName={vm.name}
        confirmLabel={t('vms.destroy')}
        loading={deleteMutation.isPending}
        error={deleteMutation.isError ? (deleteMutation.error as Error).message : undefined}
      />

      <Modal isOpen={snapshotModal} onClose={() => setSnapshotModal(false)} title={t('vmDetail.createSnapshot')}>
        <form
          onSubmit={(e) => { e.preventDefault(); snapshotMutation.mutate(); }}
          className="space-y-4"
        >
          <input
            required
            pattern="[-a-z0-9]+"
            value={snapshotName}
            onChange={(e) => setSnapshotName(e.target.value.toLowerCase())}
            className={formInputClass}
            placeholder={t('vmDetail.snapshotPlaceholder')}
          />
          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => setSnapshotModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={snapshotMutation.isPending} className="btn-primary">
              {t('common.create')}
            </button>
          </div>
        </form>
      </Modal>

      <DeployVMWizard open={cloneOpen} onClose={() => setCloneOpen(false)} cloneFrom={vm} />
    </div>
    </RefreshingPanel>
  );
}
