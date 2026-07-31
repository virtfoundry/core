import { Server, HardDrive, Globe, Shield, Network } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { listVMs, listVolumes, listVPCs, listSecurityGroups, listNetworks } from '../lib/platform-api';
import { authService } from '../lib/auth';
import { queryKeys } from '../lib/query-keys';
import { RefreshingPanel } from '../components/RefreshingPanel';
import { isVMTransitional } from '../hooks/useRealtimeEvents';

const pollOpts = { refetchInterval: 12_000 as const };

export function Dashboard() {
  const needsTenant = authService.isRoot() && !localStorage.getItem('tenant_id');
  const enabled = !needsTenant;

  const { data: vmData, isFetching: vmFetching, isLoading: vmLoading } = useQuery({
    queryKey: queryKeys.vms,
    queryFn: listVMs,
    enabled,
    refetchInterval: (q) => {
      const vms = q.state.data?.vms || [];
      if (vms.some((v) => isVMTransitional(v.state))) return 3000;
      return pollOpts.refetchInterval;
    },
  });
  const { data: volData, isFetching: volFetching } = useQuery({ queryKey: queryKeys.volumes, queryFn: listVolumes, enabled, ...pollOpts });
  const { data: vpcData, isFetching: vpcFetching } = useQuery({ queryKey: queryKeys.vpcs, queryFn: listVPCs, enabled, ...pollOpts });
  const { data: sgData, isFetching: sgFetching } = useQuery({ queryKey: queryKeys.securityGroups, queryFn: listSecurityGroups, enabled, ...pollOpts });
  const { data: netData, isFetching: netFetching } = useQuery({ queryKey: queryKeys.networks, queryFn: listNetworks, enabled, ...pollOpts });

  const isFetching = vmFetching || volFetching || vpcFetching || sgFetching || netFetching;

  const vms = vmData?.vms || [];
  const running = vms.filter((v) => v.state?.toLowerCase() === 'running').length;

  const stats = [
    { label: 'VMs', value: vms.length, sub: `${running} running`, icon: Server, color: 'bg-blue-500' },
    { label: 'Volumes', value: volData?.volumes?.length || 0, icon: HardDrive, color: 'bg-purple-500' },
    { label: 'VPCs', value: vpcData?.vpcs?.length || 0, icon: Globe, color: 'bg-nimbus-500' },
    { label: 'Security Groups', value: sgData?.security_groups?.length || 0, icon: Shield, color: 'bg-green-500' },
    { label: 'Networks', value: netData?.networks?.length || 0, icon: Network, color: 'bg-indigo-500' },
  ];

  if (needsTenant) {
    return (
      <div className="text-center py-16 text-amber-600">
        Selecione um tenant no header para ver o dashboard do ambiente.
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
        <p className="text-gray-500">Visão geral do tenant · sincronização automática</p>
      </div>

      <RefreshingPanel isFetching={isFetching} isLoading={vmLoading}>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
        {stats.map((stat) => (
          <div key={stat.label} className="bg-white dark:bg-dark-100 rounded-xl p-5 border">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">{stat.label}</p>
                <p className="text-2xl font-bold mt-1">{stat.value}</p>
                {stat.sub && <p className="text-xs text-gray-400 mt-1">{stat.sub}</p>}
              </div>
              <div className={`${stat.color} p-2.5 rounded-lg`}>
                <stat.icon size={20} className="text-white" />
              </div>
            </div>
          </div>
        ))}
      </div>
      </RefreshingPanel>

      <div className="bg-white dark:bg-dark-100 rounded-xl p-6 border">
        <h3 className="font-semibold mb-4">Ações rápidas</h3>
        <div className="flex flex-wrap gap-3">
          {[
            { to: '/vms', label: 'Deploy VM' },
            { to: '/volumes', label: 'Criar volume' },
            { to: '/vpcs', label: 'Nova VPC' },
            { to: '/snapshots', label: 'Snapshot' },
          ].map((a) => (
            <Link key={a.to} to={a.to} className="btn-primary text-sm">
              {a.label}
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}

