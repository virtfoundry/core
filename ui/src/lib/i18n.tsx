import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';

export type Locale = 'pt' | 'en';

const STORAGE_KEY = 'nimbus_locale';

const dict = {
  pt: {
    'nav.dashboard': 'Dashboard',
    'nav.vms': 'Máquinas Virtuais',
    'nav.sshKeys': 'Chaves SSH',
    'nav.vmSnapshots': 'Snapshots de VM',
    'nav.volumes': 'Volumes',
    'nav.volumeSnapshots': 'Snapshots de Volume',
    'nav.vpcs': 'VPCs',
    'nav.networks': 'Redes',
    'nav.securityGroups': 'Grupos de Segurança',
    'nav.tenants': 'Tenants',
    'nav.logout': 'Sair',
    'nav.selectTenant': '— Selecionar tenant —',
    'common.create': 'Criar',
    'common.cancel': 'Cancelar',
    'common.save': 'Salvar',
    'common.edit': 'Editar',
    'common.delete': 'Remover',
    'common.loading': 'Carregando...',
    'common.search': 'Buscar',
    'common.name': 'Nome',
    'common.state': 'Estado',
    'common.confirmDelete': 'Tem certeza que deseja remover?',
    'common.confirmDeleteTitle': 'Confirmar remoção',
    'common.confirmDeleteMessage': 'Esta ação não pode ser desfeita. O recurso abaixo será removido permanentemente.',
    'common.selectTenant': 'Selecione um tenant para continuar.',
    'common.errorLoad': 'Erro ao carregar dados',
    'common.retry': 'Tentar novamente',
    'vpcs.title': 'VPCs',
    'vpcs.subtitle': 'redes privadas isoladas',
    'vpcs.create': 'Criar VPC',
    'vpcs.empty': 'Nenhuma VPC',
    'vpcs.modalCreate': 'Criar VPC',
    'vpcs.modalEdit': 'Editar VPC',
    'vpcs.cidrLabel': 'Intervalo IP da VPC',
    'vpcs.cidrHint': 'Escolha um bloco privado RFC1918. Sub-redes serão alocadas dentro deste intervalo.',
    'vpcs.privateNet': 'Rede privada',
    'vpcs.defaultSubnetInfo': 'Uma sub-rede padrão será criada automaticamente dentro desta VPC, como na GCP, para que VMs possam ser conectadas imediatamente.',
    'vpcs.deleteTitle': 'Remover VPC',
    'networks.title': 'Redes',
    'networks.subtitle': 'sub-redes configuradas',
    'networks.create': 'Criar Rede',
    'networks.empty': 'Nenhuma rede encontrada',
    'networks.modalCreate': 'Criar sub-rede',
    'networks.modalEdit': 'Editar rede',
    'networks.vpc': 'VPC',
    'networks.selectVpc': 'Selecione uma VPC',
    'networks.cidr': 'CIDR',
    'networks.insideVpc': 'Dentro da VPC',
    'networks.ipRange': 'Intervalo IP',
    'networks.defaultBadge': 'Sub-rede padrão',
    'networks.defaultHint': 'Criada automaticamente com a VPC — necessária para conectar VMs (padrão GCP).',
    'networks.emptyHint': 'Crie uma VPC primeiro — cada VPC provisiona uma sub-rede padrão automaticamente.',
    'networks.deleteTitle': 'Remover sub-rede',
    'sg.title': 'Grupos de Segurança',
    'sg.subtitle': 'regras de firewall',
    'sg.create': 'Criar Grupo',
    'sg.empty': 'Nenhum grupo encontrado',
    'sg.modalCreate': 'Criar Grupo de Segurança',
    'sg.modalEdit': 'Editar Grupo de Segurança',
    'sg.description': 'Descrição',
    'sg.noDescription': 'Sem descrição',
    'sg.rules': 'Regras',
    'sg.rulesCount': 'Regras',
    'sg.deleteTitle': 'Remover grupo de segurança',
    'logs.title': 'Logs da instância',
    'logs.refresh': 'Atualizar logs',
    'logs.loading': 'Carregando...',
    'logs.openVelas': 'Abrir no Velas',
    'logs.startVm': 'Inicie a VM para visualizar logs.',
    'logs.clickRefresh': 'Clique em Atualizar logs para carregar.',
    'logs.integration': 'integração Velas',
    'lang.pt': 'PT',
    'lang.en': 'EN',
    'ssh.title': 'Chaves SSH',
    'ssh.subtitle': 'pares de chaves para autenticação',
    'ssh.create': 'Gerar chave',
    'ssh.register': 'Registrar chave',
    'ssh.empty': 'Nenhuma chave SSH',
    'ssh.fingerprint': 'Fingerprint',
    'ssh.publicKey': 'Chave pública',
    'ssh.privateKeyTitle': 'Chave privada gerada',
    'ssh.privateKeyWarning': 'Esta chave privada não é armazenada na plataforma. Copie e guarde em local seguro — ela não será exibida novamente.',
    'ssh.copy': 'Copiar',
    'ssh.copied': 'Copiado!',
    'ssh.deleteTitle': 'Remover chave SSH',
    'ssh.registerHint': 'Cole a chave pública (ssh-ed25519, ssh-rsa, ecdsa...)',
    'ssh.namePlaceholder': 'minha-chave-prod',
    'ssh.deployHint': 'Selecione uma chave para injetar via cloud-init (Linux). Não é possível alterar após o deploy.',
    'ssh.exposeHint': 'Expõe SSH via NodePort no host (porta externa automática).',
    'ssh.dataVolumeHint': 'Anexa um volume de dados e formata automaticamente em /mnt/iops.',
    'ssh.connectTitle': 'Acesso SSH',
    'ssh.expose': 'Expor SSH',
    'ssh.notExposed': 'SSH não exposto externamente.',
    'ssh.command': 'Comando',
  },
  en: {
    'nav.dashboard': 'Dashboard',
    'nav.vms': 'Virtual Machines',
    'nav.sshKeys': 'SSH Keys',
    'nav.vmSnapshots': 'VM Snapshots',
    'nav.volumes': 'Volumes',
    'nav.volumeSnapshots': 'Volume Snapshots',
    'nav.vpcs': 'VPCs',
    'nav.networks': 'Networks',
    'nav.securityGroups': 'Security Groups',
    'nav.tenants': 'Tenants',
    'nav.logout': 'Sign out',
    'nav.selectTenant': '— Select tenant —',
    'common.create': 'Create',
    'common.cancel': 'Cancel',
    'common.save': 'Save',
    'common.edit': 'Edit',
    'common.delete': 'Delete',
    'common.loading': 'Loading...',
    'common.search': 'Search',
    'common.name': 'Name',
    'common.state': 'State',
    'common.confirmDelete': 'Are you sure you want to delete this?',
    'common.confirmDeleteTitle': 'Confirm deletion',
    'common.confirmDeleteMessage': 'This action cannot be undone. The resource below will be permanently removed.',
    'common.selectTenant': 'Select a tenant to continue.',
    'common.errorLoad': 'Failed to load data',
    'common.retry': 'Retry',
    'vpcs.title': 'VPCs',
    'vpcs.subtitle': 'isolated private networks',
    'vpcs.create': 'Create VPC',
    'vpcs.empty': 'No VPCs',
    'vpcs.modalCreate': 'Create VPC',
    'vpcs.modalEdit': 'Edit VPC',
    'vpcs.cidrLabel': 'VPC IP range',
    'vpcs.cidrHint': 'Choose a private RFC1918 block. Subnets will be allocated inside this range.',
    'vpcs.privateNet': 'Private network',
    'vpcs.defaultSubnetInfo': 'A default subnet will be auto-created inside this VPC (GCP-style) so VMs can be attached immediately.',
    'vpcs.deleteTitle': 'Delete VPC',
    'networks.title': 'Networks',
    'networks.subtitle': 'configured subnets',
    'networks.create': 'Create Network',
    'networks.empty': 'No networks found',
    'networks.modalCreate': 'Create subnet',
    'networks.modalEdit': 'Edit network',
    'networks.vpc': 'VPC',
    'networks.selectVpc': 'Select a VPC',
    'networks.cidr': 'CIDR',
    'networks.insideVpc': 'Inside VPC',
    'networks.ipRange': 'IP range',
    'networks.defaultBadge': 'Default subnet',
    'networks.defaultHint': 'Auto-created with the VPC — required to attach VMs (GCP-style).',
    'networks.emptyHint': 'Create a VPC first — each VPC auto-provisions a default subnet.',
    'networks.deleteTitle': 'Delete subnet',
    'sg.title': 'Security Groups',
    'sg.subtitle': 'firewall rules',
    'sg.create': 'Create Group',
    'sg.empty': 'No security groups found',
    'sg.modalCreate': 'Create Security Group',
    'sg.modalEdit': 'Edit Security Group',
    'sg.description': 'Description',
    'sg.noDescription': 'No description',
    'sg.rules': 'Rules',
    'sg.rulesCount': 'Rules',
    'sg.deleteTitle': 'Delete security group',
    'logs.title': 'Instance logs',
    'logs.refresh': 'Refresh logs',
    'logs.loading': 'Loading...',
    'logs.openVelas': 'Open in Velas',
    'logs.startVm': 'Start the VM to view logs.',
    'logs.clickRefresh': 'Click Refresh logs to load.',
    'logs.integration': 'Velas integration',
    'lang.pt': 'PT',
    'lang.en': 'EN',
    'ssh.title': 'SSH Keys',
    'ssh.subtitle': 'key pairs for authentication',
    'ssh.create': 'Generate key',
    'ssh.register': 'Register key',
    'ssh.empty': 'No SSH keys',
    'ssh.fingerprint': 'Fingerprint',
    'ssh.publicKey': 'Public key',
    'ssh.privateKeyTitle': 'Generated private key',
    'ssh.privateKeyWarning': 'This private key is not stored on the platform. Copy and save it securely — it will not be shown again.',
    'ssh.copy': 'Copy',
    'ssh.copied': 'Copied!',
    'ssh.deleteTitle': 'Delete SSH key',
    'ssh.registerHint': 'Paste the public key (ssh-ed25519, ssh-rsa, ecdsa...)',
    'ssh.namePlaceholder': 'my-prod-key',
    'ssh.deployHint': 'Select a key to inject via cloud-init (Linux). Cannot be changed after deploy.',
    'ssh.exposeHint': 'Exposes SSH via NodePort on the host (automatic external port).',
    'ssh.dataVolumeHint': 'Attaches a data volume and auto-formats at /mnt/iops.',
    'ssh.connectTitle': 'SSH access',
    'ssh.expose': 'Expose SSH',
    'ssh.notExposed': 'SSH is not exposed externally.',
    'ssh.command': 'Command',
  },
} as const;

export type TranslationKey = keyof typeof dict.pt;

type I18nContextValue = {
  locale: Locale;
  setLocale: (l: Locale) => void;
  t: (key: TranslationKey) => string;
};

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    return saved === 'en' ? 'en' : 'pt';
  });

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    localStorage.setItem(STORAGE_KEY, l);
  }, []);

  const t = useCallback((key: TranslationKey) => dict[locale][key] ?? key, [locale]);

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error('useI18n must be used within I18nProvider');
  return ctx;
}

export function LanguageToggle() {
  const { locale, setLocale, t } = useI18n();
  return (
    <div className="flex rounded-lg border overflow-hidden text-sm">
      <button
        type="button"
        onClick={() => setLocale('pt')}
        className={`px-3 py-1.5 ${locale === 'pt' ? 'bg-nimbus-500 text-white' : 'hover:bg-gray-100'}`}
      >
        {t('lang.pt')}
      </button>
      <button
        type="button"
        onClick={() => setLocale('en')}
        className={`px-3 py-1.5 ${locale === 'en' ? 'bg-nimbus-500 text-white' : 'hover:bg-gray-100'}`}
      >
        {t('lang.en')}
      </button>
    </div>
  );
}
