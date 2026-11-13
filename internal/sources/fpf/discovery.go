package fpf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

type DiscoveryWarning struct {
	Stage   string `json:"stage"`
	MatchID string `json:"match_id,omitempty"`
	PhaseID string `json:"phase_id,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error"`
}

type DiscoveryResult struct {
	Competition *Competition       `json:"competition"`
	Matches     []Match            `json:"matches"`
	Warnings    []DiscoveryWarning `json:"warnings,omitempty"`
	Clubs       []Club             `json:"clubs"`
}

type phaseResponse struct {
	HTML  string `json:"html"`
	Round int    `json:"rodada"`
}

type matchDocumentResult struct {
	index     int
	documents []Document
	warning   *DiscoveryWarning
}

// DiscoverCompetition retrieves all phases, matches and
// available documents from an FPF competition.
func (c *Client) DiscoverCompetition(
	ctx context.Context,
	competitionURL string,
	options canonical.DiscoveryOptions,
) (*DiscoveryResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	competitionHTML, err := c.GetHTML(
		ctx,
		competitionURL,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("fetch FPF competition: %w", err)
	}

	competition, err := ParseCompetition(
		competitionURL,
		competitionHTML,
	)
	if err != nil {
		return nil, fmt.Errorf("parse FPF competition: %w", err)
	}

	result := &DiscoveryResult{
		Competition: competition,
		Matches:     make([]Match, 0),
		Clubs:       make([]Club, 0),
	}

	phaseEndpoint, err := resolveEndpoint(
		competitionURL,
		"jogos_ajax_ver2.php",
	)
	if err != nil {
		return nil, err
	}

	seenMatches := make(map[string]bool)
	seenClubs := make(map[string]bool)

	classificationEndpoint, err := resolveEndpoint(
		competitionURL,
		"classificacao_ajax_ver2.php",
	)
	if err != nil {
		return nil, err
	}

	for _, phase := range competition.Phases {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		initialRound := phase.InitialRound
		if initialRound < 1 {
			initialRound = 1
		}

		form := url.Values{
			"idcampeonato": {competition.ExternalID},
			"idfase":       {phase.ExternalID},
			"rodada":       {fmt.Sprintf("%d", initialRound)},
		}

		responseBody, err := c.PostForm(
			ctx,
			phaseEndpoint,
			competitionURL,
			form,
		)
		if err != nil {
			result.Warnings = append(
				result.Warnings,
				DiscoveryWarning{
					Stage:   "phase_matches",
					PhaseID: phase.ExternalID,
					URL:     phaseEndpoint,
					Error:   err.Error(),
				},
			)
			continue
		}

		var response phaseResponse

		if err := json.Unmarshal(responseBody, &response); err != nil {
			result.Warnings = append(
				result.Warnings,
				DiscoveryWarning{
					Stage:   "phase_matches",
					PhaseID: phase.ExternalID,
					URL:     phaseEndpoint,
					Error:   fmt.Sprintf("decode phase response: %v", err),
				},
			)
			continue
		}

		classificationForm := url.Values{
			"idcampeonato": {competition.ExternalID},
			"idfase":       {phase.ExternalID},
		}

		classificationBody, err := c.PostForm(
			ctx,
			classificationEndpoint,
			competitionURL,
			classificationForm,
		)

		if err != nil {
			result.Warnings = append(result.Warnings, DiscoveryWarning{
				Stage:   "phase_clubs",
				PhaseID: phase.ExternalID,
				URL:     classificationEndpoint,
				Error:   err.Error(),
			})
		} else {
			clubs, err := ParseClubs(string(classificationBody))

			if err != nil {
				result.Warnings = append(result.Warnings, DiscoveryWarning{
					Stage:   "phase_clubs",
					PhaseID: phase.ExternalID,
					URL:     classificationEndpoint,
					Error:   err.Error(),
				})
			} else {
				for _, club := range clubs {
					if seenClubs[club.ExternalID] {
						continue
					}

					seenClubs[club.ExternalID] = true
					result.Clubs = append(result.Clubs, club)
				}
			}
		}

		matches, err := ParseMatches(
			competitionURL,
			competition.ExternalID,
			phase.ExternalID,
			phase.Name,
			response.HTML,
		)
		if err != nil {
			result.Warnings = append(
				result.Warnings,
				DiscoveryWarning{
					Stage:   "phase_matches",
					PhaseID: phase.ExternalID,
					URL:     phaseEndpoint,
					Error:   err.Error(),
				},
			)
			continue
		}

		for _, match := range matches {
			if seenMatches[match.ExternalID] {
				continue
			}

			seenMatches[match.ExternalID] = true
			result.Matches = append(result.Matches, match)
		}
	}

	if err := ctx.Err(); err != nil {
		return result, err
	}

	concurrency := options.MaxConcurrentMatches
	if concurrency <= 0 {
		concurrency = 3
	}
	if concurrency > 5 {
		concurrency = 5
	}
	if concurrency > len(result.Matches) {
		concurrency = len(result.Matches)
	}

	if concurrency > 0 {
		matches, warnings := discoverMatchDocuments(
			ctx,
			c,
			result.Matches,
			concurrency,
		)

		result.Matches = matches
		result.Warnings = append(result.Warnings, warnings...)
	}

	sort.Slice(result.Matches, func(i, j int) bool {
		return result.Matches[i].ExternalID < result.Matches[j].ExternalID
	})

	sort.Slice(result.Warnings, func(i, j int) bool {
		if result.Warnings[i].Stage != result.Warnings[j].Stage {
			return result.Warnings[i].Stage < result.Warnings[j].Stage
		}
		if result.Warnings[i].PhaseID != result.Warnings[j].PhaseID {
			return result.Warnings[i].PhaseID < result.Warnings[j].PhaseID
		}
		return result.Warnings[i].MatchID < result.Warnings[j].MatchID
	})

	return result, nil
}

func discoverMatchDocuments(
	ctx context.Context,
	client *Client,
	matches []Match,
	concurrency int,
) ([]Match, []DiscoveryWarning) {
	if len(matches) == 0 {
		return matches, nil
	}

	jobs := make(chan int)
	results := make(chan matchDocumentResult, len(matches))

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for index := range jobs {
				match := matches[index]

				documents, err := client.DiscoverMatchDocuments(
					ctx,
					match,
				)

				if err != nil {
					results <- matchDocumentResult{
						index: index,
						warning: &DiscoveryWarning{
							Stage:   "match_documents",
							MatchID: match.ExternalID,
							URL:     match.URL,
							Error:   err.Error(),
						},
					}
					continue
				}

				results <- matchDocumentResult{
					index:     index,
					documents: documents,
				}
			}
		}()
	}

	go func() {
		defer close(jobs)

		for index := range matches {
			select {
			case jobs <- index:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	output := make([]Match, len(matches))
	copy(output, matches)

	var warnings []DiscoveryWarning

	for item := range results {
		if item.warning != nil {
			warnings = append(warnings, *item.warning)
			continue
		}

		output[item.index].Documents = item.documents
	}

	if err := ctx.Err(); err != nil {
		for i := range output {
			if output[i].Documents == nil {
				warnings = append(warnings, DiscoveryWarning{
					Stage:   "match_documents",
					MatchID: output[i].ExternalID,
					URL:     output[i].URL,
					Error:   err.Error(),
				})
			}
		}
	}

	return output, warnings
}

// DiscoverMatchDocuments loads the match page, invokes its
// infoJogo AJAX endpoint and parses the published documents.
func (c *Client) DiscoverMatchDocuments(
	ctx context.Context,
	match Match,
) ([]Document, error) {
	detailHTML, err := c.GetHTML(
		ctx,
		match.URL,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("fetch match page: %w", err)
	}

	if !strings.Contains(detailHTML, `id="infoJogo"`) {
		return nil, fmt.Errorf("match page does not contain infoJogo")
	}

	endpoint, err := resolveEndpoint(
		match.URL,
		"infoJogo.php",
	)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"idjogo": {match.ExternalID},
	}

	response, err := c.PostForm(
		ctx,
		endpoint,
		match.URL,
		form,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch match information: %w", err)
	}

	documents, err := ParseMatchDocuments(
		match.URL,
		string(response),
	)
	if err != nil {
		return nil, fmt.Errorf("parse match documents: %w", err)
	}

	hasFinancialBulletin := false

	for _, document := range documents {
		if document.Type == DocumentFinancialBulletin {
			hasFinancialBulletin = true
			break
		}
	}

	if !hasFinancialBulletin {
		documents = append(documents, Document{
			Type:   DocumentFinancialBulletin,
			Status: DocumentPending,
		})
	}

	return documents, nil
}

func resolveEndpoint(baseURL, endpoint string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	reference, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}

	return base.ResolveReference(reference).String(), nil
}
