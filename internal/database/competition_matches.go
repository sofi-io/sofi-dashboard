package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/sofi-io/sofi-dashboard/internal/competition"
)

func (r *CompetitionRepository) saveMatchesAndDocuments(
	ctx context.Context,
	tx pgx.Tx,
	result *competition.Result,
	seasonID string,
) error {
	source := strings.ToLower(strings.TrimSpace(result.Competition.Source))

	seen := make(map[string]bool)

	for _, match := range result.Matches {
		externalID := strings.TrimSpace(match.ExternalID)

		if externalID == "" {
			return fmt.Errorf("match external ID is required")
		}

		if seen[externalID] {
			return fmt.Errorf("duplicate match external ID: %s", externalID)
		}
		seen[externalID] = true

		if err := r.saveMatch(
			ctx,
			tx,
			source,
			seasonID,
			match,
		); err != nil {
			return fmt.Errorf(
				"save match %s: %w",
				externalID,
				err,
			)
		}

		if err := r.saveMatchDocuments(
			ctx,
			tx,
			source,
			externalID,
			match.Documents,
		); err != nil {
			return fmt.Errorf(
				"save documents for match %s: %w",
				externalID,
				err,
			)
		}
	}

	return nil
}

func (r *CompetitionRepository) saveMatch(
	ctx context.Context,
	tx pgx.Tx,
	source string,
	seasonID string,
	match competition.Match,
) error {
	externalID := strings.TrimSpace(match.ExternalID)

	lockKey := fmt.Sprintf("match:%s:%s", source, externalID)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		lockKey,
	); err != nil {
		return fmt.Errorf("lock match identity: %w", err)
	}

	var matchID string

	err := tx.QueryRow(
		ctx,
		`
		SELECT match_id
		FROM match_sources
		WHERE source_key = $1
		  AND external_id = $2
		`,
		source,
		externalID,
	).Scan(&matchID)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("resolve match identity: %w", err)
	}

	var homeClubID, awayClubID, phaseID *string

	if match.HomeClubExternalID != "" {
		id, err := resolveClubID(
			ctx, tx, source, match.HomeClubExternalID,
		)
		if err != nil {
			return fmt.Errorf("resolve home club: %w", err)
		}
		homeClubID = &id
	}

	if match.AwayClubExternalID != "" {
		id, err := resolveClubID(
			ctx, tx, source, match.AwayClubExternalID,
		)
		if err != nil {
			return fmt.Errorf("resolve away club: %w", err)
		}
		awayClubID = &id
	}

	if homeClubID != nil && awayClubID != nil &&
		*homeClubID == *awayClubID {
		return fmt.Errorf("home and away clubs cannot be identical")
	}

	if match.PhaseExternalID != "" {
		id, err := resolvePhaseID(
			ctx, tx, source, match.PhaseExternalID, seasonID,
		)
		if err != nil {
			return fmt.Errorf("resolve phase: %w", err)
		}
		phaseID = &id
	}

	if homeClubID == nil || awayClubID == nil {
		return fmt.Errorf(
			"match %s has incomplete club identities",
			externalID,
		)
	}

	if err == pgx.ErrNoRows {
		err = tx.QueryRow(
			ctx,
			`
			INSERT INTO matches (
				season_id,
				home_club_id,
				away_club_id,
				phase_id,
				round_number,
				round_name,
				venue,
				status,
				source_date_text,
				source_time_text,
				kickoff_at
			)
			VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, NULL
			)
			RETURNING id
			`,
			seasonID,
			*homeClubID,
			*awayClubID,
			phaseID,
			nullableInt(match.Round),
			roundName(match.Round),
			nullableString(match.Venue),
			string(match.Status),
			nullableString(match.DateText),
			nullableString(match.TimeText),
		).Scan(&matchID)

		if err != nil {
			return fmt.Errorf("insert match: %w", err)
		}

	} else {
		err = tx.QueryRow(
			ctx,
			`
			UPDATE matches
			SET season_id = $1,
			    home_club_id = $2,
			    away_club_id = $3,
			    phase_id = COALESCE($4, phase_id),
			    round_number = $5,
			    round_name = $6,
			    venue = $7,
			    status = $8,
			    source_date_text = $9,
			    source_time_text = $10,
			    updated_at = now()
			WHERE id = $11
			RETURNING id
			`,
			seasonID,
			*homeClubID,
			*awayClubID,
			phaseID,
			nullableInt(match.Round),
			roundName(match.Round),
			nullableString(match.Venue),
			string(match.Status),
			nullableString(match.DateText),
			nullableString(match.TimeText),
			matchID,
		).Scan(&matchID)

		if err != nil {
			return fmt.Errorf("update match: %w", err)
		}
	}

	payload, err := json.Marshal(match)
	if err != nil {
		return fmt.Errorf("marshal match payload: %w", err)
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO match_sources (
			match_id,
			source_key,
			external_id,
			source_url,
			source_payload
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (source_key, external_id)
		DO UPDATE SET
			match_id = EXCLUDED.match_id,
			source_url = EXCLUDED.source_url,
			source_payload = EXCLUDED.source_payload,
			discovered_at = now()
		`,
		matchID,
		source,
		externalID,
		match.URL,
		payload,
	)

	if err != nil {
		return fmt.Errorf("save match source identity: %w", err)
	}

	return nil
}

func resolveClubID(
	ctx context.Context,
	tx pgx.Tx,
	source string,
	externalID string,
) (string, error) {
	var id string

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
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf(
			"club %s from source %s not found: %w",
			externalID,
			source,
			err,
		)
	}

	return id, nil
}

func resolvePhaseID(
	ctx context.Context,
	tx pgx.Tx,
	source string,
	externalID string,
	seasonID string,
) (string, error) {
	var id string

	err := tx.QueryRow(
		ctx,
		`
		SELECT ps.phase_id
		FROM phase_sources ps
		JOIN phases p ON p.id = ps.phase_id
		WHERE ps.source_key = $1
		  AND ps.external_id = $2
		  AND p.season_id = $3
		`,
		source,
		externalID,
		seasonID,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf(
			"phase %s from source %s not found in season: %w",
			externalID,
			source,
			err,
		)
	}

	return id, nil
}

func (r *CompetitionRepository) saveMatchDocuments(
	ctx context.Context,
	tx pgx.Tx,
	source string,
	matchExternalID string,
	documents []competition.Document,
) error {
	var matchID string

	err := tx.QueryRow(
		ctx,
		`
		SELECT match_id
		FROM match_sources
		WHERE source_key = $1
		  AND external_id = $2
		`,
		source,
		matchExternalID,
	).Scan(&matchID)

	if err != nil {
		return fmt.Errorf("resolve match for documents: %w", err)
	}

	for _, document := range documents {
		documentURL := strings.TrimSpace(document.URL)
		documentType := strings.TrimSpace(string(document.Type))
		status := strings.TrimSpace(string(document.Status))

		if documentType == "" {
			return fmt.Errorf("document type is required")
		}

		if status == "" {
			return fmt.Errorf("document status is required")
		}

		if documentURL == "" {
			_, err := tx.Exec(
				ctx,
				`
				INSERT INTO documents (
					match_id,
					source_key,
					source_url,
					document_type,
					status
				)
				VALUES ($1, $2, NULL, $3, $4)
				ON CONFLICT (
					match_id,
					source_key,
					document_type
				)
				WHERE source_url IS NULL
				DO UPDATE SET
					status = CASE
						WHEN documents.status IN (
							'downloaded', 'processed', 'submitted'
						) THEN documents.status
						ELSE EXCLUDED.status
					END
				`,
				matchID,
				source,
				documentType,
				status,
			)

			if err != nil {
				return fmt.Errorf("save pending document: %w", err)
			}

			continue
		}

		_, err := tx.Exec(
			ctx,
			`
			INSERT INTO documents (
				match_id,
				source_key,
				source_url,
				document_type,
				status
			)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (source_key, source_url)
			DO UPDATE SET
				match_id = EXCLUDED.match_id,
				document_type = EXCLUDED.document_type,
				status = CASE
					WHEN documents.status IN (
						'downloaded', 'processed', 'submitted'
					) THEN documents.status
					ELSE EXCLUDED.status
				END
			`,
			matchID,
			source,
			documentURL,
			documentType,
			status,
		)

		if err != nil {
			return fmt.Errorf("save discovered document: %w", err)
		}
	}

	return nil
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func roundName(round int) any {
	if round <= 0 {
		return nil
	}
	return fmt.Sprintf("Rodada %d", round)
}
