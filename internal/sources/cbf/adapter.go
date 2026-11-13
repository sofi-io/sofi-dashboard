package cbf

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

type Adapter struct {
	client *Client
}

func NewAdapter(client *Client) *Adapter {
	return &Adapter{client: client}
}

var _ canonical.Discoverer = (*Adapter)(nil)

func (a *Adapter) DiscoverCompetition(
	ctx context.Context,
	competitionURL string,
	options canonical.DiscoveryOptions,
) (*canonical.Result, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("CBF adapter client is not initialized")
	}

	discovery, err := a.client.DiscoverCompetition(
		ctx,
		competitionURL,
		DiscoveryOptions{
			MaxConcurrentClubFetches:  options.MaxConcurrentClubs,
			MaxConcurrentMatchFetches: options.MaxConcurrentMatches,
		},
	)
	if err != nil {
		return nil, err
	}

	externalID, season, err := cbfCompetitionIdentity(competitionURL)
	if err != nil {
		return nil, err
	}

	result := &canonical.Result{
		Competition: canonical.Competition{
			Source:     "cbf",
			ExternalID: externalID,
			Name:       discovery.Competition.Name,
			Season:     season,
			URL:        competitionURL,
		},
		Clubs:    make([]canonical.Club, 0, len(discovery.Clubs)),
		Phases:   make([]canonical.Phase, 0),
		Matches:  make([]canonical.Match, 0, len(discovery.Matches)),
		Warnings: make([]canonical.Warning, 0, len(discovery.Warnings)),
	}

	for _, club := range discovery.Clubs {
		result.Clubs = append(result.Clubs, canonical.Club{
			ExternalID: club.ExternalID,
			Name:       club.Name,
			URL:        club.URL,
		})
	}

	for _, match := range discovery.Matches {
		result.Matches = append(result.Matches, canonical.Match{
			ExternalID: match.ExternalID,
			URL:        match.URL,
			Status:     canonical.MatchStatus(match.Status),
			Documents:  convertDocuments(match.Documents),
		})
	}

	for _, warning := range discovery.Warnings {
		entityID := warning.MatchID
		if entityID == "" {
			entityID = warning.ClubID
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

func cbfCompetitionIdentity(rawURL string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("parse CBF competition URL: %w", err)
	}

	if u.Host == "" {
		return "", "", fmt.Errorf("CBF competition URL has no host")
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")

	if len(segments) == 0 || segments[0] == "" {
		return "", "", fmt.Errorf("CBF competition URL has no path identifier")
	}

	last := segments[len(segments)-1]

	if len(segments) >= 2 {
		if year, err := strconv.Atoi(last); err == nil &&
			year >= 1900 && year <= 2200 {
			return segments[len(segments)-2], last, nil
		}
	}

	return last, "", nil
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
