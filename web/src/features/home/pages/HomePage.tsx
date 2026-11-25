import { useEffect, useState } from 'react'

import type {
  GeneralStats,
  CompetitionStats
} from '../../../types/financial'

function StatsCard({ stats }: { stats: GeneralStats | null }) {
  if (!stats) {
    return null
  }

  const retention = stats.total_retention ?? 0
  const progress = Math.max(0, Math.min(100, retention))

  const isNegative = retention < 0

  const grossRevenueFormatted = (
    stats.total_gross_revenue / 1000
    ).toLocaleString('pt-BR', {
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    })

console.log('Valor formatado:', grossRevenueFormatted)

  return (
    <div className="w-full rounded-2xl bg-sofi-midnight font-sans">
      {/* Título */}
      <h2 className="font-heading text-2xl font-semibold leading-snug text-sofi-white lg:text-3xl">
        Quanto da bilheteria
        <br />
        fica no futebol?
      </h2>

      {/* Margem de retenção */}
      <div className="mt-9 flex flex-wrap items-baseline gap-x-4 gap-y-2">
        <span
          className={`font-heading text-6xl font-extrabold leading-none tracking-tight lg:text-7xl ${
            isNegative ? 'text-sofi-orange' : 'text-sofi-mist'
          }`}
        >
          {retention.toLocaleString('pt-BR', {
            minimumFractionDigits: 1,
            maximumFractionDigits: 1,
          })}%
        </span>

        <span className="text-sm text-sofi-mist/70">
          de retenção
        </span>
      </div>

      {/* Barra de retenção */}
      <div
        className="mt-9 h-2.5 w-full overflow-hidden rounded-full bg-sofi-mist/15"
        role="progressbar"
        aria-label="Margem de retenção"
        aria-valuenow={progress}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className="h-full rounded-l-full bg-sofi-blue transition-all duration-500"
          style={{ width: `${progress}%` }}
        />
      </div>

      {/* Indicadores financeiros */}
      <div className="mt-7 grid grid-cols-2 gap-4">

        {/* Receita bruta */}
        <div>
          <p className="font-heading text-xl font-bold text-sofi-white sm:text-2xl">
            R$ {(stats.total_gross_revenue / 1000).toLocaleString('pt-BR', {
              maximumFractionDigits: 0,
            })} mi
          </p>

          <p className="mt-1.5 text-xs text-sofi-mist/70">
            Receita bruta
          </p>
        </div>

        {/* Resultado líquido */}
        <div>
          <p
            className={`font-heading text-xl font-bold sm:text-2xl ${
              (stats.total_revenue ?? 0) < 0
                ? 'text-sofi-orange'
                : 'text-sofi-white'
            }`}
          >
            R$ {(stats.total_revenue / 1000).toLocaleString('pt-BR', {
              maximumFractionDigits: 0,
            })} mi
          </p>

          <p className="mt-1.5 text-xs text-sofi-mist/70">
            Resultado líquido
          </p>
        </div>

      </div>

      {/* Rodapé */}
      <p className="mt-7 text-[11px] leading-relaxed text-sofi-mist/60">
        {stats.total_competitions} competições analisadas
        {' · '}
        {stats.total_matches} partidas
        {' · '}
        {stats.total_teams} clubes
        {' · '}
        {stats.total_seasons} temporadas
      </p>
    </div>
  )
}

function StatsSection({ stats }: { stats: GeneralStats | null }) {
  if (!stats) {
    return null
  }

  return (
    <section className="w-full bg-sofi-white py-12 lg:py-20">
      <div className="mx-auto max-w-7xl px-6 lg:px-12">
        <div className="grid items-start gap-12 lg:grid-cols-[1.6fr_1fr] lg:gap-16">
          <div className="flex flex-col items-start">
            <span className="text-xs font-bold uppercase tracking-[0.2em] text-sofi-blue">
              Dados abertos. Futebol às claras.
            </span>
            <h1 className="mt-6 max-w-3xl font-heading text-4xl font-extrabold leading-[1.1] tracking-tight text-sofi-midnight sm:text-5xl lg:text-6xl">
              O jogo também
              <br className="hidden lg:block" />
              acontece nas contas.
            </h1>
            <p className="mt-6 max-w-xl text-base leading-relaxed text-gray-500 lg:text-lg">
              Entenda o dinheiro que movimenta os estaduais.
              <br className="hidden sm:block" />
              Receitas, custos e público, do campeonato ao seu clube.
            </p>
            <div className="mt-7 flex flex-wrap items-center gap-3">
              <a
                href="/competitions"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-sofi-blue px-5 py-3.5 text-sm font-semibold text-white transition-colors hover:bg-sofi-blue/90"
              >
                Explorar competições
                <span aria-hidden="true">↗</span>
              </a>
              <a
                href="#methodology"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-5 py-3.5 text-sm font-semibold text-sofi-midnight transition-colors hover:bg-sofi-mist"
              >
                Como ler os dados
                <span aria-hidden="true">↓</span>
              </a>
            </div>

            <div className="mt-7 flex flex-wrap items-center gap-x-7 gap-y-3 text-xs">
              <span className="font-semibold text-sofi-midnight">
                {stats?.total_competitions || 0} competições
              </span>
              <span className="font-semibold text-sofi-midnight">
                {stats?.total_seasons || 0} temporadas
              </span>
            </div>
          </div>

          <div className="w-full rounded-2xl bg-sofi-midnight p-7 sm:p-8 lg:p-9">
            <StatsCard stats={stats} />
          </div>
        </div>
      </div>
    </section>
  )
}

function DetailsCard({ title, value, description }: { title: string, value: string | number, description: string }) {
  return (
    <div className="flex h-full flex-col rounded-2xl border border-gray-200 bg-white p-5 lg:p-6">
      <span className="text-sm font-medium text-gray-500">
        {title}
      </span>

      <span className="mt-3 font-heading text-3xl font-extrabold tracking-tight text-sofi-midnight lg:text-4xl">
        {value}
      </span>

      <span className="mt-2 text-xs leading-relaxed text-gray-500">
        {description}
      </span>
    </div>
  )
}

function DetailsSection({ stats }: { stats: GeneralStats | null }) {
  if (!stats) {
    return null
  }

  return (
    <section className="w-full bg-sofi-white pb-12">
      <div className="mx-auto max-w-7xl px-6 lg:px-12">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">

          <DetailsCard
            title="Receita bruta"
            value={formatNumber(stats.total_gross_revenue)}
            description="Bilheteria antes das deduções"
          />

          <DetailsCard
            title="Resultado líquido"
            value={formatNumber(stats.total_revenue)}
            description="Receita após custos, taxas e impostos"
          />

          <DetailsCard
            title="Público total"
            value={formatNumber(stats.total_attendance)}
            description="Público pagante nos estádios"
          />

          <DetailsCard
            title="Jogos deficitários"
            value={formatNumber(stats.total_deficit_matches)}
            description="Partidas com receita negativa"
          />
        </div>
      </div>
    </section>
  )
}

function CompetitionCard({
  competition,
}: {
  competition: CompetitionStats
}) {
  const formatMillions = (value: number) =>
    `R$ ${(value / 1_000_000).toLocaleString('pt-BR', {
      minimumFractionDigits: 1,
      maximumFractionDigits: 1,
    })} mi`

  return (
    <div className="mt-6 flex h-full flex-col rounded-xl border border-gray-200 bg-white p-6 transition-colors hover:border-sofi-blue/40 lg:p-7">

      {/* Nome da competição */}
      <h3 className="font-heading text-2xl font-bold leading-snug tracking-tight text-sofi-midnight">
        {competition.name}
      </h3>

      {/* Informações gerais */}
      <p className="mt-5 text-sm text-gray-500">
        {competition.total_teams} clubes
        {' · '}
        {competition.total_matches} jogos
        {' · '}
        {competition.seasons.join(', ')}
      </p>

      {/* Indicadores financeiros */}
      <div className="mt-6 grid grid-cols-2 gap-4">

        <div>
          <p className="text-sm text-gray-500">
            Receita bruta
          </p>

          <p className="mt-2 font-heading text-2xl font-bold text-sofi-midnight">
            {formatMillions(competition.gross_revenue)}
          </p>
        </div>

        <div className="text-right">
          <p className="text-sm text-gray-500">
            Resultado líquido
          </p>

          <p
            className={`mt-2 font-heading text-2xl font-bold ${
              competition.total_revenue < 0
                ? 'text-sofi-orange'
                : 'text-sofi-blue'
            }`}
          >
            {formatMillions(competition.total_revenue)}
          </p>
        </div>

      </div>

      {/* Rodapé */}
      <div className="mt-auto flex items-center justify-between gap-3 pt-7">

        <span className="text-sm text-gray-500">
          {competition.total_attendance.toLocaleString('pt-BR', {
            notation: 'compact',
            maximumFractionDigits: 1,
          })} presentes
        </span>

        <a
          href={`/competitions/${competition.id}`}
          className="inline-flex shrink-0 items-center gap-2 text-sm font-semibold text-sofi-blue transition-colors hover:text-sofi-midnight"
        >
          Explorar

          <svg
            className="h-4 w-4"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M7 17 17 7M7 7h10v10" />
          </svg>
        </a>

      </div>

    </div>
  )
}

function CompetitionsSection({ competitions }: { competitions: CompetitionStats[] }) {
  if (!competitions || competitions.length === 0) {
    return null
  }

  return (
    <section className="w-full bg-sofi-white pb-12">
      <div className="mx-auto max-w-7xl px-6 lg:px-12">
        <h2 className="text-lg font-bold">Um país, muitos campeonatos</h2>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {/* add key */}
          {competitions.map((competition) => (
            <CompetitionCard
              key={competition.id}
              competition={competition}
            />
          ))} 
        </div>
      </div>
    </section>
  )
}

function formatNumber(value: number): string {
  return value.toLocaleString('pt-BR', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  })
}

async function fetchGeneralStats(season: string): Promise<GeneralStats> {
    const response = await fetch(`/api/financial/overview?season=${season}`)
    if (!response.ok) {
        throw new Error('Erro ao buscar estatísticas gerais.')
    }
    const data: GeneralStats = await response.json()

    return data
}

function SeasonSelector({
  seasons,
  selectedSeason,
  onChange,
}: {
  seasons: string[]
  selectedSeason: string
  onChange: (season: string) => void
}) {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <div
      className="relative inline-block self-start text-right"
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget)) {
          setIsOpen(false)
        }
      }}
    >
      <button
        type="button"
        aria-expanded={isOpen}
        onClick={() => setIsOpen(!isOpen)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') setIsOpen(false)
        }}
        className="flex cursor-pointer items-center gap-2 border-none bg-transparent py-2 font-sans text-base font-medium text-gray-500 outline-none transition-colors hover:text-sofi-midnight focus-visible:text-sofi-blue"
      >
        Temporada {selectedSeason}

        <svg
          className={`h-3 w-3 transition-transform duration-200 ${
            isOpen ? 'rotate-180' : ''
          }`}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      {isOpen && (
        <div
          className="absolute left-0 top-full z-20 mt-1 min-w-full rounded-lg bg-sofi-white py-1"
          onKeyDown={(e) => {
            if (e.key === 'Escape') setIsOpen(false)
          }}
        >
          {seasons.map((season) => (
            <button
              key={season}
              type="button"
              onClick={() => {
                onChange(season)
                setIsOpen(false)
              }}
              className={`block w-full cursor-pointer rounded-md px-3 py-2 text-left text-sm outline-none transition-colors hover:bg-sofi-mist focus-visible:bg-sofi-mist ${
                selectedSeason === season
                  ? 'font-semibold text-sofi-blue'
                  : 'text-gray-500'
              }`}
            >
              Temporada {season}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

export function HomePage() {
  const [stats, setStats] = useState<GeneralStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedSeason, setSelectedSeason] = useState<string>('2026')

  useEffect(() => {
      fetchGeneralStats(selectedSeason)
          .then((data) => {
              setStats(data)
              setLoading(false)
          })
          .catch((err) => {
              setError(err.message)
              setLoading(false)
          })
  }, [selectedSeason])

  if (loading) {
      return <p>Carregando...</p>
  }
  if (error) {
    console.error(error)
  }

  if (!stats) {
    console.error('Stats data is null or undefined')  
  }

  return (
    <main className="bg-sofi-white">
      <div className="flex w-full justify-end">
        <SeasonSelector
          seasons={['2026', '2025']}
          selectedSeason={selectedSeason}
          onChange={setSelectedSeason}
        />
      </div>
      <StatsSection stats={stats} />

      <DetailsSection stats={stats} />

      <CompetitionsSection competitions={stats?.competitions || []} />
    </main>
  )
}