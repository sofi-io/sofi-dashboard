package fpf

type Competition struct {
	ExternalID string  `json:"external_id"`
	Name       string  `json:"name"`
	Season     string  `json:"season"`
	URL        string  `json:"url"`
	Phases     []Phase `json:"phases"`
}

type Phase struct {
	ExternalID   string `json:"external_id"`
	Name         string `json:"name"`
	InitialRound int    `json:"initial_round"`
	Selected     bool   `json:"selected"`
}

type Club struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
}

type MatchStatus string

const (
	MatchScheduled MatchStatus = "scheduled"
	MatchCompleted MatchStatus = "completed"
)

type Match struct {
	ExternalID    string `json:"external_id"`
	CompetitionID string `json:"competition_id"`
	PhaseID       string `json:"phase_id"`
	PhaseName     string `json:"phase_name"`
	Round         int    `json:"round"`

	DateText string `json:"date_text"`
	TimeText string `json:"time_text"`
	Venue    string `json:"venue"`

	HomeClubExternalID string `json:"home_club_external_id"`
	AwayClubExternalID string `json:"away_club_external_id"`

	HomeScore *int `json:"home_score,omitempty"`
	AwayScore *int `json:"away_score,omitempty"`

	Status    MatchStatus `json:"status"`
	URL       string      `json:"url"`
	Documents []Document  `json:"documents,omitempty"`
}

type DocumentType string

const (
	DocumentMatchReport       DocumentType = "match_report"
	DocumentFinancialBulletin DocumentType = "financial_bulletin"
	DocumentActionPlan        DocumentType = "action_plan"
)

type DocumentStatus string

const (
	DocumentDiscovered DocumentStatus = "discovered"
	DocumentPending    DocumentStatus = "pending"
	DocumentDownloaded DocumentStatus = "downloaded"
	DocumentFailed     DocumentStatus = "failed"
)

type Document struct {
	Type   DocumentType   `json:"type"`
	URL    string         `json:"url,omitempty"`
	Status DocumentStatus `json:"status"`
}
