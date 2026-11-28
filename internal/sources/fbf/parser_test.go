package fbf

import (
	"os"
	"testing"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

const testCompetitionURL = "https://fbf.org.br/competicoes/1/2026"

func TestParseCompetition(t *testing.T) {
	content, err := os.ReadFile(
		"testdata/fbf_competition.html",
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ParseCompetition(
		testCompetitionURL,
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.ExternalID != "1:2026" {
		t.Errorf(
			"unexpected competition ID: %s",
			result.ExternalID,
		)
	}

	if result.Season != "2026" {
		t.Errorf(
			"unexpected season: %s",
			result.Season,
		)
	}

	if result.Name != "Baianão Série A" {
		t.Errorf(
			"unexpected competition name: %s",
			result.Name,
		)
	}
}

func TestParsePhases(t *testing.T) {
	content, err := os.ReadFile(
		"testdata/fbf_competition.html",
	)
	if err != nil {
		t.Fatal(err)
	}

	phases, err := ParsePhases(
		testCompetitionURL,
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(phases) != 3 {
		t.Fatalf(
			"expected 3 phases, got %d",
			len(phases),
		)
	}

	expectedIDs := []string{
		"433",
		"447",
		"448",
	}

	for i, phase := range phases {
		if phase.ExternalID != expectedIDs[i] {
			t.Errorf(
				"unexpected phase ID: %s",
				phase.ExternalID,
			)
		}
	}
}

func TestParseClubs(t *testing.T) {
	content, err := os.ReadFile(
		"testdata/fbf_phase.html",
	)
	if err != nil {
		t.Fatal(err)
	}

	clubs, err := ParseClubs(
		testCompetitionURL+"/433",
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(clubs) != 10 {
		t.Fatalf(
			"expected 10 clubs, got %d",
			len(clubs),
		)
	}

	found := false

	for _, club := range clubs {
		if club.ExternalID == "ECB" {
			found = true

			if club.Name != "Bahia" {
				t.Error("unexpected club name")
			}

			if club.LogoURL == "" {
				t.Error("missing club logo")
			}
		}
	}

	if !found {
		t.Error("Bahia not found")
	}
}

func TestParseMatches(t *testing.T) {
	content, err := os.ReadFile(
		"testdata/fbf_phase.html",
	)
	if err != nil {
		t.Fatal(err)
	}

	phase := canonical.Phase{
		ExternalID: "433",
		Name:       "Fase 1 - Classificação",
		Order:      1,
	}

	matches, err := ParseMatches(
		testCompetitionURL+"/433",
		string(content),
		phase,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 45 {
		t.Fatalf(
			"expected 45 matches, got %d",
			len(matches),
		)
	}

	match := matches[0]

	if match.ExternalID != "11826" {
		t.Errorf("unexpected match ID: %s", match.ExternalID)
	}

	if match.Round != 1 {
		t.Errorf("unexpected round: %d", match.Round)
	}

	if match.PhaseExternalID != "433" {
		t.Error("unexpected phase identifier")
	}

	if match.HomeClubExternalID != "ECV" {
		t.Error("unexpected home club")
	}

	if match.AwayClubExternalID != "AAC" {
		t.Error("unexpected away club")
	}

	if match.HomeScore == nil || *match.HomeScore != 0 {
		t.Error("unexpected home score")
	}

	if match.AwayScore == nil || *match.AwayScore != 0 {
		t.Error("unexpected away score")
	}

	if match.DateText != "10/01/2026" {
		t.Errorf("unexpected match date: %s", match.DateText)
	}

	if match.TimeText != "16:00" {
		t.Errorf("unexpected match time: %s", match.TimeText)
	}

	if match.Venue != "Manoel Barradas" {
		t.Errorf("unexpected venue: %s", match.Venue)
	}

	if match.Status != canonical.MatchCompleted {
		t.Error("unexpected match status")
	}
}

func TestParseDocuments(t *testing.T) {
	content, err := os.ReadFile(
		"testdata/fbf_match.html",
	)
	if err != nil {
		t.Fatal(err)
	}

	documents, err := ParseDocuments(
		"https://fbf.org.br/jogo/detalhes/11826",
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(documents) != 2 {
		t.Fatalf(
			"expected 2 documents, got %d",
			len(documents),
		)
	}

	expected := map[canonical.DocumentType]string{
		canonical.DocumentMatchSummary: "https://conteudo.cbf.com.br/federacoes/25/sumulas/2026/50998.pdf",

		canonical.DocumentFinancialBulletin: "https://conteudo.cbf.com.br/federacoes/25/borderos/2026/50998b.pdf",
	}

	for _, document := range documents {
		expectedURL, exists := expected[document.Type]

		if !exists {
			t.Errorf(
				"unexpected document type: %s",
				document.Type,
			)
			continue
		}

		if document.URL != expectedURL {
			t.Errorf(
				"unexpected document URL: %s",
				document.URL,
			)
		}

		if document.Status != canonical.DocumentDiscovered {
			t.Errorf(
				"unexpected document status: %s",
				document.Status,
			)
		}

		delete(expected, document.Type)
	}

	if len(expected) != 0 {
		t.Errorf("missing expected documents: %v", expected)
	}
}
