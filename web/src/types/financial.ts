export interface GeneralStats {
    total_teams: number
    total_matches: number
    total_competitions: number
    total_seasons: number

    total_revenue: number
    total_gross_revenue: number
    total_attendance: number
    total_retention: number
    total_deficit_matches: number

    competitions: CompetitionStats[]
    competition_history: CompetitionSeasonStats[]
}

export interface CompetitionStats {
    id: string
    name: string
    slug: string
    seasons: string[]
    
    total_matches: number
    total_attendance: number
    total_teams: number
    total_revenue: number
    total_deficit_matches: number

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

    total_revenue?: number
    match_date?: string
}

export type CompetitionSeasonStats = {
    competition_id: string
    competition_name: string
    season: string

    gross_revenue: number
    total_revenue: number
    total_attendance: number
    average_attendance: number
    retention_margin: number
    total_matches: number
    total_deficit_matches: number
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