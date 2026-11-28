import { useMemo, useState } from 'react'

import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import type {
  CompetitionStats,
  CompetitionSeasonStats,
} from '../../../types/financial'

type Metric =
  | 'gross_revenue'
  | 'total_revenue'
  | 'total_attendance'
  | 'average_attendance'
  | 'retention_margin'
  | 'total_deficit_matches'

type View = 'evolution' | 'ranking'

const METRICS: Record<Metric, string> = {
  gross_revenue: 'Receita bruta',
  total_revenue: 'Resultado líquido',
  total_attendance: 'Público total',
  average_attendance: 'Média de público',
  retention_margin: 'Margem de retenção',
  total_deficit_matches: 'Jogos deficitários',
}

const COLORS = [
  '#2563EB',
  '#0D9488',
  '#D97706',
  '#7C3AED',
  '#DB2777',
  '#0891B2',
  '#65A30D',
  '#DC2626',
  '#64748B',
  '#9333EA',
]

function formatValue(value: number, metric: Metric) {
  if (metric === 'retention_margin') {
    return `${value.toLocaleString('pt-BR', {
      maximumFractionDigits: 1,
    })}%`
  }

  if (
    metric === 'gross_revenue' ||
    metric === 'total_revenue'
  ) {
    return value.toLocaleString('pt-BR', {
      style: 'currency',
      currency: 'BRL',
      notation: 'compact',
      maximumFractionDigits: 1,
    })
  }

  return value.toLocaleString('pt-BR', {
    notation: 'compact',
    maximumFractionDigits: 1,
  })
}

type Props = {
  competitions: CompetitionStats[]
  history: CompetitionSeasonStats[]
  selectedSeason: string
}

export function CompetitionRanking({
  competitions,
  history,
  selectedSeason,
}: Props) {
  const [view, setView] = useState<View>('evolution')

  const [metric, setMetric] =
    useState<Metric>('gross_revenue')

  const series = useMemo(() => {
    const items = new Map<string, string>()

    history.forEach((item) => {
      items.set(
        item.competition_id,
        item.competition_name,
      )
    })

    competitions.forEach((item) => {
      items.set(item.id, item.name)
    })

    return Array.from(items.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([id, name], index) => ({
        id,
        name,
        color: COLORS[index % COLORS.length],
      }))
  }, [history, competitions])

  const evolution = useMemo(() => {
    const seasons = new Map<
      string,
      Record<string, string | number | null>
    >()

    history.forEach((item) => {
      if (!seasons.has(item.season)) {
        seasons.set(item.season, {
          season: item.season,
        })
      }

      const current = seasons.get(item.season)!

      current[item.competition_id] = item[metric]
    })

    return Array.from(seasons.values()).sort(
      (a, b) =>
        String(a.season).localeCompare(
          String(b.season),
          undefined,
          { numeric: true },
        ),
    )
  }, [history, metric])

  const ranking = useMemo(() => {
    return competitions
      .map((item) => ({
        id: item.id,
        name: item.name,
        value: item[metric] ?? 0,
        color:
          series.find((s) => s.id === item.id)?.color
          ?? COLORS[0],
      }))
      .sort((a, b) => b.value - a.value)
  }, [competitions, metric, series])

  const historicalRanking = useMemo(() => {
    const lastValues = new Map<string, number>()

    evolution.forEach((item) => {
      series.forEach(({ id }) => {
        if (typeof item[id] === 'number') {
          lastValues.set(id, item[id] as number)
        }
      })
    })

    return series
      .filter(({ id }) => lastValues.has(id))
      .map((item) => ({
        ...item,
        value: lastValues.get(item.id)!,
      }))
      .sort((a, b) => b.value - a.value)
  }, [evolution, series])

  return (
    <section className="w-full bg-sofi-white pb-12">
      <div className="mx-auto max-w-7xl px-6 lg:px-12">

        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-4">
          <h2 className="font-heading text-xl font-bold text-sofi-midnight">
            Ranking de competições
          </h2>

          <select
            value={metric}
            onChange={(e) =>
              setMetric(e.target.value as Metric)
            }
            className="cursor-pointer rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-sofi-midnight outline-none"
          >
            {Object.entries(METRICS).map(
              ([key, label]) => (
                <option key={key} value={key}>
                  {label}
                </option>
              ),
            )}
          </select>
        </div>

        {/* Tabs */}
        <div className="mt-5 flex gap-6 border-b border-gray-200">
          {(['evolution', 'ranking'] as View[]).map(
            (item) => (
              <button
                key={item}
                type="button"
                onClick={() => setView(item)}
                className={`border-b-2 px-1 pb-3 text-sm transition-colors ${
                  view === item
                    ? 'border-sofi-blue font-semibold text-sofi-blue'
                    : 'border-transparent text-gray-400 hover:text-sofi-midnight'
                }`}
              >
                {item === 'evolution'
                  ? 'Evolução'
                  : 'Ranking'}
              </button>
            ),
          )}
        </div>

        {/* Chart */}
        <div className="mt-6">

          {view === 'evolution' ? (
            <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_220px]">

              <div className="h-[360px] min-w-0">
                {evolution.length === 0 ? (
                  <div className="flex h-full items-center justify-center text-sm text-gray-400">
                    Sem dados históricos
                  </div>
                ) : (
                  <ResponsiveContainer
                    width="100%"
                    height="100%"
                  >
                    <LineChart
                      data={evolution}
                      margin={{
                        top: 16,
                        right: 12,
                        bottom: 0,
                        left: 8,
                      }}
                    >
                      <CartesianGrid
                        vertical={false}
                        stroke="#E5E7EB"
                        strokeDasharray="3 3"
                      />

                      <XAxis
                        dataKey="season"
                        axisLine={false}
                        tickLine={false}
                        tickMargin={12}
                        tick={{
                          fontSize: 12,
                          fill: '#9CA3AF',
                        }}
                      />

                      <YAxis
                        axisLine={false}
                        tickLine={false}
                        width={76}
                        tick={{
                          fontSize: 11,
                          fill: '#9CA3AF',
                        }}
                        tickFormatter={(value) =>
                          formatValue(value, metric)
                        }
                      />

                      <Tooltip
                        formatter={(value) =>
                          formatValue(Number(value), metric)
                        }
                        contentStyle={{
                          border: '1px solid #E5E7EB',
                          borderRadius: 8,
                          fontSize: 12,
                          boxShadow: 'none',
                        }}
                      />

                      {series.map((item) => (
                        <Line
                          key={item.id}
                          type="linear"
                          dataKey={item.id}
                          name={item.name}
                          stroke={item.color}
                          strokeWidth={2}
                          dot={{ r: 3 }}
                          activeDot={{ r: 5 }}
                          connectNulls={false}
                          isAnimationActive={false}
                        />
                      ))}
                    </LineChart>
                  </ResponsiveContainer>
                )}
              </div>

              {/* Ranking lateral */}
              <div className="max-h-[360px] overflow-y-auto">
                {historicalRanking.map((item, index) => (
                  <div
                    key={item.id}
                    className="flex items-center gap-2 py-2 text-xs"
                  >
                    <span className="w-5 shrink-0 tabular-nums text-gray-400">
                      {index + 1}
                    </span>

                    <span
                      className="h-2 w-2 shrink-0 rounded-full"
                      style={{
                        backgroundColor: item.color,
                      }}
                    />

                    <span className="flex-1 truncate font-medium text-sofi-midnight">
                      {item.name}
                    </span>

                    <span className="shrink-0 tabular-nums text-gray-500">
                      {formatValue(item.value, metric)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <>
              <p className="mb-3 text-xs text-gray-400">
                {selectedSeason}
              </p>

              <div
                style={{
                  height: Math.max(
                    180,
                    ranking.length * 42 + 28,
                  ),
                }}
              >
                <ResponsiveContainer
                  width="100%"
                  height="100%"
                >
                  <BarChart
                    data={ranking}
                    layout="vertical"
                    margin={{
                      top: 0,
                      right: 24,
                      bottom: 0,
                      left: 0,
                    }}
                  >
                    <CartesianGrid
                      horizontal={false}
                      stroke="#F1F1F1"
                    />

                    <XAxis
                      type="number"
                      axisLine={false}
                      tickLine={false}
                      tick={{
                        fontSize: 11,
                        fill: '#9CA3AF',
                      }}
                      tickFormatter={(value) =>
                        formatValue(value, metric)
                      }
                    />

                    <YAxis
                      type="category"
                      dataKey="name"
                      axisLine={false}
                      tickLine={false}
                      width={145}
                      tick={{
                        fontSize: 12,
                        fill: '#374151',
                      }}
                    />

                    <Tooltip
                      cursor={{ fill: '#F9FAFB' }}
                      formatter={(value) =>
                        formatValue(Number(value), metric)
                      }
                      contentStyle={{
                        border: '1px solid #E5E7EB',
                        borderRadius: 8,
                        fontSize: 12,
                      }}
                    />

                    <Bar
                      dataKey="value"
                      name={METRICS[metric]}
                      barSize={18}
                      radius={3}
                    >
                      {ranking.map((item) => (
                        <Cell
                          key={item.id}
                          fill={item.color}
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </>
          )}
        </div>
      </div>
    </section>
  )
}