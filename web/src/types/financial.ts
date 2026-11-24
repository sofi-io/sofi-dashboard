export interface GeneralStats {
    totalTeams: number
    totalMatches: number
    totalCompetitions: number
    totalSeasons: number

    totalRevenue: number
    totalAttendance: number
    totalRetention: number
    totalDeficitMatches: number

    competitions: CompetitionStats[]
}

export interface CompetitionStats {
    id: string
    name: string
    slug: string
    
    total_matches: number
    total_attendance: number
    total_teams: number
    gross_revenue: number
    average_attendance: number
    retention_margin: number

    matches: MatchStats[]
}

export interface MatchStats {
    id: string
    
    home_team: string
    away_team: string
    home_logo: string | null
    away_logo: string | null

    gross_revenue: number
    attendance: number
    retention_margin: number
}

export interface ClubStats {
    id: string
    name: string
    logo_url: string | null

    total_matches: number
    gross_revenue: number
    total_attendance: number
    average_attendance: number
    retention_margin: number

    matches: MatchStats[]
}