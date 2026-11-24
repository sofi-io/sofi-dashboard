package fpf

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

type Adapter struct {
	client *Client
}

func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

var canonicalKeySeparator = regexp.MustCompile(`[^a-z0-9]+`)

func makeCanonicalCompetitionKey(source, name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = canonicalKeySeparator.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")

	return strings.ToLower(source) + ":" + normalized
}

var _ canonical.Discoverer = (*Adapter)(nil)

func (a *Adapter) DiscoverCompetition(
	ctx context.Context,
	competitionURL string,
	options canonical.DiscoveryOptions,
) (*canonical.Result, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("FPF adapter client is not initialized")
	}

	discovery, err := a.client.DiscoverCompetition(
		ctx,
		competitionURL,
		options,
	)
	if err != nil {
		return nil, err
	}

	if discovery.Competition == nil {
		return nil, fmt.Errorf("FPF discovery returned nil competition")
	}

	result := &canonical.Result{
		Competition: canonical.Competition{
			Source:     "fpf",
			ExternalID: discovery.Competition.ExternalID,
			CanonicalKey: makeCanonicalCompetitionKey(
				"fpf",
				discovery.Competition.Name,
			),
			Name:   discovery.Competition.Name,
			Season: discovery.Competition.Season,
			URL:    discovery.Competition.URL,
		},
		Clubs:    make([]canonical.Club, 0),
		Phases:   make([]canonical.Phase, 0, len(discovery.Competition.Phases)),
		Matches:  make([]canonical.Match, 0, len(discovery.Matches)),
		Warnings: make([]canonical.Warning, 0, len(discovery.Warnings)),
	}

	for _, club := range discovery.Clubs {
		result.Clubs = append(result.Clubs, canonical.Club{
			ExternalID: club.ExternalID,
			Name:       club.Name,
			LogoURL:    club.LogoURL,
		})
	}

	for i, phase := range discovery.Competition.Phases {
		result.Phases = append(result.Phases, canonical.Phase{
			ExternalID: phase.ExternalID,
			Name:       phase.Name,
			Order:      i + 1,
		})
	}

	for _, match := range discovery.Matches {
		result.Matches = append(result.Matches, canonical.Match{
			ExternalID: match.ExternalID,
			URL:        match.URL,

			PhaseExternalID: match.PhaseID,
			PhaseName:       match.PhaseName,
			Round:           match.Round,

			DateText: match.DateText,
			TimeText: match.TimeText,
			Venue:    match.Venue,

			HomeClubExternalID: match.HomeClubExternalID,
			AwayClubExternalID: match.AwayClubExternalID,

			HomeScore: match.HomeScore,
			AwayScore: match.AwayScore,

			Status:    canonical.MatchStatus(match.Status),
			Documents: convertDocuments(match.Documents),
		})
	}

	for _, warning := range discovery.Warnings {
		entityID := warning.MatchID
		if entityID == "" {
			entityID = warning.PhaseID
		}

		result.Warnings = append(result.Warnings, canonical.Warning{
			Stage:    warning.Stage,
			EntityID: entityID,
			URL:      warning.URL,
			Error:    warning.Error,
		})
	}

	return result, nil
}

func convertDocuments(documents []Document) []canonical.Document {
	result := make([]canonical.Document, 0, len(documents))

	for _, document := range documents {
		result = append(result, canonical.Document{
			Type:   canonical.DocumentType(document.Type),
			URL:    document.URL,
			Status: canonical.DocumentStatus(document.Status),
		})
	}

	return result
}
