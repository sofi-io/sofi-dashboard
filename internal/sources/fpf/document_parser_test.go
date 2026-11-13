package fpf

import (
	"os"
	"testing"
)

func TestParseMatchDocuments(t *testing.T) {
	raw, err := os.ReadFile("testdata/match-info.html")
	if err != nil {
		t.Fatal(err)
	}

	documents, err := ParseMatchDocuments(
		"https://www.fpf-pe.com.br/pt/competicoes/infoJogo.php",
		string(raw),
	)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[DocumentType]string{
		DocumentMatchReport: "https://conteudo.cbf.com.br/federacoes/9/sumulas/2026/51836.pdf",

		DocumentFinancialBulletin: "https://conteudo.cbf.com.br/federacoes/9/borderos/2026/51836b.pdf",
	}

	for documentType, expectedURL := range expected {
		var found bool

		for _, document := range documents {
			if document.Type != documentType {
				continue
			}

			found = true

			if document.URL != expectedURL {
				t.Errorf(
					"%s: got %s, want %s",
					documentType,
					document.URL,
					expectedURL,
				)
			}

			if document.Status != DocumentDiscovered {
				t.Errorf(
					"%s: unexpected status %s",
					documentType,
					document.Status,
				)
			}
		}

		if !found {
			t.Errorf("document %s not found", documentType)
		}
	}
}
