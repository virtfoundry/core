import { Link } from 'react-router-dom';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft, Play, Power, Trash2, Monitor, Camera, Save, HardDrive, Unlink, Plus,
} from 'lucide-react';
import {
  getVM, updateVM, startVM, stopVM, deleteVM, createVMSnapshot,
  listVMSnapshots, fetchVMLogs, listServiceOfferings,
  listVMVolumes, listVolumes, attachVolumeToVM, detachVolumeFromVM,
  listNetworks, listVPCs,
  type PlatformVM, type Network, type VPC,
} from '../lib/platform-api';
import {
  offeringsForTemplate, offeringLabel, findOfferingBySpec, findOfferingByName,
} from '../lib/offerings';
import { isPublicNetwork } from '../lib/networks';
import { Modal } from '../components/Modal';
import { openConsole } from '../lib/console-url';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional } from '../hooks/useRealtimeEvents';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useNeedsTenant } from '../store/hooks';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, TabBar, TenantRequiredNotice, InfoBanner,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, formSelectClass,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';

function fmtMem(mi: number) {
  if (mi >= 1024) return `${(mi / 1024).toFixed(1)} GB`;
  return `${mi} MiB`;
}

/** Public pool IP for tenants; prefer named public NIC, then vm.ip if not cluster overlay. */
function publicPoolIP(vm: PlatformVM): string {
  const publicNic = vm.nics?.find((n) => n.name === 'public' && n.ip);
  if (publicNic?.ip) return publicNic.ip;
  if (vm.ip && !vm.ip.startsWith('10.233.')) return vm.ip;
  return '';
}

type RootNicRow = {
  name: string;
  type: string;
  ip?: string;
  mac?: string;
  network_id?: string;
  nad_namespace?: string;
  nad_name?: string;
  roleKey: 'vmDetail.nicRolePublic' | 'vmDetail.nicRoleSubnet' | 'vmDetail.nicRolePod' | 'vmDetail.nicRoleOther';
  network?: Network;
  vpc?: VPC;
};

/** Root NIC table: fill public IP from pool; order public → other → pod; join network/VPC. */
function nicsForRootTable(vm: PlatformVM, networks: Network[], vpcs: VPC[]): RootNicRow[] {
  const pool = publicPoolIP(vm);
  const byID = new Map(networks.map((n) => [n.id, n]));
  const byName = new Map(networks.map((n) => [n.name, n]));
  const byNAD = new Map(
    networks
      .filter((n) => n.nad_name)
      .map((n) => [`${n.nad_namespace || ''}/${n.nad_name}`, n]),
  );
  const vpcByID = new Map(vpcs.map((v) => [v.id, v]));

  const nics = (vm.nics?.length ? vm.nics : [{ name: 'default', ip: vm.ip, type: 'default' }]).map((nic) => {
    let type = nic.type || '—';
    if (nic.name === 'pod') type = 'pod';
    else if (nic.name === 'public') type = 'multus';
    else if (type === 'pod' && nic.name !== 'pod') type = 'multus';
    const ip = nic.name === 'public' && !nic.ip ? pool : nic.ip;

    let network =
      (nic.network_id && byID.get(nic.network_id)) ||
      (nic.nad_name && byNAD.get(`${nic.nad_namespace || ''}/${nic.nad_name}`)) ||
      byName.get(nic.name);

    let roleKey: RootNicRow['roleKey'] = 'vmDetail.nicRoleOther';
    if (nic.name === 'pod' || type === 'pod') {
      roleKey = 'vmDetail.nicRolePod';
      network = undefined;
    } else if (!network) {
      roleKey = nic.name === 'public' ? 'vmDetail.nicRolePublic' : 'vmDetail.nicRoleOther';
    } else if (isPublicNetwork(network)) {
      roleKey = 'vmDetail.nicRolePublic';
    } else {
      roleKey = 'vmDetail.nicRoleSubnet';
    }

    const vpc = network?.vpc_id ? vpcByID.get(network.vpc_id) : undefined;
    return {
      name: nic.name,
      type,
      ip,
      mac: nic.mac,
      network_id: nic.network_id || network?.id,
      nad_namespace: nic.nad_namespace || network?.nad_namespace,
      nad_name: nic.nad_name || network?.nad_name,
      roleKey,
      network,
      vpc,
    };
  });
  const rank = (name: string) => (name === 'public' ? 0 : name === 'pod' ? 2 : 1);
  return [...nics].sort((a, b) => rank(a.name) - rank(b.name));
}

type Tab = 'overview' | 'networking' | 'storage' | 'logs' | 'snapshots';

export function VMDetail() {
  const { name = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t, formatDate } = useI18n();
  const isRoot = authService.isRoot();
  const [tab, setTab] = useState<Tab>('overview');
  const [snapshotModal, setSnapshotModal] = useState(false);
  const [snapshotName, setSnapshotName] = useState('');
  const [editMode, setEditMode] = useState(false);
  const [editForm, setEditForm] = useState({ display_name: '', offering: '' });
  const [logText, setLogText] = useState<string | null>(null);
  const [logError, setLogError] = useState<string | null>(null);
  const [attachVolumeId, setAttachVolumeId] = useState('');
  const [logLoading, setLogLoading] = useState(false);

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
      const vm = q.state.data?.vm;
      if (vm && isVMTransitional(vm.state)) return 3_000;
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

  const { data: networksData } = useQuery({
    queryKey: queryKeys.networks,
    queryFn: listNetworks,
    enabled: !needsTenant && isRoot && tab === 'networking',
  });
  const { data: vpcsData } = useQuery({
    queryKey: queryKeys.vpcs,
    queryFn: listVPCs,
    enabled: !needsTenant && isRoot && tab === 'networking',
  });

  const vm = data?.vm;
  const velasUrl = data?.velas_url;
  const vmSnaps = (snapData?.vm_snapshots || []).filter((s) => s.vm_name === name);
  const networks = networksData?.networks || [];
  const vpcs = vpcsData?.vpcs || [];

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
    mutationFn: () => updateVM(name, {
      display_name: editForm.display_name,
      service_offering_id: editForm.offering,
    }),
    onSuccess: () => {
      invalidate();
      setEditMode(false);
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
      <div className="space-y-4">
        <Link to="/vms" className="inline-flex items-center gap-2 text-primary-fixed-dim"><ArrowLeft size={18} /> {t('common.back')}</Link>
        <p className="text-error">{(error as Error)?.message || t('vmDetail.notFound')}</p>
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
    return `${v.cpu} vCPU, ${fmtMem(v.memory_mi)}`;
  };

  const stopped = vm.state?.toLowerCase() === 'stopped';
  const running = vm.state?.toLowerCase() === 'running';

  const vmVolumes = vmVolData?.volumes || [];
  const availableVolumes = (allVolData?.volumes || []).filter((v) => !v.vm_id);

  const tabs: { id: Tab; label: string }[] = [
    { id: 'overview', label: t('vmDetail.overview') },
    { id: 'networking', label: t('vmDetail.networking') },
    { id: 'storage', label: t('vmDetail.storage') },
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
              {running ? (
                <button type="button" onClick={() => stopMutation.mutate()} className="btn-danger-soft">
                  <Power size={16} /> {t('vms.stop')}
                </button>
              ) : (
                <button type="button" onClick={() => startMutation.mutate()} className="btn-success-soft">
                  <Play size={16} /> {t('vms.start')}
                </button>
              )}
              <button
                type="button"
                onClick={() => openConsole(name!, vm.namespace)}
                disabled={!running}
                className="btn-outline-sm"
              >
                <Monitor size={16} /> Console
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
                onClick={() => deleteMutation.mutate()}
                className="btn-danger-outline"
              >
                <Trash2 size={16} /> {t('vms.destroy')}
              </button>
            </>
          }
        />
      </div>

      {vm.error_message && (
        <InfoBanner variant="warning">{vm.error_message}</InfoBanner>
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
                    disabled={updateMutation.isPending || !stopped}
                    className="btn-ghost-brand flex items-center gap-1 disabled:opacity-40"
                    title={!stopped ? t('vmDetail.stopToResize') : undefined}
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
                <div><dt className="text-on-surface-variant">vCPUs</dt><dd className="text-on-surface">{vm.cpu}</dd></div>
                <div><dt className="text-on-surface-variant">RAM</dt><dd className="text-on-surface">{fmtMem(vm.memory_mi)}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.primaryIp')}</dt><dd className="font-data-mono text-on-surface">{vm.ip || '—'}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.createdAt')}</dt><dd className="text-on-surface">{formatDate(vm.created_at)}</dd></div>
                <div><dt className="text-on-surface-variant">{t('vmDetail.updatedAt')}</dt><dd className="text-on-surface">{formatDate(vm.updated_at)}</dd></div>
              </dl>
            )}
          </SurfaceCard>
          <SurfaceCard>
            <h2 className="font-headline text-headline-md font-semibold text-on-surface mb-4">{t('vmDetail.quickActions')}</h2>
            <p className="text-sm text-on-surface-variant">
              {t('vmDetail.syncHint')}
            </p>
          </SurfaceCard>
        </div>
      )}

      {tab === 'networking' && (
        <SurfaceCard padding={isRoot ? 'none' : undefined}>
          {isRoot ? (
            <div className="overflow-x-auto">
              <PageTable>
                <PageTableHead>
                  <PageTableTh>NIC</PageTableTh>
                  <PageTableTh>{t('vmDetail.nicRole')}</PageTableTh>
                  <PageTableTh>IP</PageTableTh>
                  <PageTableTh>{t('vmDetail.gateway')}</PageTableTh>
                  <PageTableTh>{t('vmDetail.vpc')}</PageTableTh>
                  <PageTableTh>{t('vmDetail.subnet')}</PageTableTh>
                  <PageTableTh>{t('vmDetail.cidr')}</PageTableTh>
                  <PageTableTh>{t('vmDetail.nad')}</PageTableTh>
                  <PageTableTh>MAC</PageTableTh>
                </PageTableHead>
                <PageTableBody>
                  {nicsForRootTable(vm, networks, vpcs).map((nic) => {
                    const nad =
                      nic.nad_namespace && nic.nad_name
                        ? `${nic.nad_namespace}/${nic.nad_name}`
                        : nic.nad_name || '';
                    return (
                      <PageTableRow key={nic.name}>
                        <PageTableTd className="font-medium text-on-surface">{nic.name}</PageTableTd>
                        <PageTableTd>
                          <span className="text-on-surface">{t(nic.roleKey)}</span>
                          <span className="block text-xs text-on-surface-variant font-data-mono">{nic.type}</span>
                        </PageTableTd>
                        <PageTableTd className="font-data-mono">{nic.ip || '—'}</PageTableTd>
                        <PageTableTd className="font-data-mono">{nic.network?.gateway || '—'}</PageTableTd>
                        <PageTableTd>
                          {nic.vpc ? (
                            <>
                              <span className="text-on-surface">{nic.vpc.name}</span>
                              <span className="block text-xs text-on-surface-variant font-data-mono">{nic.vpc.cidr}</span>
                            </>
                          ) : (
                            '—'
                          )}
                        </PageTableTd>
                        <PageTableTd>{nic.network?.name || '—'}</PageTableTd>
                        <PageTableTd className="font-data-mono">{nic.network?.cidr || '—'}</PageTableTd>
                        <PageTableTd className="font-data-mono text-xs">{nad || '—'}</PageTableTd>
                        <PageTableTd className="font-data-mono text-xs">{nic.mac || '—'}</PageTableTd>
                      </PageTableRow>
                    );
                  })}
                </PageTableBody>
              </PageTable>
            </div>
          ) : (
            <div>
              <h2 className="font-headline text-headline-md font-semibold text-on-surface mb-2">
                {t('vmDetail.publicIp')}
              </h2>
              <p className="font-data-mono text-on-surface text-lg">
                {publicPoolIP(vm) || '—'}
              </p>
              <p className="text-sm text-on-surface-variant mt-2">
                {t('vmDetail.publicIpHint')}
              </p>
            </div>
          )}
        </SurfaceCard>
      )}

      {tab === 'storage' && (
        <SurfaceCard>
          <div className="space-y-4">
            <p className="text-sm text-on-surface-variant">{t('vmDetail.storageHint')}</p>
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
                <Plus size={16} /> {t('vmDetail.attachVolume')}
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
                          <Unlink size={14} /> {t('vmDetail.detachVolume')}
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
    </div>
    </RefreshingPanel>
  );
}
