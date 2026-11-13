package fpf

import (
	"os"
	"testing"
)

func TestParseCompetition(t *testing.T) {
	content, err := os.ReadFile("testdata/competition.html")
	if err != nil {
		t.Fatal(err)
	}

	result, err := ParseCompetition(
		"https://www.fpf-pe.com.br/pt/competicoes/jogos.php?q=1651",
		string(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.ExternalID != "1651" {
		t.Errorf("unexpected competition ID: %s", result.ExternalID)
	}

	if result.Name != "Pernambucano A1 / 2026" {
		t.Errorf("unexpected name: %s", result.Name)
	}

	if result.Season != "2026" {
		t.Errorf("unexpected season: %s", result.Season)
	}

	if len(result.Phases) != 4 {
		t.Fatalf("got %d phases, want 4", len(result.Phases))
	}

	expected := []struct {
		id   string
		name string
	}{
		{"3798", "Primeira Fase"},
		{"3799", "Fase Eliminatória"},
		{"3800", "Semifinais"},
		{"3801", "Final"},
	}

	for i, want := range expected {
		got := result.Phases[i]

		if got.ExternalID != want.id {
			t.Errorf("phase %d: got ID %s, want %s",
				i, got.ExternalID, want.id)
		}

		if got.Name != want.name {
			t.Errorf("phase %d: got name %s, want %s",
				i, got.Name, want.name)
		}
	}

	selected := 0
	for _, phase := range result.Phases {
		if phase.Selected {
			selected++

			if phase.ExternalID != "3801" {
				t.Errorf("unexpected selected phase: %s",
					phase.ExternalID)
			}
		}
	}

	if selected != 1 {
		t.Errorf("got %d selected phases, want 1", selected)
	}
}
