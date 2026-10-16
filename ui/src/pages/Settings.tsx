import clsx from 'clsx';
import { PageHeader, SurfaceCard } from '../components/shell';
import { useI18n, type TranslationKey } from '../lib/i18n';
import { useTheme, type Theme } from '../lib/theme';
import { APP_VERSION } from '../lib/version';
import { roleBadgeLabel } from '../lib/vm-display';
import { useAppSelector } from '../store/hooks';
import { selectUser } from '../store/authSlice';

function Choice({ on, title, hint, onClick }: { on: boolean; title: string; hint: string; onClick: () => void }) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={on}
      onClick={onClick}
      className={clsx(
        'p-4 rounded-[10px] border-2 text-left flex flex-col gap-1.5 transition-colors focus-visible:ring-2 focus-visible:ring-accent',
        on ? 'border-accent bg-accent-soft' : 'border-outline-variant bg-card hover:bg-sunk',
      )}
    >
      <span className="font-semibold text-on-surface">{title}</span>
      <span className="text-[13px] leading-[19px] text-on-surface-variant">{hint}</span>
    </button>
  );
}

const grid = 'grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-3';

export function Settings() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const user = useAppSelector(selectUser);

  const themes: { id: Theme; label: TranslationKey; hint: TranslationKey }[] = [
    { id: 'light', label: 'sidebar.themeLight', hint: 'settings.lightHint' },
    { id: 'dark', label: 'sidebar.themeDark', hint: 'settings.darkHint' },
    { id: 'system', label: 'sidebar.themeSystem', hint: 'settings.systemHint' },
  ];

  const kv: [string, string | undefined, boolean?][] = [
    [t('settings.username'), user?.username, true],
    [t('settings.email'), user?.email],
    [t('settings.role'), user?.role ? roleBadgeLabel(user.role) : undefined, true],
    [t('settings.tenant'), user?.tenant_id, true],
  ];

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title={t('sidebar.settings')} subtitle={t('settings.subtitle')} />

      <div className="flex flex-wrap gap-4 items-stretch">
        <SurfaceCard title={t('settings.profile')} className="flex-[1_1_480px] min-w-0">
          <dl className="m-0 flex flex-col">
            {kv.filter(([, v]) => v).map(([k, v, mono]) => (
              <div key={k} className="flex justify-between gap-4 py-[9px] border-t border-line3 first:border-t-0">
                <dt className="text-on-surface-variant">{k}</dt>
                <dd className={clsx('m-0 text-right text-on-surface', mono && 'font-mono')}>{v}</dd>
              </div>
            ))}
          </dl>
        </SurfaceCard>

        <SurfaceCard title={t('settings.about')} className="flex-[1_1_360px] min-w-0">
          <dl className="m-0 flex flex-col">
            <div className="flex justify-between gap-4 py-[9px]">
              <dt className="text-on-surface-variant">{t('settings.version')}</dt>
              <dd className="m-0 font-mono text-on-surface">v{APP_VERSION}</dd>
            </div>
          </dl>
        </SurfaceCard>

        <SurfaceCard title={t('sidebar.theme')} className="flex-[1_1_100%] min-w-0">
          <div role="radiogroup" aria-label={t('sidebar.theme')} className={grid}>
            {themes.map((o) => (
              <Choice key={o.id} on={theme === o.id} title={t(o.label)} hint={t(o.hint)} onClick={() => setTheme(o.id)} />
            ))}
          </div>
        </SurfaceCard>

        <SurfaceCard title={t('sidebar.language')} className="flex-[1_1_100%] min-w-0">
          <div role="radiogroup" aria-label={t('sidebar.language')} className={grid}>
            <Choice on={locale === 'pt'} title="Português" hint="pt-BR" onClick={() => setLocale('pt')} />
            <Choice on={locale === 'en'} title="English" hint="en-US" onClick={() => setLocale('en')} />
          </div>
        </SurfaceCard>
      </div>
    </div>
  );
}
