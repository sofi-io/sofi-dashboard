import { useEffect, useRef, useState } from 'react'
import { toPng, toCanvas } from 'html-to-image'
import { jsPDF } from 'jspdf'
import {
  FileImage,
  FileText,
} from 'lucide-react'
  import { Link, useParams } from 'react-router-dom'
  import {
    ArrowLeft,
    RefreshCw,
    Trophy,
    CalendarDays,
    Building2,
  } from 'lucide-react'

  import { CompetitionTabs } from '../components/CompetitionTabs'
  import { MatchesList } from '../components/MatchesList'
  import { FinancialOverview } from '../components/FinancialOverview'

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

  export function CompetitionDetailPage() {
    const { id } = useParams<{ id: string }>()

    const [competition, setCompetition] = useState<Competition | null>(null)
    const [activeTab, setActiveTab] = useState<'overview' | 'matches'>('overview')
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [reloadKey, setReloadKey] = useState(0)

    const overviewExportRef = useRef<HTMLDivElement>(null)
    const financialCardRef = useRef<HTMLDivElement>(null)

    function getExportFilename(extension: string) {
      const slug = competition?.slug ?? 'competicao'

      return `sofi-${slug}-overview.${extension}`
    }

    async function exportFinancialCard() {
      try {
        const element = financialCardRef.current

        if (!element) {
          throw new Error('Card financeiro não encontrado.')
        }

        const dataUrl = await toPng(element, {
          pixelRatio: 2,
          backgroundColor: '#ffffff',
          cacheBust: true,
        })

        const link = document.createElement('a')
        link.download = getExportFilename('png')
        link.href = dataUrl

        document.body.appendChild(link)
        link.click()
        link.remove()

      } catch (error) {
        console.error('Erro ao exportar imagem:', error)
        alert(
          error instanceof Error
            ? error.message
            : 'Erro ao exportar imagem.'
        )
      }
    }

    async function exportOverviewPDF() {
      try {
        const element = overviewExportRef.current

        if (!element) {
          throw new Error('Área do relatório não encontrada.')
        }

        const canvas = await toCanvas(element, {
          pixelRatio: 2,
          backgroundColor: '#FAF9F6',
          cacheBust: true,
        })

        const pdf = new jsPDF('p', 'mm', 'a4')

        const pageWidth = 190
        const pageHeight = 277

        const sourcePageHeight =
          canvas.width * (pageHeight / pageWidth)

        let sourceY = 0
        let pageNumber = 0

        while (sourceY < canvas.height) {
          const sliceHeight = Math.min(
            sourcePageHeight,
            canvas.height - sourceY,
          )

          const pageCanvas = document.createElement('canvas')
          pageCanvas.width = canvas.width
          pageCanvas.height = sliceHeight

          const context = pageCanvas.getContext('2d')

          if (!context) {
            throw new Error('Não foi possível criar a imagem do PDF.')
          }

          context.drawImage(
            canvas,
            0,
            sourceY,
            canvas.width,
            sliceHeight,
            0,
            0,
            canvas.width,
            sliceHeight,
          )

          if (pageNumber > 0) {
            pdf.addPage()
          }

          const imageData = pageCanvas.toDataURL('image/png')

          pdf.addImage(
            imageData,
            'PNG',
            10,
            10,
            pageWidth,
            (sliceHeight / canvas.width) * pageWidth,
          )

          sourceY += sliceHeight
          pageNumber++
        }

        pdf.save(getExportFilename('pdf'))

      } catch (error) {
        console.error('Erro ao exportar PDF:', error)
        alert(
          error instanceof Error
            ? error.message
            : 'Erro ao exportar PDF.'
        )
      }
    }

    useEffect(() => {
      const controller = new AbortController()

      async function fetchCompetition() {
        try {
          const response = await fetch(
            `/api/competitions/${id}`,
            { signal: controller.signal },
          )

          if (!response.ok) {
            throw new Error('Não foi possível carregar a competição.')
          }

          const data = await response.json()

          if (!controller.signal.aborted) {
            setCompetition(data)
            setError(null)
          }
        } catch (err) {
          if (controller.signal.aborted) return

          setError(
            err instanceof Error
              ? err.message
              : 'Erro inesperado ao carregar competição.',
          )
        } finally {
          if (!controller.signal.aborted) {
            setLoading(false)
          }
        }
      }

      void fetchCompetition()

      return () => controller.abort()
    }, [id, reloadKey])

    function retryLoading() {
      setLoading(true)
      setError(null)
      setReloadKey((current) => current + 1)
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
        <CompetitionTabs
          activeTab={activeTab}
          onChange={setActiveTab}
        />

        {/* Overview */}
        {activeTab === 'overview' && (
          <div className="space-y-10">
            {/* Export buttons */}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={exportFinancialCard}
                className="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-semibold text-gray-600 transition-colors hover:border-sofi-blue/30 hover:text-sofi-blue"
              >
                <FileImage size={16} />
                Exportar imagem
              </button>

              <button
                type="button"
                onClick={exportOverviewPDF}
                className="inline-flex items-center gap-2 rounded-xl bg-sofi-blue px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-blue-700"
              >
                <FileText size={16} />
                Exportar PDF
              </button>
            </div>

            {/* Financial overview */}
            <div ref={overviewExportRef} className="space-y-10">

              <FinancialOverview
                competitionId={competition.id}
                seasons={competition.seasons}
                financialCardRef={financialCardRef}
              />

              {/* Seasons */}
              <div className="space-y-4">
                <div>
                  <h3 className="font-heading text-lg font-semibold text-sofi-midnight">
                    Temporadas cadastradas
                  </h3>

                  <p className="mt-1 text-sm text-gray-500">
                    Histórico de temporadas disponíveis nesta competição.
                  </p>
                </div>

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
          </div>
        )}

        {/* Matches */}
        {activeTab === 'matches' && (
          <MatchesList
            competitionId={competition.id}
            seasons={competition.seasons}
          />
        )}

      </section>
    )
  }
