package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDocumentRepositoryIntegration(t *testing.T) {
	db := newIntegrationDB(t)

	resetDocuments(t, db)

	createTestDocument(
		t,
		db,
		"cbf",
		"https://example.com/document.pdf",
	)

	var count int

	err := db.QueryRow(
		t.Context(),
		`SELECT COUNT(*) FROM documents`,
	).Scan(&count)

	if err != nil {
		t.Fatalf("count documents: %v", err)
	}

	if count != 1 {
		t.Fatalf("document count = %d, want 1", count)
	}
}

func TestClaimPending(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	// Create eligible documents.
	createTestDocument(
		t, db, "cbf",
		"https://example.com/document-1.pdf",
	)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/document-2.pdf",
	)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/document-3.pdf",
	)

	// Create a document that must not be claimed.
	createTestDocument(
		t, db, "cbf",
		"https://example.com/document-4.pdf",
	)

	// Create a document that exhausted its attempts.
	createTestDocument(
		t, db, "cbf",
		"https://example.com/document-5.pdf",
	)

	// Configure document states.
	_, err := db.Exec(ctx, `
		UPDATE documents
		SET discovered_at = CASE source_url
			WHEN 'https://example.com/document-1.pdf'
				THEN now() - interval '5 minutes'
			WHEN 'https://example.com/document-2.pdf'
				THEN now() - interval '4 minutes'
			WHEN 'https://example.com/document-3.pdf'
				THEN now() - interval '3 minutes'
			ELSE discovered_at
		END
	`)
	if err != nil {
		t.Fatalf("configure discovery dates: %v", err)
	}

	_, err = db.Exec(ctx, `
		UPDATE documents
		SET status = 'downloading'
		WHERE source_url = 'https://example.com/document-4.pdf'
	`)
	if err != nil {
		t.Fatalf("configure downloading document: %v", err)
	}

	_, err = db.Exec(ctx, `
		UPDATE documents
		SET status = 'failed',
		    download_attempts = 5
		WHERE source_url = 'https://example.com/document-5.pdf'
	`)
	if err != nil {
		t.Fatalf("configure exhausted document: %v", err)
	}

	// Claim only two documents.
	documents, err := repository.ClaimPending(ctx, 2)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(documents) != 2 {
		t.Fatalf("claimed %d documents, want 2", len(documents))
	}

	claimedURLs := make(map[string]bool)

	for _, doc := range documents {
		claimedURLs[doc.SourceURL] = true

		if doc.Status != "downloading" {
			t.Errorf(
				"document status = %q, want downloading",
				doc.Status,
			)
		}

		if doc.DownloadToken == nil ||
			*doc.DownloadToken == "" {
			t.Errorf("document %s has no download token", doc.ID)
		}

		if doc.DownloadAttempts != 1 {
			t.Errorf(
				"download attempts = %d, want 1",
				doc.DownloadAttempts,
			)
		}

		if doc.DownloadStartedAt == nil {
			t.Errorf("document %s has no download start time", doc.ID)
		}
	}

	// The two oldest eligible documents should be claimed.
	expectedURLs := []string{
		"https://example.com/document-1.pdf",
		"https://example.com/document-2.pdf",
	}

	for _, expected := range expectedURLs {
		if !claimedURLs[expected] {
			t.Errorf("expected document was not claimed: %s", expected)
		}
	}

	// Verify that the other eligible document remains available.
	var remainingStatus string

	err = db.QueryRow(ctx, `
		SELECT status
		FROM documents
		WHERE source_url = $1
	`, "https://example.com/document-3.pdf").Scan(&remainingStatus)

	if err != nil {
		t.Fatalf("query remaining document: %v", err)
	}

	if remainingStatus != "discovered" {
		t.Errorf(
			"remaining document status = %q, want discovered",
			remainingStatus,
		)
	}

	// Verify that ineligible documents were not modified.
	var downloadingAttempts int

	err = db.QueryRow(ctx, `
		SELECT download_attempts
		FROM documents
		WHERE source_url = $1
	`, "https://example.com/document-4.pdf").Scan(&downloadingAttempts)

	if err != nil {
		t.Fatalf("query downloading document: %v", err)
	}

	if downloadingAttempts != 0 {
		t.Errorf(
			"downloading document attempts = %d, want 0",
			downloadingAttempts,
		)
	}

	var failedStatus string

	err = db.QueryRow(ctx, `
		SELECT status
		FROM documents
		WHERE source_url = $1
	`, "https://example.com/document-5.pdf").Scan(&failedStatus)

	if err != nil {
		t.Fatalf("query exhausted document: %v", err)
	}

	if failedStatus != "failed" {
		t.Errorf("exhausted document status = %q, want failed", failedStatus)
	}
}

func TestClaimPendingRejectsInvalidLimit(t *testing.T) {
	db := newIntegrationDB(t)
	repository := NewDocumentRepository(db)

	tests := []struct {
		name  string
		limit int
	}{
		{"zero", 0},
		{"negative", -1},
		{"above maximum", 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repository.ClaimPending(
				context.Background(),
				tt.limit,
			)

			if err == nil {
				t.Fatalf(
					"ClaimPending(%d) expected error",
					tt.limit,
				)
			}
		})
	}
}

func TestClaimPendingEmptyQueue(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	repository := NewDocumentRepository(db)

	documents, err := repository.ClaimPending(
		context.Background(),
		5,
	)

	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(documents) != 0 {
		t.Fatalf("claimed %d documents, want 0", len(documents))
	}
}

func TestClaimPendingUpdatesLeaseTimestamp(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/lease-test.pdf",
	)

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("claimed %d documents, want 1", len(documents))
	}

	var startedAt time.Time

	err = db.QueryRow(ctx, `
		SELECT download_started_at
		FROM documents
		WHERE id = $1
	`, documents[0].ID).Scan(&startedAt)

	if err != nil {
		t.Fatalf("query lease timestamp: %v", err)
	}

	if time.Since(startedAt) > time.Minute {
		t.Errorf("unexpected lease timestamp: %v", startedAt)
	}
}

func TestClaimPendingConcurrentWorkers(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	const (
		totalDocuments = 100
		workerCount    = 5
		batchSize      = 20
	)

	ctx := context.Background()

	// Populate the queue.
	for i := 0; i < totalDocuments; i++ {
		createTestDocument(
			t,
			db,
			"cbf",
			fmt.Sprintf(
				"https://example.com/document-%03d.pdf",
				i,
			),
		)
	}

	type workerResult struct {
		workerID  int
		documents []Document
		err       error
	}

	results := make(chan workerResult, workerCount)
	start := make(chan struct{})

	var wg sync.WaitGroup

	// Start all workers simultaneously.
	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			<-start

			repository := NewDocumentRepository(db)

			documents, err := repository.ClaimPending(
				ctx,
				batchSize,
			)

			results <- workerResult{
				workerID:  id,
				documents: documents,
				err:       err,
			}
		}(workerID)
	}

	close(start)

	wg.Wait()
	close(results)

	claimed := make(map[string]int)
	totalClaimed := 0

	for result := range results {
		if result.err != nil {
			t.Errorf(
				"worker %d failed: %v",
				result.workerID,
				result.err,
			)
			continue
		}

		if len(result.documents) != batchSize {
			t.Errorf(
				"worker %d claimed %d documents, want %d",
				result.workerID,
				len(result.documents),
				batchSize,
			)
		}

		for _, doc := range result.documents {
			claimed[doc.ID]++
			totalClaimed++

			if doc.Status != "downloading" {
				t.Errorf(
					"document %s status = %q, want downloading",
					doc.ID,
					doc.Status,
				)
			}

			if doc.DownloadToken == nil ||
				*doc.DownloadToken == "" {
				t.Errorf(
					"document %s has no download token",
					doc.ID,
				)
			}
		}
	}

	// Every document must be claimed exactly once.
	for documentID, count := range claimed {
		if count != 1 {
			t.Errorf(
				"document %s claimed %d times",
				documentID,
				count,
			)
		}
	}

	if totalClaimed != totalDocuments {
		t.Errorf(
			"total claimed = %d, want %d",
			totalClaimed,
			totalDocuments,
		)
	}

	if len(claimed) != totalDocuments {
		t.Errorf(
			"unique documents = %d, want %d",
			len(claimed),
			totalDocuments,
		)
	}

	// Verify persisted database state.
	var downloadingCount int

	err := db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM documents
        WHERE status = 'downloading'
    `).Scan(&downloadingCount)

	if err != nil {
		t.Fatalf("count downloading documents: %v", err)
	}

	if downloadingCount != totalDocuments {
		t.Errorf(
			"downloading documents = %d, want %d",
			downloadingCount,
			totalDocuments,
		)
	}

	var tokenCount int

	err = db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM documents
        WHERE status = 'downloading'
          AND download_token IS NOT NULL
    `).Scan(&tokenCount)

	if err != nil {
		t.Fatalf("count download tokens: %v", err)
	}

	if tokenCount != totalDocuments {
		t.Errorf(
			"documents with tokens = %d, want %d",
			tokenCount,
			totalDocuments,
		)
	}
}

func TestMarkDownloaded(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/downloaded.pdf",
	)

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("claimed %d documents, want 1", len(documents))
	}

	doc := documents[0]
	token := *doc.DownloadToken

	const (
		contentType = "application/pdf"
		sizeBytes   = int64(102400)
		sha256      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		storageKey  = "documents/cbf/document.pdf"
	)

	err = repository.MarkDownloaded(
		ctx,
		doc.ID,
		token,
		contentType,
		sizeBytes,
		sha256,
		storageKey,
	)
	if err != nil {
		t.Fatalf("MarkDownloaded(): %v", err)
	}

	persisted, err := repository.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID(): %v", err)
	}

	if persisted.Status != "downloaded" {
		t.Errorf("status = %q, want downloaded", persisted.Status)
	}

	if persisted.ContentType == nil ||
		*persisted.ContentType != contentType {
		t.Errorf("unexpected content type: %v", persisted.ContentType)
	}

	if persisted.SizeBytes == nil ||
		*persisted.SizeBytes != sizeBytes {
		t.Errorf("unexpected size: %v", persisted.SizeBytes)
	}

	if persisted.SHA256 == nil ||
		*persisted.SHA256 != sha256 {
		t.Errorf("unexpected SHA256: %v", persisted.SHA256)
	}

	if persisted.StorageKey == nil ||
		*persisted.StorageKey != storageKey {
		t.Errorf("unexpected storage key: %v", persisted.StorageKey)
	}

	if persisted.DownloadedAt == nil {
		t.Error("downloaded_at should not be nil")
	}

	if persisted.DownloadToken != nil {
		t.Error("download token should be cleared")
	}

	if persisted.DownloadStartedAt != nil {
		t.Error("download_started_at should be cleared")
	}

	if persisted.LastError != nil {
		t.Errorf("last_error should be nil, got %v", persisted.LastError)
	}
}

func TestMarkFailedRetryLimit(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/failing.pdf",
	)

	const maxAttempts = 5

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		documents, err := repository.ClaimPending(ctx, 1)
		if err != nil {
			t.Fatalf("attempt %d: ClaimPending(): %v", attempt, err)
		}

		if len(documents) != 1 {
			t.Fatalf(
				"attempt %d: claimed %d documents, want 1",
				attempt,
				len(documents),
			)
		}

		doc := documents[0]

		err = repository.MarkFailed(
			ctx,
			doc.ID,
			*doc.DownloadToken,
			fmt.Errorf("simulated download failure %d", attempt),
		)
		if err != nil {
			t.Fatalf("attempt %d: MarkFailed(): %v", attempt, err)
		}

		persisted, err := repository.GetByID(ctx, doc.ID)
		if err != nil {
			t.Fatalf("attempt %d: GetByID(): %v", attempt, err)
		}

		if persisted.DownloadAttempts != attempt {
			t.Errorf(
				"attempt count = %d, want %d",
				persisted.DownloadAttempts,
				attempt,
			)
		}

		expectedStatus := "discovered"
		if attempt == maxAttempts {
			expectedStatus = "failed"
		}

		if persisted.Status != expectedStatus {
			t.Errorf(
				"attempt %d: status = %q, want %q",
				attempt,
				persisted.Status,
				expectedStatus,
			)
		}

		if persisted.DownloadToken != nil {
			t.Errorf("attempt %d: token should be cleared", attempt)
		}

		if persisted.DownloadStartedAt != nil {
			t.Errorf("attempt %d: lease should be cleared", attempt)
		}

		if persisted.LastError == nil {
			t.Errorf("attempt %d: last_error should be populated", attempt)
		}
	}

	// An exhausted document must no longer be claimable.
	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimPending after exhaustion: %v", err)
	}

	if len(documents) != 0 {
		t.Fatalf(
			"claimed %d exhausted documents, want 0",
			len(documents),
		)
	}
}

func TestRecoverStale(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/stale.pdf",
	)

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("claimed %d documents, want 1", len(documents))
	}

	doc := documents[0]

	// Simulate an abandoned worker.
	_, err = db.Exec(ctx, `
		UPDATE documents
		SET download_started_at = now() - interval '10 minutes'
		WHERE id = $1
	`, doc.ID)

	if err != nil {
		t.Fatalf("simulate expired lease: %v", err)
	}

	recovered, err := repository.RecoverStale(
		ctx,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("RecoverStale(): %v", err)
	}

	if recovered != 1 {
		t.Fatalf("recovered %d documents, want 1", recovered)
	}

	persisted, err := repository.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID(): %v", err)
	}

	if persisted.Status != "discovered" {
		t.Errorf(
			"status = %q, want discovered",
			persisted.Status,
		)
	}

	if persisted.DownloadToken != nil {
		t.Error("download token should be cleared")
	}

	if persisted.DownloadStartedAt != nil {
		t.Error("download_started_at should be cleared")
	}

	if persisted.LastError == nil ||
		*persisted.LastError != "download lease expired" {
		t.Errorf("unexpected last_error: %v", persisted.LastError)
	}

	// The document must become available again.
	reclaimed, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("reclaim document: %v", err)
	}

	if len(reclaimed) != 1 {
		t.Fatalf("reclaimed %d documents, want 1", len(reclaimed))
	}

	if reclaimed[0].DownloadAttempts != 2 {
		t.Errorf(
			"download attempts = %d, want 2",
			reclaimed[0].DownloadAttempts,
		)
	}
}

func TestRecoverStaleExhaustedDocument(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/exhausted-stale.pdf",
	)

	documents, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	doc := documents[0]

	// Simulate an abandoned worker that already exhausted attempts.
	_, err = db.Exec(ctx, `
		UPDATE documents
		SET download_attempts = 5,
		    download_started_at = now() - interval '10 minutes'
		WHERE id = $1
	`, doc.ID)

	if err != nil {
		t.Fatalf("configure exhausted document: %v", err)
	}

	recovered, err := repository.RecoverStale(
		ctx,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("RecoverStale(): %v", err)
	}

	if recovered != 1 {
		t.Fatalf("recovered %d documents, want 1", recovered)
	}

	persisted, err := repository.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID(): %v", err)
	}

	if persisted.Status != "failed" {
		t.Errorf("status = %q, want failed", persisted.Status)
	}

	if persisted.DownloadToken != nil {
		t.Error("download token should be cleared")
	}
}

func TestDownloadTokenRejectsStaleWorker(t *testing.T) {
	db := newIntegrationDB(t)
	resetDocuments(t, db)

	ctx := context.Background()
	repository := NewDocumentRepository(db)

	createTestDocument(
		t, db, "cbf",
		"https://example.com/token-test.pdf",
	)

	// Worker A claims the document.
	firstClaim, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("first ClaimPending(): %v", err)
	}

	if len(firstClaim) != 1 {
		t.Fatalf("first claim returned %d documents", len(firstClaim))
	}

	oldToken := *firstClaim[0].DownloadToken
	documentID := firstClaim[0].ID

	// Simulate worker A expiration.
	_, err = db.Exec(ctx, `
		UPDATE documents
		SET download_started_at = now() - interval '10 minutes'
		WHERE id = $1
	`, documentID)

	if err != nil {
		t.Fatalf("expire first lease: %v", err)
	}

	_, err = repository.RecoverStale(ctx, time.Minute)
	if err != nil {
		t.Fatalf("RecoverStale(): %v", err)
	}

	// Worker B claims the same document.
	secondClaim, err := repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("second ClaimPending(): %v", err)
	}

	if len(secondClaim) != 1 {
		t.Fatalf("second claim returned %d documents", len(secondClaim))
	}

	newToken := *secondClaim[0].DownloadToken

	if oldToken == newToken {
		t.Fatal("new worker received the previous download token")
	}

	// Worker A must not be allowed to mark the document downloaded.
	err = repository.MarkDownloaded(
		ctx,
		documentID,
		oldToken,
		"application/pdf",
		1024,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"documents/old.pdf",
	)

	if err == nil {
		t.Fatal("stale worker successfully marked document downloaded")
	}

	// Worker A must not be allowed to mark it failed either.
	err = repository.MarkFailed(
		ctx,
		documentID,
		oldToken,
		fmt.Errorf("late failure"),
	)

	if err == nil {
		t.Fatal("stale worker successfully marked document failed")
	}

	// Worker B must still own the document.
	persisted, err := repository.GetByID(ctx, documentID)
	if err != nil {
		t.Fatalf("GetByID(): %v", err)
	}

	if persisted.Status != "downloading" {
		t.Errorf("status = %q, want downloading", persisted.Status)
	}

	if persisted.DownloadToken == nil ||
		*persisted.DownloadToken != newToken {
		t.Error("worker B token was unexpectedly modified")
	}

	// Worker B can complete successfully.
	err = repository.MarkDownloaded(
		ctx,
		documentID,
		newToken,
		"application/pdf",
		1024,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"documents/new.pdf",
	)

	if err != nil {
		t.Fatalf("worker B MarkDownloaded(): %v", err)
	}
}
