export interface MatchFinancialHighlight {
  match_id: string
  home_team: string
  away_team: string
  gross_revenue: number
  attendance: number
}

export interface ClubFinancialRanking {
  club_id: string
  club_name: string
  gross_revenue: number
  methodological_net_result: number
  retention_margin: number
  home_matches: number
  total_attendance: number
  average_attendance: number
}

export interface CompetitionFinancialOverview {
  total_matches: number
  matches_with_extraction: number
  matches_without_extraction: number

  gross_revenue: number
  operating_expenses: number
  fees: number
  taxes: number
  other_deductions: number
  methodological_net_result: number
  retention_margin: number

  total_attendance: number
  paying_attendance: number
  non_paying_attendance: number
  average_attendance: number

  deficit_matches: number

  highest_revenue_match: MatchFinancialHighlight | null
  highest_attendance_match: MatchFinancialHighlight | null
  lowest_attendance_match: MatchFinancialHighlight | null

  top_attendance_club: ClubFinancialRanking | null
  top_retention_club: ClubFinancialRanking | null

  club_rankings: ClubFinancialRanking[]
}