package cbf

type Club struct {
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}

type MatchStatus string

const (
	MatchScheduled     MatchStatus = "scheduled"
	MatchCompleted     MatchStatus = "completed"
	MatchPostponed     MatchStatus = "postponed"
	MatchCancelled     MatchStatus = "cancelled"
	MatchPendingResult MatchStatus = "pending_result"
)

type Match struct {
	ExternalID string      `json:"external_id"`
	Slug       string      `json:"slug"`
	URL        string      `json:"url"`
	Status     MatchStatus `json:"status,omitempty"`
	Documents  []Document  `json:"documents,omitempty"`
}

type DocumentType string

const (
	DocumentSummary           DocumentType = "match_summary"
	DocumentFinancialBulletin DocumentType = "financial_bulletin"
	DocumentMatchReport       DocumentType = "match_report"
)

type DocumentStatus string

const (
	DocumentPending    DocumentStatus = "pending"
	DocumentDiscovered DocumentStatus = "discovered"
	DocumentDownloaded DocumentStatus = "downloaded"
	DocumentSubmitted  DocumentStatus = "submitted"
	DocumentProcessed  DocumentStatus = "processed"
	DocumentFailed     DocumentStatus = "failed"
)

type Document struct {
	Type   DocumentType   `json:"type"`
	URL    string         `json:"url,omitempty"`
	Status DocumentStatus `json:"status"`
}

type Competition struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Season struct {
	Name string `json:"name"`
}

type MatchHistory struct {
	ClubID  string  `json:"club_id"`
	Matches []Match `json:"matches"`
}
