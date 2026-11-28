package fbf

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

const sourceName = "fbf"

var matchDatePattern = regexp.MustCompile(
	`(\d{2}/\d{2}/\d{4})\s+(\d{2}:\d{2})`,
)

func resolveURL(baseURL, targetURL string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("parse target URL: %w", err)
	}

	resolved := base.ResolveReference(target)
	return resolved.String(), nil
}

func extractClubCodes(
	doc *goquery.Document,
) map[string]string {
	codes := make(map[string]string)

	doc.Find("#carouselExample .flex-confronto-card").Each(
		func(_ int, selection *goquery.Selection) {
			teams := selection.Find(".time-confronto")
			logos := selection.Find("img")

			if teams.Length() != 2 || logos.Length() != 2 {
				return
			}

			for i := 0; i < 2; i++ {
				code := strings.TrimSpace(
					teams.Eq(i).Text(),
				)

				logo, exists := logos.Eq(i).Attr("src")

				if exists && code != "" {
					codes[logo] = code
				}
			}
		},
	)

	return codes
}

func ParseCompetition(competitionURL string, rawHTML string) (canonical.Competition, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return canonical.Competition{}, fmt.Errorf(
			"parse competition HTML: %w",
			err,
		)
	}

	parsedURL, err := url.Parse(competitionURL)
	if err != nil {
		return canonical.Competition{}, fmt.Errorf(
			"parse competition URL: %w",
			err,
		)
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "competicoes" {
		return canonical.Competition{}, fmt.Errorf(
			"invalid competition URL format: %s",
			competitionURL,
		)
	}

	competitionID := parts[1]
	season := parts[2]

	var competitionName string
	doc.Find("a[href]").EachWithBreak(
		func(_ int, s *goquery.Selection) bool {
			href, _ := s.Attr("href")

			link, err := url.Parse(href)
			if err != nil {
				return true // continue
			}

			expectedPath := "/competicoes/" + competitionID
			if strings.TrimSuffix(link.Path, "/") == expectedPath {
				competitionName = strings.TrimSpace(s.Text())
				return false // break
			}

			return true // continue
		},
	)

	if competitionName == "" {
		return canonical.Competition{}, fmt.Errorf(
			"competition name not found in HTML for URL: %s",
			competitionURL,
		)
	}

	parsedURL.Path = fmt.Sprintf("/competicoes/%s/%s", competitionID, season)
	parsedURL.RawPath = ""
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	return canonical.Competition{
		Source:       sourceName,
		ExternalID:   competitionID + ":" + season,
		CanonicalKey: "fbf-" + competitionID,
		Name:         competitionName,
		Season:       season,
		URL:          parsedURL.String(),
	}, nil
}

func ParsePhases(competitionURL string, rawHTML string) ([]canonical.Phase, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf(
			"parse competition HTML: %w",
			err,
		)
	}

	phases := make([]canonical.Phase, 0)
	seen := make(map[string]struct{})

	doc.Find(".fase-links a[href]").Each(
		func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")

			link, err := url.Parse(href)
			if err != nil {
				return
			}

			parts := strings.Split(strings.Trim(link.Path, "/"), "/")
			if len(parts) != 4 || parts[0] != "competicoes" {
				return
			}

			phaseID := parts[3]
			if _, exists := seen[phaseID]; exists {
				return
			}

			competitionName := strings.TrimSpace(s.Text())
			if phaseID == "" && competitionName == "" {
				return
			}

			seen[phaseID] = struct{}{}

			phases = append(
				phases,
				canonical.Phase{
					ExternalID: phaseID,
					Name:       competitionName,
					Order:      len(phases) + 1,
				},
			)
		},
	)

	if len(phases) == 0 {
		return nil, fmt.Errorf(
			"no phases found in HTML for URL: %s",
			competitionURL,
		)
	}

	return phases, nil
}

func ParseClubs(competitionURL string, rawHTML string) ([]canonical.Club, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf(
			"parse competition HTML: %w",
			err,
		)
	}

	codes := extractClubCodes(doc)

	clubs := make([]canonical.Club, 0)
	seen := make(map[string]struct{})

	var parseError error

	doc.Find("table tbody tr").EachWithBreak(
		func(_ int, s *goquery.Selection) bool {
			name := strings.TrimSpace(s.Find(".team-meta__place").Text())
			if name == "" {
				parseError = fmt.Errorf(
					"missing club name in HTML for URL: %s",
					competitionURL,
				)
				return false // break
			}

			logo, exists := s.Find("img").First().Attr("src")
			if !exists {
				parseError = fmt.Errorf(
					"missing club logo in HTML for URL: %s",
					competitionURL,
				)
				return false // break
			}

			clubID, exists := codes[logo]
			if !exists {
				parseError = fmt.Errorf(
					"missing club code for logo %s in HTML for URL: %s",
					logo,
					competitionURL,
				)
				return false // break
			}

			if _, exists := seen[clubID]; exists {
				return true // continue
			}

			logoURL, err := resolveURL(competitionURL, logo)
			if err != nil {
				parseError = fmt.Errorf(
					"resolve club logo URL: %w",
					err,
				)
				return false // break
			}

			seen[clubID] = struct{}{}

			clubs = append(
				clubs,
				canonical.Club{
					ExternalID: clubID,
					Name:       name,
					LogoURL:    logoURL,
				},
			)

			return true // continue
		},
	)

	if parseError != nil {
		return nil, parseError
	}

	if len(clubs) == 0 {
		return nil, fmt.Errorf(
			"no clubs found in HTML for URL: %s",
			competitionURL,
		)
	}

	return clubs, nil
}

func ParseMatches(phaseURL string, rawHTML string, phase canonical.Phase) ([]canonical.Match, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf(
			"parse competition HTML: %w",
			err,
		)
	}

	matches := make([]canonical.Match, 0)
	seen := make(map[string]struct{})

	var parseError error

	doc.Find("#carouselExample .carousel-item").EachWithBreak(
		func(_ int, roundSelection *goquery.Selection) bool {
			roundText := strings.TrimSpace(roundSelection.Find(".carousel-titulo h4").Text())

			round, err := strconv.Atoi(strings.TrimPrefix(roundText, "Rodada "))
			if err != nil {
				parseError = fmt.Errorf(
					"parse round number: %w",
					err,
				)
				return false // break
			}

			roundSelection.Find(".flex-column.pt-2").EachWithBreak(
				func(_ int, matchSelection *goquery.Selection) bool {
					link, exists := matchSelection.Find(".detalhe-do-jogo a").Attr("href")
					if !exists {
						parseError = fmt.Errorf(
							"missing match link in HTML for URL: %s",
							phaseURL,
						)
						return true // continue
					}

					matchURL, err := resolveURL(phaseURL, link)
					if err != nil {
						parseError = fmt.Errorf(
							"resolve match URL: %w",
							err,
						)
						return false // break
					}

					parsedURL, err := url.Parse(matchURL)
					if err != nil {
						parseError = fmt.Errorf(
							"parse match URL: %w",
							err,
						)
						return false // break
					}

					parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
					if len(parts) != 3 || parts[0] != "jogo" || parts[1] == "detalhes" {
						parseError = fmt.Errorf(
							"invalid match URL format: %s",
							matchURL,
						)
						return false // break
					}

					matchID := parts[2]

					if _, exists := seen[matchID]; exists {
						return true // continue
					}

					teams := matchSelection.Find(".time-confronto")
					if teams.Length() != 2 {
						parseError = fmt.Errorf(
							"invalid number of teams in HTML for match URL: %s",
							matchURL,
						)
						return false // break
					}

					homeID := strings.TrimSpace(teams.First().Text())
					awayID := strings.TrimSpace(teams.Last().Text())

					if homeID == "" || awayID == "" {
						parseError = fmt.Errorf(
							"missing team IDs in HTML for match URL: %s",
							matchURL,
						)
						return false // break
					}

					venue, date, matchTime := parseMatchDateTime(
						strings.TrimSpace(matchSelection.Find(".data-hora-card-jogo").Text()),
					)

					homeScore, awayScore := parseMatchScore(
						matchSelection.Find(".hora-confronto-card"),
					)

					status := canonical.MatchPending
					if homeScore != nil && awayScore != nil {
						status = canonical.MatchCompleted
					}

					matches = append(
						matches,
						canonical.Match{
							ExternalID:         matchID,
							URL:                matchURL,
							PhaseExternalID:    phase.ExternalID,
							PhaseName:          phase.Name,
							Round:              round,
							DateText:           date,
							TimeText:           matchTime,
							Venue:              venue,
							HomeClubExternalID: homeID,
							AwayClubExternalID: awayID,
							HomeScore:          homeScore,
							AwayScore:          awayScore,
							Status:             status,
						},
					)

					seen[matchID] = struct{}{}

					return true // continue
				},
			)

			return parseError == nil // continue if no error
		},
	)

	if parseError != nil {
		return nil, parseError
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf(
			"no matches found in HTML for URL: %s",
			phaseURL,
		)
	}

	return matches, nil
}

func ParseDocuments(matchURL string, rawHTML string) ([]canonical.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf(
			"parse match HTML: %w",
			err,
		)
	}

	documents := make([]canonical.Document, 0)

	section := doc.Find("h6").FilterFunction(
		func(_ int, s *goquery.Selection) bool {
			return strings.EqualFold(strings.TrimSpace(s.Text()), "DOCUMENTOS")
		},
	).First()
	if section.Length() == 0 {
		return nil, fmt.Errorf(
			"documents section not found in HTML for URL: %s",
			matchURL,
		)
	}

	var parseError error
	seen := make(map[string]struct{})

	section.Parent().Find(".row.mb-3").EachWithBreak(
		func(_ int, s *goquery.Selection) bool {
			title := strings.TrimSpace(s.Find("p").First().Text())

			documentType, recognized := parseDocumentType(title)
			if !recognized {
				return true
			}

			href, exists := s.Find("a[href]").First().Attr("href")
			if !exists || strings.TrimSpace(href) == "" {
				parseError = fmt.Errorf(
					"missing document link in HTML for URL: %s",
					matchURL,
				)
				return true // break
			}

			documentURL, err := resolveURL(matchURL, href)
			if err != nil {
				parseError = fmt.Errorf(
					"resolve document URL: %w",
					err,
				)
				return false
			}

			parsedURL, err := url.Parse(documentURL)
			if err != nil {
				parseError = fmt.Errorf(
					"parse document URL: %w",
					err,
				)
				return false
			}

			if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
				parseError = fmt.Errorf(
					"invalid document URL scheme: %s",
					documentURL,
				)
				return true
			}

			if !strings.HasSuffix(strings.ToLower(parsedURL.Path), ".pdf") {
				parseError = fmt.Errorf(
					"invalid document URL format (not a PDF): %s",
					documentURL,
				)
				return true
			}

			parsedURL.Fragment = ""
			documentURL = parsedURL.String()

			if _, exists := seen[documentURL]; exists {
				return true
			}

			seen[documentURL] = struct{}{}

			documents = append(
				documents,
				canonical.Document{
					Type:   documentType,
					URL:    documentURL,
					Status: canonical.DocumentDiscovered,
				},
			)

			return true
		},
	)

	if parseError != nil {
		return nil, parseError
	}

	return documents, nil
}

func parseMatchDateTime(content string) (venue, date, matchTime string) {
	content = strings.TrimSpace(content)

	indexes := matchDatePattern.FindStringSubmatchIndex(
		content,
	)

	if indexes == nil {
		return content, "", ""
	}

	venue = strings.TrimSpace(content[:indexes[0]])
	date = content[indexes[2]:indexes[3]]
	matchTime = content[indexes[4]:indexes[5]]

	return venue, date, matchTime
}

func parseMatchScore(selection *goquery.Selection) (*int, *int) {
	spans := selection.Children().Filter("span")

	if spans.Length() < 2 {
		return nil, nil
	}

	home, homeErr := strconv.Atoi(
		strings.TrimSpace(spans.First().Text()),
	)

	away, awayErr := strconv.Atoi(
		strings.TrimSpace(spans.Last().Text()),
	)

	if homeErr != nil || awayErr != nil {
		return nil, nil
	}

	if home < 0 || away < 0 {
		return nil, nil
	}

	return &home, &away
}

func parseDocumentType(title string) (canonical.DocumentType, bool) {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "súmula", "sumula":
		return canonical.DocumentMatchSummary, true

	case "boletim financeiro":
		return canonical.DocumentFinancialBulletin, true

	case "relatório de jogo", "relatorio de jogo":
		return canonical.DocumentMatchReport, true

	case "plano de ação", "plano de acao":
		return canonical.DocumentActionPlan, true

	default:
		return "", false
	}
}