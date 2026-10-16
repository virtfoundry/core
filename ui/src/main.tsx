import React from 'react'
import ReactDOM from 'react-dom/client'
import './index.css'

async function boot() {
  if (import.meta.env.VITE_DEMO) (await import('./demo/demo')).installDemo()
  // App (and the redux store) must load after installDemo so jwt_token is set before auth state initialises.
  const { default: App } = await import('./App')
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>,
  )
}

void boot()
