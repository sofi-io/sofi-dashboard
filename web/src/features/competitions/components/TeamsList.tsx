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
        {/* <p className="text-sm text-gray-500">
          Receita Bruta: R${' '}
          {formatCurrency(team.gross_revenue)}
        </p> */}
      </div>
    </div>
  )
}

export function TeamsList({ competitionId, seasons }: { competitionId: string; seasons: any[] }) {
  const [teams, setTeams] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedSeason, setSelectedSeason] = useState(
    seasons[0]?.id ?? '',
  )
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchTeams() {
      setLoading(true)
      try {
        const response = await fetch(
          `/api/competitions/${competitionId}/teams?season=${selectedSeason}`
        )
        const data = await response.json()
        setTeams(data.items || [])
      } catch (error) {
        setError('Erro ao buscar equipes.')
        console.error('Error fetching teams:', error)
      }
      setLoading(false)
    }

    fetchTeams()
  }, [competitionId, selectedSeason])

  if (loading) {
    return <p>Carregando equipes...</p>
  }
  if (error) {
    return <p>{error}</p>
  }

  if (teams.length === 0) {
    return <p>Nenhuma equipe encontrada para esta competição e temporada.</p>
  }

  return (
    <section className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        {/* <div>
          <h2 className="font-heading text-xl font-bold text-sofi-midnight">
            Partidas
          </h2>

          <p className="mt-1 text-sm text-gray-500">
            Consulte os jogos e a disponibilidade dos borderôs.
          </p>
        </div> */}

        {seasons.length > 0 && (
          <select
            aria-label="Selecionar temporada"
            value={selectedSeason}
            onChange={(event) => {
              setLoading(true)
              setError(null)
              setSelectedSeason(event.target.value)
            }}
            className="rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-sofi-midnight outline-none transition-colors focus:border-sofi-blue"
          >
            {seasons.map((season) => (
              <option key={season.id} value={season.id}>
                Temporada {season.name}
              </option>
            ))}
          </select>
        )}
      </div>
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
    </section>
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


