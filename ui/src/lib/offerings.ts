import type { ServiceOffering, VMTemplate } from './platform-api';

export function isWindowsTemplate(tmpl?: VMTemplate | null) {
  return tmpl?.os_type?.toLowerCase() === 'windows';
}

export function offeringsForTemplate(offerings: ServiceOffering[], tmpl?: VMTemplate | null) {
  if (isWindowsTemplate(tmpl)) {
    return offerings.filter((o) => o.name === 'windows-large');
  }
  return offerings.filter((o) => o.name !== 'windows-large');
}

export function offeringLabel(o: ServiceOffering) {
  const base = o.display_name
    ? o.display_name
    : (() => {
        const mem = o.memory_mi >= 1024 ? `${(o.memory_mi / 1024).toFixed(0)} GiB` : `${o.memory_mi} MiB`;
        return `${o.name} (${o.cpu} vCPU, ${mem})`;
      })();
  return o.dedicated_cpu ? `${base} · dedicated` : base;
}

export function findOfferingBySpec(offerings: ServiceOffering[], cpu: number, memoryMi: number) {
  return offerings.find((o) => o.cpu === cpu && o.memory_mi === memoryMi);
}

export function findOfferingByName(offerings: ServiceOffering[], name: string) {
  return offerings.find((o) => o.name === name);
}
