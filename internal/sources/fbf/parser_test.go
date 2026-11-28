package fbf

import (
	"os"
	"testing"
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
