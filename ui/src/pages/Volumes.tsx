import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, HardDrive } from 'lucide-react';
import { listVolumes, createVolume } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';

export function Volumes() {
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ name: '', size_gi: 10 });
  const queryClient = useQueryClient();
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.volumes,
    queryFn: listVolumes,
    enabled: !needsTenant,
    refetchInterval: 12_000,
  });

  const createMutation = useMutation({
    mutationFn: createVolume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.volumes });
      setCreateModal(false);
      setForm({ name: '', size_gi: 10 });
    },
  });

  const volumes = data?.volumes || [];
  const filtered = volumes.filter((v) => v.name?.toLowerCase().includes(search.toLowerCase()));

  if (needsTenant) {
    return <div className="text-center py-12 text-amber-600">Selecione um tenant para gerenciar volumes.</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Volumes</h1>
          <p className="text-gray-500">{volumes.length} volumes no ambiente</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary">
            <Plus size={18} /> Criar Volume
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Buscar volumes..."
          className="w-full pl-10 pr-4 py-3 border rounded-lg" />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="bg-white dark:bg-dark-100 rounded-xl border overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-dark-200">
            <tr>
              <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 uppercase">Volume</th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 uppercase">Tamanho</th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 uppercase">Status</th>
              <th className="px-6 py-4 text-left text-xs font-semibold text-gray-500 uppercase">PVC</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {isLoading ? (
              <tr><td colSpan={4} className="text-center py-8">Carregando...</td></tr>
            ) : filtered.length === 0 ? (
              <tr><td colSpan={4} className="text-center py-8 text-gray-500">Nenhum volume encontrado</td></tr>
            ) : (
              filtered.map((vol) => (
                <tr key={vol.id} className="hover:bg-gray-50 dark:hover:bg-dark-200">
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <HardDrive size={16} className="text-nimbus-500" />
                      <span className="font-medium">{vol.name}</span>
                    </div>
                  </td>
                  <td className="px-6 py-4">{vol.size_gi} Gi</td>
                  <td className="px-6 py-4">
                    <span className="px-2 py-1 rounded text-xs font-medium bg-green-100 text-green-700">{vol.state}</span>
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-gray-500">{vol.pvc_name}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title="Criar Volume (PVC)">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">Nome</label>
            <input required pattern="[-a-z0-9]+" value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value.toLowerCase() })}
              className="w-full px-4 py-2 border rounded-lg" placeholder="data-disk-01" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Tamanho (Gi)</label>
            <input type="number" min={1} required value={form.size_gi}
              onChange={(e) => setForm({ ...form, size_gi: parseInt(e.target.value, 10) })}
              className="w-full px-4 py-2 border rounded-lg" />
          </div>
          {createMutation.isError && (
            <p className="text-red-500 text-sm">{(createMutation.error as Error).message}</p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <button type="button" onClick={() => setCreateModal(false)} className="btn-secondary">Cancelar</button>
            <button type="submit" disabled={createMutation.isPending} className="btn-primary">
              {createMutation.isPending ? 'Criando...' : 'Criar'}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
