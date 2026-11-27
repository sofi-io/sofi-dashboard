package cbf

import (
	"context"
	"fmt"
	"net/url"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
	"github.com/sofi-io/sofi-dashboard/internal/sources/httpclient"
)

var _ canonical.Discoverer = (*Adapter)(nil)

type Adapter struct {
	client *httpclient.Client
}

func NewAdapter(client *httpclient.Client) *Adapter {
	return &Adapter{
		client: client,
	}
}

func (a *Adapter) DiscoverCompetition(
	ctx context.Context,
	competitionURL string,
	options canonical.DiscoveryOptions,
) (*canonical.Result, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("CBF adapter client is not initialized")
	}

	competitionHTML, err := a.client.GetHTML(ctx, competitionURL, "")
	if err != nil {
		return nil, fmt.Errorf("fetch CBF competition: %w", err)
	}

	// PARSING PROCESS
	// +++++++++++++++
	competition, err := ParseCompetition(competitionURL, competitionHTML)
	if err != nil {
		return nil, fmt.Errorf("parse CBF competition: %w", err)
	}

	clubs, err := ParseClubs(competitionURL, competitionHTML)
	if err != nil {
		return nil, fmt.Errorf("parse CBF clubs: %w", err)
	}

	phases, err := ParsePhases(competitionURL, competitionHTML)
	if err != nil {
		return nil, fmt.Errorf("parse CBF phases: %w", err)
	}

	matches, err := a.discoverMatches(ctx, clubs)
	if err != nil {
		return nil, fmt.Errorf("discover CBF matches: %w", err)
	}

	if err := a.discoverMatchDetails(ctx, matches, phases); err != nil {
		return nil, fmt.Errorf("discover CBF match details: %w", err)
	}

	return &canonical.Result{
		Competition: competition,
		Clubs:       clubs,
		Phases:      phases,
		Matches:     matches,
	}, nil
}

func (a *Adapter) discoverMatches(
	ctx context.Context,
	clubs []canonical.Club,
) ([]canonical.Match, error) {
	matches := make([]canonical.Match, 0)
	seen := make(map[string]struct{})

	for _, club := range clubs {
		if club.URL == "" {
			return nil, fmt.Errorf(
				"missing URL for club %s",
				club.ExternalID,
			)
		}

		clubURL, err := url.Parse(club.URL)
		if err != nil {
			return nil, fmt.Errorf(
				"parse club URL: %w",
				err,
			)
		}

		query := clubURL.Query()
		query.Set("tab", "historico-de-partidas")
		clubURL.RawQuery = query.Encode()

		clubHTML, err := a.client.GetHTML(
			ctx,
			clubURL.String(),
			club.URL,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"fetch matches for club %s: %w",
				club.ExternalID,
				err,
			)
		}

		clubMatches, err := ParseMatches(
			clubURL.String(),
			clubHTML,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"parse matches for club %s: %w",
				club.ExternalID,
				err,
			)
		}

		for _, match := range clubMatches {
			if _, exists := seen[match.ExternalID]; exists {
				continue
			}

			seen[match.ExternalID] = struct{}{}
			matches = append(matches, match)
		}
	}

	return matches, nil
}

func (a *Adapter) discoverMatchDetails(
	ctx context.Context,
	matches []canonical.Match,
	phases []canonical.Phase,
) error {
	for i := range matches {
		matchURL, err := url.Parse(matches[i].URL)
		if err != nil {
			return fmt.Errorf(
				"parse match URL: %w",
				err,
			)
		}

		query := matchURL.Query()
		query.Set("view", "documentos")
		matchURL.RawQuery = query.Encode()

		content, err := a.client.GetHTML(
			ctx,
			matchURL.String(),
			matches[i].URL,
		)
		if err != nil {
			return fmt.Errorf(
				"fetch match %s details: %w",
				matches[i].ExternalID,
				err,
			)
		}

		updatedMatch, err := ParseMatchDetails(
			matches[i],
			phases,
			content,
		)
		if err != nil {
			return fmt.Errorf(
				"parse match %s details: %w",
				matches[i].ExternalID,
				err,
			)
		}

		matches[i] = updatedMatch
	}

	return nil
}
