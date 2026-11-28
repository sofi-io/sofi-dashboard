package fbf

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

const sourceName = "fbf"

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