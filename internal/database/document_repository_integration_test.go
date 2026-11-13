package database

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupDocumentRepositoryTest(
	t *testing.T,
) (*pgxpool.Pool, *DocumentRepository) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not configured; skipping PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	adminConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}

	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}

	if err := adminPool.Ping(ctx); err != nil {
		adminPool.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	schema := fmt.Sprintf("test_documents_%d", time.Now().UnixNano())

	_, err = adminPool.Exec(
		ctx,
		"CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize(),
	)
	if err != nil {
		adminPool.Close()
		t.Fatalf("create isolated schema: %v", err)
	}

	testConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		adminPool.Exec(
			ctx,
			"DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE",
		)
		adminPool.Close()
		t.Fatalf("parse test database config: %v", err)
	}

	if testConfig.ConnConfig.RuntimeParams == nil {
		testConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}

	testConfig.ConnConfig.RuntimeParams["search_path"] = schema

	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		adminPool.Exec(
			ctx,
			"DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE",
		)
		adminPool.Close()
		t.Fatalf("create test connection pool: %v", err)
	}

	_, err = testPool.Exec(ctx, `
		CREATE TABLE documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			match_id UUID,
			source_key TEXT NOT NULL,
			source_url TEXT,
			document_type TEXT NOT NULL DEFAULT 'unknown',
			content_type TEXT,
			size_bytes BIGINT,
			sha256 CHAR(64),
			storage_key TEXT,
			status TEXT NOT NULL DEFAULT 'discovered',
			download_attempts INTEGER NOT NULL DEFAULT 0,
			download_started_at TIMESTAMPTZ,
			download_token UUID,
			last_error TEXT,
			discovered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			downloaded_at TIMESTAMPTZ
		)
	`)
	if err != nil {
		testPool.Close()
		adminPool.Exec(
			ctx,
			"DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE",
		)
		adminPool.Close()
		t.Fatalf("create documents table: %v", err)
	}

	t.Cleanup(func() {
		testPool.Close()

		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()

		_, err := adminPool.Exec(
			cleanupCtx,
			"DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE",
		)
		if err != nil {
			t.Errorf("drop test schema: %v", err)
		}

		adminPool.Close()
	})

	return testPool, NewDocumentRepository(testPool)
}

func insertTestDocument(
	t *testing.T,
	pool *pgxpool.Pool,
	status string,
	attempts int,
	startedAt *time.Time,
	token *string,
) string {
	t.Helper()

	ctx := context.Background()

	var id string

	err := pool.QueryRow(ctx, `
		INSERT INTO documents (
			source_key,
			source_url,
			document_type,
			status,
			download_attempts,
			download_started_at,
			download_token
		)
		VALUES (
			'cbf',
			'https://example.test/document.pdf',
			'financial_bulletin',
			$1,
			$2,
			$3,
			$4
		)
		RETURNING id
	`, status, attempts, startedAt, token).Scan(&id)

	if err != nil {
		t.Fatalf("insert test document: %v", err)
	}

	return id
}

func TestDocumentRepositoryClaimPending(t *testing.T) {
	pool, repository := setupDocumentRepositoryTest(t)

	id := insertTestDocument(
		t, pool, "discovered", 0, nil, nil,
	)

	ctx := context.Background()

	documents, err := repository.ClaimPending(ctx, 5)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("got %d documents, want 1", len(documents))
	}

	doc := documents[0]

	if doc.ID != id {
		t.Fatalf("document ID = %s, want %s", doc.ID, id)
	}

	if doc.Status != "downloading" {
		t.Fatalf("status = %s, want downloading", doc.Status)
	}

	if doc.DownloadAttempts != 1 {
		t.Fatalf("attempts = %d, want 1", doc.DownloadAttempts)
	}

	if doc.DownloadToken == nil || *doc.DownloadToken == "" {
		t.Fatal("download token should be generated")
	}

	if doc.DownloadStartedAt == nil {
		t.Fatal("download_started_at should be populated")
	}

	documents, err = repository.ClaimPending(ctx, 5)
	if err != nil {
		t.Fatalf("second ClaimPending(): %v", err)
	}

	if len(documents) != 0 {
		t.Fatalf("got %d documents, want 0", len(documents))
	}
}

func TestDocumentRepositoryConcurrentClaim(t *testing.T) {
	pool, repository := setupDocumentRepositoryTest(t)

	insertTestDocument(
		t, pool, "discovered", 0, nil, nil,
	)

	const workers = 2

	start := make(chan struct{})
	results := make(chan int, workers)
	errors := make(chan error, workers)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			documents, err := repository.ClaimPending(
				context.Background(),
				1,
			)

			if err != nil {
				errors <- err
				return
			}

			results <- len(documents)
		}()
	}

	close(start)
	wg.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Errorf("concurrent claim failed: %v", err)
	}

	totalClaimed := 0

	for count := range results {
		totalClaimed += count
	}

	if totalClaimed != 1 {
		t.Fatalf(
			"total claimed = %d, want exactly 1",
			totalClaimed,
		)
	}
}

func TestDocumentRepositoryMarkDownloaded(t *testing.T) {
	pool, repository := setupDocumentRepositoryTest(t)

	id := insertTestDocument(
		t, pool, "discovered", 0, nil, nil,
	)

	ctx := context.Background()

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	doc := documents[0]
	token := *doc.DownloadToken

	err = repository.MarkDownloaded(
		ctx,
		id,
		token,
		"application/pdf",
		1024,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"documents/test.pdf",
	)
	if err != nil {
		t.Fatalf("MarkDownloaded(): %v", err)
	}

	persisted, err := repository.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID(): %v", err)
	}

	if persisted.Status != "downloaded" {
		t.Fatalf("status = %s, want downloaded", persisted.Status)
	}

	if persisted.ContentType == nil ||
		*persisted.ContentType != "application/pdf" {
		t.Fatal("content type was not persisted correctly")
	}

	if persisted.SizeBytes == nil || *persisted.SizeBytes != 1024 {
		t.Fatal("size was not persisted correctly")
	}

	if persisted.SHA256 == nil || *persisted.SHA256 != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatal("SHA256 was not persisted correctly")
	}

	if persisted.StorageKey == nil ||
		*persisted.StorageKey != "documents/test.pdf" {
		t.Fatal("storage key was not persisted correctly")
	}

	if persisted.DownloadToken != nil {
		t.Fatal("download token should be cleared")
	}

	if persisted.DownloadedAt == nil {
		t.Fatal("downloaded_at should be populated")
	}
}

func TestDocumentRepositoryMarkFailedRetries(t *testing.T) {
	pool, repository := setupDocumentRepositoryTest(t)

	id := insertTestDocument(
		t, pool, "discovered", 0, nil, nil,
	)

	ctx := context.Background()

	for attempt := 1; attempt <= maxDocumentAttempts; attempt++ {
		documents, err := repository.ClaimPending(ctx, 1)
		if err != nil {
			t.Fatalf("attempt %d: ClaimPending(): %v", attempt, err)
		}

		if len(documents) != 1 {
			t.Fatalf(
				"attempt %d: expected one claimed document, got %d",
				attempt,
				len(documents),
			)
		}

		err = repository.MarkFailed(
			ctx,
			id,
			*documents[0].DownloadToken,
			fmt.Errorf("simulated download failure"),
		)
		if err != nil {
			t.Fatalf("attempt %d: MarkFailed(): %v", attempt, err)
		}

		persisted, err := repository.GetByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}

		expectedStatus := "discovered"
		if attempt == maxDocumentAttempts {
			expectedStatus = "failed"
		}

		if persisted.Status != expectedStatus {
			t.Fatalf(
				"attempt %d: status = %s, want %s",
				attempt,
				persisted.Status,
				expectedStatus,
			)
		}

		if persisted.DownloadAttempts != attempt {
			t.Fatalf(
				"attempts = %d, want %d",
				persisted.DownloadAttempts,
				attempt,
			)
		}
	}

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(documents) != 0 {
		t.Fatal("document exceeding retry limit must not be claimed")
	}
}

func TestDocumentRepositoryRecoverStale(t *testing.T) {
	pool, repository := setupDocumentRepositoryTest(t)

	oldTime := time.Now().Add(-10 * time.Minute)
	oldToken := "00000000-0000-0000-0000-000000000001"

	id := insertTestDocument(
		t,
		pool,
		"downloading",
		1,
		&oldTime,
		&oldToken,
	)

	ctx := context.Background()

	recovered, err := repository.RecoverStale(
		ctx,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("RecoverStale(): %v", err)
	}

	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	persisted, err := repository.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if persisted.Status != "discovered" {
		t.Fatalf("status = %s, want discovered", persisted.Status)
	}

	if persisted.DownloadToken != nil {
		t.Fatal("stale download token should be cleared")
	}

	if persisted.DownloadStartedAt != nil {
		t.Fatal("download_started_at should be cleared")
	}
}

func TestDocumentRepositoryRejectsStaleToken(t *testing.T) {
	pool, repository := setupDocumentRepositoryTest(t)

	oldTime := time.Now().Add(-10 * time.Minute)
	oldToken := "00000000-0000-0000-0000-000000000001"

	id := insertTestDocument(
		t,
		pool,
		"downloading",
		1,
		&oldTime,
		&oldToken,
	)

	ctx := context.Background()

	_, err := repository.RecoverStale(ctx, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(documents) != 1 {
		t.Fatal("expected document to be reclaimed")
	}

	newToken := *documents[0].DownloadToken

	if newToken == oldToken {
		t.Fatal("new worker must receive a different token")
	}

	err = repository.MarkDownloaded(
		ctx,
		id,
		oldToken,
		"application/pdf",
		1024,
		"abcdef123456",
		"documents/stale.pdf",
	)

	if err == nil {
		t.Fatal("old worker must not update document")
	}

	err = repository.MarkDownloaded(
		ctx,
		id,
		newToken,
		"application/pdf",
		1024,
		"abcdef123456",
		"documents/current.pdf",
	)

	if err != nil {
		t.Fatalf("current worker should update document: %v", err)
	}
}
