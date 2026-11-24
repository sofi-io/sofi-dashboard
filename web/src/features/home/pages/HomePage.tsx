import { useEffect, useState } from 'react'

import type {
  GeneralStats,
  CompetitionStats
} from '../../../types/financial'

function StatsCard({ stats }: { stats: GeneralStats | null }) {
  if (!stats) {
    return null
  }
  return (
    <div className="flex flex-col gap-2 rounded-lg border border-gray-200/70 bg-white/90 p-4 shadow-[0_4px_24px_rgba(17,24,39,0.05)] backdrop-blur-xl">
      <h2 className="text-lg font-semibold">Estatísticas Gerais</h2>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        <div className="flex flex-col gap-1 rounded-lg bg-sofi-mist p-3">
          <span className="text-sm text-gray-600">Total de Competições</span>
          <span className="text-xl font-bold">{stats.totalCompetitions}</span>
        </div>
        <div className="flex flex-col gap-1 rounded-lg bg-sofi-mist p-3">
          <span className="text-sm text-gray-600">Total de Clubes</span>
          <span className="text-xl font-bold">{stats.totalTeams}</span>
        </div>
        <div className="flex flex-col gap-1 rounded-lg bg-sofi-mist p-3">
          <span className="text-sm text-gray-600">Total de Partidas</span>
          <span className="text-xl font-bold">{stats.totalMatches}</span>
        </div>
      </div>
    </div>
  )
}

function DetailsCard({ title, value, description }: { title: string; value: string | number; description: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-lg bg-sofi-mist p-3">
      <span className="text-sm text-gray-600">{title}</span>
      <span className="text-xl font-bold">{value}</span>
      <span className="text-xs text-gray-500">{description}</span>
    </div>
  )
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

function fetchGeneralStats(): Promise<GeneralStats> {
  return fetch('/api/financial/general-stats')
      .then((response) => {
          if (!response.ok) {
              throw new Error('Erro ao buscar estatísticas gerais.')
          }
          return response.json()
      })
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
      <div className="flex flex-col gap-4">
          <h1 className="text-2xl font-bold">Página Inicial</h1>
          <p>Bem-vindo ao painel de controle da Sofi!</p>
          <StatsCard stats={stats} />
      </div>
  )
}