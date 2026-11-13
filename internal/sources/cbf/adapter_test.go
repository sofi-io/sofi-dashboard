package cbf

import (
	"testing"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

func TestCBFCompetitionIdentity(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		externalID string
		season     string
	}{
		{
			name:       "serie B 2026",
			url:        "https://www.cbf.com.br/futebol-brasileiro/tabelas/campeonato-brasileiro/serie-b/2026",
			externalID: "serie-b",
			season:     "2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, season, err := cbfCompetitionIdentity(tt.url)
			if err != nil {
				t.Fatal(err)
			}

			if id != tt.externalID {
				t.Errorf("external ID = %q, want %q", id, tt.externalID)
			}

			if season != tt.season {
				t.Errorf("season = %q, want %q", season, tt.season)
			}
		})
	}
}

func TestConvertCBFDocuments(t *testing.T) {
	documents := []Document{
		{
			Type:   DocumentFinancialBulletin,
			URL:    "https://example.com/bordero.pdf",
			Status: DocumentDiscovered,
		},
		{
			Type:   DocumentMatchReport,
			URL:    "https://example.com/sumula.pdf",
			Status: DocumentPending,
		},
	}

	result := convertDocuments(documents)

	if len(result) != 2 {
		t.Fatalf("got %d documents, want 2", len(result))
	}

	if result[0].Type != canonical.DocumentFinancialBulletin {
		t.Errorf("unexpected document type: %s", result[0].Type)
	}

	if result[0].Status != canonical.DocumentDiscovered {
		t.Errorf("unexpected document status: %s", result[0].Status)
	}
}

func TestCBFAdapterImplementsDiscoverer(t *testing.T) {
	var discoverer canonical.Discoverer = NewAdapter(NewClient())

	if discoverer == nil {
		t.Fatal("adapter should implement Discoverer")
	}
}
