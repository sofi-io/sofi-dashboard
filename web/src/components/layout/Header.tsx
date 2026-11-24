
import { NavLink } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'

export function Header() {
  return (
    <header className="sticky top-0 z-50 px-4 pt-4">
      <div className="mx-auto flex h-[72px] max-w-7xl items-center justify-between rounded-full border border-gray-200/70 bg-white/90 px-5 shadow-[0_4px_24px_rgba(17,24,39,0.05)] backdrop-blur-xl sm:px-8">

        {/* Brand */}
        <NavLink
          to="/"
          aria-label="SOFI - Página inicial"
          className="shrink-0 font-heading text-[30px] font-extrabold leading-none tracking-[-0.07em] text-sofi-midnight"
        >
          sofi
        </NavLink>

        {/* Navigation */}
        <nav className="flex items-center gap-1 sm:gap-2">

          <NavLink
            to="/competitions"
            end
            className={({ isActive }) =>
              `rounded-full px-3 py-2 text-xs font-semibold transition-colors sm:px-5 sm:text-sm ${
                isActive
                  ? 'bg-sofi-mist text-sofi-blue'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-sofi-midnight'
              }`
            }
          >
            Competições
          </NavLink>

          {/* Intelligence dropdown */}
          <details className="group relative">
            <summary className="flex cursor-pointer list-none items-center gap-1.5 rounded-full px-3 py-2 text-xs font-semibold text-gray-600 transition-colors hover:bg-gray-100 hover:text-sofi-midnight sm:px-5 sm:text-sm [&::-webkit-details-marker]:hidden">
              Inteligência
              <ChevronDown
                size={15}
                className="transition-transform duration-200 group-open:rotate-180"
              />
            </summary>

            <div className="absolute right-0 top-full z-50 mt-3 w-56 rounded-2xl border border-gray-200/80 bg-white p-2 shadow-xl shadow-gray-900/10">
              <NavLink
                to="/inteligencia"
                onClick={(event) => {
                  const details = event.currentTarget.closest('details')
                  details?.removeAttribute('open')
                }}
                className={({ isActive }) =>
                  `block rounded-xl px-4 py-3 text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-sofi-mist text-sofi-blue'
                      : 'text-gray-700 hover:bg-gray-50 hover:text-sofi-midnight'
                  }`
                }
              >
                <span className="block">Visão geral</span>
                <span className="mt-1 block text-xs font-normal text-gray-400">
                  Indicadores e análises
                </span>
              </NavLink>
            </div>
          </details>

        </nav>

        {/* Right spacing */}
        <div className="hidden w-12 sm:block" />

      </div>
    </header>
  )
}
