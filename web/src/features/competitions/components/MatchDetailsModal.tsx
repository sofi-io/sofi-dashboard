
import { useEffect } from 'react'
import {
  CalendarDays,
  Clock3,
  MapPin,
  Users,
  Wallet,
  Receipt,
  TrendingUp,
} from 'lucide-react'

import type { Match } from './MatchesList'

interface MatchDetailsModalProps {
  match: Match
  onClose: () => void
}

function formatMatchDate(match: Match) {
  if (match.kickoff_at) {
    const date = new Date(match.kickoff_at)

    if (!Number.isNaN(date.getTime())) {
      return new Intl.DateTimeFormat('pt-BR', {
        day: '2-digit',
        month: 'long',
        year: 'numeric',
        timeZone: 'America/Recife',
      }).format(date)
    }
  }

  return match.source_date_text || 'Data não informada'
}

function formatMatchTime(match: Match) {
  if (match.kickoff_at) {
    const date = new Date(match.kickoff_at)

    if (!Number.isNaN(date.getTime())) {
      return new Intl.DateTimeFormat('pt-BR', {
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'America/Recife',
      }).format(date)
    }
  }

  return match.source_time_text || 'Horário não informado'
}

interface FinancialIndicatorProps {
  label: string
  value: string
  icon: React.ElementType
  description?: string
}

function FinancialIndicator({
  label,
  value,
  icon: Icon,
  description,
}: FinancialIndicatorProps) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-gray-500">
          {label}
        </p>

        <Icon size={18} className="text-sofi-blue" />
      </div>

      <p className="mt-4 font-heading text-2xl font-bold text-sofi-midnight">
        {value}
      </p>

      {description && (
        <p className="mt-1 text-xs text-gray-400">
          {description}
        </p>
      )}
    </div>
  )
}

export function MatchDetailsModal({
  match,
  onClose,
}: MatchDetailsModalProps) {
  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('keydown', handleKeyDown)

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [onClose])

  const round = match.round_name ||
    (match.round_number
      ? `Rodada ${match.round_number}`
      : 'Rodada não informada')

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-3 sm:p-6">
      <button
        type="button"
        aria-label="Fechar detalhes da partida"
        onClick={onClose}
        className="absolute inset-0 cursor-default bg-sofi-midnight/50 backdrop-blur-sm"
      />

      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby="match-modal-title"
        className="relative z-10 max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-3xl bg-sofi-white shadow-2xl"
      >
        {/* Compact match header */}
        <header className="border-b border-gray-200 bg-white p-6 sm:p-8">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-widest text-sofi-blue">
                {match.season} · {round}
              </p>

              <p className="mt-1 text-xs text-gray-400">
                Detalhes financeiros da partida
              </p>
            </div>
          </div>

          <div className="mt-6 grid grid-cols-[1fr_auto_1fr] items-center gap-3">
            <div className="min-w-0">
              <h2
                id="match-modal-title"
                className="break-words font-heading text-lg font-bold text-sofi-midnight sm:text-xl"
              >
                {match.home_club.name}
              </h2>
              <p className="mt-1 text-xs text-gray-400">
                Mandante
              </p>
            </div>

            <span className="font-heading text-sm font-bold text-gray-300">
              VS
            </span>

            <div className="min-w-0 text-right">
              <h3 className="break-words font-heading text-lg font-bold text-sofi-midnight sm:text-xl">
                {match.away_club.name}
              </h3>
              <p className="mt-1 text-xs text-gray-400">
                Visitante
              </p>
            </div>
          </div>

          <div className="mt-5 flex flex-wrap gap-x-5 gap-y-2 border-t border-gray-100 pt-4 text-xs text-gray-500">
            <span className="inline-flex items-center gap-1.5">
              <CalendarDays size={14} />
              {formatMatchDate(match)}
            </span>

            <span className="inline-flex items-center gap-1.5">
              <Clock3 size={14} />
              {formatMatchTime(match)}
            </span>

            {match.venue && (
              <span className="inline-flex items-center gap-1.5">
                <MapPin size={14} />
                {match.venue}
              </span>
            )}
          </div>
        </header>

        {/* Financial indicators */}
        <main className="space-y-6 p-6 sm:p-8">
          <div>
            <p className="text-xs font-semibold uppercase tracking-widest text-sofi-blue">
              Análise financeira
            </p>

            <h3 className="mt-2 font-heading text-xl font-bold text-sofi-midnight">
              Indicadores financeiros
            </h3>

            <p className="mt-1 text-sm text-gray-500">
              Demonstrativo financeiro da partida.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FinancialIndicator
              label="Público total"
              value="—"
              icon={Users}
            />

            <FinancialIndicator
              label="Renda bruta"
              value="—"
              icon={Wallet}
            />

            <FinancialIndicator
              label="Despesas operacionais"
              value="—"
              icon={Receipt}
            />

            <FinancialIndicator
              label="Resultado líquido"
              value="—"
              icon={TrendingUp}
            />
          </div>

          {/* Additional financial breakdown */}
          <section className="rounded-2xl border border-dashed border-gray-300 bg-white p-5">
            <h4 className="font-heading text-sm font-bold text-sofi-midnight">
              Detalhamento financeiro
            </h4>

            <div className="mt-4 space-y-3">
              {[
                'Público pagante',
                'Público não pagante',
                'Tributos e taxas',
                'Outras deduções',
              ].map((item) => (
                <div
                  key={item}
                  className="flex items-center justify-between gap-4 border-b border-gray-100 pb-3 last:border-0 last:pb-0"
                >
                  <span className="text-sm text-gray-500">
                    {item}
                  </span>

                  <span className="text-sm font-semibold text-gray-400">
                    —
                  </span>
                </div>
              ))}
            </div>
          </section>
        </main>
      </section>
    </div>
  )
}
