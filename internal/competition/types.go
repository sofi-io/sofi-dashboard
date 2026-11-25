package competition

import "context"

type Competition struct {
	Source       string `json:"source"`
	ExternalID   string `json:"external_id"`
	CanonicalKey string `json:"canonical_key"`
	Name         string `json:"name"`
	Season       string `json:"season,omitempty"`
	URL          string `json:"url"`
}

type Club struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url,omitempty"`
	LogoURL    string `json:"logo_url,omitempty"`
}

type Phase struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Order      int    `json:"order,omitempty"`
}

type MatchStatus string

const (
	MatchScheduled MatchStatus = "scheduled"
	MatchCompleted MatchStatus = "completed"
	MatchPostponed MatchStatus = "postponed"
	MatchCancelled MatchStatus = "cancelled"
	MatchPending   MatchStatus = "pending_result"
)

type Match struct {
	ExternalID string `json:"external_id"`
	URL        string `json:"url"`

	PhaseExternalID string `json:"phase_external_id,omitempty"`
	PhaseName       string `json:"phase_name,omitempty"`
	Round           int    `json:"round,omitempty"`

	DateText string `json:"date_text,omitempty"`
	TimeText string `json:"time_text,omitempty"`
	Venue    string `json:"venue,omitempty"`

	HomeClubExternalID string `json:"home_club_external_id,omitempty"`
	AwayClubExternalID string `json:"away_club_external_id,omitempty"`

	HomeScore *int `json:"home_score,omitempty"`
	AwayScore *int `json:"away_score,omitempty"`

	Status    MatchStatus `json:"status"`
	Documents []Document  `json:"documents,omitempty"`
}

type DocumentType string

const (
	DocumentMatchReport       DocumentType = "match_report"
	DocumentFinancialBulletin DocumentType = "financial_bulletin"
	DocumentActionPlan        DocumentType = "action_plan"
	DocumentMatchSummary      DocumentType = "match_summary"
)

type DocumentStatus string

const (
	DocumentPending    DocumentStatus = "pending"
	DocumentDiscovered DocumentStatus = "discovered"
	DocumentDownloaded DocumentStatus = "downloaded"
	DocumentProcessed  DocumentStatus = "processed"
	DocumentFailed     DocumentStatus = "failed"
	DocumentSubmitted  DocumentStatus = "submitted"
)

type Document struct {
	Type   DocumentType   `json:"type"`
	URL    string         `json:"url,omitempty"`
	Status DocumentStatus `json:"status"`
}

type Warning struct {
	Stage    string `json:"stage"`
	EntityID string `json:"entity_id,omitempty"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error"`
}

type Result struct {
	Competition Competition `json:"competition"`
	Clubs       []Club      `json:"clubs"`
	Phases      []Phase     `json:"phases"`
	Matches     []Match     `json:"matches"`
	Warnings    []Warning   `json:"warnings,omitempty"`
}

type DiscoveryOptions struct {
	MaxConcurrentClubs   int
	MaxConcurrentMatches int
}

type Discoverer interface {
	DiscoverCompetition(
		ctx context.Context,
		competitionURL string,
		options DiscoveryOptions,
	) (*Result, error)
}

type GeneralStats struct {
	TotalTeams        int `json:"total_teams"`
	TotalMatches      int `json:"total_matches"`
	TotalCompetitions int `json:"total_competitions"`
	TotalSeasons      int `json:"total_seasons"`

	TotalRevenue        float64 `json:"total_revenue"`
	TotalGrossRevenue   float64 `json:"total_gross_revenue"`
	TotalAttendance     int     `json:"total_attendance"`
	TotalRetention      float64 `json:"total_retention"`
	TotalDeficitMatches int     `json:"total_deficit_matches"`

	Competitions []CompetitionStats `json:"competitions"`
}

type CompetitionStats struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`

	TotalMatches      int      `json:"total_matches"`
	TotalAttendance   int      `json:"total_attendance"`
	TotalTeams        int      `json:"total_teams"`
	TotalRevenue      float64  `json:"total_revenue"`
	GrossRevenue      float64  `json:"gross_revenue"`
	AverageAttendance float64  `json:"average_attendance"`
	RetentionMargin   float64  `json:"retention_margin"`
	Seasons           []string `json:"seasons"`

	Matches []MatchStats `json:"matches"`
}

type MatchStats struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`

	HomeTeam string `json:"home_team"`
	AwayTeam string `json:"away_team"`
	HomeLogo string `json:"home_logo,omitempty"`
	AwayLogo string `json:"away_logo,omitempty"`

	GrossRevenue    float64 `json:"gross_revenue"`
	Attendance      int     `json:"attendance"`
	RetentionMargin float64 `json:"retention_margin"`
}

type ClubStats struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logo_url,omitempty"`

	TotalMatches      int     `json:"total_matches"`
	GrossRevenue      float64 `json:"gross_revenue"`
	TotalAttendance   int     `json:"total_attendance"`
	AverageAttendance float64 `json:"average_attendance"`
	RetentionMargin   float64 `json:"retention_margin"`

	Matches []MatchStats `json:"matches"`
}
