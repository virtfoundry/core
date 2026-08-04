import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authService } from '../lib/auth';
import { VirtFoundryLogo } from '../components/VirtFoundryLogo';
import { LanguageToggle, useI18n } from '../lib/i18n';
import { ThemeToggle } from '../lib/theme';
import { formInputClass } from '../components/shell';
import loginBg from '../assets/login-bg.svg';

export function Login() {
  const { t } = useI18n();
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
    } catch {
      setError(t('login.invalidCredentials'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex bg-background">
      <div
        className="hidden lg:flex lg:w-1/2 bg-cover bg-center p-12 flex-col justify-between relative"
        style={{ backgroundImage: `url(${loginBg})` }}
      >
        <div className="absolute inset-0 bg-gradient-to-br from-primary-container/95 to-brand-800/95" />

        <div className="relative z-10 flex items-start justify-between gap-4">
          <VirtFoundryLogo height={44} variant="dark" />
          <div className="flex items-center gap-2">
            <ThemeToggle onDark compact />
            <LanguageToggle onDark />
          </div>
        </div>

        <div className="relative z-10 space-y-6">
          <h2 className="font-headline text-headline-xl font-bold text-white leading-tight">
            {t('login.headline')}
          </h2>
          <p className="text-xl text-on-primary-container/90 max-w-md">
            {t('login.subheadline')}
          </p>
        </div>

        <div className="relative z-10 grid grid-cols-3 gap-6 text-white">
          <div>
            <div className="font-headline text-headline-lg font-bold">99.9%</div>
            <div className="text-on-primary-container/80 text-sm">{t('login.uptimeSla')}</div>
          </div>
          <div>
            <div className="font-headline text-headline-lg font-bold">10K+</div>
            <div className="text-on-primary-container/80 text-sm">{t('login.managedVms')}</div>
          </div>
          <div>
            <div className="font-headline text-headline-lg font-bold">24/7</div>
            <div className="text-on-primary-container/80 text-sm">{t('login.support')}</div>
          </div>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center p-8 bg-background">
        <div className="w-full max-w-md">
          <div className="mb-8 flex items-center justify-between gap-3">
            <VirtFoundryLogo fullWidth className="max-w-[280px]" />
            <div className="flex items-center gap-2">
              <ThemeToggle compact />
              <LanguageToggle />
            </div>
          </div>

          <h2 className="font-headline text-headline-md font-bold text-on-surface mb-2">{t('login.welcome')}</h2>
          <p className="text-on-surface-variant mb-8">{t('login.subtitle')}</p>

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

            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2">
                <input type="checkbox" className="w-4 h-4 rounded border-outline-variant text-primary-container" />
                <span className="text-sm text-on-surface-variant">{t('login.remember')}</span>
              </label>
              <a href="#" className="text-sm text-primary hover:text-primary-fixed-dim">{t('login.forgotPassword')}</a>
            </div>

            {error && (
              <div className="p-3 rounded-lg border border-error/30 text-error text-sm inner-glow" style={{ backgroundColor: 'color-mix(in srgb, var(--vf-error-container) 20%, transparent)' }}>
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="btn-primary w-full py-3 font-semibold"
            >
              {loading ? t('login.submitting') : t('login.submit')}
            </button>
          </form>

          <p className="mt-8 text-center text-sm text-on-surface-variant">
            {t('login.version')}
          </p>
        </div>
      </div>
    </div>
  );
}
