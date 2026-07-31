import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  optimizeDeps: {
    include: ['@novnc/novnc'],
    esbuildOptions: {
      target: 'esnext',
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    target: 'esnext',
  },
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
})
