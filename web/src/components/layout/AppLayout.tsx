
import { Outlet } from 'react-router-dom'
import { Header } from './Header'

export function AppLayout() {
  return (
    <div className="flex min-h-screen flex-col bg-sofi-white">

      <Header />

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 pb-16 pt-8 sm:px-6 sm:pt-10 lg:px-8">
        <Outlet />
      </main>

      <footer className="mt-auto w-full">
        <div className="mx-auto max-w-7xl border-t border-gray-200/80 px-4 py-7 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-xs font-medium text-gray-500">
              SOFI — Observatório Financeiro do Futebol
            </p>

            <p className="text-xs text-gray-400">
              Transparência, dados e inteligência para o futebol.
            </p>
          </div>
        </div>
      </footer>

    </div>
  )
}
