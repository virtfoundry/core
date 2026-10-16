import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authService } from '../lib/auth';
import { VirtFoundryLogo } from '../components/VirtFoundryLogo';
import { useI18n } from '../lib/i18n';
import { useTheme } from '../lib/theme';
import { Moon, Sun } from 'lucide-react';
import { formInputClass } from '../components/shell';
import { APP_VERSION } from '../lib/version';

const prefBtn =
  'h-9 border border-outline-variant rounded-lg bg-card text-on-surface cursor-pointer flex items-center justify-center focus-visible:ring-2 focus-visible:ring-accent';

function LoginPreferences() {
  const { t, locale, setLocale } = useI18n();
  const { isDark, toggleTheme } = useTheme();
  return (
    <div className="flex gap-2">
      <button
        type="button"
        onClick={() => setLocale(locale === 'pt' ? 'en' : 'pt')}
        aria-label={t('topbar.langAria')}
        className={`${prefBtn} px-3 text-[13px] font-semibold font-mono`}
      >
        {locale.toUpperCase()}
      </button>
      <button type="button" onClick={toggleTheme} aria-label={t('topbar.themeAria')} className={`${prefBtn} w-9`}>
        {isDark ? <Sun size={16} strokeWidth={1.8} /> : <Moon size={16} strokeWidth={1.8} />}
      </button>
    </div>
  );
}

export function Login() {
  const { t } = useI18n();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [retryIn, setRetryIn] = useState(0);
  const navigate = useNavigate();

  useEffect(() => {
    if (retryIn <= 0) return;
    const id = setTimeout(() => setRetryIn((s) => s - 1), 1000);
    return () => clearTimeout(id);
  }, [retryIn]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await authService.login({ username, password });
      navigate('/');
    } catch (err) {
      const e = err as { attemptsRemaining?: number; retryAfter?: number };
      if (e.retryAfter) {
        setRetryIn(e.retryAfter);
      } else {
        setError(
          e.attemptsRemaining === undefined
            ? t('login.invalidCredentials')
            : `${t('login.invalidCredentials')}. ${t('login.attemptsRemaining').replace('{n}', String(e.attemptsRemaining))}`,
        );
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-wrap bg-background text-on-surface text-[15px]">
      <section className="hidden lg:flex flex-[1_1_520px] flex-col justify-between box-border px-14 py-10 text-[#DAE2FD] bg-gradient-to-br from-[#0B1326] to-[#14264F]">
        <div className="w-60"><VirtFoundryLogo fullWidth variant="dark" /></div>
        <div className="flex flex-col gap-5 max-w-[460px]">
          <h1 className="m-0 text-[40px] leading-[46px] font-semibold tracking-[-0.03em]">{t('login.pitch')}</h1>
          <ul className="m-0 p-0 list-none flex flex-col gap-3 text-[#B4C5FF]">
            {(['login.point1', 'login.point2', 'login.point3'] as const).map((k) => (
              <li key={k} className="flex gap-2.5 items-center">
                <span className="w-2 h-2 rounded-full bg-[#2563EB]" aria-hidden />
                {t(k)}
              </li>
            ))}
          </ul>
        </div>
        <span className="font-mono text-[13px] text-[#8E9BC0]">v{APP_VERSION}</span>
      </section>

      <section className="flex flex-[1_1_480px] flex-col box-border px-6 py-8 sm:px-14">
        <div className="flex items-center justify-between gap-4">
          <div className="w-[180px] lg:hidden"><VirtFoundryLogo fullWidth /></div>
          <div className="ml-auto"><LoginPreferences /></div>
        </div>

        <form onSubmit={handleSubmit} className="my-auto w-full max-w-[400px] self-center flex flex-col gap-[18px] py-8">
          <div className="flex flex-col gap-1.5">
            <h2 className="m-0 text-[28px] leading-[34px] font-semibold tracking-[-0.02em]">{t('login.heading')}</h2>
            <span className="text-on-surface-variant">{t('login.intro')}</span>
          </div>

          <label className="flex flex-col gap-1.5">
            <span className="font-medium">{t('login.username')}</span>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              className={formInputClass}
              placeholder={t('login.usernamePlaceholder')}
              required
            />
          </label>

          <label className="flex flex-col gap-1.5">
            <span className="font-medium">{t('login.password')}</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              className={formInputClass}
              placeholder="••••••••"
              required
            />
          </label>

          {retryIn > 0 && (
            <div role="alert" className="p-3 rounded-lg border border-bad-fg/30 bg-bad-bg text-bad-fg text-sm">
              {t('login.tooManyAttempts').replace(
                '{time}',
                `${Math.floor(retryIn / 60)}:${String(retryIn % 60).padStart(2, '0')}`,
              )}
            </div>
          )}

          {error && retryIn <= 0 && (
            <div role="alert" className="p-3 rounded-lg border border-bad-fg/30 bg-bad-bg text-bad-fg text-sm">
              {error}
            </div>
          )}

          <button type="submit" disabled={loading || retryIn > 0} className="btn-primary w-full h-11">
            {loading ? t('login.submitting') : t('login.submit')}
          </button>

          <a href="https://github.com/virtfoundry/core/tree/main/core/ui/docs" target="_blank" rel="noopener noreferrer" className="text-link text-sm no-underline hover:underline">
            {t('login.docsHint')}
          </a>
        </form>

        <span className="text-on-surface-variant text-[13px] self-center w-full max-w-[400px]">
          {t('login.tenantAdminHint').replace('{slug}', 'tenant')}
        </span>
      </section>
    </div>
  );
}
