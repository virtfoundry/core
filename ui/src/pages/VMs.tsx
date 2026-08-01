import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Play, Trash2, Plus, Search, Power, Monitor, Camera } from 'lucide-react';
import {
  listVMs, startVM, stopVM, deleteVM, deployVM, createVMSnapshot, listNetworks,
  listSSHKeys, listVolumes, VM_IMAGES, VM_SIZES, VM_WINDOWS_IMAGES,
  PlatformVM,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { openConsole } from '../lib/console-url';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional } from '../hooks/useRealtimeEvents';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';
import { useI18n } from '../lib/i18n';
import { isIsolatedNetwork, isPublicNetwork } from '../lib/networks';

function stateColor(state: string) {
  switch (state?.toLowerCase()) {
    case 'running': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
    case 'stopped': return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
    case 'starting': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400';
    case 'stopping': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
    case 'error': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
    default: return 'bg-gray-100 text-gray-800';
  }
}

function fmtMem(mi: number) {
  if (mi >= 1024) return `${(mi / 1024).toFixed(1)} GB`;
  return `${mi} MiB`;
}

export function VMs() {
  const { t } = useI18n();
  const [search, setSearch] = useState('');
  const [deployModal, setDeployModal] = useState(false);
  const [snapshotModal, setSnapshotModal] = useState<{ vmName: string } | null>(null);
  const [snapshotForm, setSnapshotForm] = useState({ name: '' });
  const allImages = [...VM_IMAGES, ...VM_WINDOWS_IMAGES];
  const isWindowsImage = (image: string) => VM_WINDOWS_IMAGES.some((t) => t.image === image);
  const sizesForImage = (image: string) =>
    isWindowsImage(image)
      ? VM_SIZES.filter((s) => s.id === 'windows-large')
      : VM_SIZES.filter((s) => s.id !== 'windows-large');

  const [form, setForm] = useState({
    name: '',
    image: VM_IMAGES[0].image,
    offering: 'small',
    network_ids: [] as string[],
    ssh_key_id: '',
    data_volume_id: '',
    expose_ssh: false,
  });
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');
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
  const networks = netData?.networks || [];
  const privateNetworks = networks.filter(isIsolatedNetwork);
  const hasPublicNetwork = networks.some(isPublicNetwork);
  const sshKeys = sshData?.ssh_keys || [];
  const volumes = volData?.volumes || [];

  const { data, isLoading, isFetching, refetch, error, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
    refetchInterval: (q) => {
      const vms = q.state.data?.vms || [];
      if (vms.some((vm) => isVMTransitional(vm.state))) return 3000;
      return 12_000;
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
  const deployMutation = useMutation({
    mutationFn: deployVM,
    onSuccess: () => {
      invalidate();
      setDeployModal(false);
      setForm({ name: '', image: VM_IMAGES[0].image, offering: 'small', network_ids: [], ssh_key_id: '', data_volume_id: '', expose_ssh: false });
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
    const size = VM_SIZES.find((s) => s.id === form.offering) || VM_SIZES[0];
    const linux = !isWindowsImage(form.image);
    deployMutation.mutate({
      name: form.name,
      image: form.image,
      cpu: size.cpu,
      memory_mi: size.memory_mi,
      ...(form.network_ids.length ? { network_ids: form.network_ids } : {}),
      ...(linux && form.ssh_key_id ? { ssh_key_id: form.ssh_key_id } : {}),
      ...(linux && form.data_volume_id ? { data_volume_id: form.data_volume_id } : {}),
      ...(linux && form.expose_ssh ? { expose_ssh: true } : {}),
    });
  };

  if (needsTenant) {
    return (
      <div className="text-center py-12 text-amber-600">
        {t('vms.selectTenant')}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t('nav.vms')}</h1>
          <p className="text-gray-500">{vms.length} {t('vms.subtitle')}</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton
            onRefresh={() => refetch()}
            isFetching={isFetching}
            dataUpdatedAt={dataUpdatedAt}
          />
          <button onClick={() => setDeployModal(true)} className="btn-primary">
            <Plus size={18} /> Deploy VM
          </button>
        </div>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-600 text-sm">
          {(error as Error).message}
        </div>
      )}

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('vms.searchPlaceholder')}
          className="w-full pl-10 pr-4 py-3 border rounded-lg bg-white dark:bg-dark-100"
        />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="bg-white dark:bg-dark-100 rounded-xl border overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-dark-200 text-gray-600">
            <tr>
              <th className="text-left px-4 py-3 font-medium">{t('common.name')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('vms.col.displayName')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('common.state')}</th>
              <th className="text-left px-4 py-3 font-medium">IP</th>
              <th className="text-left px-4 py-3 font-medium">Zone</th>
              <th className="text-left px-4 py-3 font-medium">Host</th>
              <th className="text-left px-4 py-3 font-medium">Offering</th>
              <th className="text-left px-4 py-3 font-medium">Template</th>
              <th className="text-right px-4 py-3 font-medium">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={9} className="text-center py-12 text-gray-500">{t('common.loading')}</td></tr>
            ) : filteredVMs.length === 0 ? (
              <tr><td colSpan={9} className="text-center py-12 text-gray-500">{t('vms.empty')}</td></tr>
            ) : (
              filteredVMs.map((vm: PlatformVM) => (
                <tr key={vm.id || vm.name} className="border-t hover:bg-gray-50/80 dark:hover:bg-dark-200/50">
                  <td className="px-4 py-3">
                    <Link to={`/vms/${vm.name}`} className="font-medium text-brand-600 hover:underline">
                      {vm.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3">{vm.display_name || vm.name}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${stateColor(vm.state)}`}>
                      {vm.state}
                    </span>
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
              className="w-full px-4 py-2 border rounded-lg"
              placeholder="web-server-01"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">{t('common.image')}</label>
              <select
                value={form.image}
                onChange={(e) => {
                  const image = e.target.value;
                  setForm({
                    ...form,
                    image,
                    offering: isWindowsImage(image) ? 'windows-large' : form.offering === 'windows-large' ? 'small' : form.offering,
                  });
                }}
                className="w-full px-4 py-2 border rounded-lg"
              >
                <optgroup label="Linux">
                  {VM_IMAGES.map((t) => <option key={t.id} value={t.image}>{t.label}</option>)}
                </optgroup>
                <optgroup label="Windows">
                  {VM_WINDOWS_IMAGES.map((t) => <option key={t.id} value={t.image}>{t.label}</option>)}
                </optgroup>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Offering</label>
              <select value={form.offering} onChange={(e) => setForm({ ...form, offering: e.target.value })} className="w-full px-4 py-2 border rounded-lg">
                {sizesForImage(form.image).map((o) => <option key={o.id} value={o.id}>{o.label}</option>)}
              </select>
            </div>
          </div>
          {hasPublicNetwork && (
            <div className="rounded-lg border border-purple-200 bg-purple-50 px-4 py-3 text-sm text-purple-900">
              <p className="font-medium">{t('vms.publicNetworkDefault')}</p>
              <p className="text-xs mt-1 text-purple-800">{t('vms.publicNetworkDefaultHint')}</p>
            </div>
          )}
          {privateNetworks.length > 0 && (
            <div>
              <label className="block text-sm font-medium mb-1">{t('vms.privateNetworks')}</label>
              <select
                multiple
                value={form.network_ids}
                onChange={(e) => {
                  const selected = Array.from(e.target.selectedOptions, (o) => o.value);
                  setForm({ ...form, network_ids: selected });
                }}
                className="w-full px-4 py-2 border rounded-lg min-h-[88px]"
              >
                {privateNetworks.map((n) => (
                  <option key={n.id} value={n.id}>{n.name} ({n.cidr}) · {t('networks.typeIsolated')}</option>
                ))}
              </select>
              <p className="text-xs text-gray-500 mt-1">{t('vms.multiSelectHint')}</p>
            </div>
          )}
          {!isWindowsImage(form.image) && (
            <>
              <div>
                <label className="block text-sm font-medium mb-1">{t('vms.sshKeyOptional')}</label>
                <select
                  value={form.ssh_key_id}
                  onChange={(e) => setForm({ ...form, ssh_key_id: e.target.value })}
                  className="w-full px-4 py-2 border rounded-lg"
                >
                  <option value="">{t('common.noneFem')}</option>
                  {sshKeys.map((k) => (
                    <option key={k.id} value={k.id}>{k.name} ({k.fingerprint})</option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 mt-1">
                  {t('ssh.deployHint')}{' '}
                  <Link to="/ssh-keys" className="text-brand-600 hover:underline">{t('vms.manageKeys')}</Link>
                </p>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">{t('vms.dataVolumeOptional')}</label>
                <select
                  value={form.data_volume_id}
                  onChange={(e) => setForm({ ...form, data_volume_id: e.target.value })}
                  className="w-full px-4 py-2 border rounded-lg"
                >
                  <option value="">{t('common.none')}</option>
                  {volumes.map((v) => (
                    <option key={v.id} value={v.id}>{v.name} ({v.size_gi} Gi)</option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 mt-1">{t('ssh.dataVolumeHint')}</p>
              </div>
              <label className="flex items-center gap-2 text-sm cursor-pointer">
                <input
                  type="checkbox"
                  checked={form.expose_ssh}
                  onChange={(e) => setForm({ ...form, expose_ssh: e.target.checked })}
                  className="rounded border-gray-300"
                />
                {t('vms.exposeSsh')}
                <span className="text-xs text-gray-500">({t('ssh.exposeHint')})</span>
              </label>
            </>
          )}
          {deployMutation.isError && (
            <p className="text-red-500 text-sm">{(deployMutation.error as Error)?.message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setDeployModal(false)} className="btn-secondary">{t('common.cancel')}</button>
            <button type="submit" disabled={deployMutation.isPending} className="btn-primary">
              {deployMutation.isPending ? t('common.deploying') : 'Deploy'}
            </button>
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
            className="w-full px-4 py-2 border rounded-lg"
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
