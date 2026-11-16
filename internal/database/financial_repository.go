package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

	TotalAttendance     int     `json:"total_attendance"`
	PayingAttendance    int     `json:"paying_attendance"`
	NonPayingAttendance int     `json:"non_paying_attendance"`
	AverageAttendance   float64 `json:"average_attendance"`

	HighestRevenueMatch    *MatchFinancialHighlight `json:"highest_revenue_match"`
	HighestAttendanceMatch *MatchFinancialHighlight `json:"highest_attendance_match"`

	ClubRankings []ClubFinancialRanking `json:"club_rankings"`
}

type MatchFinancialHighlight struct {
	MatchID      string  `json:"match_id"`
	HomeTeam     string  `json:"home_team"`
	AwayTeam     string  `json:"away_team"`
	GrossRevenue float64 `json:"gross_revenue"`
	Attendance   int     `json:"attendance"`
}

type ClubFinancialRanking struct {
	ClubID                  string  `json:"club_id"`
	ClubName                string  `json:"club_name"`
	GrossRevenue            float64 `json:"gross_revenue"`
	MethodologicalNetResult float64 `json:"methodological_net_result"`
	RetentionMargin         float64 `json:"retention_margin"`
}

type financialExtractionResult struct {
	Match struct {
		HomeTeam string `json:"home_team"`
		AwayTeam string `json:"away_team"`
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
	MatchID  string
	HomeID   string
	HomeName string
	AwayName string
	Result   []byte
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
) (CompetitionFinancialOverview, error) {
	if r == nil || r.db == nil {
		return CompetitionFinancialOverview{},
			fmt.Errorf("financial repository is not initialized")
	}

	const query = `
		WITH competition_matches AS (
			SELECT
				m.id AS match_id,
				m.home_club_id,
				hc.name AS home_name,
				ac.name AS away_name
			FROM matches m
			JOIN seasons s ON s.id = m.season_id
			JOIN competitions c ON c.id = s.competition_id
			JOIN clubs hc ON hc.id = m.home_club_id
			JOIN clubs ac ON ac.id = m.away_club_id
			WHERE c.id = $1
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
			cm.away_name,
			COALESCE(fe.raw_result::text, '')
		FROM competition_matches cm
		LEFT JOIN completed_financial_extractions fe
			ON fe.match_id = cm.match_id
		ORDER BY cm.match_id
	`

	rows, err := r.db.Query(ctx, query, competitionID)
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
			&row.AwayName,
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
			AwayTeam:     row.AwayName,
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

		club := clubTotals[row.HomeID]
		if club == nil {
			club = &ClubFinancialRanking{
				ClubID:   row.HomeID,
				ClubName: row.HomeName,
			}
			clubTotals[row.HomeID] = club
		}

		club.GrossRevenue += financial.GrossRevenue
		club.MethodologicalNetResult += financial.MethodologicalNetResult
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

	return overview, nil
}
