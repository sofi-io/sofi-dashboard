package cbf

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

const defaultClubConcurrency = 3

type DiscoveryOptions struct {
	MaxConcurrentClubFetches  int
	MaxConcurrentMatchFetches int
}

type DiscoveryWarning struct {
	Stage   string `json:"stage"`
	ClubID  string `json:"club_id,omitempty"`
	MatchID string `json:"match_id,omitempty"`
	URL     string `json:"url"`
	Error   string `json:"error"`
}

type DiscoveryResult struct {
	Competition Competition        `json:"competition"`
	Clubs       []Club             `json:"clubs"`
	Matches     []Match            `json:"matches"`
	Warnings    []DiscoveryWarning `json:"warnings,omitempty"`
}

type clubResult struct {
	matches []Match
	warning *DiscoveryWarning
}

func (c *Client) DiscoverCompetition(
	ctx context.Context,
	competitionURL string,
	options DiscoveryOptions,
) (*DiscoveryResult, error) {

	if err := validateCompetitionURL(competitionURL); err != nil {
		return nil, fmt.Errorf("validate competition URL: %w", err)
	}

	html, err := c.FetchHTML(ctx, competitionURL)
	if err != nil {
		return nil, fmt.Errorf("fetch competition: %w", err)
	}

	clubs, err := ParseClubs(competitionURL, html)
	if err != nil {
		return nil, fmt.Errorf("discover clubs: %w", err)
	}

	if len(clubs) == 0 {
		return nil, fmt.Errorf("no clubs found in competition")
	}

	result := &DiscoveryResult{
		Competition: Competition{
			Name: parseCompetitionName(html),
			URL:  competitionURL,
		},
		Clubs:   clubs,
		Matches: make([]Match, 0),
	}

	concurrency := options.MaxConcurrentClubFetches
	if concurrency <= 0 {
		concurrency = defaultClubConcurrency
	}

	if concurrency > len(clubs) {
		concurrency = len(clubs)
	}

	jobs := make(chan Club)
	results := make(chan clubResult, len(clubs))

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for club := range jobs {
				if ctx.Err() != nil {
					results <- clubResult{
						warning: &DiscoveryWarning{
							ClubID: club.ExternalID,
							URL:    club.URL,
							Error:  ctx.Err().Error(),
						},
					}
					continue
				}

				historyURL, err := clubHistoryURL(club.URL)
				if err != nil {
					results <- clubResult{
						warning: &DiscoveryWarning{
							ClubID: club.ExternalID,
							URL:    club.URL,
							Error:  err.Error(),
						},
					}
					continue
				}

				html, err := c.FetchHTML(ctx, historyURL)
				if err != nil {
					results <- clubResult{
						warning: &DiscoveryWarning{
							Stage:  "club_history",
							ClubID: club.ExternalID,
							URL:    historyURL,
							Error:  err.Error(),
						},
					}
					continue
				}

				matches, err := ParseMatches(club.URL, html)
				if err != nil {
					results <- clubResult{
						warning: &DiscoveryWarning{
							Stage:  "club_history",
							ClubID: club.ExternalID,
							URL:    historyURL,
							Error:  err.Error(),
						},
					}
					continue
				}

				results <- clubResult{
					matches: matches,
				}
			}
		}()
	}

	go func() {
		for _, club := range clubs {
			jobs <- club
		}
		close(jobs)

		wg.Wait()
		close(results)
	}()

	seen := make(map[string]bool)

	for item := range results {
		if item.warning != nil {
			result.Warnings = append(result.Warnings, *item.warning)
			continue
		}

		for _, match := range item.matches {
			if seen[match.ExternalID] {
				continue
			}

			seen[match.ExternalID] = true
			result.Matches = append(result.Matches, match)
		}
	}

	if err := ctx.Err(); err != nil {
		return result, err
	}

	matchConcurrency := options.MaxConcurrentMatchFetches
	if matchConcurrency <= 0 {
		matchConcurrency = 3
	}

	matches, warnings := discoverMatchDocuments(
		ctx,
		c,
		result.Matches,
		matchConcurrency,
	)

	if err := ctx.Err(); err != nil {
		return result, err
	}

	result.Matches = matches
	result.Warnings = append(result.Warnings, warnings...)

	sort.Slice(result.Clubs, func(i, j int) bool {
		return result.Clubs[i].ExternalID < result.Clubs[j].ExternalID
	})

	sort.Slice(result.Matches, func(i, j int) bool {
		return result.Matches[i].ExternalID < result.Matches[j].ExternalID
	})

	sort.Slice(result.Warnings, func(i, j int) bool {
		return result.Warnings[i].ClubID < result.Warnings[j].ClubID
	})

	sort.Slice(result.Warnings, func(i, j int) bool {
		if result.Warnings[i].Stage != result.Warnings[j].Stage {
			return result.Warnings[i].Stage < result.Warnings[j].Stage
		}

		return result.Warnings[i].MatchID < result.Warnings[j].MatchID
	})

	return result, nil
}

func validateCompetitionURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid competition URL: %w", err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("competition URL must use HTTPS")
	}

	host := strings.ToLower(u.Hostname())

	if host != "cbf.com.br" && !strings.HasSuffix(host, ".cbf.com.br") {
		return fmt.Errorf("unsupported source host: %s", host)
	}

	if !strings.HasPrefix(
		u.Path,
		"/futebol-brasileiro/tabelas/",
	) {
		return fmt.Errorf("URL is not a CBF competition page")
	}

	return nil
}

func parseCompetitionName(rawHTML string) string {
	doc, err := parseHTML(rawHTML)
	if err != nil {
		return ""
	}

	if title := strings.TrimSpace(doc.Find("h1").First().Text()); title != "" {
		return title
	}

	if title := strings.TrimSpace(doc.Find("title").First().Text()); title != "" {
		return title
	}

	return ""
}

func clubHistoryURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse club URL: %w", err)
	}

	query := u.Query()
	query.Set("tab", "historico-de-partidas")
	u.RawQuery = query.Encode()

	return u.String(), nil
}
