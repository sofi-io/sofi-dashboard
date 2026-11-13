package cbf

import "testing"

func TestValidateCompetitionURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name: "valid CBF competition",
			url:  "https://www.cbf.com.br/futebol-brasileiro/tabelas/campeonato-brasileiro/serie-b/2026",
		},
		{
			name:    "reject HTTP",
			url:     "http://www.cbf.com.br/futebol-brasileiro/tabelas/serie-b/2026",
			wantErr: true,
		},
		{
			name:    "reject external host",
			url:     "https://example.com/futebol-brasileiro/tabelas/serie-b/2026",
			wantErr: true,
		},
		{
			name:    "reject non competition page",
			url:     "https://www.cbf.com.br/futebol-brasileiro/times/20023",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCompetitionURL(tt.url)

			if (err != nil) != tt.wantErr {
				t.Fatalf("validateCompetitionURL() error = %v, wantErr %v",
					err, tt.wantErr)
			}
		})
	}
}

func TestParseCompetitionName(t *testing.T) {
	html := `
	<html>
		<head><title>CBF - Campeonato</title></head>
		<body><h1>Campeonato Brasileiro Série B</h1></body>
	</html>
	`

	got := parseCompetitionName(html)
	want := "Campeonato Brasileiro Série B"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseCompetitionNameFallback(t *testing.T) {
	html := `
	<html>
		<head><title>CBF - Campeonato Brasileiro</title></head>
		<body></body>
	</html>
	`

	got := parseCompetitionName(html)
	want := "CBF - Campeonato Brasileiro"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
