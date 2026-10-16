import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

async function boot() {
  if (import.meta.env.VITE_DEMO) (await import('./demo/demo')).installDemo()
  ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
}

void boot()
