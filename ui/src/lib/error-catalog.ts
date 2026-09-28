export type ErrorCatalogEntry = {
  id: string;
  match: RegExp;
  title: string;
  titleEn: string;
  fix: string;
  fixEn: string;
  docsHint: string;
};

/** Known Instance/KubeVirt error patterns → suggested fix (preview catalog). */
export const ERROR_CATALOG: ErrorCatalogEntry[] = [
  {
    id: 'nic-missing',
    match: /nic|networkattachment|multus|nad|CNI|network interface/i,
    title: 'NIC / Multus ausente',
    titleEn: 'NIC / Multus missing',
    fix: 'Confirme NAD da sub-rede, NetworkAttachmentDefinition no namespace do tenant e se o deploy usou Multus (não só pod network).',
    fixEn: 'Confirm the subnet NAD, NetworkAttachmentDefinition in the tenant namespace, and that deploy used Multus (not pod-only).',
    docsHint: 'Deploy → Network → Multus/VPC',
  },
  {
    id: 'image-deny',
    match: /image.*(deny|pull|not found|unauthorized|denied)|ErrImagePull|ImagePullBackOff/i,
    title: 'Imagem negada / pull falhou',
    titleEn: 'Image deny / pull failed',
    fix: 'Verifique registry, pull secrets do tenant e se o template aponta para uma imagem acessível no cluster.',
    fixEn: 'Check registry access, tenant pull secrets, and that the template image is reachable from the cluster.',
    docsHint: 'Templates → image URL',
  },
  {
    id: 'insufficient-resources',
    match: /insufficient|Unschedulable|FailedScheduling|out of memory|cpu.*request/i,
    title: 'Sem capacidade no nó',
    titleEn: 'Node capacity exhausted',
    fix: 'Reduza a oferta (CPU/RAM) ou libere nós. Em lab, offerings small costumam agendar melhor.',
    fixEn: 'Lower the offering (CPU/RAM) or free nodes. In labs, small offerings schedule more reliably.',
    docsHint: 'Offerings → smaller size',
  },
  {
    id: 'volume-attach',
    match: /volume.*(attach|bound|pvc|claim)|PersistentVolumeClaim|Multi-Attach/i,
    title: 'Volume / PVC',
    titleEn: 'Volume / PVC issue',
    fix: 'Confirme PVC Bound, storage class e se o volume não está attachado a outra VM.',
    fixEn: 'Confirm PVC is Bound, storage class is valid, and the volume is not attached to another VM.',
    docsHint: 'VM → Storage',
  },
  {
    id: 'guest-boot',
    match: /guest|cloud-init|qemu.?agent|boot|DataVolume/i,
    title: 'Boot / cloud-init / agent',
    titleEn: 'Boot / cloud-init / agent',
    fix: 'Abra o console VNC, confira cloud-init e se o guest agent está instalado (badge no detalhe é preview local).',
    fixEn: 'Open VNC console, check cloud-init, and whether the guest agent is installed (detail badge is local preview).',
    docsHint: 'VM detail → Console / cloud-init',
  },
  {
    id: 'ssh-key',
    match: /ssh.?key|authorized_keys|public.?key/i,
    title: 'Chave SSH',
    titleEn: 'SSH key',
    fix: 'Registre uma chave em Chaves SSH e selecione-a no deploy (Linux).',
    fixEn: 'Register a key under SSH Keys and select it at deploy (Linux).',
    docsHint: 'SSH Keys',
  },
];

export function matchErrorCatalog(message?: string | null, locale: 'pt' | 'en' = 'pt') {
  if (!message) return null;
  for (const entry of ERROR_CATALOG) {
    if (entry.match.test(message)) {
      return {
        id: entry.id,
        title: locale === 'en' ? entry.titleEn : entry.title,
        fix: locale === 'en' ? entry.fixEn : entry.fix,
        docsHint: entry.docsHint,
      };
    }
  }
  return null;
}

export function validateCloudInitYaml(text: string): { ok: boolean; issue?: string } {
  const trimmed = text.trim();
  if (!trimmed) return { ok: true };
  if (!trimmed.startsWith('#cloud-config') && !trimmed.startsWith('---') && !/^[a-zA-Z0-9_]+:/.test(trimmed)) {
    return { ok: false, issue: 'Esperado YAML cloud-config (#cloud-config ou chave: valor).' };
  }
  // Lightweight structural checks — not a full YAML parser.
  const lines = trimmed.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!line.trim() || line.trim().startsWith('#')) continue;
    const indent = line.match(/^\s*/)?.[0].length ?? 0;
    if (indent % 2 !== 0) {
      return { ok: false, issue: `Indentação ímpar na linha ${i + 1} (use 2 espaços).` };
    }
  }
  return { ok: true };
}

/** Mask password / private key fragments in cloud-init preview. */
export function maskCloudInitSecrets(text: string): string {
  return text
    .replace(/(password\s*:\s*)(['"]?)[^\n'"]+/gi, '$1$2********')
    .replace(/(passwd\s*:\s*)(['"]?)[^\n'"]+/gi, '$1$2********')
    .replace(/(-----BEGIN[^-]+PRIVATE KEY-----)[\s\S]*?(-----END[^-]+PRIVATE KEY-----)/g, '$1\n********\n$2');
}
