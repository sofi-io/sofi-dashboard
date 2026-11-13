package fpf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

func TestDiscoverCompetition(t *testing.T) {
	var phaseRequests int
	var detailRequests int
	var documentRequests int

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet &&
				r.URL.Path == "/pt/competicoes/jogos.php":

				fmt.Fprint(w, `
                    <html>
                    <h3 class="head_background_title">
                        Pernambucano A1 / 2026
                    </h3>
                    <a class="fase active"
                       data-id="1651"
                       data-fase="3801"
                       data-rodada="1">Final</a>
                    </html>
                `)

			case r.Method == http.MethodPost &&
				r.URL.Path == "/pt/competicoes/jogos_ajax_ver2.php":

				phaseRequests++

				if err := r.ParseForm(); err != nil {
					t.Errorf("parse phase form: %v", err)
				}

				if r.Form.Get("idcampeonato") != "1651" {
					t.Errorf("unexpected competition ID")
				}

				if r.Form.Get("idfase") != "3801" {
					t.Errorf("unexpected phase ID")
				}

				payload := map[string]interface{}{
					"rodada": 1,
					"html": `
                        <div class="swiper-slide">
                            <div class="title-rodada">1ª Rodada</div>

                            <div class="pr-jogos">
                                <span class="cronolog-sm">
                                    DOM, 01/03 - 18h00 - Aflitos
                                </span>

                                <img src="https://conteudo.cbf.com.br/clubes/20010/escudo.jpg">
                                <img src="https://conteudo.cbf.com.br/clubes/20023/escudo.jpg">

                                <div class="result">
                                    <span>3</span>
                                    <span class="versus">X</span>
                                    <span>1</span>
                                </div>

                                <a href="../competicoes/jogo.php?id=1651&jogo=51836">
                                    sobre o jogo
                                </a>
                            </div>
                        </div>
                    `,
				}

				json.NewEncoder(w).Encode(payload)

			case r.Method == http.MethodGet &&
				r.URL.Path == "/pt/competicoes/jogo.php":

				detailRequests++

				fmt.Fprint(w, `
                    <html>
                        <div id="infoJogo"></div>
                    </html>
                `)

			case r.Method == http.MethodPost &&
				r.URL.Path == "/pt/competicoes/infoJogo.php":

				documentRequests++

				if err := r.ParseForm(); err != nil {
					t.Errorf("parse document form: %v", err)
				}

				if r.Form.Get("idjogo") != "51836" {
					t.Errorf("unexpected match ID")
				}

				fmt.Fprint(w, `
                    <div class="row">
                        <section id="documentos">
                            <a href="https://example.com/51836.pdf">
                                Súmula
                            </a>
                            <a href="https://example.com/51836b.pdf">
                                Boletim Financeiro
                            </a>
                        </section>
                    </div>
                `)

			default:
				http.NotFound(w, r)
			}
		},
	))
	defer server.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	client.httpClient = server.Client()

	competitionURL := server.URL +
		"/pt/competicoes/jogos.php?q=1651"

	result, err := client.DiscoverCompetition(
		context.Background(),
		competitionURL,
		canonicalOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Competition.Name != "Pernambucano A1 / 2026" {
		t.Errorf("unexpected competition name: %s",
			result.Competition.Name)
	}

	if len(result.Competition.Phases) != 1 {
		t.Fatalf("got %d phases, want 1",
			len(result.Competition.Phases))
	}

	if len(result.Matches) != 1 {
		t.Fatalf("got %d matches, want 1",
			len(result.Matches))
	}

	match := result.Matches[0]

	if match.ExternalID != "51836" {
		t.Errorf("unexpected match ID: %s", match.ExternalID)
	}

	if match.HomeClubExternalID != "20010" {
		t.Errorf("unexpected home club ID")
	}

	if match.AwayClubExternalID != "20023" {
		t.Errorf("unexpected away club ID")
	}

	if match.HomeScore == nil || *match.HomeScore != 3 {
		t.Errorf("unexpected home score")
	}

	if match.AwayScore == nil || *match.AwayScore != 1 {
		t.Errorf("unexpected away score")
	}

	if len(match.Documents) != 2 {
		t.Fatalf("got %d documents, want 2",
			len(match.Documents))
	}

	if phaseRequests != 1 {
		t.Errorf("phase requests = %d, want 1", phaseRequests)
	}

	if detailRequests != 1 {
		t.Errorf("detail requests = %d, want 1", detailRequests)
	}

	if documentRequests != 1 {
		t.Errorf("document requests = %d, want 1", documentRequests)
	}
}

func canonicalOptions() canonical.DiscoveryOptions {
	return canonical.DiscoveryOptions{
		MaxConcurrentMatches: 2,
	}
}

func TestResolveEndpoint(t *testing.T) {
	got, err := resolveEndpoint(
		"https://www.fpf-pe.com.br/pt/competicoes/jogos.php?q=1651",
		"jogos_ajax_ver2.php",
	)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.Path != "/pt/competicoes/jogos_ajax_ver2.php" {
		t.Errorf("unexpected endpoint: %s", got)
	}
}
