import { Link } from 'react-router-dom';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft, Play, Power, Trash2, Monitor, Camera, Save, Key, Copy, Check,
} from 'lucide-react';
import {
  getVM, updateVM, startVM, stopVM, deleteVM, createVMSnapshot,
  listVMSnapshots, fetchVMLogs, getVMSSH, exposeVMSSH, VM_SIZES,
  type PlatformVM,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { openConsole } from '../lib/console-url';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional } from '../hooks/useRealtimeEvents';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';

function stateColor(state: string) {
  switch (state?.toLowerCase()) {
    case 'running': return 'bg-green-100 text-green-800';
    case 'stopped': return 'bg-gray-100 text-gray-800';
    case 'starting': return 'bg-blue-100 text-blue-800';
    case 'stopping': return 'bg-yellow-100 text-yellow-800';
    case 'error': return 'bg-red-100 text-red-800';
    default: return 'bg-gray-100 text-gray-800';
  }
}

function fmtMem(mi: number) {
  if (mi >= 1024) return `${(mi / 1024).toFixed(1)} GB`;
  return `${mi} MiB`;
}

type Tab = 'overview' | 'networking' | 'ssh' | 'logs' | 'snapshots';

export function VMDetail() {
  const { name = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t, formatDate } = useI18n();
  const [tab, setTab] = useState<Tab>('overview');
  const [snapshotModal, setSnapshotModal] = useState(false);
  const [snapshotName, setSnapshotName] = useState('');
  const [editMode, setEditMode] = useState(false);
  const [editForm, setEditForm] = useState({ display_name: '', offering: 'small' });
  const [logText, setLogText] = useState<string | null>(null);
  const [logError, setLogError] = useState<string | null>(null);
  const [logLoading, setLogLoading] = useState(false);
  const [sshCopied, setSshCopied] = useState(false);

  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

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

  const { data, isLoading, isFetching, error, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vm(name),
    queryFn: () => getVM(name),
    enabled: !needsTenant && !!name,
    refetchInterval: (q) => {
      const vm = q.state.data?.vm;
      if (vm && isVMTransitional(vm.state)) return 3000;
      return 12_000;
    },
  });

  const { data: snapData } = useQuery({
    queryKey: queryKeys.vmSnapshots,
    queryFn: listVMSnapshots,
    enabled: !needsTenant && tab === 'snapshots',
    refetchInterval: tab === 'snapshots' ? 10_000 : false,
  });

  const { data: sshData, refetch: refetchSSH } = useQuery({
    queryKey: ['platform-vm-ssh', name],
    queryFn: () => getVMSSH(name),
    enabled: !needsTenant && !!name && tab === 'ssh',
    refetchInterval: tab === 'ssh' ? 10_000 : false,
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
      const size = VM_SIZES.find((s) => s.id === editForm.offering) || VM_SIZES[0];
      return updateVM(name, {
        display_name: editForm.display_name,
        cpu: size.cpu,
        memory_mi: size.memory_mi,
      });
    },
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
  const exposeSSHMutation = useMutation({
    mutationFn: () => exposeVMSSH(name),
    onSuccess: () => refetchSSH(),
  });

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">{t('vmDetail.selectTenant')}</div>;
  }

  if (isLoading) return <div className="text-center py-12 text-gray-500">{t('vmDetail.loading')}</div>;
  if (error || !vm) {
    return (
      <div className="space-y-4">
        <Link to="/vms" className="inline-flex items-center gap-2 text-brand-600"><ArrowLeft size={18} /> {t('common.back')}</Link>
        <p className="text-red-600">{(error as Error)?.message || t('vmDetail.notFound')}</p>
      </div>
    );
  }

  const isWindowsVM = (v: PlatformVM) =>
    v.name.startsWith('win-') ||
    v.template?.toLowerCase().includes('windows') ||
    (v.image?.toLowerCase().includes('windows') ?? false);

  const sizesForVM = (v: PlatformVM) =>
    isWindowsVM(v)
      ? VM_SIZES.filter((s) => s.id === 'windows-large')
      : VM_SIZES.filter((s) => s.id !== 'windows-large');

  const stopped = vm.state?.toLowerCase() === 'stopped';
  const running = vm.state?.toLowerCase() === 'running';

  const tabs: { id: Tab; label: string }[] = [
    { id: 'overview', label: t('vmDetail.overview') },
    { id: 'networking', label: t('vmDetail.networking') },
    ...(!isWindowsVM(vm) ? [{ id: 'ssh' as Tab, label: t('ssh.connectTitle') }] : []),
    { id: 'logs', label: 'Logs' },
    { id: 'snapshots', label: 'Snapshots' },
  ];

  const sshCmd = sshData?.exposed && sshData.node_port
    ? `ssh -i <sua-chave-privada> ubuntu@<host> -p ${sshData.node_port}`
    : null;

  const copySSHCmd = async () => {
    if (!sshCmd) return;
    await navigator.clipboard.writeText(sshCmd);
    setSshCopied(true);
    setTimeout(() => setSshCopied(false), 2000);
  };

  return (
    <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <Link to="/vms" className="inline-flex items-center gap-2 text-sm text-brand-600 mb-2">
            <ArrowLeft size={16} /> {t('nav.vms')}
          </Link>
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {vm.display_name || vm.name}
            </h1>
            <span className={`px-2.5 py-1 rounded-full text-xs font-medium ${stateColor(vm.state)}`}>
              {vm.state}
            </span>
          </div>
          <p className="text-gray-500 text-sm mt-1">
            {vm.name}{vm.zone ? ` · ${vm.zone}` : ''}{vm.ip ? ` · ${vm.ip}` : ''}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <RefreshButton
            compact
            onRefresh={() => refetch()}
            isFetching={isFetching}
            dataUpdatedAt={dataUpdatedAt}
          />
          {running ? (
            <button onClick={() => stopMutation.mutate()} className="btn-danger-soft">
              <Power size={16} /> {t('vms.stop')}
            </button>
          ) : (
            <button onClick={() => startMutation.mutate()} className="btn-success-soft">
              <Play size={16} /> {t('vms.start')}
            </button>
          )}
          <button
            onClick={() => openConsole(name!, vm.namespace)}
            disabled={!running}
            className="btn-outline-sm"
          >
            <Monitor size={16} /> Console
          </button>
          <button
            onClick={() => { setSnapshotName(`${vm.name}-snap`); setSnapshotModal(true); }}
            disabled={!running}
            className="btn-outline-sm"
          >
            <Camera size={16} /> Snapshot
          </button>
          <button
            onClick={() => deleteMutation.mutate()}
            className="btn-danger-outline"
          >
            <Trash2 size={16} /> {t('vms.destroy')}
          </button>
        </div>
      </div>

      {vm.error_message && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{vm.error_message}</div>
      )}

      <div className="border-b flex gap-6">
        {tabs.map((tabItem) => (
          <button
            key={tabItem.id}
            onClick={() => setTab(tabItem.id)}
            className={`btn-tab ${
              tab === tabItem.id ? 'border-brand-500 text-brand-600' : 'border-transparent text-gray-500'
            }`}
          >
            {tabItem.label}
          </button>
        ))}
      </div>

      {tab === 'overview' && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 bg-white dark:bg-dark-100 rounded-xl border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold">{t('vmDetail.detailsTitle')}</h2>
              {!editMode ? (
                <button
                  onClick={() => {
                    const offering =
                      VM_SIZES.find((s) => s.cpu === vm.cpu && s.memory_mi === vm.memory_mi)?.id ||
                      (isWindowsVM(vm) ? 'windows-large' : 'small');
                    setEditForm({ display_name: vm.display_name || vm.name, offering });
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
                    className="w-full px-3 py-2 border rounded-lg"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Service offering</label>
                  <select
                    value={editForm.offering}
                    onChange={(e) => setEditForm({ ...editForm, offering: e.target.value })}
                    disabled={!stopped}
                    className="w-full px-3 py-2 border rounded-lg disabled:opacity-50"
                  >
                    {sizesForVM(vm).map((o) => (
                      <option key={o.id} value={o.id}>{o.label}</option>
                    ))}
                  </select>
                  {!stopped && (
                    <p className="text-xs text-amber-600 mt-1">{t('vmDetail.resizeRequiresStopped')}</p>
                  )}
                </div>
              </div>
            ) : (
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-4 text-sm">
                <div><dt className="text-gray-500">ID</dt><dd className="font-mono text-xs break-all">{vm.id}</dd></div>
                <div><dt className="text-gray-500">{t('vmDetail.internalName')}</dt><dd>{vm.name}</dd></div>
                <div><dt className="text-gray-500">{t('vms.col.displayName')}</dt><dd>{vm.display_name || vm.name}</dd></div>
                <div><dt className="text-gray-500">{t('common.state')}</dt><dd>{vm.state}</dd></div>
                <div><dt className="text-gray-500">{t('common.region')}</dt><dd>{vm.zone || '—'}</dd></div>
                <div><dt className="text-gray-500">{t('common.host')}</dt><dd>{vm.host_name || '—'}</dd></div>
                <div><dt className="text-gray-500">{t('vmDetail.platform')}</dt><dd>VirtFoundry Compute</dd></div>
                <div><dt className="text-gray-500">Template</dt><dd>{vm.template || '—'}</dd></div>
                <div><dt className="text-gray-500">{t('common.image')}</dt><dd className="font-mono text-xs break-all">{vm.image || '—'}</dd></div>
                <div><dt className="text-gray-500">vCPUs</dt><dd>{vm.cpu}</dd></div>
                <div><dt className="text-gray-500">RAM</dt><dd>{fmtMem(vm.memory_mi)}</dd></div>
                <div><dt className="text-gray-500">{t('vmDetail.primaryIp')}</dt><dd className="font-mono">{vm.ip || '—'}</dd></div>
                <div><dt className="text-gray-500">{t('vmDetail.createdAt')}</dt><dd>{formatDate(vm.created_at)}</dd></div>
                <div><dt className="text-gray-500">{t('vmDetail.updatedAt')}</dt><dd>{formatDate(vm.updated_at)}</dd></div>
              </dl>
            )}
          </div>
          <div className="bg-white dark:bg-dark-100 rounded-xl border p-6">
            <h2 className="font-semibold mb-4">{t('vmDetail.quickActions')}</h2>
            <p className="text-sm text-gray-500">
              {t('vmDetail.syncHint')}
            </p>
          </div>
        </div>
      )}

      {tab === 'networking' && (
        <div className="bg-white dark:bg-dark-100 rounded-xl border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-dark-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium">NIC</th>
                <th className="text-left px-4 py-3 font-medium">Tipo</th>
                <th className="text-left px-4 py-3 font-medium">IP</th>
                <th className="text-left px-4 py-3 font-medium">MAC</th>
              </tr>
            </thead>
            <tbody>
              {(vm.nics?.length ? vm.nics : [{ name: 'default', ip: vm.ip, type: 'default' }]).map((nic) => (
                <tr key={nic.name} className="border-t">
                  <td className="px-4 py-3">{nic.name}</td>
                  <td className="px-4 py-3">{nic.type || '—'}</td>
                  <td className="px-4 py-3 font-mono">{nic.ip || '—'}</td>
                  <td className="px-4 py-3 font-mono text-xs">{nic.mac || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {tab === 'ssh' && !isWindowsVM(vm) && (
        <div className="bg-white dark:bg-dark-100 rounded-xl border p-6 space-y-6 max-w-2xl">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-brand-100 rounded-lg flex items-center justify-center">
              <Key size={20} className="text-brand-600" />
            </div>
            <div>
              <h2 className="font-semibold">{t('ssh.connectTitle')}</h2>
              <p className="text-sm text-gray-500">{t('vmDetail.internalIp')}: {sshData?.vm_ip || vm.ip || '—'}</p>
            </div>
          </div>

          {sshData?.exposed && sshData.node_port ? (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <span className="px-2 py-1 rounded text-xs font-medium bg-green-100 text-green-700">{t('vmDetail.exposed')}</span>
                <span className="text-sm">NodePort: <strong>{sshData.node_port}</strong></span>
              </div>
              <div>
                <p className="text-sm font-medium mb-1">{t('ssh.command')}</p>
                <div className="flex gap-2">
                  <code className="flex-1 text-xs font-mono bg-gray-900 text-green-400 p-3 rounded-lg break-all">
                    {sshCmd}
                  </code>
                  <button type="button" onClick={copySSHCmd} className="btn-secondary shrink-0">
                    {sshCopied ? <Check size={16} /> : <Copy size={16} />}
                  </button>
                </div>
              </div>
              <p className="text-xs text-gray-500">
                {t('vmDetail.sshKeyHint')}{' '}
                <Link to="/ssh-keys" className="text-brand-600 hover:underline">{t('vmDetail.manageKeysLink')}</Link>.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-sm text-gray-500">{t('ssh.notExposed')}</p>
              <button
                type="button"
                onClick={() => exposeSSHMutation.mutate()}
                disabled={!running || exposeSSHMutation.isPending || !vm.ip}
                className="btn-primary"
              >
                {exposeSSHMutation.isPending ? t('common.loading') : t('ssh.expose')}
              </button>
              {!running && (
                <p className="text-xs text-amber-600">{t('vmDetail.vmMustBeRunning')}</p>
              )}
              {exposeSSHMutation.isError && (
                <p className="text-red-500 text-sm">{(exposeSSHMutation.error as Error).message}</p>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'logs' && (
        <div className="bg-white dark:bg-dark-100 rounded-xl border p-6 space-y-4">
          <div className="flex flex-wrap gap-2 items-center justify-between">
            <p className="text-sm text-gray-500">{t('logs.title')} · {t('logs.integration')}</p>
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
            <p className="text-amber-600 text-sm">{t('logs.startVm')}</p>
          )}
          {logError && <p className="text-red-600 text-sm">{logError}</p>}
          <pre className="text-xs font-mono bg-gray-950 text-green-100 rounded-lg p-4 overflow-auto max-h-[480px] whitespace-pre-wrap">
            {logText ?? (running ? t('logs.clickRefresh') : '—')}
          </pre>
        </div>
      )}

      {tab === 'snapshots' && (
        <div className="bg-white dark:bg-dark-100 rounded-xl border overflow-hidden">
          {vmSnaps.length === 0 ? (
            <p className="p-6 text-gray-500 text-sm">{t('vmDetail.noSnapshots')}</p>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="text-left px-4 py-3">{t('common.name')}</th>
                  <th className="text-left px-4 py-3">{t('common.phase')}</th>
                  <th className="text-left px-4 py-3">{t('common.created')}</th>
                </tr>
              </thead>
              <tbody>
                {vmSnaps.map((s) => (
                  <tr key={s.id} className="border-t">
                    <td className="px-4 py-3">{s.name}</td>
                    <td className="px-4 py-3">{s.phase}</td>
                    <td className="px-4 py-3">{formatDate(s.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
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
            className="w-full px-4 py-2 border rounded-lg"
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
