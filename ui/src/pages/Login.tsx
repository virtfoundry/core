import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authService } from '../lib/auth';
import { Cloud } from 'lucide-react';
import loginBg from '../assets/login-bg.svg';

export function Login() {
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
      setError('Credenciais inválidas');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex">
      {/* Left side - VirtForge branding */}
      <div 
        className="hidden lg:flex lg:w-1/2 bg-cover bg-center p-12 flex-col justify-between relative"
        style={{ backgroundImage: `url(${loginBg})` }}
      >
        <div className="absolute inset-0 bg-gradient-to-br from-nimbus-600/95 to-nimbus-800/95" />
        
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center">
              <Cloud size={28} className="text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold text-white">VirtForge</h1>
              <p className="text-nimbus-200">Cloud Platform</p>
            </div>
          </div>
        </div>

        <div className="relative z-10 space-y-6">
          <h2 className="text-4xl font-bold text-white leading-tight">
            Gerencie sua infraestrutura em nuvem
          </h2>
          <p className="text-xl text-nimbus-100 max-w-md">
            Plataforma IaaS completa para criar, gerenciar e escalar seus ambientes de nuvem privada.
          </p>
        </div>

        <div className="relative z-10 grid grid-cols-3 gap-6 text-white">
          <div>
            <div className="text-3xl font-bold">99.9%</div>
            <div className="text-nimbus-200 text-sm">Uptime SLA</div>
          </div>
          <div>
            <div className="text-3xl font-bold">10K+</div>
            <div className="text-nimbus-200 text-sm">VMs Gerenciadas</div>
          </div>
          <div>
            <div className="text-3xl font-bold">24/7</div>
            <div className="text-nimbus-200 text-sm">Suporte</div>
          </div>
        </div>
      </div>

      {/* Right side - Login form */}
      <div className="flex-1 flex items-center justify-center p-8 bg-gray-50">
        <div className="w-full max-w-md">
          <div className="mb-8 flex items-center gap-3">
            <div className="w-10 h-10 bg-nimbus-500 rounded-xl flex items-center justify-center">
              <Cloud size={22} className="text-white" />
            </div>
            <span className="text-2xl font-bold text-gray-900">VirtForge</span>
          </div>

          <h2 className="text-2xl font-bold text-gray-900 mb-2">Bem-vindo</h2>
          <p className="text-gray-500 mb-8">Entre com suas credenciais para acessar</p>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Usuário</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-nimbus-500 focus:border-transparent transition"
                placeholder="admin"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Senha</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-nimbus-500 focus:border-transparent transition"
                placeholder="••••••••"
                required
              />
            </div>

            <div className="flex items-center justify-between">
              <label className="flex items-center gap-2">
                <input type="checkbox" className="w-4 h-4 text-nimbus-500 rounded" />
                <span className="text-sm text-gray-600">Lembrar</span>
              </label>
              <a href="#" className="text-sm text-nimbus-500 hover:text-nimbus-600">Esqueceu senha?</a>
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
              {loading ? 'Entrando...' : 'Entrar'}
            </button>
          </form>

          <p className="mt-8 text-center text-sm text-gray-500">
            VirtForge Cloud v1.0.0
          </p>
        </div>
      </div>
    </div>
  );
}
