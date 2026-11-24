import { useEffect, useState } from 'react'

import type {
  GeneralStats
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