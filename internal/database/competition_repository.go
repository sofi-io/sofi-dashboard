package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sofi-io/sofi-dashboard/internal/competition"
)

type CompetitionRepository struct {
	db *pgxpool.Pool
}

func NewCompetitionRepository(
	db *pgxpool.Pool,
) *CompetitionRepository {
	return &CompetitionRepository{db: db}
}

// SaveDiscovery persists the competition and season.
// Further entities will be added in subsequent implementation steps.
func (r *CompetitionRepository) SaveDiscovery(
	ctx context.Context,
	result *competition.Result,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("competition repository is not initialized")
	}

	if err := validateDiscovery(result); err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin discovery transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.saveCompetitionAndSeason(ctx, tx, result); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit discovery transaction: %w", err)
	}

	return nil
}

func validateDiscovery(result *competition.Result) error {
	if result == nil {
		return fmt.Errorf("discovery result is nil")
	}

	c := result.Competition

	if strings.TrimSpace(c.Source) == "" {
		return fmt.Errorf("competition source is required")
	}
	if strings.TrimSpace(c.ExternalID) == "" {
		return fmt.Errorf("competition external ID is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("competition name is required")
	}
	if strings.TrimSpace(c.Season) == "" {
		return fmt.Errorf("competition season is required")
	}
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("competition URL is required")
	}

	return nil
}

func (r *CompetitionRepository) saveCompetitionAndSeason(
	ctx context.Context,
	tx pgx.Tx,
	result *competition.Result,
) error {
	c := result.Competition

	source := strings.ToLower(strings.TrimSpace(c.Source))
	externalID := strings.TrimSpace(c.ExternalID)
	seasonName := strings.TrimSpace(c.Season)

	// Serialize concurrent discoveries of the same source identity.
	lockKey := fmt.Sprintf("competition:%s:%s", source, externalID)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lockKey,
	); err != nil {
		return fmt.Errorf("lock competition identity: %w", err)
	}

	// Resolve existing competition identity.
	var competitionID string

	err := tx.QueryRow(
		ctx,
		`
		SELECT competition_id
		FROM competition_sources
		WHERE source_key = $1
		  AND external_id = $2
		`,
		source,
		externalID,
	).Scan(&competitionID)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("resolve competition identity: %w", err)
	}

	if err == pgx.ErrNoRows {
		// Create canonical competition.
		slug := makeSlug(source, externalID)

		err = tx.QueryRow(
			ctx,
			`
			INSERT INTO competitions (name, slug)
			VALUES ($1, $2)
			RETURNING id
			`,
			c.Name,
			slug,
		).Scan(&competitionID)

		if err != nil {
			return fmt.Errorf("insert competition: %w", err)
		}

		// Register external identity.
		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO competition_sources (
				competition_id,
				source_key,
				external_id,
				source_url
			)
			VALUES ($1, $2, $3, $4)
			`,
			competitionID,
			source,
			externalID,
			c.URL,
		)

		if err != nil {
			return fmt.Errorf("insert competition source identity: %w", err)
		}
	} else {
		// Update metadata without changing canonical identity.
		_, err = tx.Exec(
			ctx,
			`
			UPDATE competitions
			SET name = $1,
			    updated_at = now()
			WHERE id = $2
			`,
			c.Name,
			competitionID,
		)

		if err != nil {
			return fmt.Errorf("update competition: %w", err)
		}

		_, err = tx.Exec(
			ctx,
			`
			UPDATE competition_sources
			SET source_url = $1,
			    discovered_at = now()
			WHERE source_key = $2
			  AND external_id = $3
			`,
			c.URL,
			source,
			externalID,
		)

		if err != nil {
			return fmt.Errorf("update competition source: %w", err)
		}
	}

	// A season identity is scoped to its source competition.
	seasonExternalID := externalID + ":" + seasonName

	var seasonID string
	var seasonCompetitionID string

	err = tx.QueryRow(
		ctx,
		`
		SELECT ss.season_id, s.competition_id
		FROM season_sources ss
		JOIN seasons s ON s.id = ss.season_id
		WHERE ss.source_key = $1
		  AND ss.external_id = $2
		`,
		source,
		seasonExternalID,
	).Scan(&seasonID, &seasonCompetitionID)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("resolve season identity: %w", err)
	}

	if err == nil {
		if seasonCompetitionID != competitionID {
			return fmt.Errorf(
				"season identity %s belongs to another competition",
				seasonExternalID,
			)
		}

		_, err = tx.Exec(
			ctx,
			`
        UPDATE seasons
        SET name = $1
        WHERE id = $2
        `,
			seasonName,
			seasonID,
		)

		if err != nil {
			return fmt.Errorf("update season: %w", err)
		}
	} else {
		// Reuse an existing canonical season if available.
		err = tx.QueryRow(
			ctx,
			`
        INSERT INTO seasons (competition_id, name)
        VALUES ($1, $2)
        ON CONFLICT (competition_id, name)
        DO UPDATE SET name = EXCLUDED.name
        RETURNING id
        `,
			competitionID,
			seasonName,
		).Scan(&seasonID)

		if err != nil {
			return fmt.Errorf("insert or resolve season: %w", err)
		}

		_, err = tx.Exec(
			ctx,
			`
        INSERT INTO season_sources (
            season_id,
            source_key,
            external_id,
            source_url
        )
        VALUES ($1, $2, $3, $4)
        `,
			seasonID,
			source,
			seasonExternalID,
			c.URL,
		)

		if err != nil {
			return fmt.Errorf("insert season source identity: %w", err)
		}
	}

	// 1. Persist canonical clubs and phases.
	if err := r.saveClubsAndPhases(
		ctx,
		tx,
		result,
		seasonID,
	); err != nil {
		return err
	}

	// 2. Persist matches and their documents.
	if err := r.saveMatchesAndDocuments(
		ctx,
		tx,
		result,
		seasonID,
	); err != nil {
		return err
	}

	return nil

}
