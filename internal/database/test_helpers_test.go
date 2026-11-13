package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func resetDocuments(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`TRUNCATE TABLE documents CASCADE`,
	)
	if err != nil {
		t.Fatalf("reset documents: %v", err)
	}
}

func createTestDocument(
	t *testing.T,
	db *pgxpool.Pool,
	sourceKey string,
	sourceURL string,
) {
	t.Helper()

	_, err := db.Exec(
		context.Background(),
		`
		INSERT INTO documents (
			source_key,
			source_url,
			document_type,
			status
		)
		VALUES ($1, $2, 'match_report', 'discovered')
		`,
		sourceKey,
		sourceURL,
	)

	if err != nil {
		t.Fatalf("create test document: %v", err)
	}
}
