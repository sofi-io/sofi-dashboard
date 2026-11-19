package database

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type extractionFixture struct {
	repository *ExtractionRepository
	documentID string
}

func setupExtractionTest(
	t *testing.T,
) (*pgxpool.Pool, extractionFixture) {
	t.Helper()

	db := newIntegrationDB(t)

	resetDocuments(t, db)

	ctx := context.Background()

	var competitionID, seasonID string
	var homeClubID, awayClubID, matchID string

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	err := db.QueryRow(ctx, `
        INSERT INTO competitions (name, slug, canonical_key)
        VALUES ($1, $2, $3)
        RETURNING id
    `, "Test Competition", "test-competition-"+suffix, "test-competition-"+suffix).
		Scan(&competitionID)
	if err != nil {
		t.Fatalf("create competition: %v", err)
	}

	err = db.QueryRow(ctx, `
        INSERT INTO seasons (competition_id, name)
        VALUES ($1, $2)
        RETURNING id
    `, competitionID, "2026").
		Scan(&seasonID)
	if err != nil {
		t.Fatalf("create season: %v", err)
	}

	err = db.QueryRow(ctx, `
        INSERT INTO clubs (name, canonical_name, slug)
        VALUES ('Home Club', 'home club', $1)
        RETURNING id
    `, "home-"+suffix).Scan(&homeClubID)
	if err != nil {
		t.Fatalf("create home club: %v", err)
	}

	err = db.QueryRow(ctx, `
        INSERT INTO clubs (name, canonical_name, slug)
        VALUES ('Away Club', 'away club', $1)
        RETURNING id
    `, "away-"+suffix).Scan(&awayClubID)
	if err != nil {
		t.Fatalf("create away club: %v", err)
	}

	err = db.QueryRow(ctx, `
        INSERT INTO matches (
            season_id,
            home_club_id,
            away_club_id
        )
        VALUES ($1, $2, $3)
        RETURNING id
    `, seasonID, homeClubID, awayClubID).Scan(&matchID)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	var documentID string

	err = db.QueryRow(ctx, `
        INSERT INTO documents (
            match_id,
            source_key,
            source_url,
            original_filename,
            document_type,
            status,
            storage_key,
            downloaded_at
        )
        VALUES (
            $1,
            'test',
            $2,
            'report.pdf',
            'match_report',
            'downloaded',
            $3,
            now()
        )
        RETURNING id
    `,
		matchID,
		"https://example.test/"+suffix+".pdf",
		"documents/"+suffix+".pdf",
	).Scan(&documentID)

	if err != nil {
		t.Fatalf("create downloaded document: %v", err)
	}

	return db, extractionFixture{
		repository: NewExtractionRepository(db),
		documentID: documentID,
	}
}

func TestExtractionRepositoryCreatePending(t *testing.T) {
	db, fixture := setupExtractionTest(t)

	ctx := context.Background()

	created, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatalf("CreatePending(): %v", err)
	}

	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}

	// Calling again must not create another active extraction.
	created, err = fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatalf("second CreatePending(): %v", err)
	}

	if created != 0 {
		t.Fatalf("second created = %d, want 0", created)
	}

	var count int

	err = db.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM extractions
        WHERE document_id = $1
          AND provider = 'sofi-api'
    `, fixture.documentID).Scan(&count)

	if err != nil {
		t.Fatalf("count extractions: %v", err)
	}

	if count != 1 {
		t.Fatalf("extraction count = %d, want 1", count)
	}
}

func TestExtractionRepositoryClaimPending(t *testing.T) {
	db, fixture := setupExtractionTest(t)

	ctx := context.Background()

	// Create one pending extraction.
	created, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatalf("CreatePending(): %v", err)
	}

	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}

	// Claim the pending extraction.
	tasks, err := fixture.repository.ClaimPending(ctx, 5)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}

	task := tasks[0]

	if task.DocumentID != fixture.documentID {
		t.Fatalf(
			"document ID = %s, want %s",
			task.DocumentID,
			fixture.documentID,
		)
	}

	if task.StorageKey == "" {
		t.Fatal("storage key should not be empty")
	}

	if task.OriginalFilename != "report.pdf" {
		t.Fatalf(
			"filename = %s, want report.pdf",
			task.OriginalFilename,
		)
	}

	if task.Season != "2026" {
		t.Fatalf(
			"season = %s, want 2026",
			task.Season,
		)
	}

	if task.SubmissionToken == "" {
		t.Fatal("submission token should be generated")
	}

	if task.SubmissionAttempts != 1 {
		t.Fatalf(
			"submission attempts = %d, want 1",
			task.SubmissionAttempts,
		)
	}

	// The same task must not be claimed twice.
	tasks, err = fixture.repository.ClaimPending(ctx, 5)
	if err != nil {
		t.Fatalf("second ClaimPending(): %v", err)
	}

	if len(tasks) != 0 {
		t.Fatalf(
			"second claim returned %d tasks, want 0",
			len(tasks),
		)
	}

	// Verify persisted state.
	var status string
	var attempts int
	var token *string

	err = db.QueryRow(ctx, `
        SELECT
            status,
            submission_attempts,
            submission_token
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(&status, &attempts, &token)

	if err != nil {
		t.Fatalf("query extraction: %v", err)
	}

	if status != "submitting" {
		t.Fatalf("status = %s, want submitting", status)
	}

	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}

	if token == nil || *token != task.SubmissionToken {
		t.Fatal("persisted submission token does not match")
	}
}

func TestExtractionRepositoryMarkSubmitted(t *testing.T) {
	db, fixture := setupExtractionTest(t)

	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatalf("CreatePending(): %v", err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatalf("ClaimPending(): %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}

	task := tasks[0]
	externalID := "sofi-extraction-test-123"

	err = fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		externalID,
	)
	if err != nil {
		t.Fatalf("MarkSubmitted(): %v", err)
	}

	var (
		status          string
		storedID        string
		submittedAt     *time.Time
		nextPollAt      *time.Time
		submissionToken *string
		startedAt       *time.Time
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            external_extraction_id,
            submitted_at,
            next_poll_at,
            submission_token,
            submission_started_at
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&storedID,
		&submittedAt,
		&nextPollAt,
		&submissionToken,
		&startedAt,
	)

	if err != nil {
		t.Fatalf("query extraction: %v", err)
	}

	if status != "submitted" {
		t.Fatalf("status = %s, want submitted", status)
	}

	if storedID != externalID {
		t.Fatalf(
			"external ID = %s, want %s",
			storedID,
			externalID,
		)
	}

	if submittedAt == nil {
		t.Fatal("submitted_at should be populated")
	}

	if nextPollAt == nil {
		t.Fatal("next_poll_at should be populated")
	}

	if submissionToken != nil {
		t.Fatal("submission token should be cleared")
	}

	if startedAt != nil {
		t.Fatal("submission_started_at should be cleared")
	}

	// A stale worker must not update an already submitted task.
	err = fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"another-extraction-id",
	)

	if err == nil {
		t.Fatal("expected stale submission token to be rejected")
	}
}

func TestExtractionRepositoryMarkSubmissionFailed(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]
	cause := fmt.Errorf("document storage unavailable")

	err = fixture.repository.MarkSubmissionFailed(
		ctx,
		task.ID,
		task.SubmissionToken,
		cause,
	)
	if err != nil {
		t.Fatalf("MarkSubmissionFailed(): %v", err)
	}

	var status string
	var errorMessage *string
	var token *string

	err = db.QueryRow(ctx, `
        SELECT status, error_message, submission_token
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(&status, &errorMessage, &token)

	if err != nil {
		t.Fatal(err)
	}

	if status != "failed" {
		t.Fatalf("status = %s, want failed", status)
	}

	if errorMessage == nil || *errorMessage != cause.Error() {
		t.Fatal("failure message was not persisted")
	}

	if token != nil {
		t.Fatal("submission token should be cleared")
	}
}

func TestExtractionRepositoryMarkSubmissionUnknown(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]
	cause := fmt.Errorf("connection lost after request")

	err = fixture.repository.MarkSubmissionUnknown(
		ctx,
		task.ID,
		task.SubmissionToken,
		cause,
	)
	if err != nil {
		t.Fatalf("MarkSubmissionUnknown(): %v", err)
	}

	var status string
	var errorMessage *string

	err = db.QueryRow(ctx, `
        SELECT status, error_message
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(&status, &errorMessage)

	if err != nil {
		t.Fatal(err)
	}

	if status != "submission_unknown" {
		t.Fatalf(
			"status = %s, want submission_unknown",
			status,
		)
	}

	if errorMessage == nil || *errorMessage != cause.Error() {
		t.Fatal("unknown submission error was not persisted")
	}

	// Unknown submissions must never return to the automatic queue.
	tasks, err = fixture.repository.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 0 {
		t.Fatalf(
			"claimed %d tasks, want 0",
			len(tasks),
		)
	}
}

func TestExtractionRepositoryRecoverStale(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}

	task := tasks[0]

	// Simulate a worker that abandoned its submission.
	_, err = db.Exec(ctx, `
        UPDATE extractions
        SET submission_started_at = now() - interval '10 minutes'
        WHERE id = $1
    `, task.ID)
	if err != nil {
		t.Fatal(err)
	}

	recovered, err := fixture.repository.RecoverStale(
		ctx,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("RecoverStale(): %v", err)
	}

	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	var (
		status       string
		errorMessage *string
		token        *string
		startedAt    *time.Time
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            error_message,
            submission_token,
            submission_started_at
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&errorMessage,
		&token,
		&startedAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "submission_unknown" {
		t.Fatalf(
			"status = %s, want submission_unknown",
			status,
		)
	}

	if errorMessage == nil ||
		!strings.Contains(*errorMessage, "lease expired") {
		t.Fatal("lease expiration error was not recorded")
	}

	if token != nil {
		t.Fatal("submission token should be cleared")
	}

	if startedAt != nil {
		t.Fatal("submission_started_at should be cleared")
	}

	// The extraction must not be automatically resubmitted.
	tasks, err = fixture.repository.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 0 {
		t.Fatalf(
			"claimed %d tasks after recovery, want 0",
			len(tasks),
		)
	}
}

func TestExtractionRepositoryClaimDueForPolling(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}

	task := tasks[0]
	externalID := "sofi-poll-test-123"

	err = fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		externalID,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Claim extraction ready for polling.
	pollTasks, err := fixture.repository.ClaimDueForPolling(ctx, 5)
	if err != nil {
		t.Fatalf("ClaimDueForPolling(): %v", err)
	}

	if len(pollTasks) != 1 {
		t.Fatalf("got %d poll tasks, want 1", len(pollTasks))
	}

	pollTask := pollTasks[0]

	if pollTask.ID != task.ID {
		t.Fatalf("ID = %s, want %s", pollTask.ID, task.ID)
	}

	if pollTask.ExternalID != externalID {
		t.Fatalf(
			"external ID = %s, want %s",
			pollTask.ExternalID,
			externalID,
		)
	}

	if pollTask.PollToken == "" {
		t.Fatal("poll token should be generated")
	}

	// The same extraction cannot be claimed twice.
	pollTasks, err = fixture.repository.ClaimDueForPolling(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 0 {
		t.Fatalf(
			"second claim returned %d tasks, want 0",
			len(pollTasks),
		)
	}

	var (
		pollToken   *string
		pollStarted *time.Time
		status      string
	)

	err = db.QueryRow(ctx, `
        SELECT status, poll_token, poll_started_at
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&pollToken,
		&pollStarted,
	)

	if err != nil {
		t.Fatal(err)
	}

	if status != "submitted" {
		t.Fatalf("status = %s, want submitted", status)
	}

	if pollToken == nil || *pollToken != pollTask.PollToken {
		t.Fatal("persisted poll token does not match")
	}

	if pollStarted == nil {
		t.Fatal("poll_started_at should be populated")
	}
}

func TestExtractionRepositoryMarkPollPending(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]

	err = fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"sofi-poll-pending-123",
	)
	if err != nil {
		t.Fatal(err)
	}

	pollTasks, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 1 {
		t.Fatalf("got %d poll tasks, want 1", len(pollTasks))
	}

	pollTask := pollTasks[0]

	delay := 30 * time.Second

	err = fixture.repository.MarkPollPending(
		ctx,
		pollTask.ID,
		pollTask.PollToken,
		"processing",
		delay,
	)
	if err != nil {
		t.Fatalf("MarkPollPending(): %v", err)
	}

	var (
		status       string
		nextPollAt   *time.Time
		pollToken    *string
		pollStarted  *time.Time
		lastPolledAt *time.Time
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            next_poll_at,
            poll_token,
            poll_started_at,
            last_polled_at
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&nextPollAt,
		&pollToken,
		&pollStarted,
		&lastPolledAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "processing" {
		t.Fatalf("status = %s, want processing", status)
	}

	if nextPollAt == nil {
		t.Fatal("next_poll_at should be populated")
	}

	if pollToken != nil {
		t.Fatal("poll token should be cleared")
	}

	if pollStarted != nil {
		t.Fatal("poll_started_at should be cleared")
	}

	if lastPolledAt == nil {
		t.Fatal("last_polled_at should be populated")
	}

	// The extraction must not be immediately claimable again.
	pollTasks, err = fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 0 {
		t.Fatalf(
			"got %d tasks immediately after scheduling, want 0",
			len(pollTasks),
		)
	}
}

func TestExtractionRepositoryMarkPollRetry(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]

	if err := fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"sofi-poll-retry-123",
	); err != nil {
		t.Fatal(err)
	}

	pollTasks, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	pollTask := pollTasks[0]

	cause := fmt.Errorf("SOFI API returned HTTP 503")
	retryAfter := 45 * time.Second

	before := time.Now()

	err = fixture.repository.MarkPollRetry(
		ctx,
		pollTask.ID,
		pollTask.PollToken,
		cause,
		retryAfter,
	)
	if err != nil {
		t.Fatalf("MarkPollRetry(): %v", err)
	}

	var (
		status       string
		nextPollAt   *time.Time
		pollToken    *string
		pollStarted  *time.Time
		lastPolledAt *time.Time
		errorMessage *string
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            next_poll_at,
            poll_token,
            poll_started_at,
            last_polled_at,
            error_message
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&nextPollAt,
		&pollToken,
		&pollStarted,
		&lastPolledAt,
		&errorMessage,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "submitted" {
		t.Fatalf("status = %s, want submitted", status)
	}

	if nextPollAt == nil {
		t.Fatal("next_poll_at should be populated")
	}

	if nextPollAt.Before(before.Add(retryAfter)) {
		t.Fatalf("next_poll_at is too early: %v", *nextPollAt)
	}

	if pollToken != nil {
		t.Fatal("poll token should be cleared")
	}

	if pollStarted != nil {
		t.Fatal("poll_started_at should be cleared")
	}

	if lastPolledAt == nil {
		t.Fatal("last_polled_at should be populated")
	}

	if errorMessage == nil || *errorMessage != cause.Error() {
		t.Fatalf("error_message = %v, want %q", errorMessage, cause.Error())
	}

	pollTasks, err = fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 0 {
		t.Fatalf("got %d tasks immediately, want 0", len(pollTasks))
	}
}

func TestExtractionRepositoryMarkPollCompleted(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]

	if err := fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"sofi-completed-123",
	); err != nil {
		t.Fatal(err)
	}

	pollTasks, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 1 {
		t.Fatalf("got %d polling tasks, want 1", len(pollTasks))
	}

	pollTask := pollTasks[0]

	result := json.RawMessage(`{
        "club": "Clube Exemplo",
        "gross_revenue": 150000.00,
        "net_revenue": 120000.00
    }`)

	validation := json.RawMessage(`{
        "valid": true,
        "warnings": []
    }`)

	completedAt := time.Now().UTC().Truncate(time.Microsecond)

	err = fixture.repository.MarkPollCompleted(
		ctx,
		pollTask.ID,
		pollTask.PollToken,
		result,
		validation,
		&completedAt,
	)
	if err != nil {
		t.Fatalf("MarkPollCompleted(): %v", err)
	}

	var (
		status       string
		rawResult    []byte
		completed    *time.Time
		pollToken    *string
		pollStarted  *time.Time
		nextPollAt   *time.Time
		errorMessage *string
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            raw_result,
            completed_at,
            poll_token,
            poll_started_at,
            next_poll_at,
            error_message
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&rawResult,
		&completed,
		&pollToken,
		&pollStarted,
		&nextPollAt,
		&errorMessage,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "completed" {
		t.Fatalf("status = %s, want completed", status)
	}

	if completed == nil || !completed.Equal(completedAt) {
		t.Fatalf("completed_at = %v, want %v", completed, completedAt)
	}

	if pollToken != nil {
		t.Fatal("poll_token should be cleared")
	}

	if pollStarted != nil {
		t.Fatal("poll_started_at should be cleared")
	}

	if nextPollAt != nil {
		t.Fatal("next_poll_at should be cleared")
	}

	if errorMessage != nil {
		t.Fatalf("error_message should be NULL, got %s", *errorMessage)
	}

	var persisted struct {
		Result     json.RawMessage `json:"result"`
		Validation json.RawMessage `json:"validation"`
	}

	if err := json.Unmarshal(rawResult, &persisted); err != nil {
		t.Fatalf("decode raw_result: %v", err)
	}

	if !json.Valid(persisted.Result) {
		t.Fatal("persisted result is invalid JSON")
	}

	if !json.Valid(persisted.Validation) {
		t.Fatal("persisted validation is invalid JSON")
	}

	var expectedResult any
	var actualResult any

	if err := json.Unmarshal(result, &expectedResult); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(persisted.Result, &actualResult); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedResult, actualResult) {
		t.Fatalf(
			"persisted result differs from submitted result:\ngot: %#v\nwant: %#v",
			actualResult,
			expectedResult,
		)
	}

	var expectedValidation any
	var actualValidation any

	if err := json.Unmarshal(validation, &expectedValidation); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(persisted.Validation, &actualValidation); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedValidation, actualValidation) {
		t.Fatalf(
			"persisted validation differs from submitted validation",
		)
	}

	// A completed extraction must never be polled again.
	pollTasks, err = fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 0 {
		t.Fatalf("got %d tasks after completion, want 0", len(pollTasks))
	}
}

func TestExtractionRepositoryMarkPollFailed(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}

	task := tasks[0]

	if err := fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"sofi-failed-123",
	); err != nil {
		t.Fatal(err)
	}

	pollTasks, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 1 {
		t.Fatalf("got %d polling tasks, want 1", len(pollTasks))
	}

	pollTask := pollTasks[0]

	cause := fmt.Errorf("extraction failed: invalid document structure")

	err = fixture.repository.MarkPollFailed(
		ctx,
		pollTask.ID,
		pollTask.PollToken,
		cause,
	)
	if err != nil {
		t.Fatalf("MarkPollFailed(): %v", err)
	}

	var (
		status       string
		completedAt  *time.Time
		pollToken    *string
		pollStarted  *time.Time
		nextPollAt   *time.Time
		errorMessage *string
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            completed_at,
            poll_token,
            poll_started_at,
            next_poll_at,
            error_message
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&completedAt,
		&pollToken,
		&pollStarted,
		&nextPollAt,
		&errorMessage,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "failed" {
		t.Fatalf("status = %s, want failed", status)
	}

	if completedAt == nil {
		t.Fatal("completed_at should be populated")
	}

	if pollToken != nil {
		t.Fatal("poll_token should be cleared")
	}

	if pollStarted != nil {
		t.Fatal("poll_started_at should be cleared")
	}

	if nextPollAt != nil {
		t.Fatal("next_poll_at should be cleared")
	}

	if errorMessage == nil || *errorMessage != cause.Error() {
		t.Fatalf(
			"error_message = %v, want %q",
			errorMessage,
			cause.Error(),
		)
	}

	// Failed extractions must not return to the polling queue.
	pollTasks, err = fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 0 {
		t.Fatalf(
			"got %d tasks after failure, want 0",
			len(pollTasks),
		)
	}
}

func TestExtractionRepositoryRecoverStalePolls(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]

	if err := fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"sofi-stale-poll-123",
	); err != nil {
		t.Fatal(err)
	}

	pollTasks, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 1 {
		t.Fatalf("got %d polling tasks, want 1", len(pollTasks))
	}

	// Simulate an abandoned polling lease.
	_, err = db.Exec(ctx, `
        UPDATE extractions
        SET poll_started_at = now() - interval '10 minutes'
        WHERE id = $1
    `, task.ID)
	if err != nil {
		t.Fatal(err)
	}

	recovered, err := fixture.repository.RecoverStalePolls(
		ctx,
		5*time.Minute,
	)
	if err != nil {
		t.Fatalf("RecoverStalePolls(): %v", err)
	}

	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	var (
		status       string
		pollToken    *string
		pollStarted  *time.Time
		nextPollAt   *time.Time
		errorMessage *string
	)

	err = db.QueryRow(ctx, `
        SELECT
            status,
            poll_token,
            poll_started_at,
            next_poll_at,
            error_message
        FROM extractions
        WHERE id = $1
    `, task.ID).Scan(
		&status,
		&pollToken,
		&pollStarted,
		&nextPollAt,
		&errorMessage,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "submitted" {
		t.Fatalf("status = %s, want submitted", status)
	}

	if pollToken != nil {
		t.Fatal("poll_token should be cleared")
	}

	if pollStarted != nil {
		t.Fatal("poll_started_at should be cleared")
	}

	if nextPollAt == nil {
		t.Fatal("next_poll_at should be populated")
	}

	if errorMessage == nil ||
		*errorMessage != "poll lease expired; retry scheduled" {
		t.Fatalf("unexpected error_message: %v", errorMessage)
	}

	// The recovered extraction must be claimable again.
	pollTasks, err = fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(pollTasks) != 1 {
		t.Fatalf("got %d tasks after recovery, want 1", len(pollTasks))
	}

	if pollTasks[0].ID != task.ID {
		t.Fatalf("recovered task ID = %s, want %s",
			pollTasks[0].ID, task.ID)
	}
}

func TestExtractionRepositoryConcurrentPollingClaim(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	const totalExtractions = 10
	const totalWorkers = 5

	// setupExtractionTest creates one eligible document.
	// Add nine more documents for the concurrency test.
	for i := 1; i < totalExtractions; i++ {
		_, err := db.Exec(ctx, `
        INSERT INTO documents (
            match_id,
            source_key,
            source_url,
            document_type,
            status,
            storage_key,
            downloaded_at
        )
        SELECT
            match_id,
            'test-concurrency',
            $2,
            'match_report',
            'downloaded',
            $3,
            now()
        FROM documents
        WHERE id = $1
    `,
			fixture.documentID,
			fmt.Sprintf("https://example.test/concurrency-%d.pdf", i),
			fmt.Sprintf("test/concurrency-%d.pdf", i),
		)

		if err != nil {
			t.Fatalf("create concurrency document %d: %v", i, err)
		}
	}

	_, err := fixture.repository.CreatePending(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != totalExtractions {
		t.Fatalf(
			"got %d submission tasks, want %d",
			len(tasks),
			totalExtractions,
		)
	}

	for i, task := range tasks {
		externalID := fmt.Sprintf("sofi-concurrent-%d", i)

		if err := fixture.repository.MarkSubmitted(
			ctx,
			task.ID,
			task.SubmissionToken,
			externalID,
		); err != nil {
			t.Fatal(err)
		}
	}

	type claimResult struct {
		tasks []ExtractionPollTask
		err   error
	}

	results := make(chan claimResult, totalWorkers)
	start := make(chan struct{})

	var wg sync.WaitGroup

	for i := 0; i < totalWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			claimed, err := fixture.repository.ClaimDueForPolling(
				ctx,
				totalExtractions,
			)

			results <- claimResult{
				tasks: claimed,
				err:   err,
			}
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	claimedIDs := make(map[string]bool)
	totalClaimed := 0

	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent claim: %v", result.err)
		}

		for _, task := range result.tasks {
			if claimedIDs[task.ID] {
				t.Fatalf(
					"extraction %s was claimed more than once",
					task.ID,
				)
			}

			claimedIDs[task.ID] = true
			totalClaimed++
		}
	}

	if totalClaimed != totalExtractions {
		t.Fatalf(
			"total claimed = %d, want %d",
			totalClaimed,
			totalExtractions,
		)
	}
}

func TestExtractionRepositoryRejectsStalePollToken(t *testing.T) {
	db, fixture := setupExtractionTest(t)
	ctx := context.Background()

	_, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := fixture.repository.ClaimPending(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	task := tasks[0]

	if err := fixture.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		"sofi-stale-token-123",
	); err != nil {
		t.Fatal(err)
	}

	firstClaim, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(firstClaim) != 1 {
		t.Fatalf("got %d claims, want 1", len(firstClaim))
	}

	// Simulate the first worker becoming stale.
	_, err = db.Exec(ctx, `
        UPDATE extractions
        SET poll_started_at = now() - interval '10 minutes'
        WHERE id = $1
    `, task.ID)
	if err != nil {
		t.Fatal(err)
	}

	recovered, err := fixture.repository.RecoverStalePolls(
		ctx,
		5*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}

	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	secondClaim, err := fixture.repository.ClaimDueForPolling(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(secondClaim) != 1 {
		t.Fatalf("got %d second claims, want 1", len(secondClaim))
	}

	if firstClaim[0].PollToken == secondClaim[0].PollToken {
		t.Fatal("new worker received the same poll token")
	}

	err = fixture.repository.MarkPollCompleted(
		ctx,
		task.ID,
		firstClaim[0].PollToken,
		json.RawMessage(`{"result":"stale"}`),
		json.RawMessage(`{"valid":false}`),
		nil,
	)

	if err == nil {
		t.Fatal("expected stale worker to be rejected")
	}

	// The current worker must still own the extraction.
	err = fixture.repository.MarkPollCompleted(
		ctx,
		task.ID,
		secondClaim[0].PollToken,
		json.RawMessage(`{"result":"current"}`),
		json.RawMessage(`{"valid":true}`),
		nil,
	)

	if err != nil {
		t.Fatalf("current worker was rejected: %v", err)
	}
}

func TestExtractionRepositoryGetLatestByDocumentID(t *testing.T) {
	_, fixture := setupExtractionTest(t)

	ctx := context.Background()

	created, err := fixture.repository.CreatePending(ctx, 10)
	if err != nil {
		t.Fatalf("CreatePending(): %v", err)
	}

	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}

	extraction, err := fixture.repository.GetLatestByDocumentID(
		ctx,
		fixture.documentID,
	)
	if err != nil {
		t.Fatalf("GetLatestByDocumentID(): %v", err)
	}

	if extraction.DocumentID != fixture.documentID {
		t.Fatalf(
			"document ID = %s, want %s",
			extraction.DocumentID,
			fixture.documentID,
		)
	}

	if extraction.Status != "pending" {
		t.Fatalf(
			"status = %s, want pending",
			extraction.Status,
		)
	}
}
