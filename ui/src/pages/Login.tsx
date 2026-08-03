import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authService } from '../lib/auth';
import { VirtFoundryLogo } from '../components/VirtFoundryLogo';
import { LanguageToggle, useI18n } from '../lib/i18n';
import { ThemeToggle } from '../lib/theme';
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
    <div className="min-h-screen flex">
      <div
        className="hidden lg:flex lg:w-1/2 bg-cover bg-center p-12 flex-col justify-between relative"
        style={{ backgroundImage: `url(${loginBg})` }}
      >
        <div className="absolute inset-0 bg-gradient-to-br from-brand-600/95 to-brand-800/95" />

        <div className="relative z-10 flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <VirtFoundryLogo size={48} className="ring-2 ring-white/20 rounded-xl" />
            <div>
              <h1 className="text-3xl font-bold text-white">VirtFoundry</h1>
              <p className="text-brand-200">{t('login.tagline')}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <ThemeToggle onDark compact />
            <LanguageToggle onDark />
          </div>
        </div>

        <div className="relative z-10 space-y-6">
          <h2 className="text-4xl font-bold text-white leading-tight">
            {t('login.headline')}
          </h2>
          <p className="text-xl text-brand-100 max-w-md">
            {t('login.subheadline')}
          </p>
        </div>

        <div className="relative z-10 grid grid-cols-3 gap-6 text-white">
          <div>
            <div className="text-3xl font-bold">99.9%</div>
            <div className="text-brand-200 text-sm">{t('login.uptimeSla')}</div>
          </div>
          <div>
            <div className="text-3xl font-bold">10K+</div>
            <div className="text-brand-200 text-sm">{t('login.managedVms')}</div>
          </div>
          <div>
            <div className="text-3xl font-bold">24/7</div>
            <div className="text-brand-200 text-sm">{t('login.support')}</div>
          </div>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center p-8 bg-gray-50 dark:bg-dark-200">
        <div className="w-full max-w-md">
          <div className="mb-8 flex items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <VirtFoundryLogo size={40} />
              <div>
                <span className="text-2xl font-bold text-gray-900 dark:text-white">VirtFoundry</span>
                <p className="text-xs text-gray-500 lg:hidden">{t('login.tagline')}</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <ThemeToggle compact />
              <LanguageToggle />
            </div>
          </div>

          <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">{t('login.welcome')}</h2>
          <p className="text-gray-500 mb-8">{t('login.subtitle')}</p>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('login.username')}</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-transparent transition dark:bg-dark-100"
                placeholder={t('login.usernamePlaceholder')}
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('login.password')}</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-brand-500 focus:border-transparent transition dark:bg-dark-100"
                placeholder="••••••••"
                required
              />
            </div>

            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2">
                <input type="checkbox" className="w-4 h-4 text-brand-500 rounded" />
                <span className="text-sm text-gray-600 dark:text-gray-400">{t('login.remember')}</span>
              </label>
              <a href="#" className="text-sm text-brand-500 hover:text-brand-600">{t('login.forgotPassword')}</a>
            </div>

            {error && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-600 text-sm">
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

          <p className="mt-8 text-center text-sm text-gray-500">
            {t('login.version')}
          </p>
        </div>
      </div>
    </div>
  );
}
