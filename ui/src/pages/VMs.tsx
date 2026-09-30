import { useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Play, Trash2, Plus, Power, Monitor, Camera, Server, AlertCircle,
  Star, Pin, Copy, Columns2, Download, Tag, CopyPlus,
} from 'lucide-react';
import {
  listVMs, startVM, stopVM, deleteVM, createVMSnapshot, PlatformVM,
} from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { DeployVMWizard } from '../components/DeployVMWizard';
import { ComingSoonBadge } from '../components/ComingSoonBadge';
import { OverflowMenu } from '../components/OverflowMenu';
import { openConsole } from '../lib/console-url';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional, realtimePollInterval, useRealtimeConnected } from '../hooks/useRealtimeEvents';
import { queryKeys } from '../lib/query-keys';
import { useNeedsTenant } from '../store/hooks';
import { useAppSelector } from '../store/hooks';
import { selectUser } from '../store/authSlice';
import { useI18n } from '../lib/i18n';
import {
  PageHeader, SurfaceCard, SearchField, TenantRequiredNotice, EmptyState,
  PageTable, PageTableHead, PageTableTh, PageTableBody, PageTableRow, PageTableTd,
  formInputClass, InfoBanner,
} from '../components/shell';
import { StatusBadge } from '../components/StatusBadge';
import { formatVmOffering, isVmError, isVmRunning } from '../lib/vm-display';
import { matchErrorCatalog } from '../lib/error-catalog';
import {
  getAllVmTags,
  getFavoriteVMs,
  getPinnedVMs,
  getSplitView,
  pushRecentAction,
  setSplitView,
  setVmTags,
  toggleFavoriteVM,
  togglePinnedVM,
} from '../lib/preview-prefs';
import clsx from 'clsx';

type StateFilter = 'all' | 'running' | 'error' | 'stopped' | 'other';

type BulkProgress = {
  total: number;
  done: number;
  failed: number;
  action: string;
} | null;

export function VMs() {
  const { t, locale } = useI18n();
  const user = useAppSelector(selectUser);
  const [search, setSearch] = useState('');
  const [stateFilter, setStateFilter] = useState<StateFilter>('all');
  const [mineOnly, setMineOnly] = useState(false);
  const [tagFilter, setTagFilter] = useState('');
  const [deployModal, setDeployModal] = useState(false);
  const [cloneFrom, setCloneFrom] = useState<PlatformVM | null>(null);
  const [snapshotModal, setSnapshotModal] = useState<{ vmName: string } | null>(null);
  const [snapshotForm, setSnapshotForm] = useState({ name: '' });
  const [deleteTarget, setDeleteTarget] = useState<PlatformVM | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkProgress, setBulkProgress] = useState<BulkProgress>(null);
  const [favs, setFavs] = useState(() => getFavoriteVMs());
  const [pins, setPins] = useState(() => getPinnedVMs());
  const [tagMap, setTagMap] = useState(() => getAllVmTags());
  const [split, setSplit] = useState(() => getSplitView());
  const [splitName, setSplitName] = useState<string | null>(null);
  const [tagEditVm, setTagEditVm] = useState<string | null>(null);
  const [tagDraft, setTagDraft] = useState('');
  const [copiedSsh, setCopiedSsh] = useState<string | null>(null);
  const [rateLimitMsg, setRateLimitMsg] = useState<string | null>(null);

  const queryClient = useQueryClient();
  const needsTenant = useNeedsTenant();
  const wsConnected = useRealtimeConnected();

  const { data, isLoading, isRefetching, refetch, error, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled: !needsTenant,
    refetchInterval: (q) => {
      const vms = q.state.data?.vms || [];
      const transitional = vms.some((vm) => isVMTransitional(vm.state));
      return realtimePollInterval(wsConnected, transitional, { healthyMs: false });
    },
  });

  const withRateLimit = useCallback(async <T,>(fn: () => Promise<T>): Promise<T> => {
    try {
      return await fn();
    } catch (e) {
      const msg = (e as Error).message || '';
      if (/429|rate.?limit|too many/i.test(msg)) {
        setRateLimitMsg(t('preview.rateLimit'));
        window.setTimeout(() => setRateLimitMsg(null), 4000);
      }
      throw e;
    }
  }, [t]);

  const startMutation = useMutation({
    mutationFn: (name: string) => withRateLimit(() => startVM(name)),
    onMutate: async (name) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.vms });
      const prev = queryClient.getQueryData<{ vms: PlatformVM[] }>(queryKeys.vms);
      if (prev) {
        queryClient.setQueryData(queryKeys.vms, {
          vms: prev.vms.map((vm) =>
            vm.name === name ? { ...vm, state: 'Starting' } : vm,
          ),
        });
      }
      return { prev };
    },
    onError: (_e, _n, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(queryKeys.vms, ctx.prev);
    },
    // List refresh via /ws/events merge-patch — do not invalidateQueries(vms).
  });
  const stopMutation = useMutation({
    mutationFn: (name: string) => withRateLimit(() => stopVM(name)),
    onMutate: async (name) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.vms });
      const prev = queryClient.getQueryData<{ vms: PlatformVM[] }>(queryKeys.vms);
      if (prev) {
        queryClient.setQueryData(queryKeys.vms, {
          vms: prev.vms.map((vm) =>
            vm.name === name ? { ...vm, state: 'Stopping' } : vm,
          ),
        });
      }
      return { prev };
    },
    onError: (_e, _n, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(queryKeys.vms, ctx.prev);
    },
    // List refresh via /ws/events merge-patch — do not invalidateQueries(vms).
  });
  const destroyMutation = useMutation({
    mutationFn: deleteVM,
    onSuccess: () => {
      // Row removal via vm.deleted merge-patch — do not invalidateQueries(vms).
      setDeleteTarget(null);
    },
  });
  const snapshotMutation = useMutation({
    mutationFn: createVMSnapshot,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.vmSnapshots });
      setSnapshotModal(null);
      setSnapshotForm({ name: '' });
    },
  });

  const vms = data?.vms || [];

  const allTags = useMemo(() => {
    const s = new Set<string>();
    Object.values(tagMap).forEach((tags) => tags.forEach((t) => s.add(t)));
    return [...s].sort();
  }, [tagMap]);

  const filteredVMs = useMemo(() => {
    const list = vms.filter((vm: PlatformVM) => {
      const q = search.toLowerCase();
      const tags = tagMap[vm.name] || [];
      const matchesSearch =
        !q
        || vm.name?.toLowerCase().includes(q)
        || vm.display_name?.toLowerCase().includes(q)
        || vm.ip?.includes(search)
        || vm.error_message?.toLowerCase().includes(q)
        || tags.some((tg) => tg.includes(q));

      if (!matchesSearch) return false;

      const st = (vm.state || '').toLowerCase();
      if (stateFilter === 'running' && st !== 'running') return false;
      if (stateFilter === 'error' && st !== 'error') return false;
      if (stateFilter === 'stopped' && st !== 'stopped') return false;
      if (stateFilter === 'other' && ['running', 'error', 'stopped'].includes(st)) return false;

      if (tagFilter && !tags.includes(tagFilter)) return false;

      if (mineOnly && user?.username) {
        const uname = user.username.toLowerCase().replace(/-admin$/, '');
        const owned =
          vm.name?.toLowerCase().includes(uname)
          || vm.display_name?.toLowerCase().includes(uname);
        if (!owned && user.role !== 'root') {
          // soft UX hint
        }
      }

      return true;
    });

    return list.sort((a, b) => {
      const ap = pins.has(a.name) ? 0 : 1;
      const bp = pins.has(b.name) ? 0 : 1;
      if (ap !== bp) return ap - bp;
      const af = favs.has(a.name) ? 0 : 1;
      const bf = favs.has(b.name) ? 0 : 1;
      if (af !== bf) return af - bf;
      return (a.name || '').localeCompare(b.name || '');
    });
  }, [vms, search, stateFilter, mineOnly, user, tagMap, tagFilter, pins, favs]);

  const filterCounts = useMemo(() => {
    const counts = { all: vms.length, running: 0, error: 0, stopped: 0, other: 0 };
    for (const vm of vms) {
      const st = (vm.state || '').toLowerCase();
      if (st === 'running') counts.running += 1;
      else if (st === 'error') counts.error += 1;
      else if (st === 'stopped') counts.stopped += 1;
      else counts.other += 1;
    }
    return counts;
  }, [vms]);

  const splitVm = splitName ? vms.find((v) => v.name === splitName) : null;

  const toggleSelect = (name: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selected.size === filteredVMs.length) setSelected(new Set());
    else setSelected(new Set(filteredVMs.map((v) => v.name)));
  };

  const runBulk = async (action: 'start' | 'stop' | 'delete') => {
    const names = [...selected];
    if (names.length === 0) return;
    if (action === 'delete') {
      const ok = window.confirm(t('vms.bulkDeleteConfirm').replace('{n}', String(names.length)));
      if (!ok) return;
    }
    setBulkProgress({ total: names.length, done: 0, failed: 0, action });
    let done = 0;
    let failed = 0;
    for (const name of names) {
      try {
        if (action === 'start') await startVM(name);
        else if (action === 'stop') await stopVM(name);
        else await deleteVM(name);
        done += 1;
      } catch {
        failed += 1;
      }
      setBulkProgress({ total: names.length, done: done + failed, failed, action });
    }
    setSelected(new Set());
    // Bulk rows update via /ws/events merge-patch — do not invalidateQueries(vms).
    window.setTimeout(() => setBulkProgress(null), 2500);
  };

  const copySsh = async (vm: PlatformVM) => {
    if (!vm.ip) return;
    const line = `ssh ubuntu@${vm.ip}`;
    try {
      await navigator.clipboard.writeText(line);
      setCopiedSsh(vm.name);
      pushRecentAction({ label: `SSH ${vm.name}`, path: `/vms/${vm.name}` });
      window.setTimeout(() => setCopiedSsh(null), 2000);
    } catch {
      /* ignore */
    }
  };

  const exportCsv = () => {
    const rows = [
      ['name', 'display_name', 'state', 'ip', 'host', 'cpu', 'memory_mi', 'template', 'tags'].join(','),
      ...filteredVMs.map((vm) =>
        [
          vm.name,
          JSON.stringify(vm.display_name || ''),
          vm.state || '',
          vm.ip || '',
          vm.host_name || '',
          vm.cpu ?? '',
          vm.memory_mi ?? '',
          vm.template || '',
          JSON.stringify((tagMap[vm.name] || []).join(';')),
        ].join(','),
      ),
    ];
    const blob = new Blob([rows.join('\n')], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'virtfoundry-vms.csv';
    a.click();
    URL.revokeObjectURL(url);
  };

  const openClone = (vm: PlatformVM) => {
    setCloneFrom(vm);
    setDeployModal(true);
    pushRecentAction({ label: `Clone ${vm.name}`, path: `/vms/${vm.name}` });
  };

  if (needsTenant) {
    return <TenantRequiredNotice message={t('vms.selectTenant')} />;
  }

  return (
    <div className={clsx('space-y-6', split && 'xl:grid xl:grid-cols-5 xl:gap-6 xl:space-y-0')}>
      <div className={clsx('space-y-6', split && 'xl:col-span-3')}>
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
              <button
                type="button"
                className="btn-outline-sm"
                title={t('vms.exportCsv')}
                onClick={exportCsv}
                disabled={filteredVMs.length === 0}
              >
                <Download size={16} /> CSV
              </button>
              <button
                type="button"
                className={clsx('btn-outline-sm hidden xl:inline-flex', split && 'ring-1 ring-primary-container')}
                title={t('vms.splitView')}
                onClick={() => {
                  const next = !split;
                  setSplit(next);
                  setSplitView(next);
                  if (!next) setSplitName(null);
                }}
              >
                <Columns2 size={16} /> {t('vms.split')}
              </button>
              <button type="button" onClick={() => { setCloneFrom(null); setDeployModal(true); }} className="btn-primary">
                <Plus size={18} /> {t('vms.deploy')}
              </button>
            </>
          }
        />

        {rateLimitMsg && <InfoBanner variant="warning">{rateLimitMsg}</InfoBanner>}

        {error && (
          <div className="p-4 bg-error-container/30 border border-error-container rounded-lg text-on-error-container text-sm flex items-start gap-3">
            <AlertCircle size={18} className="shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="font-medium">{t('common.errorLoad')}</p>
              <p className="mt-1 opacity-90">{(error as Error).message || t('vms.errorHint')}</p>
              <button type="button" onClick={() => refetch()} className="btn-outline-sm mt-3">
                {t('common.retry')}
              </button>
            </div>
          </div>
        )}

        <div className="flex flex-col lg:flex-row gap-3 lg:items-center">
          <SearchField
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('vms.searchPlaceholder')}
            containerClassName="flex-1 max-w-xl"
          />
          <div className="flex flex-wrap gap-2 items-center" role="group" aria-label={t('common.filterState')}>
            {([
              ['all', t('common.filterAll'), filterCounts.all],
              ['running', t('vms.filterRunning'), filterCounts.running],
              ['error', t('vms.filterError'), filterCounts.error],
              ['stopped', t('vms.filterStopped'), filterCounts.stopped],
            ] as const).map(([id, label, count]) => (
              <button
                key={id}
                type="button"
                onClick={() => setStateFilter(id)}
                className={
                  stateFilter === id
                    ? 'btn-primary text-xs !h-8 !px-3'
                    : 'btn-outline-sm text-xs !h-8'
                }
              >
                {label} ({count})
              </button>
            ))}
            <label className="inline-flex items-center gap-2 text-xs text-on-surface-variant ml-1 cursor-pointer">
              <input
                type="checkbox"
                checked={mineOnly}
                onChange={(e) => setMineOnly(e.target.checked)}
              />
              {t('common.mine')}
            </label>
            {allTags.length > 0 && (
              <select
                value={tagFilter}
                onChange={(e) => setTagFilter(e.target.value)}
                className="text-xs h-8 border border-outline-variant rounded-lg bg-surface-container-high px-2"
                aria-label={t('vms.filterTag')}
              >
                <option value="">{t('vms.allTags')}</option>
                {allTags.map((tg) => (
                  <option key={tg} value={tg}>{tg}</option>
                ))}
              </select>
            )}
          </div>
        </div>

        {selected.size > 0 && (
          <div className="flex flex-wrap items-center gap-2 p-3 rounded-lg border border-outline-variant bg-surface-container-high">
            <span className="text-sm font-medium">{t('vms.bulkSelected').replace('{n}', String(selected.size))}</span>
            <ComingSoonBadge label={t('preview.uiOnly')} />
            <button type="button" className="btn-outline-sm" onClick={() => runBulk('start')}>
              <Play size={14} /> {t('vms.start')}
            </button>
            <button type="button" className="btn-outline-sm" onClick={() => runBulk('stop')}>
              <Power size={14} /> {t('vms.stop')}
            </button>
            <button type="button" className="btn-danger-outline text-sm !h-8" onClick={() => runBulk('delete')}>
              <Trash2 size={14} /> {t('vms.destroy')}
            </button>
            <button type="button" className="btn-ghost-muted text-sm" onClick={() => setSelected(new Set())}>
              {t('common.cancel')}
            </button>
          </div>
        )}

        {bulkProgress && (
          <InfoBanner>
            {t('vms.bulkProgress')
              .replace('{action}', bulkProgress.action)
              .replace('{done}', String(bulkProgress.done))
              .replace('{total}', String(bulkProgress.total))
              .replace('{failed}', String(bulkProgress.failed))}
          </InfoBanner>
        )}

        <RefreshingPanel isFetching={isRefetching} isLoading={isLoading}>
          <SurfaceCard padding="none" className="overflow-hidden">
            {isLoading ? (
              <div className="text-center py-16 text-on-surface-variant">{t('common.loading')}</div>
            ) : filteredVMs.length === 0 ? (
              <EmptyState
                icon={<Server size={40} />}
                title={vms.length === 0 ? t('vms.empty') : t('header.searchEmpty')}
                hint={vms.length === 0 ? t('vms.emptyHint') : undefined}
                action={
                  vms.length === 0 ? (
                    <button type="button" onClick={() => setDeployModal(true)} className="btn-primary">
                      <Plus size={16} /> {t('vms.emptyCta')}
                    </button>
                  ) : undefined
                }
              />
            ) : (
              <PageTable>
                <PageTableHead>
                  <PageTableTh className="w-10">
                    <input
                      type="checkbox"
                      checked={selected.size > 0 && selected.size === filteredVMs.length}
                      onChange={toggleSelectAll}
                      aria-label={t('vms.selectAll')}
                    />
                  </PageTableTh>
                  <PageTableTh className="w-16" />
                  <PageTableTh>{t('common.name')}</PageTableTh>
                  <PageTableTh>{t('vms.col.displayName')}</PageTableTh>
                  <PageTableTh>{t('common.state')}</PageTableTh>
                  <PageTableTh>IP</PageTableTh>
                  <PageTableTh>{t('vms.col.host')}</PageTableTh>
                  <PageTableTh>{t('vms.col.offering')}</PageTableTh>
                  <PageTableTh>Tags</PageTableTh>
                  <PageTableTh className="text-right">{t('common.actions')}</PageTableTh>
                </PageTableHead>
                <PageTableBody>
                  {filteredVMs.map((vm: PlatformVM) => {
                    const errored = isVmError(vm.state);
                    const running = isVmRunning(vm.state);
                    const catalog = matchErrorCatalog(vm.error_message, locale);
                    const tags = tagMap[vm.name] || [];
                    return (
                      <PageTableRow
                        key={vm.id || vm.name}
                        className={clsx(
                          errored && 'bg-error-container/5',
                          splitName === vm.name && 'ring-1 ring-inset ring-primary-container/50',
                        )}
                      >
                        <PageTableTd>
                          <input
                            type="checkbox"
                            checked={selected.has(vm.name)}
                            onChange={() => toggleSelect(vm.name)}
                            aria-label={`${t('common.select')} ${vm.name}`}
                          />
                        </PageTableTd>
                        <PageTableTd>
                          <div className="flex gap-0.5">
                            <button
                              type="button"
                              className="btn-icon-neutral !p-1"
                              title={t('vms.favorite')}
                              aria-pressed={favs.has(vm.name)}
                              onClick={() => setFavs(toggleFavoriteVM(vm.name))}
                            >
                              <Star size={14} className={favs.has(vm.name) ? 'fill-warning text-warning' : ''} />
                            </button>
                            <button
                              type="button"
                              className="btn-icon-neutral !p-1"
                              title={t('vms.pin')}
                              aria-pressed={pins.has(vm.name)}
                              onClick={() => setPins(togglePinnedVM(vm.name))}
                            >
                              <Pin size={14} className={pins.has(vm.name) ? 'text-primary' : ''} />
                            </button>
                          </div>
                        </PageTableTd>
                        <PageTableTd>
                          <button
                            type="button"
                            className="text-left"
                            onClick={() => {
                              if (split) setSplitName(vm.name);
                            }}
                          >
                            <Link
                              to={`/vms/${vm.name}`}
                              className="font-medium text-primary hover:text-primary-fixed-dim hover:underline"
                              onClick={(e) => {
                                pushRecentAction({ label: vm.display_name || vm.name, path: `/vms/${vm.name}` });
                                if (split) {
                                  e.preventDefault();
                                  setSplitName(vm.name);
                                }
                              }}
                            >
                              {vm.name}
                            </Link>
                          </button>
                          {errored && vm.error_message && (
                            <p className="mt-1 text-xs text-error line-clamp-2 max-w-xs" title={vm.error_message}>
                              {vm.error_message}
                            </p>
                          )}
                          {catalog && (
                            <p className="mt-1 text-[11px] text-on-surface-variant">
                              {t('errorCatalog.hint')}: {catalog.title}
                            </p>
                          )}
                        </PageTableTd>
                        <PageTableTd>{vm.display_name || vm.name}</PageTableTd>
                        <PageTableTd>
                          <StatusBadge status={vm.state || 'inactive'} />
                        </PageTableTd>
                        <PageTableTd className="font-mono text-xs">
                          <span className="inline-flex items-center gap-1">
                            {vm.ip || '—'}
                            {vm.ip && (
                              <button
                                type="button"
                                className="btn-icon-neutral !p-1"
                                title={t('vms.copySsh')}
                                aria-label={t('vms.copySsh')}
                                onClick={() => copySsh(vm)}
                              >
                                <Copy size={12} />
                              </button>
                            )}
                          </span>
                          {copiedSsh === vm.name && (
                            <span className="block text-[10px] text-success">{t('vms.copied')}</span>
                          )}
                        </PageTableTd>
                        <PageTableTd className="text-xs">{vm.host_name || '—'}</PageTableTd>
                        <PageTableTd className={vm.cpu === 0 && vm.memory_mi === 0 ? 'text-on-surface-variant' : undefined}>
                          {formatVmOffering(vm)}
                        </PageTableTd>
                        <PageTableTd>
                          <div className="flex flex-wrap gap-1 items-center max-w-[140px] min-h-[28px]">
                            {tags.map((tg) => (
                              <span key={tg} className="text-[10px] px-1.5 py-0.5 rounded border border-outline-variant bg-surface-container">
                                {tg}
                              </span>
                            ))}
                            {tags.length === 0 && (
                              <span className="text-xs text-on-surface-variant">—</span>
                            )}
                          </div>
                        </PageTableTd>
                        <PageTableTd>
                          <div className="flex justify-end gap-1" role="group" aria-label={t('common.actions')}>
                            {running ? (
                              <button
                                type="button"
                                onClick={() => stopMutation.mutate(vm.name)}
                                className="btn-icon-warning focus-visible:ring-2 focus-visible:ring-primary"
                                title={t('vms.stop')}
                                aria-label={t('vms.stop')}
                              >
                                <Power size={16} />
                              </button>
                            ) : (
                              <button
                                type="button"
                                onClick={() => startMutation.mutate(vm.name)}
                                className="btn-icon-success focus-visible:ring-2 focus-visible:ring-primary"
                                title={errored ? t('vmDetail.retry') : t('vms.start')}
                                aria-label={errored ? t('vmDetail.retry') : t('vms.start')}
                              >
                                <Play size={16} />
                              </button>
                            )}
                            <button
                              type="button"
                              onClick={() => openConsole(vm.name, vm.namespace)}
                              disabled={!running}
                              className="btn-icon-neutral focus-visible:ring-2 focus-visible:ring-primary"
                              title={t('vms.console')}
                              aria-label={t('vms.console')}
                            >
                              <Monitor size={16} />
                            </button>
                            <OverflowMenu
                              items={[
                                {
                                  id: 'clone',
                                  label: t('vms.clone'),
                                  icon: <CopyPlus size={14} />,
                                  onSelect: () => openClone(vm),
                                },
                                {
                                  id: 'snapshot',
                                  label: t('vms.snapshot'),
                                  icon: <Camera size={14} />,
                                  disabled: !running,
                                  onSelect: () => {
                                    setSnapshotForm({ name: `${vm.name}-snap` });
                                    setSnapshotModal({ vmName: vm.name });
                                  },
                                },
                                {
                                  id: 'tags',
                                  label: t('vms.editTags'),
                                  icon: <Tag size={14} />,
                                  onSelect: () => {
                                    setTagEditVm(vm.name);
                                    setTagDraft(tags.join(', '));
                                  },
                                },
                                {
                                  id: 'delete',
                                  label: t('vms.destroy'),
                                  icon: <Trash2 size={14} />,
                                  danger: true,
                                  onSelect: () => setDeleteTarget(vm),
                                },
                              ]}
                            />
                          </div>
                        </PageTableTd>
                      </PageTableRow>
                    );
                  })}
                </PageTableBody>
              </PageTable>
            )}
          </SurfaceCard>
        </RefreshingPanel>

        {filteredVMs.some((vm) => isVmError(vm.state) && vm.error_message) && stateFilter !== 'error' && (
          <InfoBanner variant="warning">
            {t('vms.errorFilterHint')}
          </InfoBanner>
        )}
      </div>

      {split && (
        <aside className="hidden xl:block xl:col-span-2 sticky top-20 self-start">
          <SurfaceCard padding="md" className="min-h-[320px]">
            <div className="flex items-center justify-between mb-3">
              <h2 className="font-headline text-title-md font-semibold">{t('vms.splitDetail')}</h2>
              <ComingSoonBadge />
            </div>
            {!splitVm ? (
              <p className="text-sm text-on-surface-variant">{t('vms.splitHint')}</p>
            ) : (
              <div className="space-y-3 text-sm">
                <div className="flex items-center gap-2">
                  <StatusBadge status={splitVm.state} />
                  <Link to={`/vms/${splitVm.name}`} className="text-primary hover:underline font-medium">
                    {splitVm.display_name || splitVm.name}
                  </Link>
                </div>
                <dl className="grid grid-cols-2 gap-2">
                  <div><dt className="text-on-surface-variant text-xs">IP</dt><dd className="font-data-mono">{splitVm.ip || '—'}</dd></div>
                  <div><dt className="text-on-surface-variant text-xs">{t('vms.col.host')}</dt><dd>{splitVm.host_name || '—'}</dd></div>
                  <div><dt className="text-on-surface-variant text-xs">{t('vms.col.offering')}</dt><dd>{formatVmOffering(splitVm)}</dd></div>
                  <div><dt className="text-on-surface-variant text-xs">Template</dt><dd>{splitVm.template || '—'}</dd></div>
                </dl>
                {splitVm.error_message && (
                  <p className="text-xs text-error whitespace-pre-wrap">{splitVm.error_message}</p>
                )}
                <div className="flex flex-wrap gap-2 pt-2">
                  <Link to={`/vms/${splitVm.name}`} className="btn-primary text-sm">{t('vms.viewVm')}</Link>
                  {splitVm.ip && (
                    <button type="button" className="btn-outline-sm" onClick={() => copySsh(splitVm)}>
                      <Copy size={14} /> SSH
                    </button>
                  )}
                  <button type="button" className="btn-outline-sm" onClick={() => openClone(splitVm)}>
                    <CopyPlus size={14} /> {t('vms.clone')}
                  </button>
                </div>
              </div>
            )}
          </SurfaceCard>
        </aside>
      )}

      <DeployVMWizard
        open={deployModal}
        onClose={() => { setDeployModal(false); setCloneFrom(null); }}
        cloneFrom={cloneFrom}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && destroyMutation.mutate(deleteTarget.name)}
        title={t('vms.destroyTitle')}
        message={t('vms.destroyMessage')}
        resourceName={deleteTarget?.name}
        requireTypedName={deleteTarget?.name}
        confirmLabel={t('vms.destroy')}
        loading={destroyMutation.isPending}
        error={destroyMutation.isError ? (destroyMutation.error as Error).message : undefined}
      />

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

      <Modal
        isOpen={!!tagEditVm}
        onClose={() => setTagEditVm(null)}
        title={`${t('vms.editTags')} — ${tagEditVm ?? ''}`}
      >
        <div className="space-y-3">
          <p className="text-xs text-on-surface-variant flex items-center gap-2">
            {t('vms.tagsHint')} <ComingSoonBadge label={t('preview.localOnly')} />
          </p>
          <input
            value={tagDraft}
            onChange={(e) => setTagDraft(e.target.value)}
            className={formInputClass}
            placeholder="prod, web, staging"
          />
          <div className="flex justify-end gap-2">
            <button type="button" className="btn-secondary" onClick={() => setTagEditVm(null)}>{t('common.cancel')}</button>
            <button
              type="button"
              className="btn-primary"
              onClick={() => {
                if (!tagEditVm) return;
                const tags = tagDraft.split(/[,;\s]+/).filter(Boolean);
                setVmTags(tagEditVm, tags);
                setTagMap(getAllVmTags());
                setTagEditVm(null);
              }}
            >
              {t('common.save')}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
