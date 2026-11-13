package fpf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type matchesFixture struct {
	HTML  string `json:"html"`
	Round int    `json:"rodada"`
}

func loadMatchesFixture(t *testing.T, phaseID string) string {
	t.Helper()

	path := filepath.Join(
		"testdata",
		fmt.Sprintf("phase-%s.json", phaseID),
	)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	var fixture matchesFixture

	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}

	if fixture.HTML == "" {
		t.Fatalf("fixture %s has empty HTML", path)
	}

	return fixture.HTML
}

func TestParseMatchesAllPhases(t *testing.T) {
	tests := []struct {
		name          string
		phaseID       string
		expectedCount int
	}{
		{
			name:          "Primeira Fase",
			phaseID:       "3798",
			expectedCount: 28,
		},
		{
			name:          "Fase Eliminatória",
			phaseID:       "3799",
			expectedCount: 4,
		},
		{
			name:          "Semifinais",
			phaseID:       "3800",
			expectedCount: 4,
		},
		{
			name:          "Final",
			phaseID:       "3801",
			expectedCount: 2,
		},
	}

	const baseURL = "https://www.fpf-pe.com.br/pt/competicoes/jogos.php?q=1651"

	allIDs := make(map[string]bool)

	totalMatches := 0

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawHTML := loadMatchesFixture(t, tt.phaseID)

			matches, err := ParseMatches(
				baseURL,
				"1651",
				tt.phaseID,
				tt.name,
				rawHTML,
			)

			if err != nil {
				t.Fatalf("ParseMatches returned error: %v", err)
			}

			if len(matches) != tt.expectedCount {
				t.Fatalf(
					"got %d matches, want %d",
					len(matches),
					tt.expectedCount,
				)
			}

			for _, match := range matches {
				if match.ExternalID == "" {
					t.Error("match has empty external ID")
				}

				if allIDs[match.ExternalID] {
					t.Errorf(
						"duplicate match ID across phases: %s",
						match.ExternalID,
					)
				}

				allIDs[match.ExternalID] = true

				if match.CompetitionID != "1651" {
					t.Errorf(
						"unexpected competition ID: %s",
						match.CompetitionID,
					)
				}

				if match.PhaseID != tt.phaseID {
					t.Errorf(
						"unexpected phase ID: %s",
						match.PhaseID,
					)
				}

				if match.PhaseName != tt.name {
					t.Errorf(
						"unexpected phase name: %s",
						match.PhaseName,
					)
				}

				if match.Round <= 0 {
					t.Errorf(
						"invalid round for match %s: %d",
						match.ExternalID,
						match.Round,
					)
				}

				if match.DateText == "" {
					t.Errorf("match %s has no date", match.ExternalID)
				}

				if match.TimeText == "" {
					t.Errorf("match %s has no time", match.ExternalID)
				}

				if match.Venue == "" {
					t.Errorf("match %s has no venue", match.ExternalID)
				}

				if match.HomeClubExternalID == "" {
					t.Errorf("match %s has no home club ID", match.ExternalID)
				}

				if match.AwayClubExternalID == "" {
					t.Errorf("match %s has no away club ID", match.ExternalID)
				}

				if match.HomeClubExternalID == match.AwayClubExternalID {
					t.Errorf("match %s has identical home and away clubs", match.ExternalID)
				}

				if match.URL == "" {
					t.Errorf("match %s has no URL", match.ExternalID)
				}

				if (match.HomeScore == nil) != (match.AwayScore == nil) {
					t.Errorf(
						"match %s has incomplete score",
						match.ExternalID,
					)
				}

				if match.Status == MatchCompleted {
					if match.HomeScore == nil || match.AwayScore == nil {
						t.Errorf(
							"completed match %s has no complete score",
							match.ExternalID,
						)
					}
				}

				if match.Status == MatchScheduled {
					if match.HomeScore != nil || match.AwayScore != nil {
						t.Errorf(
							"scheduled match %s unexpectedly has a score",
							match.ExternalID,
						)
					}
				}
			}

			totalMatches += len(matches)
		})
	}

	if totalMatches != 38 {
		t.Errorf("got %d total matches, want 38", totalMatches)
	}

	if len(allIDs) != 38 {
		t.Errorf("got %d unique match IDs, want 38", len(allIDs))
	}
}

func TestParseMatchesFinalKnownMatch(t *testing.T) {
	rawHTML := loadMatchesFixture(t, "3801")

	matches, err := ParseMatches(
		"https://www.fpf-pe.com.br/pt/competicoes/jogos.php?q=1651",
		"1651",
		"3801",
		"Final",
		rawHTML,
	)
	if err != nil {
		t.Fatal(err)
	}

	var found bool

	for _, match := range matches {
		if match.ExternalID != "51836" {
			continue
		}

		found = true

		if match.HomeClubExternalID != "20010" {
			t.Errorf(
				"home club: got %s, want 20010",
				match.HomeClubExternalID,
			)
		}

		if match.AwayClubExternalID != "20023" {
			t.Errorf(
				"away club: got %s, want 20023",
				match.AwayClubExternalID,
			)
		}

		if match.HomeScore == nil || match.AwayScore == nil {
			t.Fatal("known completed match has no score")
		}

		if *match.HomeScore != 3 || *match.AwayScore != 3 {
			t.Errorf(
				"score: got %d-%d, want 3-3",
				*match.HomeScore,
				*match.AwayScore,
			)
		}

		if match.Status != MatchCompleted {
			t.Errorf("got status %s, want completed", match.Status)
		}

		expectedURL := "https://www.fpf-pe.com.br/pt/competicoes/jogo.php?id=1651&jogo=51836"

		if match.URL != expectedURL {
			t.Errorf(
				"URL: got %s, want %s",
				match.URL,
				expectedURL,
			)
		}
	}

	if !found {
		t.Fatal("known match 51836 not found")
	}
}
