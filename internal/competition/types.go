package competition

import "context"

type Competition struct {
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Season     string `json:"season,omitempty"`
	URL        string `json:"url"`
}

type Club struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url,omitempty"`
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
