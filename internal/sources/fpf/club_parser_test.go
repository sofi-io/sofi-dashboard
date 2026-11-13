package fpf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseClubs(t *testing.T) {
	path := filepath.Join(
		"testdata",
		"classification-3798.html",
	)

	rawHTML, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	clubs, err := ParseClubs(string(rawHTML))
	if err != nil {
		t.Fatalf("ParseClubs() error = %v", err)
	}

	if len(clubs) != 8 {
		t.Fatalf("expected 8 clubs, got %d", len(clubs))
	}

	expected := map[string]string{
		"20023": "Náutico",
		"20010": "Sport Recife",
		"55531": "Retrô",
		"20039": "Santa Cruz",
	}

	for id, expectedName := range expected {
		found := false

		for _, club := range clubs {
			if club.ExternalID == id {
				found = true

				if club.Name != expectedName {
					t.Errorf(
						"club %s: expected name %q, got %q",
						id,
						expectedName,
						club.Name,
					)
				}
				break
			}
		}

		if !found {
			t.Errorf("club %s not found", id)
		}
	}
}

func TestParseClubsAcrossPhases(t *testing.T) {
	files := []string{
		"classification-3798.html",
		"classification-3799.html",
		"classification-3800.html",
		"classification-3801.html",
	}

	seen := make(map[string]bool)

	for _, filename := range files {
		rawHTML, err := os.ReadFile(
			filepath.Join("testdata", filename),
		)
		if err != nil {
			t.Fatal(err)
		}

		clubs, err := ParseClubs(string(rawHTML))
		if err != nil {
			t.Fatalf("%s: %v", filename, err)
		}

		for _, club := range clubs {
			seen[club.ExternalID] = true
		}
	}

	if len(seen) != 8 {
		t.Fatalf("expected 8 unique clubs, got %d", len(seen))
	}
}

func TestParseClubsEmptyHTML(t *testing.T) {
	clubs, err := ParseClubs("<html><body></body></html>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clubs) != 0 {
		t.Fatalf("expected 0 clubs, got %d", len(clubs))
	}
}

func TestParseClubsDeduplicatesWithinClassification(t *testing.T) {
	html := `
	<table>
		<tr>
			<td>1º</td>
			<td><img src="https://example.com/clubes/123/escudo.jpg"></td>
			<td>Clube A</td>
		</tr>
		<tr>
			<td>2º</td>
			<td><img src="https://example.com/clubes/123/escudo.jpg"></td>
			<td>Clube A</td>
		</tr>
	</table>`

	clubs, err := ParseClubs(html)
	if err != nil {
		t.Fatal(err)
	}

	if len(clubs) != 1 {
		t.Fatalf("expected 1 unique club, got %d", len(clubs))
	}
}
