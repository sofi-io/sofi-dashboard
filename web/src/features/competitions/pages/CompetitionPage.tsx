
import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  RefreshCw,
  Trophy,
  CalendarDays,
  Building2,
} from 'lucide-react'

interface Season {
  id: string
  name: string
  starts_on: string | null
  ends_on: string | null
}

interface Competition {
  id: string
  name: string
  slug: string
  organization: {
    name?: string
  } | null
  seasons: Season[]
  created_at: string
  updated_at: string
}

type Tab = 'overview' | 'matches'

export function CompetitionPage() {
  const { id } = useParams<{ id: string }>()

  const [competition, setCompetition] = useState<Competition | null>(null)
  const [activeTab, setActiveTab] = useState<Tab>('overview')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  async function loadCompetition(signal?: AbortSignal) {
    try {
      const response = await fetch(`/api/competitions/${id}`, {
        signal,
      })

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error('Competição não encontrada.')
        }

        throw new Error('Não foi possível carregar a competição.')
      }

      const data: Competition = await response.json()

      setCompetition(data)
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }

      setError(
        err instanceof Error
          ? err.message
          : 'Ocorreu um erro inesperado.',
      )
    } finally {
      if (!signal?.aborted) {
        setLoading(false)
      }
    }
  }

  useEffect(() => {
    const controller = new AbortController()

    void loadCompetition(controller.signal)

    return () => controller.abort()
  }, [id])

  function retryLoading() {
    setLoading(true)
    setError(null)
    void loadCompetition()
  }

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-4 w-32 rounded bg-gray-200" />
        <div className="h-10 w-2/3 rounded bg-gray-200" />
        <div className="h-32 rounded-2xl bg-white" />
      </div>
    )
  }

  if (error || !competition) {
    return (
      <section className="rounded-2xl border border-gray-200 bg-white p-10 text-center">
        <Trophy size={32} className="mx-auto text-gray-300" />

        <h1 className="mt-4 font-heading text-xl font-bold text-sofi-midnight">
          Não foi possível carregar a competição
        </h1>

        <p className="mt-2 text-sm text-gray-500">
          {error ?? 'Competição não encontrada.'}
        </p>

        <div className="mt-6 flex justify-center gap-3">
          <button
            onClick={retryLoading}
            className="rounded-full bg-sofi-blue px-5 py-2.5 text-sm font-semibold text-white hover:bg-blue-700"
          >
            Tentar novamente
          </button>

          <Link
            to="/"
            className="rounded-full border border-gray-200 px-5 py-2.5 text-sm font-medium text-gray-600 hover:bg-gray-50"
          >
            Voltar
          </Link>
        </div>
      </section>
    )
  }

  return (
    <section className="space-y-8">

      {/* Back navigation */}
      <Link
        to="/"
        className="inline-flex items-center gap-2 text-sm font-medium text-gray-500 transition-colors hover:text-sofi-blue"
      >
        <ArrowLeft size={16} />
        Todas as competições
      </Link>

      {/* Competition heading */}
      <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-3">
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-sofi-blue">
            Observatório do Futebol
          </p>

          <h1 className="font-heading text-3xl font-bold tracking-tight text-sofi-midnight sm:text-4xl">
            {competition.name}
          </h1>

          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-500">
            <span className="inline-flex items-center gap-1.5">
              <Building2 size={15} />
              {competition.organization?.name ?? 'Organização não informada'}
            </span>

            <span className="inline-flex items-center gap-1.5">
              <CalendarDays size={15} />
              {competition.seasons.length} {competition.seasons.length === 1 ? 'temporada' : 'temporadas'}
            </span>
          </div>
        </div>

        <button
          onClick={retryLoading}
          className="inline-flex items-center gap-2 self-start rounded-full border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-gray-600 transition-colors hover:border-sofi-blue/30 hover:text-sofi-blue"
        >
          <RefreshCw size={14} />
          Atualizar
        </button>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-8">
          <button
            onClick={() => setActiveTab('overview')}
            className={`border-b-2 px-1 pb-4 text-sm font-semibold transition-colors ${
              activeTab === 'overview'
                ? 'border-sofi-blue text-sofi-blue'
                : 'border-transparent text-gray-500 hover:text-gray-800'
            }`}
          >
            Overview
          </button>

          <button
            onClick={() => setActiveTab('matches')}
            className={`border-b-2 px-1 pb-4 text-sm font-semibold transition-colors ${
              activeTab === 'matches'
                ? 'border-sofi-blue text-sofi-blue'
                : 'border-transparent text-gray-500 hover:text-gray-800'
            }`}
          >
            Jogos
          </button>
        </nav>
      </div>

      {/* Overview */}
      {activeTab === 'overview' && (
        <div className="space-y-6">
          <div>
            <h2 className="font-heading text-xl font-bold text-sofi-midnight">
              Visão geral
            </h2>

            <p className="mt-1 text-sm text-gray-500">
              Informações gerais da competição e suas temporadas.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <div className="rounded-2xl border border-gray-200 bg-white p-6">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sofi-mist text-sofi-blue">
                  <Trophy size={20} />
                </div>

                <span className="text-sm font-medium text-gray-500">
                  Competição
                </span>
              </div>

              <p className="mt-5 font-heading text-lg font-bold text-sofi-midnight">
                {competition.name}
              </p>

              <p className="mt-1 text-xs text-gray-400">
                Identificador: {competition.slug}
              </p>
            </div>

            <div className="rounded-2xl border border-gray-200 bg-white p-6">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sofi-mist text-sofi-blue">
                  <CalendarDays size={20} />
                </div>

                <span className="text-sm font-medium text-gray-500">
                  Temporadas cadastradas
                </span>
              </div>

              <p className="mt-5 font-heading text-3xl font-bold text-sofi-midnight">
                {competition.seasons.length}
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="font-heading text-lg font-semibold text-sofi-midnight">
              Temporadas
            </h3>

            {competition.seasons.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-gray-300 bg-white/60 p-10 text-center">
                <p className="text-sm text-gray-500">
                  Nenhuma temporada cadastrada.
                </p>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {competition.seasons.map((season) => (
                  <div
                    key={season.id}
                    className="rounded-2xl border border-gray-200 bg-white p-5"
                  >
                    <p className="text-xs font-medium uppercase tracking-wider text-gray-400">
                      Temporada
                    </p>

                    <h4 className="mt-2 font-heading text-2xl font-bold text-sofi-midnight">
                      {season.name}
                    </h4>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Matches */}
      {activeTab === 'matches' && (
        <div className="rounded-2xl border border-dashed border-gray-300 bg-white/60 px-6 py-16 text-center">
          <CalendarDays size={32} className="mx-auto text-gray-300" />

          <h2 className="mt-4 font-heading text-lg font-semibold text-sofi-midnight">
            Jogos da competição
          </h2>

          <p className="mt-2 text-sm text-gray-500">
            A listagem de partidas será integrada nesta etapa do desenvolvimento.
          </p>
        </div>
      )}

    </section>
  )
}
