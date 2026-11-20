
import { useEffect, useState } from 'react'
import {
  Wallet,
  Users,
  Percent,
  TrendingDown,
  TrendingUp,
  Trophy,
  ArrowUpRight,
  ArrowDownRight,
  RefreshCw,
  AlertCircle,
  CalendarDays,
  Target,
} from 'lucide-react'

import type {
  CompetitionFinancialOverview,
  MatchFinancialHighlight,
  ClubFinancialRanking,
} from '../types/financial'

interface Season {
  id: string
  name: string
}

interface FinancialOverviewProps {
  competitionId: string
  seasons: Season[]
}

const currencyFormatter = new Intl.NumberFormat('pt-BR', {
  style: 'currency',
  currency: 'BRL',
  maximumFractionDigits: 0,
})

const numberFormatter = new Intl.NumberFormat('pt-BR')

const percentageFormatter = new Intl.NumberFormat('pt-BR', {
  style: 'percent',
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
})

function formatCurrency(value: number): string {
  return currencyFormatter.format(value)
}

function formatNumber(value: number): string {
  return numberFormatter.format(value)
}

function formatPercentage(value: number): string {
  return percentageFormatter.format(value)
}

function MatchHighlights({
  highest,
  lowest,
}: {
  highest: MatchFinancialHighlight | null
  lowest: MatchFinancialHighlight | null
}) {
  const items = [
    { title: 'Maior público', match: highest, type: 'highest' as const },
    { title: 'Menor público', match: lowest, type: 'lowest' as const },
  ]

  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 sm:p-6">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 sm:divide-x sm:divide-gray-100">
        {items.map(({ title, match, type }) => {
          const Icon = type === 'highest' ? ArrowUpRight : ArrowDownRight

          return (
            <div key={title} className="min-w-0 sm:px-4 first:sm:pl-0 last:sm:pr-0">
              <div className="flex items-center gap-3">
                <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-xl ${
                  type === 'highest'
                    ? 'bg-green-50 text-green-600'
                    : 'bg-orange-50 text-orange-600'
                }`}>
                  <Icon size={18} />
                </div>
                <span className="text-sm font-semibold text-gray-600">{title}</span>
              </div>

              {match ? (
                <>
                  <div className="mt-4 flex items-center justify-center gap-3">
                    <p className="min-w-0 flex-1 text-right font-heading text-base font-bold text-sofi-midnight">
                      {match.home_team}
                    </p>

                    <span className="shrink-0 text-xs font-medium text-gray-400">
                      vs.
                    </span>

                    <p className="min-w-0 flex-1 text-left font-heading text-base font-bold text-sofi-midnight">
                      {match.away_team}
                    </p>
                    <div className="border-gray-100 pt-3 pb-3 bg-sofi-mist rounded-xl px-3 flex flex-col items-center justify-center">
                      <p className="text-xs text-gray-400">Público presente</p>
                      <p className="mt-1 font-heading text-2xl font-bold text-sofi-midnight">
                        {formatNumber(match.attendance)}
                      </p>
                    </div>
                  </div>
                </>
              ) : (
                <p className="mt-5 text-sm text-gray-400">Dados indisponíveis.</p>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function MetricCard({
  title,
  value,
  description,
  icon: Icon,
  accent = 'blue',
}: {
  title: string
  value: string
  description?: string
  icon: typeof Wallet
  accent?: 'blue' | 'green' | 'orange' | 'red'
}) {
  const accents = {
    blue: 'bg-sofi-mist text-sofi-blue',
    green: 'bg-green-50 text-green-600',
    orange: 'bg-orange-50 text-orange-600',
    red: 'bg-red-50 text-red-600',
  }

  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 transition-shadow hover:shadow-sm sm:p-6">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-medium text-gray-500">
          {title}
        </p>

        <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${accents[accent]}`}>
          <Icon size={19} />
        </div>
      </div>

      <p className="mt-5 break-words font-heading text-2xl font-bold tracking-tight text-sofi-midnight sm:text-3xl">
        {value}
      </p>

      {description && (
        <p className="mt-2 text-xs text-gray-400">
          {description}
        </p>
      )}
    </div>
  )
}

function ClubHighlights({
  attendanceClub,
  retentionClub,
}: {
  attendanceClub: ClubFinancialRanking | null
  retentionClub: ClubFinancialRanking | null
}) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 sm:p-6">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 sm:divide-x sm:divide-gray-100">
        <div className="min-w-0 sm:pr-5">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-sofi-mist text-sofi-blue">
              <Users size={19} />
            </div>
            <h4 className="text-sm font-semibold text-gray-600">
              Maior média de público
            </h4>
          </div>

          {attendanceClub ? (
            <>
              <p className="mt-5 font-heading text-xl font-bold text-sofi-midnight">
                {attendanceClub.club_name}
              </p>
              <p className="mt-2 font-heading text-3xl font-bold text-sofi-blue">
                {formatNumber(Math.round(attendanceClub.average_attendance))}
              </p>
              <p className="mt-1 text-xs text-gray-400">
                Média de público como mandante
              </p>
              <p className="mt-3 text-xs text-gray-500">
                {attendanceClub.home_matches} partidas como mandante
              </p>
            </>
          ) : (
            <p className="mt-5 text-sm text-gray-400">Dados indisponíveis.</p>
          )}
        </div>

        <div className="min-w-0 border-t border-gray-100 pt-5 sm:border-l sm:border-t-0 sm:pl-5 sm:pt-0">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-green-50 text-green-600">
              <Target size={19} />
            </div>
            <h4 className="text-sm font-semibold text-gray-600">
              Maior taxa de retenção
            </h4>
          </div>

          {retentionClub ? (
            <>
              <p className="mt-5 font-heading text-xl font-bold text-sofi-midnight">
                {retentionClub.club_name}
              </p>
              <p className="mt-2 font-heading text-3xl font-bold text-green-600">
                {formatPercentage(retentionClub.retention_margin)}
              </p>
              <p className="mt-1 text-xs text-gray-400">Taxa de retenção</p>
              <div className="mt-4 border-t border-gray-100 pt-3">
                <p className="text-xs text-gray-400">Valor total retido</p>
                <p className="mt-1 font-heading text-lg font-bold text-sofi-midnight">
                  {formatCurrency(retentionClub.methodological_net_result)}
                </p>
              </div>
            </>
          ) : (
            <p className="mt-5 text-sm text-gray-400">Dados indisponíveis.</p>
          )}
        </div>
      </div>
    </div>
  )
}

function FinancialSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[1, 2, 3, 4].map((item) => (
          <div key={item} className="h-36 rounded-2xl bg-gray-200" />
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {[1, 2, 3, 4].map((item) => (
          <div key={item} className="h-48 rounded-2xl bg-gray-200" />
        ))}
      </div>
    </div>
  )
}

export function FinancialOverview({
  competitionId,
  seasons,
}: FinancialOverviewProps) {
  const [selectedSeason, setSelectedSeason] = useState('')
  const [overview, setOverview] =
    useState<CompetitionFinancialOverview | null>(null)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    const controller = new AbortController()

    async function fetchOverview() {
      setLoading(true)
      setError(null)

      try {
        const params = new URLSearchParams()

        if (selectedSeason) {
          params.set('season', selectedSeason)
        }

        const query = params.toString()
        const url = `/api/competitions/${competitionId}/financial-overview${
          query ? `?${query}` : ''
        }`

        const response = await fetch(url, {
          signal: controller.signal,
        })

        if (!response.ok) {
          throw new Error(
            'Não foi possível carregar os dados financeiros.',
          )
        }

        const data: CompetitionFinancialOverview =
          await response.json()

        if (!controller.signal.aborted) {
          setOverview(data)
        }
      } catch (err) {
        if (controller.signal.aborted) return

        setError(
          err instanceof Error
            ? err.message
            : 'Erro inesperado ao carregar dados financeiros.',
        )
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false)
        }
      }
    }

    void fetchOverview()

    return () => controller.abort()
  }, [competitionId, selectedSeason, reloadKey])

  function retryLoading() {
    setReloadKey((current) => current + 1)
  }

  return (
    <section className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="font-heading text-xl font-bold text-sofi-midnight">
            Panorama financeiro
          </h2>

          <p className="mt-1 text-sm text-gray-500">
            Indicadores financeiros e de público da competição.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <CalendarDays
              size={15}
              className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
            />

            <select
              aria-label="Selecionar período financeiro"
              value={selectedSeason}
              onChange={(event) => setSelectedSeason(event.target.value)}
              className="rounded-xl border border-gray-200 bg-white py-2.5 pl-9 pr-4 text-sm font-medium text-sofi-midnight outline-none transition-colors focus:border-sofi-blue"
            >
              <option value="">Consolidado histórico</option>

              {seasons.map((season) => (
                <option key={season.id} value={season.id}>
                  Temporada {season.name}
                </option>
              ))}
            </select>
          </div>

          <button
            type="button"
            onClick={retryLoading}
            disabled={loading}
            aria-label="Atualizar panorama financeiro"
            className="flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200 bg-white text-gray-500 transition-colors hover:border-sofi-blue/30 hover:text-sofi-blue disabled:opacity-50"
          >
            <RefreshCw
              size={16}
              className={loading ? 'animate-spin' : ''}
            />
          </button>
        </div>
      </div>

      {/* Loading */}
      {loading && <FinancialSkeleton />}

      {/* Error */}
      {!loading && error && (
        <div className="rounded-2xl border border-gray-200 bg-white p-10 text-center">
          <AlertCircle
            size={32}
            className="mx-auto text-gray-300"
          />

          <p className="mt-3 text-sm text-gray-600">
            {error}
          </p>

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

      {/* Financial data */}
      {!loading && !error && overview && (
        <>
          {overview.matches_with_extraction === 0 && (
            <div className="rounded-xl border border-blue-100 bg-blue-50 px-4 py-3">
              <p className="text-sm text-blue-800">
                Ainda não existem dados financeiros extraídos para este período.
                Os indicadores serão atualizados conforme os borderôs forem processados.
              </p>
            </div>
          )}

          {/* Main metrics */}
          {/* Financial hero */}
          <div className="overflow-hidden rounded-3xl bg-white border border-gray-200 rounded-2xl shadow-sm">
            <div className="p-6 sm:p-8">
              <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
                {/* Gross revenue */}
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-white/10 text-blue-300">
                      <Wallet size={19} />
                    </div>

                    <span className="text-sm font-medium text-sofi-midnight">
                      Receita bruta movimentada
                    </span>
                  </div>

                  <div>
                    <p className="font-heading text-4xl font-extrabold tracking-tight sm:text-5xl">
                      {formatCurrency(overview.gross_revenue)}
                    </p>

                    <p className="mt-2 text-sm text-gray-400">
                      {formatNumber(overview.matches_with_extraction)} partidas com dados financeiros
                    </p>
                  </div>
                </div>

                {/* Retention and net result */}
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:min-w-[390px]">
                  <div className="rounded-2xl border border-white/10 bg-sofi-white p-5">
                    <div className="flex items-center gap-2 text-gray">
                      <Percent size={16} />
                      <span className="text-xs font-medium ">
                        Taxa de retenção
                      </span>
                    </div>

                    <p className="mt-3 font-heading text-3xl font-bold text-emerald-400">
                      {formatPercentage(overview.retention_margin)}
                    </p>

                    <p className="mt-1 text-xs text-gray-400">
                      Da receita bruta
                    </p>
                  </div>

                  <div className="rounded-2xl border border-white/10 bg-sofi-white p-5">
                    <div className="flex items-center gap-2 text-gray">
                      <TrendingUp size={16} />
                      <span className="text-xs font-medium">
                        Resultado líquido
                      </span>
                    </div>

                    <p className="mt-3 font-heading text-3xl font-bold text-sofi-midnight">
                      {formatCurrency(overview.methodological_net_result)}
                    </p>

                    <p className="mt-1 text-xs text-gray-400">
                      Resultado metodológico
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Secondary indicators */}
            <div className="grid grid-cols-2 divide-x divide-white/10 border-t border-white/10 sm:grid-cols-3">
              <div className="p-5 sm:px-8">
                <div className="flex items-center gap-2 text-gray-400">
                  <Users size={17} />
                  <span className="text-xs font-medium">Público total</span>
                </div>

                <p className="mt-2 font-heading text-2xl font-bold text-sofi-midnight">
                  {formatNumber(overview.total_attendance)}
                </p>

                <p className="mt-1 text-xs text-gray-500">
                  {formatNumber(overview.paying_attendance)} pagantes
                </p>
              </div>

              <div className="p-5 sm:px-8">
                <div className="flex items-center gap-2 text-gray-400">
                  <TrendingDown size={17} />
                  <span className="text-xs font-medium">Jogos deficitários</span>
                </div>

                <p className={`mt-2 font-heading text-2xl font-bold ${
                  overview.deficit_matches > 0
                    ? 'text-orange-400'
                    : 'text-emerald-400'
                }`}>
                  {formatNumber(overview.deficit_matches)}
                </p>

                <p className="mt-1 text-xs text-gray-500">
                  De {formatNumber(overview.matches_with_extraction)} jogos analisados
                </p>
              </div>

              <div className="col-span-2 border-t border-white/10 p-5 sm:col-span-1 sm:border-l sm:border-t-0 sm:px-8">
                <div className="flex items-center gap-2 text-gray-400">
                  <CalendarDays size={17} />
                  <span className="text-xs font-medium">Média de público</span>
                </div>

                <p className="mt-2 font-heading text-2xl font-bold text-sofi-midnight">
                  {formatNumber(Math.round(overview.average_attendance))}
                </p>

                <p className="mt-1 text-xs text-gray-500">
                  Por partida com dados extraídos
                </p>
              </div>
            </div>
          </div>

          {/* Data coverage */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 sm:p-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-sofi-midnight">
                  Cobertura dos dados
                </h3>
                <p className="mt-1 text-xs text-gray-400">
                  Borderôs com extração financeira concluída
                </p>
              </div>

              <div className="text-right">
                <p className="font-heading text-xl font-bold text-sofi-midnight">
                  {overview.total_matches > 0
                    ? formatPercentage(
                        overview.matches_with_extraction / overview.total_matches,
                      )
                    : '0%'}
                </p>
                <p className="text-xs text-gray-400">
                  {overview.matches_with_extraction} de {overview.total_matches} partidas
                </p>
              </div>
            </div>

            <div className="mt-4 h-2 overflow-hidden rounded-full bg-gray-100">
              <div
                className="h-full rounded-full bg-sofi-blue transition-all duration-500"
                style={{
                  width: `${
                    overview.total_matches > 0
                      ? Math.min(
                          100,
                          (overview.matches_with_extraction /
                            overview.total_matches) * 100,
                        )
                      : 0
                  }%`,
                }}
              />
            </div>

            {overview.matches_without_extraction > 0 && (
              <p className="mt-3 text-xs text-gray-400">
                {overview.matches_without_extraction} partidas ainda sem dados financeiros disponíveis.
              </p>
            )}
          </div>

          {/* Competition highlights */}
          <div className="space-y-4">
            <MatchHighlights
              highest={overview.highest_attendance_match}
              lowest={overview.lowest_attendance_match}
            />

            <ClubHighlights
              attendanceClub={overview.top_attendance_club}
              retentionClub={overview.top_retention_club}
            />
          </div>

          {/* Club ranking */}
          {overview.club_rankings.length > 0 && (
            <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white">
              <div className="border-b border-gray-100 p-5 sm:p-6">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sofi-mist text-sofi-blue">
                    <Trophy size={19} />
                  </div>

                  <div>
                    <h3 className="font-heading font-bold text-sofi-midnight">
                      Ranking financeiro dos clubes
                    </h3>
                    <p className="mt-1 text-xs text-gray-400">
                      Ordenado pela receita bruta acumulada como mandante.
                    </p>
                  </div>
                </div>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full min-w-[650px] text-left text-sm">
                  <thead className="bg-gray-50 text-xs uppercase tracking-wider text-gray-400">
                    <tr>
                      <th className="px-5 py-4 font-semibold">Clube</th>
                      <th className="px-5 py-4 text-right font-semibold">Renda bruta</th>
                      <th className="px-5 py-4 text-right font-semibold">Valor retido</th>
                      <th className="px-5 py-4 text-right font-semibold">Retenção</th>
                      <th className="px-5 py-4 text-right font-semibold">Média de público</th>
                    </tr>
                  </thead>

                  <tbody className="divide-y divide-gray-100">
                    {overview.club_rankings.map((club, index) => (
                      <tr
                        key={club.club_id}
                        className="transition-colors hover:bg-gray-50"
                      >
                        <td className="px-5 py-4">
                          <div className="flex items-center gap-3">
                            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-gray-100 text-xs font-bold text-gray-500">
                              {index + 1}
                            </span>
                            <span className="font-semibold text-sofi-midnight">
                              {club.club_name}
                            </span>
                          </div>
                        </td>

                        <td className="px-5 py-4 text-right font-medium text-gray-700">
                          {formatCurrency(club.gross_revenue)}
                        </td>

                        <td className="px-5 py-4 text-right font-medium text-gray-700">
                          {formatCurrency(club.methodological_net_result)}
                        </td>

                        <td className="px-5 py-4 text-right">
                          <span className="font-semibold text-green-600">
                            {formatPercentage(club.retention_margin)}
                          </span>
                        </td>

                        <td className="px-5 py-4 text-right font-medium text-gray-700">
                          {formatNumber(Math.round(club.average_attendance))}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </section>
  )
}
