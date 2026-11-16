
type CompetitionTab = 'overview' | 'matches'

interface CompetitionTabsProps {
  activeTab: CompetitionTab
  onChange: (tab: CompetitionTab) => void
}

const tabs: { id: CompetitionTab; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'matches', label: 'Jogos' },
]

export function CompetitionTabs({
  activeTab,
  onChange,
}: CompetitionTabsProps) {
  return (
    <div className="border-b border-gray-200">
      <nav className="flex gap-8" aria-label="Navegação da competição">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id

          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => onChange(tab.id)}
              aria-current={isActive ? 'page' : undefined}
              className={`relative border-b-2 px-1 pb-4 text-sm font-semibold transition-colors ${
                isActive
                  ? 'border-sofi-blue text-sofi-blue'
                  : 'border-transparent text-gray-500 hover:text-sofi-midnight'
              }`}
            >
              {tab.label}
            </button>
          )
        })}
      </nav>
    </div>
  )
}
