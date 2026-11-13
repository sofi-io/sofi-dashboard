package cbf

type Club struct {
	ExternalID string
	Name       string
	URL        string
}

type Match struct {
	ExternalID string
	Slug       string
	URL        string
}

type DocumentType string

const (
	DocumentSummary           DocumentType = "match_summary"
	DocumentFinancialBulletin DocumentType = "financial_bulletin"
	DocumentMatchReport       DocumentType = "match_report"
)

type Document struct {
	Type DocumentType
	URL  string
}

type Competition struct {
	Name string
	URL  string
}

type Season struct {
	Name string
}

type MatchHistory struct {
	ClubID  string
	Matches []Match
}
