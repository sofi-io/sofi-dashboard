
import { useEffect, useState } from 'react'
import {
  CalendarDays,
  Clock3,
  MapPin,
  FileCheck2,
  FileX2,
  RefreshCw,
  AlertCircle,
} from 'lucide-react'

import { MatchDetailsModal } from './MatchDetailsModal'

interface Season {
  id: string
  name: string
  starts_on: string | null
  ends_on: string | null
}

interface MatchDocument {
  id: string
  type: string
  status: string
}

export interface Match {
  id: string
  season_id: string
  season: string
  round_name: string
  round_number: number | null
  kickoff_at: string | null
  source_date_text: string
  source_time_text: string
  venue: string
  status: string
  home_club: {
    id: string
    name: string
  }
  away_club: {
    id: string
    name: string
  }
  documents: MatchDocument[]
}

interface MatchesResponse {
  items: Match[]
  total: number
  limit: number
  offset: number
}

interface MatchesListProps {
  competitionId: string
  seasons: Season[]
}

function formatDate(match: Match): string {
  if (match.kickoff_at) {
    const date = new Date(match.kickoff_at)

    if (!Number.isNaN(date.getTime())) {
      return new Intl.DateTimeFormat('pt-BR', {
        day: '2-digit',
        month: 'short',
        year: 'numeric',
        timeZone: 'America/Recife',
      }).format(date)
    }
  }

  return match.source_date_text || 'Data a definir'
}

function formatTime(match: Match): string {
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

  return match.source_time_text || 'Horário a definir'
}

function getRoundName(match: Match): string {
  if (match.round_name) return match.round_name

  if (match.round_number !== null) {
    return `Rodada ${match.round_number}`
  }

  return 'Rodada não informada'
}

function hasFinancialBulletin(match: Match): boolean {
  return match.documents.some(
    (document) =>
      document.type === 'financial_bulletin' &&
      ['downloaded', 'processed', 'submitted'].includes(
        document.status,
      ),
  )
}

function MatchCard({ match, onClick, }: {
  match: Match
  onClick: () => void
}) {
  const hasBulletin = hasFinancialBulletin(match)

  return (
    <button
      type="button"
      onClick={onClick}
      className="group w-full rounded-2xl border border-gray-200 bg-white p-5 text-left transition-all hover:border-sofi-blue/30 hover:shadow-md sm:p-6"
      >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3 text-xs text-gray-500">
          <span className="inline-flex items-center gap-1.5">
            <CalendarDays size={14} />
            {formatDate(match)}
          </span>

          <span className="inline-flex items-center gap-1.5">
            <Clock3 size={14} />
            {formatTime(match)}
          </span>
        </div>

        <span className="rounded-full bg-sofi-mist px-3 py-1 text-xs font-semibold text-sofi-blue">
          {match.status}
        </span>
      </div>

      <div className="my-6 grid grid-cols-[1fr_auto_1fr] items-center gap-3 sm:gap-6">
        <div className="min-w-0 text-left">
          <p className="break-words font-heading text-base font-bold text-sofi-midnight sm:text-lg">
            {match.home_club.name}
          </p>
          <p className="mt-1 text-xs text-gray-400">Mandante</p>
        </div>

        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-100">
          <span className="text-xs font-bold text-gray-400">VS</span>
        </div>

        <div className="min-w-0 text-right">
          <p className="break-words font-heading text-base font-bold text-sofi-midnight sm:text-lg">
            {match.away_club.name}
          </p>
          <p className="mt-1 text-xs text-gray-400">Visitante</p>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-4">
        <div className="flex min-w-0 items-center gap-1.5 text-xs text-gray-500">
          {match.venue ? (
            <>
              <MapPin size={14} className="shrink-0" />
              <span className="truncate">{match.venue}</span>
            </>
          ) : (
            <span>Estádio não informado</span>
          )}
        </div>

        <div
          className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-semibold ${
            hasBulletin
              ? 'bg-green-50 text-green-700'
              : 'bg-gray-100 text-gray-500'
          }`}
        >
          {hasBulletin ? (
            <FileCheck2 size={14} />
          ) : (
            <FileX2 size={14} />
          )}

          {hasBulletin ? 'Borderô disponível' : 'Sem borderô'}
        </div>
      </div>
    </button>
  )
}

export function MatchesList({
  competitionId,
  seasons,
}: MatchesListProps) {
  const [selectedSeason, setSelectedSeason] = useState(
    seasons[0]?.id ?? '',
  )

  const [matches, setMatches] = useState<Match[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)
  const [selectedMatch, setSelectedMatch] = useState<Match | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    async function fetchMatches() {
      try {
        const params = new URLSearchParams({
          limit: '100',
        })

        if (selectedSeason) {
          params.set('season', selectedSeason)
        }

        const response = await fetch(
          `/api/competitions/${competitionId}/matches?${params}`,
          { signal: controller.signal },
        )

        if (!response.ok) {
          throw new Error('Não foi possível carregar as partidas.')
        }

        const data: MatchesResponse = await response.json()

        if (!controller.signal.aborted) {
          setMatches(data.items)
          setTotal(data.total)
          setError(null)
        }
      } catch (err) {
        if (controller.signal.aborted) return

        setError(
          err instanceof Error
            ? err.message
            : 'Erro inesperado ao carregar partidas.',
        )
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false)
        }
      }
    }

    void fetchMatches()

    return () => controller.abort()
  }, [competitionId, selectedSeason, reloadKey])

  const groupedMatches = matches.reduce<Record<string, Match[]>>(
    (groups, match) => {
      const round = getRoundName(match)

      if (!groups[round]) {
        groups[round] = []
      }

      groups[round].push(match)

      return groups
    },
    {},
  )

  function retryLoading() {
    setLoading(true)
    setError(null)
    setReloadKey((current) => current + 1)
  }

  return (
    <section className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="font-heading text-xl font-bold text-sofi-midnight">
            Partidas
          </h2>

          <p className="mt-1 text-sm text-gray-500">
            Consulte os jogos e a disponibilidade dos borderôs.
          </p>
        </div>

        {seasons.length > 0 && (
          <select
            aria-label="Selecionar temporada"
            value={selectedSeason}
            onChange={(event) => {
              setLoading(true)
              setError(null)
              setSelectedSeason(event.target.value)
            }}
            className="rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-sofi-midnight outline-none transition-colors focus:border-sofi-blue"
          >
            {seasons.map((season) => (
              <option key={season.id} value={season.id}>
                Temporada {season.name}
              </option>
            ))}
          </select>
        )}
      </div>

      {!loading && !error && (
        <div className="flex items-center justify-between border-b border-gray-200 pb-4">
          <span className="text-sm text-gray-500">
            Partidas encontradas
          </span>

          <span className="font-heading text-2xl font-bold text-sofi-midnight">
            {total}
          </span>
        </div>
      )}

      {loading && (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {[1, 2, 3, 4].map((item) => (
            <div
              key={item}
              className="h-52 animate-pulse rounded-2xl bg-gray-100"
            />
          ))}
        </div>
      )}

      {!loading && error && (
        <div className="rounded-2xl border border-gray-200 bg-white p-10 text-center">
          <AlertCircle
            size={32}
            className="mx-auto text-gray-300"
          />

          <p className="mt-3 text-sm text-gray-600">{error}</p>

          <button
            type="button"
            onClick={retryLoading}
            className="mt-5 inline-flex items-center gap-2 rounded-full bg-sofi-blue px-5 py-2.5 text-sm font-semibold text-white hover:bg-blue-700"
          >
            <RefreshCw size={14} />
            Tentar novamente
          </button>
        </div>
      )}

      {!loading && !error && matches.length === 0 && (
        <div className="rounded-2xl border border-dashed border-gray-300 bg-white/60 p-12 text-center">
          <CalendarDays
            size={32}
            className="mx-auto text-gray-300"
          />

          <h3 className="mt-4 font-heading font-semibold text-sofi-midnight">
            Nenhuma partida encontrada
          </h3>

          <p className="mt-2 text-sm text-gray-500">
            Não existem partidas cadastradas para esta temporada.
          </p>
        </div>
      )}

      {!loading && !error && matches.length > 0 && (
        <div className="space-y-8">
          {Object.entries(groupedMatches).map(([round, roundMatches]) => (
            <div key={round} className="space-y-4">
              <div className="flex items-center gap-3">
                <h3 className="font-heading text-sm font-bold uppercase tracking-wider text-sofi-midnight">
                  {round}
                </h3>

                <div className="h-px flex-1 bg-gray-200" />

                <span className="text-xs text-gray-400">
                  {roundMatches.length}{' '}
                  {roundMatches.length === 1 ? 'jogo' : 'jogos'}
                </span>
              </div>

              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                {roundMatches.map((match) => (
                  <MatchCard
                    key={match.id}
                    match={match}
                    onClick={() => setSelectedMatch(match)}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
      {selectedMatch && (
        <MatchDetailsModal
          match={selectedMatch}
          onClose={() => setSelectedMatch(null)}
        />
      )}
    </section>
  )
}
