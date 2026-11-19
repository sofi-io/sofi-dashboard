package database

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type financialTestFixture struct {
	db            *pgxpool.Pool
	competitionID string
	seasonID      string
	clubIDs       map[string]string
	matchIDs      []string
}

func newFinancialTestFixture(t *testing.T) *financialTestFixture {
	t.Helper()

	db := newIntegrationDB(t)
	ctx := context.Background()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	fixture := &financialTestFixture{
		db:       db,
		clubIDs:  make(map[string]string),
		matchIDs: make([]string, 0),
	}

	err := db.QueryRow(ctx, `
		INSERT INTO competitions (name, slug, canonical_key)
		VALUES ($1, $2, $3)
		RETURNING id
	`, "Financial Test Competition", "financial-test-"+suffix, "financial-test-"+suffix).
		Scan(&fixture.competitionID)
	if err != nil {
		t.Fatalf("create competition: %v", err)
	}

	err = db.QueryRow(ctx, `
		INSERT INTO seasons (competition_id, name)
		VALUES ($1, '2026')
		RETURNING id
	`, fixture.competitionID).Scan(&fixture.seasonID)
	if err != nil {
		t.Fatalf("create season: %v", err)
	}

	for _, name := range []string{"Alpha FC", "Beta FC", "Gamma FC", "Delta FC"} {
		var id string

		err := db.QueryRow(ctx, `
			INSERT INTO clubs (name, canonical_name, slug)
			VALUES ($1, $2, $3)
			RETURNING id
		`, name, name, fmt.Sprintf("financial-%s-%s",
			slugForTest(name), suffix)).Scan(&id)

		if err != nil {
			t.Fatalf("create club %s: %v", name, err)
		}

		fixture.clubIDs[name] = id
	}

	t.Cleanup(func() {
		cleanupFinancialFixture(t, fixture)
	})

	return fixture
}

func slugForTest(value string) string {
	switch value {
	case "Alpha FC":
		return "alpha"
	case "Beta FC":
		return "beta"
	case "Gamma FC":
		return "gamma"
	case "Delta FC":
		return "delta"
	default:
		return "unknown"
	}
}

func cleanupFinancialFixture(
	t *testing.T,
	fixture *financialTestFixture,
) {
	t.Helper()

	ctx := context.Background()
	db := fixture.db

	_, err := db.Exec(ctx, `
		DELETE FROM extractions e
		USING documents d
		WHERE e.document_id = d.id
		  AND d.match_id IN (
			SELECT id FROM matches WHERE season_id = $1
		  )
	`, fixture.seasonID)
	if err != nil {
		t.Errorf("cleanup extractions: %v", err)
	}

	_, err = db.Exec(ctx, `
		DELETE FROM documents
		WHERE match_id IN (
			SELECT id FROM matches WHERE season_id = $1
		)
	`, fixture.seasonID)
	if err != nil {
		t.Errorf("cleanup documents: %v", err)
	}

	_, err = db.Exec(ctx, `
		DELETE FROM matches WHERE season_id = $1
	`, fixture.seasonID)
	if err != nil {
		t.Errorf("cleanup matches: %v", err)
	}

	_, err = db.Exec(ctx, `
		DELETE FROM seasons WHERE id = $1
	`, fixture.seasonID)
	if err != nil {
		t.Errorf("cleanup season: %v", err)
	}

	for _, id := range fixture.clubIDs {
		_, err = db.Exec(ctx, `
			DELETE FROM clubs WHERE id = $1
		`, id)
		if err != nil {
			t.Errorf("cleanup club: %v", err)
		}
	}

	_, err = db.Exec(ctx, `
		DELETE FROM competitions WHERE id = $1
	`, fixture.competitionID)
	if err != nil {
		t.Errorf("cleanup competition: %v", err)
	}
}

func (f *financialTestFixture) createMatch(
	t *testing.T,
	home string,
	away string,
) string {
	t.Helper()

	var id string

	err := f.db.QueryRow(context.Background(), `
		INSERT INTO matches (
			season_id,
			home_club_id,
			away_club_id
		)
		VALUES ($1, $2, $3)
		RETURNING id
	`,
		f.seasonID,
		f.clubIDs[home],
		f.clubIDs[away],
	).Scan(&id)

	if err != nil {
		t.Fatalf("create match %s x %s: %v", home, away, err)
	}

	f.matchIDs = append(f.matchIDs, id)

	return id
}

func (f *financialTestFixture) createExtraction(
	t *testing.T,
	matchID string,
	grossRevenue float64,
	operatingExpenses float64,
	fees float64,
	taxes float64,
	otherDeductions float64,
	netResult float64,
	totalAttendance int,
	payingAttendance int,
	externalID string,
) {
	t.Helper()

	ctx := context.Background()

	var documentID string

	err := f.db.QueryRow(ctx, `
		INSERT INTO documents (
			match_id,
			source_key,
			source_url,
			document_type,
			status
		)
		VALUES ($1, 'financial-test', $2, 'financial_bulletin', 'downloaded')
		RETURNING id
	`, matchID, "https://example.test/"+externalID+".pdf").
		Scan(&documentID)

	if err != nil {
		t.Fatalf("create financial document: %v", err)
	}

	result := map[string]interface{}{
		"result": map[string]interface{}{
			"match": map[string]interface{}{
				"home_team": "",
				"away_team": "",
			},
			"financial": map[string]interface{}{
				"gross_revenue":             grossRevenue,
				"operating_expenses":        operatingExpenses,
				"fees":                      fees,
				"taxes":                     taxes,
				"other_deductions":          otherDeductions,
				"methodological_net_result": netResult,
			},
			"attendance": map[string]interface{}{
				"total":      totalAttendance,
				"paying":     payingAttendance,
				"non_paying": totalAttendance - payingAttendance,
			},
		},
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal financial result: %v", err)
	}

	_, err = f.db.Exec(ctx, `
		INSERT INTO extractions (
			document_id,
			provider,
			external_extraction_id,
			status,
			raw_result,
			completed_at
		)
		VALUES ($1, 'sofi-api', $2, 'completed', $3, now())
	`, documentID, externalID, raw)

	if err != nil {
		t.Fatalf("create extraction: %v", err)
	}
}

func TestGetOverview_RepositoryNotInitialized(t *testing.T) {
	var repository *CompetitionFinancialRepository

	_, err := repository.GetOverview(
		context.Background(),
		"00000000-0000-0000-0000-000000000001",
	)

	if err == nil {
		t.Fatal("expected error for uninitialized repository")
	}
}

func TestGetOverview_EmptyCompetition(t *testing.T) {
	fixture := newFinancialTestFixture(t)

	repository := NewCompetitionFinancialRepository(fixture.db)

	overview, err := repository.GetOverview(
		context.Background(),
		fixture.competitionID,
	)
	if err != nil {
		t.Fatalf("GetOverview(): %v", err)
	}

	if overview.TotalMatches != 0 {
		t.Errorf("total matches = %d, want 0", overview.TotalMatches)
	}

	if overview.MatchesWithExtraction != 0 {
		t.Errorf("matches with extraction = %d, want 0",
			overview.MatchesWithExtraction)
	}

	if overview.GrossRevenue != 0 {
		t.Errorf("gross revenue = %f, want 0", overview.GrossRevenue)
	}

	if len(overview.ClubRankings) != 0 {
		t.Errorf("club rankings = %d, want 0", len(overview.ClubRankings))
	}
}

func TestGetOverview_FinancialAggregation(t *testing.T) {
	fixture := newFinancialTestFixture(t)

	match1 := fixture.createMatch(t, "Alpha FC", "Beta FC")
	match2 := fixture.createMatch(t, "Alpha FC", "Gamma FC")
	match3 := fixture.createMatch(t, "Alpha FC", "Delta FC")

	fixture.createExtraction(
		t, match1,
		100000, 40000, 8000, 5000, 0, 47000,
		10000, 9000, "match-1",
	)

	fixture.createExtraction(
		t, match2,
		200000, 80000, 16000, 10000, 0, 94000,
		15000, 14000, "match-2",
	)

	_ = match3

	// Third match intentionally has no completed extraction.

	repository := NewCompetitionFinancialRepository(fixture.db)

	overview, err := repository.GetOverview(
		context.Background(),
		fixture.competitionID,
	)
	if err != nil {
		t.Fatalf("GetOverview(): %v", err)
	}

	if overview.MatchesWithExtraction != 2 {
		t.Errorf("matches with extraction = %d, want 2",
			overview.MatchesWithExtraction)
	}

	if overview.MatchesWithoutExtraction != 1 {
		t.Errorf("matches without extraction = %d, want 1",
			overview.MatchesWithoutExtraction)
	}

	assertFinancialFloat(t, "gross revenue", overview.GrossRevenue, 300000)
	assertFinancialFloat(t, "operating expenses", overview.OperatingExpenses, 120000)
	assertFinancialFloat(t, "fees", overview.Fees, 24000)
	assertFinancialFloat(t, "taxes", overview.Taxes, 15000)
	assertFinancialFloat(t, "other deductions", overview.OtherDeductions, 0)
	assertFinancialFloat(t, "methodological net result",
		overview.MethodologicalNetResult, 141000)

	if overview.TotalAttendance != 25000 {
		t.Errorf("total attendance = %d, want 25000",
			overview.TotalAttendance)
	}

	if overview.PayingAttendance != 23000 {
		t.Errorf("paying attendance = %d, want 23000",
			overview.PayingAttendance)
	}

	if overview.NonPayingAttendance != 2000 {
		t.Errorf("non-paying attendance = %d, want 2000",
			overview.NonPayingAttendance)
	}

	assertFinancialFloat(t, "average attendance",
		overview.AverageAttendance, 12500)
}

func TestGetOverview_HighestRevenueAndAttendance(t *testing.T) {
	fixture := newFinancialTestFixture(t)

	match1 := fixture.createMatch(t, "Alpha FC", "Beta FC")
	match2 := fixture.createMatch(t, "Beta FC", "Gamma FC")

	fixture.createExtraction(
		t, match1,
		100000, 40000, 8000, 5000, 0, 47000,
		10000, 9000, "highlight-1",
	)

	fixture.createExtraction(
		t, match2,
		250000, 80000, 20000, 12500, 0, 137500,
		20000, 18000, "highlight-2",
	)

	repository := NewCompetitionFinancialRepository(fixture.db)

	overview, err := repository.GetOverview(
		context.Background(),
		fixture.competitionID,
	)
	if err != nil {
		t.Fatalf("GetOverview(): %v", err)
	}

	if overview.HighestRevenueMatch == nil {
		t.Fatal("highest revenue match is nil")
	}

	if overview.HighestRevenueMatch.MatchID != match2 {
		t.Errorf("highest revenue match = %s, want %s",
			overview.HighestRevenueMatch.MatchID, match2)
	}

	if overview.HighestAttendanceMatch == nil {
		t.Fatal("highest attendance match is nil")
	}

	if overview.HighestAttendanceMatch.MatchID != match2 {
		t.Errorf("highest attendance match = %s, want %s",
			overview.HighestAttendanceMatch.MatchID, match2)
	}
}

func TestGetOverview_ClubRankings(t *testing.T) {
	fixture := newFinancialTestFixture(t)

	match1 := fixture.createMatch(t, "Alpha FC", "Beta FC")
	match2 := fixture.createMatch(t, "Alpha FC", "Gamma FC")
	match3 := fixture.createMatch(t, "Beta FC", "Gamma FC")

	fixture.createExtraction(
		t, match1,
		100000, 40000, 8000, 5000, 0, 47000,
		10000, 9000, "ranking-1",
	)

	fixture.createExtraction(
		t, match2,
		200000, 80000, 16000, 10000, 0, 94000,
		15000, 14000, "ranking-2",
	)

	fixture.createExtraction(
		t, match3,
		50000, 20000, 4000, 2500, 0, 23500,
		5000, 4500, "ranking-3",
	)

	repository := NewCompetitionFinancialRepository(fixture.db)

	overview, err := repository.GetOverview(
		context.Background(),
		fixture.competitionID,
	)
	if err != nil {
		t.Fatalf("GetOverview(): %v", err)
	}

	if len(overview.ClubRankings) != 2 {
		t.Fatalf("club rankings = %d, want 2", len(overview.ClubRankings))
	}

	rankings := make(map[string]ClubFinancialRanking)

	for _, ranking := range overview.ClubRankings {
		rankings[ranking.ClubName] = ranking
	}

	alpha := rankings["Alpha FC"]
	assertFinancialFloat(t, "Alpha revenue", alpha.GrossRevenue, 300000)
	assertFinancialFloat(t, "Alpha net result",
		alpha.MethodologicalNetResult, 141000)
	assertFinancialFloat(t, "Alpha retention margin",
		alpha.RetentionMargin, 0.47)

	beta := rankings["Beta FC"]
	assertFinancialFloat(t, "Beta revenue", beta.GrossRevenue, 50000)
	assertFinancialFloat(t, "Beta net result",
		beta.MethodologicalNetResult, 23500)
	assertFinancialFloat(t, "Beta retention margin",
		beta.RetentionMargin, 0.47)

	if overview.ClubRankings[0].GrossRevenue <
		overview.ClubRankings[1].GrossRevenue {
		t.Error("club rankings are not ordered by gross revenue descending")
	}
}

func assertFinancialFloat(
	t *testing.T,
	name string,
	got float64,
	want float64,
) {
	t.Helper()

	const epsilon = 0.000001

	if got-want > epsilon || want-got > epsilon {
		t.Errorf("%s = %f, want %f", name, got, want)
	}
}
