package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CompetitionQueryRepository struct {
	db *pgxpool.Pool
}

func NewCompetitionQueryRepository(
	db *pgxpool.Pool,
) *CompetitionQueryRepository {
	return &CompetitionQueryRepository{db: db}
}

type CompetitionListItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Organization *string   `json:"organization,omitempty"`
	Sources      []string  `json:"sources"`
	SeasonsCount int64     `json:"seasons_count"`
	CreatedAt    time.Time `json:"created_at"`
}

func (r *CompetitionQueryRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]CompetitionListItem, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf(
			"competition query repository is not initialized",
		)
	}

	if limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("limit must be between 1 and 100")
	}

	if offset < 0 {
		return nil, 0, fmt.Errorf("offset must be non-negative")
	}

	var total int64

	err := r.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM competitions`,
	).Scan(&total)

	if err != nil {
		return nil, 0, fmt.Errorf(
			"count competitions: %w",
			err,
		)
	}

	rows, err := r.db.Query(
		ctx,
		`
		SELECT
			c.id,
			c.name,
			c.slug,
			o.name AS organization,
			COALESCE(
				array_agg(DISTINCT cs.source_key)
					FILTER (WHERE cs.source_key IS NOT NULL),
				ARRAY[]::text[]
			) AS sources,
			COUNT(DISTINCT s.id) AS seasons_count,
			c.created_at
		FROM competitions c
		LEFT JOIN organizations o
			ON o.id = c.organization_id
		LEFT JOIN competition_sources cs
			ON cs.competition_id = c.id
		LEFT JOIN seasons s
			ON s.competition_id = c.id
		GROUP BY
			c.id,
			c.name,
			c.slug,
			o.name,
			c.created_at
		ORDER BY c.created_at DESC, c.name ASC
		LIMIT $1 OFFSET $2
		`,
		limit,
		offset,
	)

	if err != nil {
		return nil, 0, fmt.Errorf(
			"query competitions: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]CompetitionListItem, 0)

	for rows.Next() {
		var item CompetitionListItem

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Organization,
			&item.Sources,
			&item.SeasonsCount,
			&item.CreatedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf(
				"scan competition: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf(
			"iterate competitions: %w",
			err,
		)
	}

	return items, total, nil
}
