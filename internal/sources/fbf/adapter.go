package fbf

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf(
			"FBF adapter client is not initialized",
		)
	}

	content, err := a.client.GetHTML(ctx, competitionURL, "")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch competition page: %w",
			err,
		)
	}

	competition, err := ParseCompetition(competitionURL, content)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse competition page: %w",
			err,
		)
	}

	phases, err := ParsePhases(competitionURL, content)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse competition phases: %w",
			err,
		)
	}

	clubs, matches, err := a.discoverPhases(ctx, competitionURL, phases)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to discover competition clubs and matches: %w",
			err,
		)
	}

	if err := a.discoverDocuments(ctx, matches); err != nil {
		return nil, fmt.Errorf(
			"failed to discover match documents: %w",
			err,
		)
	}

	return &canonical.Result{
		Competition: competition,
		Clubs:       clubs,
		Phases:      phases,
		Matches:     matches,
	}, nil
}

func (a *Adapter) discoverPhases(ctx context.Context, competitionURL string, phases []canonical.Phase) ([]canonical.Club, []canonical.Match, error) {
	clubs := make([]canonical.Club, 0)
	matches := make([]canonical.Match, 0)

	seenClubs := make(map[string]struct{})
	seenMatches := make(map[string]struct{})

	for _, phase := range phases {
		phaseURL := fmt.Sprintf(
			"%s/%s",
			competitionURL,
			phase.ExternalID,
		)

		content, err := a.client.GetHTML(
			ctx,
			phaseURL,
			competitionURL,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"fetch phase %s: %w",
				phase.ExternalID,
				err,
			)
		}

		phaseClubs, err := ParseClubs(
			phaseURL,
			content,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"parse clubs from phase %s: %w",
				phase.ExternalID,
				err,
			)
		}

		phaseMatches, err := ParseMatches(
			phaseURL,
			content,
			phase,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"parse matches from phase %s: %w",
				phase.ExternalID,
				err,
			)
		}

		for _, club := range phaseClubs {
			if _, exists := seenClubs[club.ExternalID]; exists {
				continue
			}

			seenClubs[club.ExternalID] = struct{}{}
			clubs = append(clubs, club)
		}

		for _, match := range phaseMatches {
			if _, exists := seenMatches[match.ExternalID]; exists {
				continue
			}

			seenMatches[match.ExternalID] = struct{}{}
			matches = append(matches, match)
		}
	}

	return clubs, matches, nil
}

func (a *Adapter) discoverDocuments(ctx context.Context, matches []canonical.Match) error {
	for i := range matches {
		if matches[i].Status != canonical.MatchCompleted {
			continue
		}

		content, err := a.client.GetHTML(
			ctx,
			matches[i].URL,
			"",
		)
		if err != nil {
			return fmt.Errorf(
				"fetch match documents for %s: %w",
				matches[i].ExternalID,
				err,
			)
		}

		documents, err := ParseDocuments(matches[i].URL, content)
		if err != nil {
			return fmt.Errorf(
				"parse match documents for %s: %w",
				matches[i].ExternalID,
				err,
			)
		}

		matches[i].Documents = documents
	}
	return nil
}
