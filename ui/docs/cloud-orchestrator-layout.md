# Cloud Orchestrator layout (UI experiment)

Branch `feat/cloud-orchestrator-layout` — light + **dark mode** from style guide and HTML mockups.

## Design tokens

| File | Role |
|------|------|
| `src/styles/design-tokens.css` | CSS variables for `:root` (light) and `.dark` |
| `tailwind.config.js` | Maps utilities to `var(--vf-*)` |
| `src/index.css` | `vf-card`, `inner-glow`, buttons, typography |

### Style guide mapping (prints)

| Token | Light | Dark |
|-------|-------|------|
| Primary | `#2563EB` / `#004ac6` | `#b4c5ff` (text) + `#2563eb` (container) |
| Secondary | `#565e74` | `#7bd0ff` |
| Background | `#f8f9fa` | `#0b1326` |
| Card | `#ffffff` | `#1e293b` (border `#334155`) |
| Success | `#16a34a` | `#10b981` |

### Typography

| Role | Font |
|------|------|
| All UI text | JetBrains Mono (`font-sans` / `font-mono` in Tailwind) |
| Headlines | `.font-headline` |
| Labels / data | `.font-label`, `.font-data-mono` |

## Run locally

```bash
cd ui && npm install && npm run dev
```

Toggle dark mode with the header theme button. Compare Dashboard + VMs in both themes.

## Components

- `Layout.tsx` — sidebar `surface-container`, nav glow, fixed header
- `shell/PageHeader`, `SurfaceCard`, `PagePrimitives` — reusable page building blocks
- `Modal`, `ConfirmDialog`, `RefreshButton`, `RefreshingPanel` — token-based overlays/actions
- `CIDRPicker`, `SGRulesEditor`, `ResourceActions` — form helpers on design tokens
- `StatusBadge` — semantic success/warning/error pills

## Pages migrated

| Page | Pattern |
|------|---------|
| Dashboard | Bento grid + stats |
| VMs | PageHeader + table + deploy modals |
| VPCs, Networks, PublicNetwork | Grid cards + CIDR |
| SecurityGroups | Grid cards + rules |
| IAM | TabBar + tables |
| Volumes, Snapshots, VMSnapshots | Tables |
| Templates, SSHKeys | Tables / cards |
| Tenants | Root-only table |
| VMDetail | TabBar + sections |
| Login | Split hero + token form panel |

**Standalone:** `VMConsole` — full-screen VNC (dark chrome, tokenized `btn-console`)

## Backend APIs (layout shell)

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/dashboard/summary` | Aggregated counts, health, recent VM activity |
| `GET /api/v1/search?q=` | Global search across VMs, volumes, VPCs, networks, SGs |
| `GET /api/v1/notifications` | Active alerts (VM errors, transitional states) |

## Next steps

1. Networking topology card (mockup centerpiece)
2. Optional Material Symbols for icon parity
3. Remove or migrate orphan `DataTable` / `StatsCard` if reused later
