import { useEffect, useState } from 'react'

function TeamCard({ team, onClick }: { team: any; onClick: () => void }) {
  return (
    <div
      className="flex cursor-pointer items-center gap-4 rounded-lg border border-gray-200 p-4 hover:bg-gray-50"
      onClick={onClick}
    >
      <img
        src={team.logo_url || '/default-logo.png'}
        alt={team.name}
        className="h-12 w-12 rounded-full object-cover"
      />
      <div>
        <h3 className="font-heading text-lg font-bold text-sofi-midnight">
          {team.name}
        </h3>
        <p className="text-sm text-gray-500">
          Receita Bruta: R${' '}
          {formatCurrency(team.gross_revenue)}
        </p>
      </div>
    </div>
  )
}

export function TeamsList({ competitionId, season }: { competitionId: string; season: string }) {
  const [teams, setTeams] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchTeams() {
      setLoading(true)
      try {
        const response = await fetch(
          `/api/competitions/${competitionId}/teams?season=${season}`
        )
        const data = await response.json()
        setTeams(data.items || [])
      } catch (error) {
        console.error('Error fetching teams:', error)
      }
      setLoading(false)
    }

    fetchTeams()
  }, [competitionId, season])

  if (loading) {
    return <p>Carregando equipes...</p>
  }

  if (teams.length === 0) {
    return <p>Nenhuma equipe encontrada para esta competição e temporada.</p>
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      {teams.map((team) => (
        <TeamCard
          key={team.id}
          team={team}
          onClick={() => {
            // Handle team click, e.g., navigate to team details page
            console.log('Clicked on team:', team.name)
          }}
        />
      ))}
    </div>
  )
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

export default TeamsList  


