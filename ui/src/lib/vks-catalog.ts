/** Keep in sync with vks-image-factory/VERSIONS.md published node tags. */
export const PUBLISHED_NODE_VERSIONS = [
  { k8s: 'v1.36.5', templateHint: 'ubuntu-node-1-36-5' },
] as const;

const NODE_RE = /^ubuntu-node-(\d+)-(\d+)-(\d+)$/;

export function parseTemplateK8sVersion(templateName: string): string | null {
  const m = NODE_RE.exec(templateName);
  if (!m) return null;
  return `v${m[1]}.${m[2]}.${m[3]}`;
}

export function resolveVKSVersionOptions(
  templates: Array<{ name: string }>,
): Array<{ kubernetes_version: string; template: string }> {
  const byVersion = new Map<string, string>();
  for (const t of templates) {
    const ver = parseTemplateK8sVersion(t.name);
    if (!ver) continue;
    if (!PUBLISHED_NODE_VERSIONS.some((p) => p.k8s === ver)) continue;
    if (!byVersion.has(ver)) byVersion.set(ver, t.name);
  }
  return [...byVersion.entries()]
    .sort(([a], [b]) => b.localeCompare(a, undefined, { numeric: true }))
    .map(([kubernetes_version, template]) => ({ kubernetes_version, template }));
}
