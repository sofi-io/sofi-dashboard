
import { Outlet } from 'react-router-dom'
import { Header } from './Header'

export function AppLayout() {
  return (
    <div className="min-h-screen bg-sofi-white">
      <Header />

      <main className="mx-auto w-full max-w-7xl px-6 py-10">
        <Outlet />
      </main>

      <footer className="mx-auto mt-16 max-w-7xl border-t border-gray-200 px-6 py-6">
        <p className="text-xs text-gray-500">
          SOFI — Observatório Financeiro do Futebol
        </p>
      </footer>
    </div>
  )
}
