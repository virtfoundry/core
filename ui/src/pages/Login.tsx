import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import { authService } from '../lib/auth';
import { VirtFoundryLogo } from '../components/VirtFoundryLogo';
import { LanguageToggle, useI18n } from '../lib/i18n';
import { ThemeToggle } from '../lib/theme';
import { useAppSelector } from '../store/hooks';
import { selectIsDarkTheme } from '../store/themeSlice';
import { formInputClass } from '../components/shell';
import loginBg from '../assets/login-bg.svg';
import { appVersionLabel } from '../lib/version';

function LoginPreferences({ onDark = false }: { onDark?: boolean }) {
  return (
    <div className="flex items-center gap-2 shrink-0">
      <ThemeToggle onDark={onDark} compact />
      <LanguageToggle onDark={onDark} />
    </div>
  );
}

export function Login() {
  const { t } = useI18n();
  const isDark = useAppSelector(selectIsDarkTheme);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await authService.login({ username, password });
      navigate('/');
    } catch (err) {
      const msg = err instanceof Error ? err.message : '';
      setError(msg && msg !== 'Rejected' ? msg : t('login.invalidCredentials'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col bg-background">
      {/* Full-width top bar — edge to edge */}
      <header
        className={clsx(
          'relative z-20 w-full shrink-0 flex items-center justify-between gap-6 px-6 sm:px-8 lg:px-10 xl:px-14 py-5 bg-background border-b border-outline-variant/50',
          'lg:absolute lg:inset-x-0 lg:top-0 lg:border-none lg:bg-transparent',
        )}
      >
        <div
          className={clsx(
            'w-[200px] sm:w-[240px] lg:w-[280px] shrink-0',
            isDark && 'lg:drop-shadow-[0_2px_12px_rgba(0,0,0,0.5)]',
          )}
        >
          <VirtFoundryLogo
            fullWidth
            variant={isDark ? 'dark' : 'light'}
            className="hidden lg:block w-full"
          />
          <VirtFoundryLogo fullWidth className="lg:hidden w-full" />
        </div>
        <LoginPreferences onDark={isDark} />
      </header>

      <div className="flex flex-1 flex-col lg:flex-row min-h-0">
        {/* Brand panel — desktop only */}
        <aside
          className="hidden lg:flex lg:w-1/2 xl:w-[52%] flex-col relative isolate min-h-[calc(100vh-0px)]"
          style={{ backgroundImage: `url(${loginBg})`, backgroundSize: 'cover', backgroundPosition: 'center' }}
        >
          <div
            className={
              isDark
                ? 'absolute inset-0 bg-gradient-to-br from-brand-800 via-brand-900 to-[#0a1628] pointer-events-none'
                : 'absolute inset-0 bg-gradient-to-br from-brand-50 via-surface-container-low to-brand-100 pointer-events-none'
            }
            aria-hidden="true"
          />

          {/* Spacer for overlapping top bar */}
          <div className="relative z-10 h-[88px] shrink-0" aria-hidden="true" />

          <div className="relative z-10 flex-1 flex flex-col justify-center px-10 xl:px-14 py-8">
            <div className="max-w-lg space-y-5">
              <h2
                className={
                  isDark
                    ? 'font-headline text-headline-xl font-bold text-white leading-tight'
                    : 'font-headline text-headline-xl font-bold text-brand-900 leading-tight'
                }
              >
                {t('login.headline')}
              </h2>
              <p
                className={
                  isDark
                    ? 'text-lg text-white/90 leading-relaxed'
                    : 'text-lg text-on-surface-variant leading-relaxed'
                }
              >
                {t('login.subheadline')}
              </p>
            </div>
          </div>

          <footer
            className={
              isDark
                ? 'relative z-10 grid grid-cols-3 gap-8 px-10 xl:px-14 pb-10 text-white'
                : 'relative z-10 grid grid-cols-3 gap-8 px-10 xl:px-14 pb-10 text-brand-900'
            }
          >
            <div>
              <div className="font-headline text-headline-lg font-bold">99.9%</div>
              <div className={isDark ? 'text-white/80 text-sm mt-1' : 'text-on-surface-variant text-sm mt-1'}>
                {t('login.uptimeSla')}
              </div>
            </div>
            <div>
              <div className="font-headline text-headline-lg font-bold">10K+</div>
              <div className={isDark ? 'text-white/80 text-sm mt-1' : 'text-on-surface-variant text-sm mt-1'}>
                {t('login.managedVms')}
              </div>
            </div>
            <div>
              <div className="font-headline text-headline-lg font-bold">24/7</div>
              <div className={isDark ? 'text-white/80 text-sm mt-1' : 'text-on-surface-variant text-sm mt-1'}>
                {t('login.support')}
              </div>
            </div>
          </footer>
        </aside>

        {/* Sign-in panel */}
        <main className="flex-1 flex flex-col min-h-0 relative lg:pt-[88px]">
          <div className="flex-1 flex items-center justify-center px-6 py-10 sm:px-10 lg:px-16">
            <div className="w-full max-w-[420px]">
              <div className="mb-8 space-y-2">
                <h2 className="font-headline text-headline-md font-bold text-on-surface">{t('login.welcome')}</h2>
                <p className="text-on-surface-variant">{t('login.subtitle')}</p>
              </div>

              <form onSubmit={handleSubmit} className="space-y-5">
                <div>
                  <label className="block text-sm font-medium text-on-surface mb-1">{t('login.username')}</label>
                  <input
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    className={formInputClass}
                    placeholder={t('login.usernamePlaceholder')}
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-on-surface mb-1">{t('login.password')}</label>
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={formInputClass}
                    placeholder="••••••••"
                    required
                  />
                </div>

                <div className="flex items-center justify-between gap-4">
                  <label className="flex items-center gap-2 min-w-0">
                    <input type="checkbox" className="w-4 h-4 rounded border-outline-variant text-primary-container shrink-0" />
                    <span className="text-sm text-on-surface-variant">{t('login.remember')}</span>
                  </label>
                  <a href="#" className="text-sm text-primary hover:text-primary-fixed-dim shrink-0">
                    {t('login.forgotPassword')}
                  </a>
                </div>

                {error && (
                  <div
                    className="p-3 rounded-lg border border-error/30 text-error text-sm inner-glow"
                    style={{ backgroundColor: 'color-mix(in srgb, var(--vf-error-container) 20%, transparent)' }}
                  >
                    {error}
                  </div>
                )}

                <button type="submit" disabled={loading} className="btn-primary w-full py-3 font-semibold">
                  {loading ? t('login.submitting') : t('login.submit')}
                </button>
              </form>

              <p className="mt-4 text-center text-xs text-on-surface-variant">
                {t('login.tenantAdminHint').replace('{slug}', 'tenant')}
              </p>

              <p className="mt-8 text-center text-sm text-on-surface-variant">{appVersionLabel()}</p>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}
