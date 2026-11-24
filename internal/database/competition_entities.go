package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/sofi-io/sofi-dashboard/internal/competition"
)

func (r *CompetitionRepository) saveClubsAndPhases(
	ctx context.Context,
	tx pgx.Tx,
	result *competition.Result,
	seasonID string,
) error {
	source := strings.ToLower(strings.TrimSpace(result.Competition.Source))

	// Persist clubs.
	seenClubs := make(map[string]bool)

	for _, club := range result.Clubs {
		externalID := strings.TrimSpace(club.ExternalID)
		name := strings.TrimSpace(club.Name)

		if externalID == "" || name == "" {
			return fmt.Errorf("club external ID and name are required")
		}

		if seenClubs[externalID] {
			return fmt.Errorf("duplicate club external ID: %s", externalID)
		}
		seenClubs[externalID] = true

		if err := r.saveClub(ctx, tx, source, club); err != nil {
			return fmt.Errorf(
				"save club %s (%s): %w",
				name,
				externalID,
				err,
			)
		}
	}

	// Persist phases.
	seenPhases := make(map[string]bool)

	for _, phase := range result.Phases {
		externalID := strings.TrimSpace(phase.ExternalID)
		name := strings.TrimSpace(phase.Name)

		if externalID == "" || name == "" {
			return fmt.Errorf("phase external ID and name are required")
		}

		if seenPhases[externalID] {
			return fmt.Errorf("duplicate phase external ID: %s", externalID)
		}
		seenPhases[externalID] = true

		if err := r.savePhase(
			ctx,
			tx,
			source,
			seasonID,
			phase,
			result.Competition.URL,
		); err != nil {
			return fmt.Errorf(
				"save phase %s (%s): %w",
				name,
				externalID,
				err,
			)
		}
	}

	return nil
}

func (r *CompetitionRepository) saveClub(
	ctx context.Context,
	tx pgx.Tx,
	source string,
	club competition.Club,
) error {
	externalID := strings.TrimSpace(club.ExternalID)
	name := strings.TrimSpace(club.Name)
	sourceURL := strings.TrimSpace(club.URL)
	logoURL := strings.TrimSpace(club.LogoURL)

	lockKey := fmt.Sprintf("club:%s:%s", source, externalID)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lockKey,
	); err != nil {
		return fmt.Errorf("lock club identity: %w", err)
	}

	var clubID string

	err := tx.QueryRow(
		ctx,
		`
		SELECT club_id
		FROM club_sources
		WHERE source_key = $1
		  AND external_id = $2
		`,
		source,
		externalID,
	).Scan(&clubID)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("resolve club identity: %w", err)
	}

	if err == pgx.ErrNoRows {
		slug := makeSlug(source, externalID)
		canonicalName := normalizeCanonicalName(name)

		err = tx.QueryRow(
			ctx,
			`
			INSERT INTO clubs (
				name,
				canonical_name,
				slug,
				logo_url
			)
			VALUES ($1, $2, $3, NULLIF($4, ''))
			RETURNING id
			`,
			name,
			canonicalName,
			slug,
			logoURL,
		).Scan(&clubID)

		if err != nil {
			return fmt.Errorf("insert club: %w", err)
		}

		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO club_sources (
				club_id,
				source_key,
				external_id,
				source_url
			)
			VALUES ($1, $2, $3, NULLIF($4, ''))
			`,
			clubID,
			source,
			externalID,
			sourceURL,
		)

		if err != nil {
			return fmt.Errorf("insert club source identity: %w", err)
		}

		return nil
	}

	// Preserve the canonical UUID and update source metadata.
	_, err = tx.Exec(
		ctx,
		`
		UPDATE clubs
		SET name = $1,
		    canonical_name = $2,
		    logo_url = NULLIF($3, ''),
		    updated_at = now()
		WHERE id = $4
		`,
		name,
		normalizeCanonicalName(name),
		club.LogoURL,
		clubID,
	)

	if err != nil {
		return fmt.Errorf("update club: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
		UPDATE club_sources
		SET source_url = NULLIF($1, ''),
		    discovered_at = now()
		WHERE source_key = $2
		  AND external_id = $3
		`,
		sourceURL,
		source,
		externalID,
	)

	if err != nil {
		return fmt.Errorf("update club source identity: %w", err)
	}

	return nil
}

func (r *CompetitionRepository) savePhase(
	ctx context.Context,
	tx pgx.Tx,
	source string,
	seasonID string,
	phase competition.Phase,
	sourceURL string,
) error {
	externalID := strings.TrimSpace(phase.ExternalID)
	name := strings.TrimSpace(phase.Name)

	lockKey := fmt.Sprintf("phase:%s:%s", source, externalID)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lockKey,
	); err != nil {
		return fmt.Errorf("lock phase identity: %w", err)
	}

	var phaseID string
	var existingSeasonID string

	err := tx.QueryRow(
		ctx,
		`
		SELECT ps.phase_id, p.season_id
		FROM phase_sources ps
		JOIN phases p ON p.id = ps.phase_id
		WHERE ps.source_key = $1
		  AND ps.external_id = $2
		`,
		source,
		externalID,
	).Scan(&phaseID, &existingSeasonID)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("resolve phase identity: %w", err)
	}

	if err == nil {
		if existingSeasonID != seasonID {
			return fmt.Errorf(
				"phase identity %s belongs to another season",
				externalID,
			)
		}

		_, err = tx.Exec(
			ctx,
			`
			UPDATE phases
			SET name = $1,
			    phase_order = $2
			WHERE id = $3
			`,
			name,
			phase.Order,
			phaseID,
		)

		if err != nil {
			return fmt.Errorf("update phase: %w", err)
		}

		_, err = tx.Exec(
			ctx,
			`
			UPDATE phase_sources
			SET source_url = $1,
			    discovered_at = now()
			WHERE source_key = $2
			  AND external_id = $3
			`,
			sourceURL,
			source,
			externalID,
		)

		if err != nil {
			return fmt.Errorf("update phase source identity: %w", err)
		}

		return nil
	}

	// Reuse a phase with the same name in this season if it exists.
	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO phases (
			season_id,
			name,
			phase_order
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (season_id, name)
		DO UPDATE SET
			phase_order = COALESCE(
				EXCLUDED.phase_order,
				phases.phase_order
			)
		RETURNING id
		`,
		seasonID,
		name,
		phase.Order,
	).Scan(&phaseID)

	if err != nil {
		return fmt.Errorf("insert or resolve phase: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO phase_sources (
			phase_id,
			source_key,
			external_id,
			source_url
		)
		VALUES ($1, $2, $3, $4)
		`,
		phaseID,
		source,
		externalID,
		sourceURL,
	)

	if err != nil {
		return fmt.Errorf("insert phase source identity: %w", err)
	}

	return nil
}

func normalizeCanonicalName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}
