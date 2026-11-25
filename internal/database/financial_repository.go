package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sofi-io/sofi-dashboard/internal/competition"
)

var ErrSeasonNotFound = errors.New("season not found in competition")
var ErrClubNotFound = errors.New("club not found")

type TeamFinancialOverview struct {
	TotalMatches             int `json:"total_matches"`
	MatchesWithExtraction    int `json:"matches_with_extraction"`
	MatchesWithoutExtraction int `json:"matches_without_extraction"`

	GrossRevenue            float64 `json:"gross_revenue"`
	OperatingExpenses       float64 `json:"operating_expenses"`
	Fees                    float64 `json:"fees"`
	Taxes                   float64 `json:"taxes"`
	OtherDeductions         float64 `json:"other_deductions"`
	MethodologicalNetResult float64 `json:"methodological_net_result"`
	RetentionMargin         float64 `json:"retention_margin"`

	TotalAttendance     int     `json:"total_attendance"`
	PayingAttendance    int     `json:"paying_attendance"`
	NonPayingAttendance int     `json:"non_paying_attendance"`
	AverageAttendance   float64 `json:"average_attendance"`

	DeficitMatches         int                      `json:"deficit_matches"`
	HighestRevenueMatch    *MatchFinancialHighlight `json:"highest_revenue_match"`
	HighestAttendanceMatch *MatchFinancialHighlight `json:"highest_attendance_match"`
	LowestAttendanceMatch  *MatchFinancialHighlight `json:"lowest_attendance_match"`
}

type CompetitionFinancialOverview struct {
	TotalMatches             int `json:"total_matches"`
	MatchesWithExtraction    int `json:"matches_with_extraction"`
	MatchesWithoutExtraction int `json:"matches_without_extraction"`

	GrossRevenue            float64 `json:"gross_revenue"`
	OperatingExpenses       float64 `json:"operating_expenses"`
	Fees                    float64 `json:"fees"`
	Taxes                   float64 `json:"taxes"`
	OtherDeductions         float64 `json:"other_deductions"`
	MethodologicalNetResult float64 `json:"methodological_net_result"`
	RetentionMargin         float64 `json:"retention_margin"`

	TotalAttendance     int     `json:"total_attendance"`
	PayingAttendance    int     `json:"paying_attendance"`
	NonPayingAttendance int     `json:"non_paying_attendance"`
	AverageAttendance   float64 `json:"average_attendance"`

	DeficitMatches int `json:"deficit_matches"`

	HighestRevenueMatch    *MatchFinancialHighlight `json:"highest_revenue_match"`
	HighestAttendanceMatch *MatchFinancialHighlight `json:"highest_attendance_match"`
	LowestAttendanceMatch  *MatchFinancialHighlight `json:"lowest_attendance_match"`

	TopAttendanceClub *ClubFinancialRanking `json:"top_attendance_club"`
	TopRetentionClub  *ClubFinancialRanking `json:"top_retention_club"`

	ClubRankings []ClubFinancialRanking `json:"club_rankings"`
}

type MatchFinancialHighlight struct {
	MatchID      string  `json:"match_id"`
	HomeTeam     string  `json:"home_team"`
	HomeLogo     string  `json:"home_logo"`
	AwayTeam     string  `json:"away_team"`
	AwayLogo     string  `json:"away_logo"`
	GrossRevenue float64 `json:"gross_revenue"`
	Attendance   int     `json:"attendance"`
}

type ClubFinancialRanking struct {
	ClubID                  string  `json:"club_id"`
	ClubName                string  `json:"club_name"`
	ClubeLogoURL            string  `json:"club_logo_url,omitempty"`
	GrossRevenue            float64 `json:"gross_revenue"`
	MethodologicalNetResult float64 `json:"methodological_net_result"`
	RetentionMargin         float64 `json:"retention_margin"`

	HomeMatches       int     `json:"home_matches"`
	TotalAttendance   int     `json:"total_attendance"`
	AverageAttendance float64 `json:"average_attendance"`
}

type financialExtractionResult struct {
	Match struct {
		HomeTeam string `json:"home_team"`
		HomeLogo string `json:"home_logo"`
		AwayTeam string `json:"away_team"`
		AwayLogo string `json:"away_logo"`
	} `json:"match"`

	Financial struct {
		GrossRevenue            float64 `json:"gross_revenue"`
		OperatingExpenses       float64 `json:"operating_expenses"`
		Fees                    float64 `json:"fees"`
		Taxes                   float64 `json:"taxes"`
		OtherDeductions         float64 `json:"other_deductions"`
		MethodologicalNetResult float64 `json:"methodological_net_result"`
	} `json:"financial"`

	Attendance struct {
		Total     int `json:"total"`
		Paying    int `json:"paying"`
		NonPaying int `json:"non_paying"`
	} `json:"attendance"`
}

type financialMatchRow struct {
	MatchID     string
	HomeID      string
	HomeName    string
	HomeLogoURL string
	AwayName    string
	AwayLogoURL string
	Result      []byte
}

type CompetitionFinancialRepository struct {
	db *pgxpool.Pool
}

func NewCompetitionFinancialRepository(
	db *pgxpool.Pool,
) *CompetitionFinancialRepository {
	return &CompetitionFinancialRepository{db: db}
}

func (r *CompetitionFinancialRepository) GetOverview(
	ctx context.Context,
	competitionID string,
	seasonID string,
) (CompetitionFinancialOverview, error) {
	if r == nil || r.db == nil {
		return CompetitionFinancialOverview{},
			fmt.Errorf("financial repository is not initialized")
	}

	var competitionExists bool

	err := r.db.QueryRow(ctx, `
    SELECT EXISTS (
        SELECT 1
        FROM competitions
        WHERE id::text = $1
    )
`, competitionID).Scan(&competitionExists)

	if err != nil {
		return CompetitionFinancialOverview{},
			fmt.Errorf("check competition: %w", err)
	}

	if !competitionExists {
		return CompetitionFinancialOverview{}, ErrCompetitionNotFound
	}

	if seasonID != "" {
		var seasonExists bool

		err := r.db.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM seasons
            WHERE id::text = $1
              AND competition_id::text = $2
        )
    `, seasonID, competitionID).Scan(&seasonExists)

		if err != nil {
			return CompetitionFinancialOverview{},
				fmt.Errorf("check season: %w", err)
		}

		if !seasonExists {
			return CompetitionFinancialOverview{}, ErrSeasonNotFound
		}
	}

	const query = `
		WITH competition_matches AS (
			SELECT
				m.id AS match_id,
				m.home_club_id,
				hc.name AS home_name,
				COALESCE(hc.logo_url, '') AS home_logo_url,
				ac.name AS away_name,
				COALESCE(ac.logo_url, '') AS away_logo_url,
				s.id AS season_id
			FROM matches m
			JOIN seasons s ON s.id = m.season_id
			JOIN competitions c ON c.id = s.competition_id
			JOIN clubs hc ON hc.id = m.home_club_id
			JOIN clubs ac ON ac.id = m.away_club_id
			WHERE c.id = $1
				AND ($2 = '' OR s.id::text = $2)
		),
		completed_financial_extractions AS (
			SELECT DISTINCT ON (d.match_id)
				d.match_id,
				e.raw_result
			FROM documents d
			JOIN extractions e ON e.document_id = d.id
			WHERE d.document_type = 'financial_bulletin'
			AND e.status = 'completed'
			AND d.match_id IS NOT NULL
			ORDER BY
				d.match_id,
				e.updated_at DESC,
				e.id DESC
		)
		SELECT
			cm.match_id,
			cm.home_club_id,
			cm.home_name,
			cm.home_logo_url,
			cm.away_name,
			cm.away_logo_url,
			COALESCE(fe.raw_result::text, '')
		FROM competition_matches cm
		LEFT JOIN completed_financial_extractions fe
			ON fe.match_id = cm.match_id
		ORDER BY cm.match_id
	`

	rows, err := r.db.Query(ctx, query, competitionID, seasonID)
	if err != nil {
		return CompetitionFinancialOverview{},
			fmt.Errorf("query competition financial data: %w", err)
	}
	defer rows.Close()

	overview := CompetitionFinancialOverview{
		ClubRankings: make([]ClubFinancialRanking, 0),
	}

	clubTotals := make(map[string]*ClubFinancialRanking)

	for rows.Next() {
		var row financialMatchRow
		var rawResult string

		if err := rows.Scan(
			&row.MatchID,
			&row.HomeID,
			&row.HomeName,
			&row.HomeLogoURL,
			&row.AwayName,
			&row.AwayLogoURL,
			&rawResult,
		); err != nil {
			return CompetitionFinancialOverview{},
				fmt.Errorf("scan financial match: %w", err)
		}

		overview.TotalMatches++

		if rawResult == "" {
			continue
		}

		var envelope struct {
			Result financialExtractionResult `json:"result"`
		}

		if err := json.Unmarshal([]byte(rawResult), &envelope); err != nil {
			return CompetitionFinancialOverview{},
				fmt.Errorf("decode financial result for match %s: %w",
					row.MatchID, err)
		}

		result := envelope.Result

		overview.MatchesWithExtraction++

		financial := result.Financial
		attendance := result.Attendance

		overview.GrossRevenue += financial.GrossRevenue
		overview.OperatingExpenses += financial.OperatingExpenses
		overview.Fees += financial.Fees
		overview.Taxes += financial.Taxes
		overview.OtherDeductions += financial.OtherDeductions
		overview.MethodologicalNetResult += financial.MethodologicalNetResult

		overview.TotalAttendance += attendance.Total
		overview.PayingAttendance += attendance.Paying
		overview.NonPayingAttendance += attendance.NonPaying

		highlight := MatchFinancialHighlight{
			MatchID:      row.MatchID,
			HomeTeam:     row.HomeName,
			HomeLogo:     row.HomeLogoURL,
			AwayTeam:     row.AwayName,
			AwayLogo:     row.AwayLogoURL,
			GrossRevenue: financial.GrossRevenue,
			Attendance:   attendance.Total,
		}

		if overview.HighestRevenueMatch == nil ||
			highlight.GrossRevenue > overview.HighestRevenueMatch.GrossRevenue {
			copy := highlight
			overview.HighestRevenueMatch = &copy
		}

		if overview.HighestAttendanceMatch == nil ||
			highlight.Attendance > overview.HighestAttendanceMatch.Attendance {
			copy := highlight
			overview.HighestAttendanceMatch = &copy
		}

		if overview.LowestAttendanceMatch == nil ||
			highlight.Attendance < overview.LowestAttendanceMatch.Attendance {

			copy := highlight
			overview.LowestAttendanceMatch = &copy
		}

		club := clubTotals[row.HomeID]
		if club == nil {
			club = &ClubFinancialRanking{
				ClubID:       row.HomeID,
				ClubName:     row.HomeName,
				ClubeLogoURL: row.HomeLogoURL,
			}
			clubTotals[row.HomeID] = club
		}

		club.HomeMatches++
		club.TotalAttendance += attendance.Total

		club.GrossRevenue += financial.GrossRevenue
		club.MethodologicalNetResult += financial.MethodologicalNetResult
		if financial.MethodologicalNetResult <= 0 {
			overview.DeficitMatches++
		}

	}

	if err := rows.Err(); err != nil {
		return CompetitionFinancialOverview{},
			fmt.Errorf("iterate financial matches: %w", err)
	}

	overview.MatchesWithoutExtraction =
		overview.TotalMatches - overview.MatchesWithExtraction

	if overview.MatchesWithExtraction > 0 {
		overview.AverageAttendance =
			float64(overview.TotalAttendance) /
				float64(overview.MatchesWithExtraction)
	}

	for _, club := range clubTotals {
		if club.GrossRevenue > 0 {
			club.RetentionMargin =
				club.MethodologicalNetResult / club.GrossRevenue
		}

		if club.HomeMatches > 0 {
			club.AverageAttendance =
				float64(club.TotalAttendance) /
					float64(club.HomeMatches)
		}

		overview.ClubRankings = append(
			overview.ClubRankings,
			*club,
		)
	}

	sort.Slice(overview.ClubRankings, func(i, j int) bool {
		a := overview.ClubRankings[i]
		b := overview.ClubRankings[j]

		if a.GrossRevenue != b.GrossRevenue {
			return a.GrossRevenue > b.GrossRevenue
		}

		return a.ClubName < b.ClubName
	})

	for i := range overview.ClubRankings {
		club := &overview.ClubRankings[i]

		if overview.TopAttendanceClub == nil ||
			club.AverageAttendance >
				overview.TopAttendanceClub.AverageAttendance {

			copy := *club
			overview.TopAttendanceClub = &copy
		}

		if club.GrossRevenue > 0 &&
			(overview.TopRetentionClub == nil ||
				club.RetentionMargin >
					overview.TopRetentionClub.RetentionMargin) {

			copy := *club
			overview.TopRetentionClub = &copy
		}
	}

	if overview.GrossRevenue > 0 {
		overview.RetentionMargin =
			overview.MethodologicalNetResult / overview.GrossRevenue
	}

	return overview, nil
}

func (r *CompetitionFinancialRepository) GetTeamOverview(
	ctx context.Context,
	competitionID string,
	seasonID string,
	clubID string,
) (TeamFinancialOverview, error) {
	if r == nil || r.db == nil {
		return TeamFinancialOverview{},
			fmt.Errorf("financial repository is not initialized")
	}

	var competitionExists bool

	err := r.db.QueryRow(ctx, `
	SELECT EXISTS (
		SELECT 1
		FROM competitions
		WHERE id::text = $1
	)
	`, competitionID).Scan(&competitionExists)
	if err != nil {
		return TeamFinancialOverview{},
			fmt.Errorf("check competition: %w", err)
	}

	if !competitionExists {
		return TeamFinancialOverview{}, ErrCompetitionNotFound
	}

	var seasonExists bool

	err = r.db.QueryRow(ctx, `
	SELECT EXISTS (
		SELECT 1
		FROM seasons
		WHERE id::text = $1
	)
	`, seasonID).Scan(&seasonExists)
	if err != nil {
		return TeamFinancialOverview{},
			fmt.Errorf("check season: %w", err)
	}

	if !seasonExists {
		return TeamFinancialOverview{}, ErrSeasonNotFound
	}

	var clubExists bool

	err = r.db.QueryRow(ctx, `
	SELECT EXISTS (
		SELECT 1
		FROM clubs
		WHERE id::text = $1
	)
	`, clubID).Scan(&clubExists)
	if err != nil {
		return TeamFinancialOverview{},
			fmt.Errorf("check club: %w", err)
	}

	if !clubExists {
		return TeamFinancialOverview{}, ErrClubNotFound
	}

	return TeamFinancialOverview{}, nil
}

func (r *CompetitionFinancialRepository) GetGeneralStats(
	ctx context.Context,
	seasonName string,
) (competition.GeneralStats, error) {
	if r == nil || r.db == nil {
		return competition.GeneralStats{},
			fmt.Errorf("financial repository is not initialized")
	}

	var stats competition.GeneralStats
	var competitionsJSON []byte

	const query = `
        WITH filtered_seasons AS (
            SELECT
                id,
                name,
                competition_id
            FROM seasons
            WHERE ($1 = '' OR name = $1)
        ),

        filtered_matches AS (
            SELECT
                m.id,
                m.season_id,
                m.home_club_id,
                m.away_club_id,
                s.competition_id
            FROM matches m
            JOIN filtered_seasons s
                ON s.id = m.season_id
        ),

        filtered_clubs AS (
            SELECT home_club_id AS club_id
            FROM filtered_matches

            UNION

            SELECT away_club_id AS club_id
            FROM filtered_matches
        ),

        filtered_competitions AS (
            SELECT DISTINCT
                c.id,
                c.name,
                c.slug
            FROM competitions c
            JOIN filtered_seasons s
                ON s.competition_id = c.id
        ),

        completed_financial_extractions AS (
            SELECT DISTINCT ON (d.match_id)
                d.match_id,
                e.raw_result
            FROM documents d
            JOIN extractions e
                ON e.document_id = d.id
            JOIN filtered_matches m
                ON m.id = d.match_id
            WHERE d.document_type = 'financial_bulletin'
              AND e.status = 'completed'
              AND d.match_id IS NOT NULL
            ORDER BY
                d.match_id,
                e.updated_at DESC,
                e.id DESC
        ),

        financial_data AS (
            SELECT
                match_id,

                (raw_result #>>
                    '{result,financial,gross_revenue}'
                )::numeric AS gross_revenue,

                (raw_result #>>
                    '{result,financial,methodological_net_result}'
                )::numeric AS net_revenue,

                (raw_result #>>
                    '{result,attendance,total}'
                )::bigint AS attendance

            FROM completed_financial_extractions
        ),

        financial_summary AS (
            SELECT
                COALESCE(SUM(gross_revenue), 0)
                    AS total_gross_revenue,

                COALESCE(SUM(net_revenue), 0)
                    AS total_revenue,

                COALESCE(SUM(attendance), 0)
                    AS total_attendance,

                COUNT(*) FILTER (
                    WHERE net_revenue < 0
                ) AS total_deficit_matches

            FROM financial_data
        ),

        competition_matches AS (
            SELECT
                m.competition_id,
                m.season_id,
                m.id AS match_id,
                m.home_club_id,
                m.away_club_id,

                fd.gross_revenue,
                fd.net_revenue,
                fd.attendance

            FROM filtered_matches m
            LEFT JOIN financial_data fd
                ON fd.match_id = m.id
        ),

        competition_teams AS (
            SELECT
                competition_id,
                home_club_id AS club_id
            FROM competition_matches

            UNION

            SELECT
                competition_id,
                away_club_id AS club_id
            FROM competition_matches
        ),

        competition_team_counts AS (
            SELECT
                competition_id,
                COUNT(DISTINCT club_id) AS total_teams
            FROM competition_teams
            GROUP BY competition_id
        ),

        competition_seasons AS (
            SELECT
                competition_id,
                jsonb_agg(
                    DISTINCT name ORDER BY name
                ) AS seasons
            FROM filtered_seasons
            GROUP BY competition_id
        ),

        competition_summary AS (
            SELECT
                c.id::text AS id,
                c.name,
                c.slug,

                COUNT(cm.match_id) AS total_matches,

                COALESCE(SUM(cm.attendance), 0)
                    AS total_attendance,

                COALESCE(ct.total_teams, 0)
                    AS total_teams,

                COALESCE(SUM(cm.net_revenue), 0)
                    AS total_revenue,

                COALESCE(SUM(cm.gross_revenue), 0)
                    AS gross_revenue,

                COALESCE(AVG(cm.attendance), 0)
                    AS average_attendance,

                COALESCE(
                    SUM(cm.net_revenue) * 100.0 /
                    NULLIF(SUM(cm.gross_revenue), 0),
                    0
                ) AS retention_margin,

                COALESCE(cs.seasons, '[]'::jsonb)
                    AS seasons

            FROM filtered_competitions c

            LEFT JOIN competition_matches cm
                ON cm.competition_id = c.id

            LEFT JOIN competition_team_counts ct
                ON ct.competition_id = c.id

            LEFT JOIN competition_seasons cs
                ON cs.competition_id = c.id

            GROUP BY
                c.id,
                c.name,
                c.slug,
                ct.total_teams,
                cs.seasons
        )

        SELECT
            (
                SELECT COUNT(DISTINCT club_id)
                FROM filtered_clubs
            ) AS total_teams,

            (
                SELECT COUNT(*)
                FROM filtered_matches
            ) AS total_matches,

            (
                SELECT COUNT(*)
                FROM filtered_competitions
            ) AS total_competitions,

            (
                SELECT COUNT(*)
                FROM filtered_seasons
            ) AS total_seasons,

            fs.total_revenue,
            fs.total_gross_revenue,
            fs.total_attendance,

            COALESCE(
                fs.total_revenue * 100.0 /
                NULLIF(fs.total_gross_revenue, 0),
                0
            ) AS total_retention,

            fs.total_deficit_matches,

            COALESCE(
                (
                    SELECT jsonb_agg(
                        jsonb_build_object(
                            'id', cs.id,
                            'name', cs.name,
                            'slug', cs.slug,
                            'total_matches', cs.total_matches,
                            'total_attendance', cs.total_attendance,
                            'total_teams', cs.total_teams,
                            'total_revenue', cs.total_revenue,
                            'gross_revenue', cs.gross_revenue,
                            'average_attendance', cs.average_attendance,
                            'retention_margin', cs.retention_margin,
                            'seasons', cs.seasons,
                            'matches', '[]'::jsonb
                        )
                        ORDER BY cs.name
                    )
                    FROM competition_summary cs
                ),
                '[]'::jsonb
            ) AS competitions

        FROM financial_summary fs
    `

	err := r.db.QueryRow(ctx, query, seasonName).Scan(
		&stats.TotalTeams,
		&stats.TotalMatches,
		&stats.TotalCompetitions,
		&stats.TotalSeasons,
		&stats.TotalRevenue,
		&stats.TotalGrossRevenue,
		&stats.TotalAttendance,
		&stats.TotalRetention,
		&stats.TotalDeficitMatches,
		&competitionsJSON,
	)

	if err != nil {
		return competition.GeneralStats{},
			fmt.Errorf("query general stats: %w", err)
	}

	if err := json.Unmarshal(
		competitionsJSON,
		&stats.Competitions,
	); err != nil {
		return competition.GeneralStats{},
			fmt.Errorf("decode competition stats: %w", err)
	}

	return stats, nil
}
