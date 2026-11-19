
import { useEffect, useState } from 'react'
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

interface FinancialItem {
  source_label: string
  category: string
  amount: number
  notes: string | null
}

interface FinancialExtraction {
  id: string
  document_id: string
  status: string
  error_message: string | null
  raw_result: {
    result: {
      match: {
        date: string
        venue: string
        competition: string
        home_team: string
        away_team: string
      }
      financial: {
        gross_revenue: number
        operating_expenses: number
        fees: number
        taxes: number
        other_deductions: number
        methodological_net_result: number
        reported_net_result: number
        retention_margin: number
      }
      attendance: {
        total: number
        paying: number
        non_paying: number
      }
      financial_items: FinancialItem[]
    }
    validation: {
      valid: boolean
      issues: string[]
      review_required: boolean
    }
  } | null
}

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

function formatCurrency(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value)) {
    return '—'
  }

  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(value)
}

function formatNumber(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value)) {
    return '—'
  }

  return new Intl.NumberFormat('pt-BR').format(value)
}

function formatPercentage(value: number | undefined): string {
  if (value === undefined || !Number.isFinite(value)) {
    return '—'
  }

  return new Intl.NumberFormat('pt-BR', {
    style: 'percent',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)
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

  const financialBulletinId = match.documents.find(
    (document) => document.type === 'financial_bulletin',
  )?.id ?? null

  const [extractionState, setExtractionState] = useState<{
    documentId: string
    data: FinancialExtraction | null
    error: string | null
  } | null>(null)

  const extraction = extractionState?.documentId === financialBulletinId
    ? extractionState.data
    : null

  const extractionError = extractionState?.documentId === financialBulletinId
    ? extractionState.error
    : null

  const loadingExtraction = Boolean(
    financialBulletinId &&
    extractionState?.documentId !== financialBulletinId
  )

  useEffect(() => {
    if (!financialBulletinId) return

    const documentId = financialBulletinId

    const controller = new AbortController()

    async function fetchExtraction() {
      try {
        const response = await fetch(
          `/api/documents/${documentId}/extraction`,
          { signal: controller.signal },
        )

        if (response.status === 404) {
          throw new Error(
            'A extração deste borderô ainda não está disponível.',
          )
        }

        if (!response.ok) {
          throw new Error(
            'Não foi possível consultar os dados financeiros.',
          )
        }

        const data: FinancialExtraction = await response.json()

        if (!controller.signal.aborted) {
          setExtractionState({
            documentId,
            data,
            error: null,
          })
        }
      } catch (err) {
        if (controller.signal.aborted) return

        setExtractionState({
          documentId,
          data: null,
          error: err instanceof Error
            ? err.message
            : 'Erro inesperado ao consultar o borderô.',
        })
      }
    }

    void fetchExtraction()

    return () => controller.abort()
  }, [financialBulletinId])

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

          {loadingExtraction && (
            <div className="rounded-2xl border border-gray-200 bg-white p-8 text-center">
              <div className="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-gray-200 border-t-sofi-blue" />
              <p className="mt-3 text-sm text-gray-500">
                Consultando dados financeiros...
              </p>
            </div>
          )}

          {!loadingExtraction && extractionError && (
            <div className="rounded-2xl border border-gray-200 bg-white p-6">
              <p className="text-sm font-medium text-gray-700">
                {extractionError}
              </p>
            </div>
          )}

          {!loadingExtraction && !extractionError && !financialBulletinId && (
            <div className="rounded-2xl border border-dashed border-gray-300 bg-white p-8 text-center">
              <p className="text-sm text-gray-500">
                Esta partida ainda não possui borderô financeiro disponível.
              </p>
            </div>
          )}

          {!loadingExtraction && extraction && (
            <>
              {extraction.status !== 'completed' && (
                <div className="rounded-2xl border border-gray-200 bg-white p-6">
                  <p className="text-sm font-semibold text-sofi-midnight">
                    {extraction.status === 'failed'
                      ? 'Falha no processamento do borderô'
                      : 'Extração em processamento'}
                  </p>

                  {extraction.error_message && (
                    <p className="mt-2 text-sm text-gray-500">
                      {extraction.error_message}
                    </p>
                  )}
                </div>
              )}

              {extraction.status === 'completed' &&
                extraction.raw_result && (
                  <>
                    {(() => {
                      const { financial, attendance, financial_items } =
                        extraction.raw_result.result

                      return (
                        <>
                          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            <FinancialIndicator
                              label="Público total"
                              value={formatNumber(attendance.total)}
                              icon={Users}
                            />

                            <FinancialIndicator
                              label="Renda bruta"
                              value={formatCurrency(financial.gross_revenue)}
                              icon={Wallet}
                            />

                            <FinancialIndicator
                              label="Despesas operacionais"
                              value={formatCurrency(financial.operating_expenses)}
                              icon={Receipt}
                            />

                            <FinancialIndicator
                              label="Resultado líquido"
                              value={formatCurrency(financial.methodological_net_result)}
                              icon={TrendingUp}
                              description="Resultado líquido metodológico"
                            />
                          </div>

                          <section className="rounded-2xl border border-gray-200 bg-white p-5">
                            <h4 className="font-heading text-sm font-bold text-sofi-midnight">
                              Detalhamento financeiro
                            </h4>

                            <div className="mt-4 space-y-3">
                              {[
                                {
                                  label: 'Público pagante',
                                  value: formatNumber(attendance.paying),
                                },
                                {
                                  label: 'Público não pagante',
                                  value: formatNumber(attendance.non_paying),
                                },
                                {
                                  label: 'Taxas',
                                  value: formatCurrency(financial.fees),
                                },
                                {
                                  label: 'Tributos',
                                  value: formatCurrency(financial.taxes),
                                },
                                {
                                  label: 'Outras deduções',
                                  value: formatCurrency(financial.other_deductions),
                                },
                                {
                                  label: 'Margem de retenção',
                                  value: formatPercentage(financial.retention_margin),
                                },
                              ].map((item) => (
                                <div
                                  key={item.label}
                                  className="flex items-center justify-between gap-4 border-b border-gray-100 pb-3 last:border-0 last:pb-0"
                                >
                                  <span className="text-sm text-gray-500">
                                    {item.label}
                                  </span>

                                  <span className="text-sm font-semibold text-sofi-midnight">
                                    {item.value}
                                  </span>
                                </div>
                              ))}
                            </div>
                          </section>

                          <section className="rounded-2xl border border-gray-200 bg-white p-5">
                            <h4 className="font-heading text-sm font-bold text-sofi-midnight">
                              Itens do borderô
                            </h4>

                            <div className="mt-4 space-y-3">
                              {financial_items.map((item, index) => (
                                <div
                                  key={`${item.source_label}-${index}`}
                                  className="flex items-start justify-between gap-4 border-b border-gray-100 pb-3 last:border-0 last:pb-0"
                                >
                                  <div className="min-w-0">
                                    <p className="text-sm text-gray-700">
                                      {item.source_label}
                                    </p>
                                    <p className="mt-1 text-xs text-gray-400">
                                      {item.category}
                                    </p>
                                  </div>

                                  <span className="shrink-0 text-sm font-semibold text-sofi-midnight">
                                    {formatCurrency(item.amount)}
                                  </span>
                                </div>
                              ))}
                            </div>
                          </section>

                          <div className="flex items-center justify-between rounded-xl bg-gray-50 px-4 py-3">
                            <span className="text-xs text-gray-500">
                              Validação da extração
                            </span>

                            <span className={`text-xs font-semibold ${
                              extraction.raw_result.validation.valid
                                ? 'text-green-700'
                                : 'text-amber-700'
                            }`}>
                              {extraction.raw_result.validation.valid
                                ? 'Dados válidos'
                                : 'Requer atenção'}
                            </span>
                          </div>
                        </>
                      )
                    })()}
                  </>
                )}

              {extraction.status === 'completed' &&
                !extraction.raw_result && (
                  <div className="rounded-2xl border border-gray-200 bg-white p-6">
                    <p className="text-sm text-gray-500">
                      A extração foi concluída, mas não retornou dados financeiros.
                    </p>
                  </div>
                )}
            </>
          )}
        </main>
      </section>
    </div>
  )
}
