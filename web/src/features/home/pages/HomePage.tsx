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
            R$ {(stats.total_gross_revenue ?? 0).toLocaleString('pt-BR', {
              minimumFractionDigits: 1,
              maximumFractionDigits: 1,
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
            R$ {(stats.total_revenue ?? 0).toLocaleString('pt-BR', {
              minimumFractionDigits: 1,
              maximumFractionDigits: 1,
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



function formatNumber(value: number): string {
  return value.toLocaleString('pt-BR', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  })
}

function CompetitionStatsCard({ competitions }: { competitions: CompetitionStats[] }) {
  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-2 rounded-lg border border-gray-200/70 bg-white/90 p-4 shadow-[0_4px_24px_rgba(17,24,39,0.05)] backdrop-blur-xl">
        <h2 className="text-lg font-semibold">Os estaduais, em números</h2>
      </div>
    
      {/*Cards*/}
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {competitions.slice(0, 3).map((competition) => (
          <div key={competition.id} className="flex flex-col gap-1 rounded-lg bg-sofi-mist p-3">
            <span className="text-sm text-gray-600">{competition.name}</span>
            <span className="text-xs text-gray-500">{competition.total_teams} clubes</span>
            <span className="text-xl font-bold">{competition.total_matches} partidas</span>
            <span className="text-xs text-gray-500">R$ {competition.gross_revenue.toLocaleString()}</span>
          </div>
        ))}
      </div>
    </section>
  ) 
}

async function fetchGeneralStats(): Promise<GeneralStats> {
    const response = await fetch('/api/financial/overview')
    if (!response.ok) {
        throw new Error('Erro ao buscar estatísticas gerais.')
    }
    const data: GeneralStats = await response.json()

    return data
}

export function HomePage() {
  const [stats, setStats] = useState<GeneralStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
      fetchGeneralStats()
          .then((data) => {
              setStats(data)
              setLoading(false)
          })
          .catch((err) => {
              setError(err.message)
              setLoading(false)
          })
  }, [])

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
      <StatsSection stats={stats} />

      <DetailsSection stats={stats} />
    </main>
  )
}