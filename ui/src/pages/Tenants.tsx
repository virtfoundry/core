import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Users } from 'lucide-react';
import { listTenants, createTenant } from '../lib/platform-api';
import { Modal } from '../components/Modal';
import { RefreshButton } from '../components/RefreshButton';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { queryKeys } from '../lib/query-keys';
import { authService } from '../lib/auth';

export function Tenants() {
  const [search, setSearch] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [form, setForm] = useState({ name: '', slug: '', admin_password: '' });
  const queryClient = useQueryClient();
  const isRoot = authService.isRoot();

  const { data, isLoading, isFetching, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.tenants,
    queryFn: listTenants,
    enabled: isRoot,
    refetchInterval: 15_000,
  });

  const createMutation = useMutation({
    mutationFn: createTenant,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants });
      setCreateModal(false);
      setForm({ name: '', slug: '', admin_password: '' });
    },
  });

  if (!isRoot) {
    return (
      <div className="text-center py-12 text-gray-500">
        Apenas usuários root podem gerenciar tenants.
      </div>
    );
  }

  const tenants = data?.tenants || [];
  const filtered = tenants.filter((t) =>
    t.name.toLowerCase().includes(search.toLowerCase()) ||
    t.slug.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Tenants</h1>
          <p className="text-gray-500">{tenants.length} tenants na plataforma</p>
        </div>
        <div className="flex gap-3">
          <RefreshButton onRefresh={() => refetch()} isFetching={isFetching} dataUpdatedAt={dataUpdatedAt} />
          <button onClick={() => setCreateModal(true)} className="btn-primary">
            <Plus size={18} /> Criar Tenant
          </button>
        </div>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={20} />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Buscar tenants..."
          className="w-full pl-10 pr-4 py-3 border rounded-lg"
        />
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={isLoading}>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {isLoading ? (
          <div className="col-span-full text-center py-12">Carregando...</div>
        ) : filtered.length === 0 ? (
          <div className="col-span-full text-center py-12">
            <Users size={48} className="mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">Nenhum tenant criado</p>
          </div>
        ) : (
          filtered.map((t) => (
            <div key={t.id} className="bg-white dark:bg-dark-100 rounded-xl border p-5">
              <h3 className="font-semibold text-lg">{t.name}</h3>
              <p className="text-sm text-gray-500 mb-3">{t.slug}</p>
              <div className="text-sm space-y-1">
                <p><span className="text-gray-500">Região:</span> {t.slug}</p>
                <p><span className="text-gray-500">Estado:</span> {t.state}</p>
              </div>
            </div>
          ))
        )}
      </div>
      </RefreshingPanel>

      <Modal isOpen={createModal} onClose={() => setCreateModal(false)} title="Criar Tenant">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            createMutation.mutate(form);
          }}
          className="space-y-4"
        >
          <div>
            <label className="block text-sm font-medium mb-1">Nome</label>
            <input
              required
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg"
              placeholder="Acme Corp"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Slug</label>
            <input
              required
              pattern="[-a-z0-9]+"
              value={form.slug}
              onChange={(e) => setForm({ ...form, slug: e.target.value.toLowerCase() })}
              className="w-full px-4 py-2 border rounded-lg"
              placeholder="acme"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Senha do admin do tenant</label>
            <input
              required
              type="password"
              value={form.admin_password}
              onChange={(e) => setForm({ ...form, admin_password: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg"
            />
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
