package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrCompetitionNotFound = errors.New("competition not found")

type CompetitionDetail struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Slug         string              `json:"slug"`
	Organization *OrganizationDetail `json:"organization"`
	Seasons      []SeasonDetail      `json:"seasons"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type OrganizationDetail struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Slug       string  `json:"slug"`
	WebsiteURL *string `json:"website_url,omitempty"`
}

type SeasonDetail struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	StartsOn *time.Time `json:"starts_on"`
	EndsOn   *time.Time `json:"ends_on"`
}

func (r *CompetitionRepository) GetByID(
	ctx context.Context,
	id string,
) (CompetitionDetail, error) {
	const query = `
		SELECT
			c.id::text,
			c.name,
			c.slug,
			o.id::text,
			o.name,
			o.slug,
			o.website_url,
			c.created_at,
			c.updated_at
		FROM competitions c
		LEFT JOIN organizations o
			ON o.id = c.organization_id
		WHERE c.id = $1
	`

	var item CompetitionDetail
	var orgID, orgName, orgSlug, orgWebsite *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.Slug,
		&orgID,
		&orgName,
		&orgSlug,
		&orgWebsite,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return CompetitionDetail{}, ErrCompetitionNotFound
	}

	if err != nil {
		return CompetitionDetail{}, fmt.Errorf(
			"get competition: %w", err,
		)
	}

	if orgID != nil {
		item.Organization = &OrganizationDetail{
			ID:         *orgID,
			Name:       *orgName,
			Slug:       *orgSlug,
			WebsiteURL: orgWebsite,
		}
	}

	item.Seasons = make([]SeasonDetail, 0)

	const seasonsQuery = `
		SELECT
			id::text,
			name,
			starts_on,
			ends_on
		FROM seasons
		WHERE competition_id = $1
		ORDER BY name DESC, id
	`

	rows, err := r.db.Query(ctx, seasonsQuery, id)
	if err != nil {
		return CompetitionDetail{}, fmt.Errorf(
			"list competition seasons: %w", err,
		)
	}
	defer rows.Close()

	for rows.Next() {
		var season SeasonDetail

		if err := rows.Scan(
			&season.ID,
			&season.Name,
			&season.StartsOn,
			&season.EndsOn,
		); err != nil {
			return CompetitionDetail{}, fmt.Errorf(
				"scan competition season: %w", err,
			)
		}

		item.Seasons = append(item.Seasons, season)
	}

	if err := rows.Err(); err != nil {
		return CompetitionDetail{}, fmt.Errorf(
			"iterate competition seasons: %w", err,
		)
	}

	return item, nil
}
