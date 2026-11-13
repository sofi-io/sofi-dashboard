package cbf

import (
	"testing"
)

func TestParseClubsDeduplicates(t *testing.T) {
	html := `
	<a href="/futebol-brasileiro/times/serie-b/2026/20023">
		<span>Náutico</span>
	</a>
	<a href="/futebol-brasileiro/times/serie-b/2026/20023">
		<span>Náutico</span>
	</a>
	<a href="/futebol-brasileiro/times/serie-b/2026/20010">
		<span>Outro Clube</span>
	</a>
	`

	clubs, err := ParseClubs("https://www.cbf.com.br", html)
	if err != nil {
		t.Fatal(err)
	}

	if len(clubs) != 2 {
		t.Fatalf("got %d clubs, want 2", len(clubs))
	}

	if clubs[0].ExternalID != "20023" {
		t.Errorf("unexpected club ID: %s", clubs[0].ExternalID)
	}
}

func TestParseMatches(t *testing.T) {
	html := `
	<a href="/futebol-brasileiro/jogos/serie-b/2026/londrina-x-nautico/833285">
		Londrina x Náutico
	</a>
	<a href="/futebol-brasileiro/jogos/serie-b/2026/londrina-x-nautico/833285">
		Londrina x Náutico
	</a>
	`

	matches, err := ParseMatches("https://www.cbf.com.br", html)
	if err != nil {
		t.Fatal(err)
	}

	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1", len(matches))
	}

	if matches[0].ExternalID != "833285" {
		t.Errorf("unexpected match ID: %s", matches[0].ExternalID)
	}
}

func TestParseDocuments(t *testing.T) {
	html := `
	<a href="https://conteudo.cbf.com.br/sumulas/2026/242374se.pdf">
		Súmula
	</a>
	<a href="https://conteudo.cbf.com.br/sumulas/2026/242374b.pdf">
		Boletim Financeiro
	</a>
	<a href="https://conteudo.cbf.com.br/sumulas/2026/242374rdj.pdf">
		Relatório de Jogo
	</a>
	`

	documents, err := ParseDocuments("https://www.cbf.com.br", html)
	if err != nil {
		t.Fatal(err)
	}

	if len(documents) != 3 {
		t.Fatalf("got %d documents, want 3", len(documents))
	}

	if documents[1].Type != DocumentFinancialBulletin {
		t.Errorf("expected financial bulletin, got %s",
			documents[1].Type)
	}
}
