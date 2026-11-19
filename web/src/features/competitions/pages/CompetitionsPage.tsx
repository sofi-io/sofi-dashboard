
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowUpRight, Trophy, RefreshCw } from 'lucide-react'

interface Competition {
  id: string
  name: string
  slug: string
}

interface CompetitionListResponse {
  items: Competition[]
  total: number
  limit: number
  offset: number
}

export function CompetitionsPage() {
  const [competitions, setCompetitions] = useState<Competition[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  function retryLoading() {
    setLoading(true)
    setError(null)
    void loadCompetitions()
  }

  async function loadCompetitions(signal?: AbortSignal) {
    try {
      const response = await fetch('/api/competitions?limit=100&offset=0', {
        signal,
      })

      if (!response.ok) {
        throw new Error('Não foi possível carregar as competições.')
      }

      const data: CompetitionListResponse = await response.json()

      setCompetitions(data.items ?? [])
      setTotal(data.total ?? 0)
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

    void loadCompetitions(controller.signal)

    return () => controller.abort()
  }, [])

  return (
    <section className="space-y-10">

      {/* Page heading */}
      <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-3">

          <h1 className="font-heading text-4xl font-bold tracking-tight text-sofi-midnight sm:text-5xl">
            Competições
          </h1>

          <p className="max-w-2xl text-sm leading-relaxed text-gray-500 sm:text-base">
            Acompanhe os dados financeiros e as informações públicas
            das competições do futebol brasileiro.
          </p>
        </div>

        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Trophy size={17} className="text-sofi-blue" />
          <span>
            {loading ? 'Carregando...' : `${total} ${total === 1 ? 'competição' : 'competições'}`}
          </span>
        </div>
      </div>

      {/* Competition list */}
      <div className="space-y-5">

        <div className="flex items-center justify-between border-b border-gray-200 pb-4">
          <h2 className="font-heading text-lg font-semibold text-sofi-midnight">
            Todas as competições
          </h2>

          <button
            type="button"
            onClick={retryLoading}
            disabled={loading}
            aria-label="Atualizar competições"
            className="flex items-center gap-2 rounded-full px-3 py-2 text-xs font-medium text-gray-500 transition-colors hover:bg-sofi-mist hover:text-sofi-blue disabled:opacity-50"
          >
            <RefreshCw
              size={14}
              className={loading ? 'animate-spin' : ''}
            />
            Atualizar
          </button>
        </div>

        {loading && (
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map((item) => (
              <div
                key={item}
                className="animate-pulse rounded-2xl border border-gray-200 bg-white p-6"
              >
                <div className="h-10 w-10 rounded-xl bg-gray-100" />
                <div className="mt-6 h-5 w-3/4 rounded bg-gray-100" />
                <div className="mt-3 h-3 w-1/2 rounded bg-gray-100" />
                <div className="mt-8 h-4 w-1/3 rounded bg-gray-100" />
              </div>
            ))}
          </div>
        )}

        {!loading && error && (
          <div className="rounded-2xl border border-red-200 bg-white p-8 text-center">
            <p className="font-medium text-sofi-midnight">
              Não foi possível carregar as competições.
            </p>

            <p className="mt-2 text-sm text-gray-500">
              {error}
            </p>

            <button
              type="button"
              onClick={retryLoading}
              className="mt-5 rounded-full bg-sofi-blue px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-blue-700"
            >
              Tentar novamente
            </button>
          </div>
        )}

        {!loading && !error && competitions.length === 0 && (
          <div className="rounded-2xl border border-dashed border-gray-300 bg-white/60 px-6 py-16 text-center">
            <Trophy size={32} className="mx-auto text-gray-300" />

            <h3 className="mt-4 font-heading text-lg font-semibold text-sofi-midnight">
              Nenhuma competição disponível
            </h3>

            <p className="mt-2 text-sm text-gray-500">
              As competições cadastradas serão apresentadas aqui.
            </p>
          </div>
        )}

        {!loading && !error && competitions.length > 0 && (
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2 lg:grid-cols-3">
            {competitions.map((competition) => (
              <Link
                key={competition.id}
                to={`/competitions/${competition.id}`}
                className="group flex flex-col justify-between rounded-2xl border border-gray-200/80 bg-white p-6 transition-all duration-200 hover:-translate-y-1 hover:border-sofi-blue/30 hover:shadow-lg hover:shadow-gray-900/5"
              >
                <div>
                  <div className="flex items-start justify-between">
                    <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-sofi-mist text-sofi-blue">
                      <Trophy size={21} />
                    </div>

                    <ArrowUpRight
                      size={19}
                      className="text-gray-300 transition-colors group-hover:text-sofi-blue"
                    />
                  </div>

                  <h3 className="mt-6 font-heading text-lg font-bold text-sofi-midnight transition-colors group-hover:text-sofi-blue">
                    {competition.name}
                  </h3>
                </div>

                <div className="mt-8 flex items-center justify-between border-t border-gray-100 pt-4">
                  <span className="text-xs font-medium text-gray-400">
                    Explorar competição
                  </span>

                  <span className="text-sm font-semibold text-sofi-blue">
                    →
                  </span>
                </div>
              </Link>
            ))}
          </div>
        )}

      </div>
    </section>
  )
}
