package cbf

import (
	"os"
	"testing"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

func TestParseMatches(t *testing.T) {
	content, err := os.ReadFile("../../../club.html")
	if err != nil {
		t.Fatal(err)
	}

	const clubURL = "https://www.cbf.com.br/futebol-brasileiro/times/campeonato-brasileiro/serie-b/2026/20023"

	matches, err := ParseMatches(
		clubURL,
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 38 {
		t.Fatalf(
			"expected 38 matches, got %d",
			len(matches),
		)
	}

	match := matches[0]

	if match.ExternalID != "833285" {
		t.Errorf("unexpected match ID: %s", match.ExternalID)
	}

	if match.Round != 38 {
		t.Errorf("unexpected round: %d", match.Round)
	}

	if match.HomeClubExternalID != "62726" {
		t.Error("unexpected home club")
	}

	if match.AwayClubExternalID != "20023" {
		t.Error("unexpected away club")
	}

	if match.HomeScore == nil || *match.HomeScore != 2 {
		t.Error("unexpected home score")
	}

	if match.AwayScore == nil || *match.AwayScore != 1 {
		t.Error("unexpected away score")
	}

	if match.Status != canonical.MatchCompleted {
		t.Error("unexpected match status")
	}

	if match.URL == "" {
		t.Error("empty match URL")
	}
}

func TestParseMatchDetails(t *testing.T) {
	content, err := os.ReadFile("../../../match.html")
	if err != nil {
		t.Fatal(err)
	}

	match := canonical.Match{
		ExternalID: "833285",
	}

	phases := []canonical.Phase{
		{
			ExternalID: "2009",
			Name:       "1ª Fase",
			Order:      1,
		},
		{
			ExternalID: "2010",
			Name:       "Playoff de Acesso",
			Order:      2,
		},
	}

	result, err := ParseMatchDetails(
		match,
		phases,
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.PhaseExternalID != "2009" {
		t.Errorf(
			"unexpected phase ID: %s",
			result.PhaseExternalID,
		)
	}

	if result.PhaseName != "1ª Fase" {
		t.Errorf(
			"unexpected phase name: %s",
			result.PhaseName,
		)
	}

	if len(result.Documents) != 3 {
		t.Fatalf(
			"expected 3 documents, got %d",
			len(result.Documents),
		)
	}

	foundFinancialBulletin := false

	for _, document := range result.Documents {
		if document.Status != canonical.DocumentDiscovered {
			t.Errorf("unexpected document status")
		}

		if document.Type == canonical.DocumentFinancialBulletin {
			foundFinancialBulletin = true

			expectedURL := "https://conteudo.cbf.com.br/sumulas/2026/242374b.pdf"

			if document.URL != expectedURL {
				t.Errorf(
					"unexpected financial bulletin URL: %s",
					document.URL,
				)
			}
		}
	}

	if !foundFinancialBulletin {
		t.Error("financial bulletin not found")
	}
}
