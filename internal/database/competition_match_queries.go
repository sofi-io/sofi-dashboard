package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type MatchDocumentItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type CompetitionMatchItem struct {
	ID          string              `json:"id"`
	SeasonID    string              `json:"season_id"`
	Season      string              `json:"season"`
	Phase       *MatchPhaseItem     `json:"phase,omitempty"`
	RoundName   string              `json:"round_name,omitempty"`
	RoundNumber *int                `json:"round_number,omitempty"`
	KickoffAt   *time.Time          `json:"kickoff_at,omitempty"`
	SourceDate  string              `json:"source_date_text,omitempty"`
	SourceTime  string              `json:"source_time_text,omitempty"`
	Venue       string              `json:"venue,omitempty"`
	Status      string              `json:"status"`
	HomeClub    MatchClubItem       `json:"home_club"`
	AwayClub    MatchClubItem       `json:"away_club"`
	Documents   []MatchDocumentItem `json:"documents"`
}

type MatchPhaseItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PhaseOrder int    `json:"phase_order"`
}

type MatchClubItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logo_url,omitempty"`
}

func (r *CompetitionRepository) ListMatches(
	ctx context.Context,
	competitionID string,
	seasonID string,
	limit int,
	offset int,
) ([]CompetitionMatchItem, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("competition repository is not initialized")
	}

	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM competitions
			WHERE id::text = $1
		)
	`, competitionID).Scan(&exists)

	if err != nil {
		return nil, 0, fmt.Errorf("check competition: %w", err)
	}

	if !exists {
		return nil, 0, ErrCompetitionNotFound
	}

	var total int64

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM competitions c
		JOIN seasons s ON s.competition_id = c.id
		JOIN matches m ON m.season_id = s.id
		WHERE c.id::text = $1
		  AND ($2 = '' OR s.id::text = $2)
	`, competitionID, seasonID).Scan(&total)

	if err != nil {
		return nil, 0, fmt.Errorf("count competition matches: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			m.id::text,
			s.id::text,
			s.name,
			p.id::text,
			p.name,
			p.phase_order,
			COALESCE(m.round_name, ''),
			m.round_number,
			m.kickoff_at,
			COALESCE(m.source_date_text, ''),
			COALESCE(m.source_time_text, ''),
			COALESCE(m.venue, ''),
			m.status,
			home.id::text,
			home.name,
			COALESCE(home.logo_url, ''),
			away.id::text,
			away.name,
			COALESCE(away.logo_url, ''),
			COALESCE(
				jsonb_agg(
					jsonb_build_object(
						'id', d.id::text,
						'type', d.document_type,
						'status', d.status
					)
					ORDER BY d.document_type
				) FILTER (WHERE d.id IS NOT NULL),
				'[]'::jsonb
			)
		FROM competitions c
		JOIN seasons s ON s.competition_id = c.id
		JOIN matches m ON m.season_id = s.id
		LEFT JOIN phases p ON p.id = m.phase_id
		JOIN clubs home ON home.id = m.home_club_id
		JOIN clubs away ON away.id = m.away_club_id
		LEFT JOIN documents d ON d.match_id = m.id
		WHERE c.id::text = $1
		  AND ($2 = '' OR s.id::text = $2)
		GROUP BY
			m.id,
			s.id,
			p.id,
			home.id,
			away.id
		ORDER BY
			s.name,
			m.round_number NULLS LAST,
			m.kickoff_at NULLS LAST,
			m.id
		LIMIT $3 OFFSET $4
	`, competitionID, seasonID, limit, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("query competition matches: %w", err)
	}
	defer rows.Close()

	items := make([]CompetitionMatchItem, 0)

	for rows.Next() {
		var item CompetitionMatchItem
		var documentsJSON []byte
		var phaseID, phaseName *string
		var phaseOrder *int

		err := rows.Scan(
			&item.ID,
			&item.SeasonID,
			&item.Season,
			&phaseID,
			&phaseName,
			&phaseOrder,
			&item.RoundName,
			&item.RoundNumber,
			&item.KickoffAt,
			&item.SourceDate,
			&item.SourceTime,
			&item.Venue,
			&item.Status,
			&item.HomeClub.ID,
			&item.HomeClub.Name,
			&item.HomeClub.LogoURL,
			&item.AwayClub.ID,
			&item.AwayClub.Name,
			&item.AwayClub.LogoURL,
			&documentsJSON,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("scan competition match: %w", err)
		}

		if phaseID != nil && phaseName != nil {
			item.Phase = &MatchPhaseItem{
				ID:         *phaseID,
				Name:       *phaseName,
				PhaseOrder: *phaseOrder,
			}
		}

		if err := json.Unmarshal(documentsJSON, &item.Documents); err != nil {
			return nil, 0, fmt.Errorf("decode match documents: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate competition matches: %w", err)
	}

	return items, total, nil
}

func (r *CompetitionRepository) ListTeams(
	ctx context.Context,
	competitionID string,
	seasonID string,
) ([]MatchClubItem, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("competition repository is not initialized")
	}

	var exists bool
	err := r.db.QueryRow(ctx, `
	SELECT EXISTS (
		SELECT 1
		FROM competitions
		WHERE id::text = $1
		)
		`, competitionID).Scan(&exists)

	if err != nil {
		return nil, 0, fmt.Errorf("check competition: %w", err)
	}

	if !exists {
		return nil, 0, ErrCompetitionNotFound
	}

	var total int64

	err = r.db.QueryRow(ctx, `
	SELECT COUNT(DISTINCT c.id)
	FROM competitions comp
	JOIN seasons s ON s.competition_id = comp.id
	JOIN matches m ON m.season_id = s.id
	JOIN clubs c ON c.id = m.home_club_id OR c.id = m.away_club_id
	WHERE comp.id::text = $1
	AND ($2 = '' OR s.id::text = $2)
	`, competitionID, seasonID).Scan(&total)

	if err != nil {
		return nil, 0, fmt.Errorf("count competition teams: %w", err)
	}

	rows, err := r.db.Query(ctx, `
	SELECT DISTINCT c.id::text, c.name, COALESCE(c.logo_url, '')
	FROM competitions comp
	JOIN seasons s ON s.competition_id = comp.id
	JOIN matches m ON m.season_id = s.id
	JOIN clubs c ON c.id = m.home_club_id OR c.id = m.away_club_id
	WHERE comp.id::text = $1
	AND ($2 = '' OR s.id::text = $2)
	ORDER BY c.name
	LIMIT $3 OFFSET $4
	`, competitionID, seasonID, 1000, 0)

	if err != nil {
		return nil, 0, fmt.Errorf("query competition teams: %w", err)
	}
	defer rows.Close()

	items := make([]MatchClubItem, 0)

	for rows.Next() {
		var item MatchClubItem

		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.LogoURL,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("scan competition team: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate competition teams: %w", err)
	}

	return items, total, nil
}
