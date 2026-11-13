package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultExtractionLease = 5 * time.Minute
	maxExtractionBatch     = 100
	defaultSOFIProvider    = "sofi-api"
)

type ExtractionTask struct {
	ID                 string
	DocumentID         string
	StorageKey         string
	OriginalFilename   string
	Season             string
	SubmissionToken    string
	SubmissionAttempts int
}

type ExtractionPollTask struct {
	ID         string
	ExternalID string
	PollToken  string
}

type ExtractionRepository struct {
	db *pgxpool.Pool
}

func NewExtractionRepository(db *pgxpool.Pool) *ExtractionRepository {
	return &ExtractionRepository{db: db}
}

// CreatePending creates extraction records for downloaded
// documents that have never been submitted to this provider.
func (r *ExtractionRepository) CreatePending(
	ctx context.Context,
	limit int,
) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("extraction repository is not initialized")
	}

	if limit < 1 || limit > maxExtractionBatch {
		return 0, fmt.Errorf("limit must be between 1 and 100")
	}

	tag, err := r.db.Exec(ctx, `
		WITH candidates AS (
			SELECT d.id
			FROM documents d
			JOIN matches m ON m.id = d.match_id
			JOIN seasons s ON s.id = m.season_id
			WHERE d.status = 'downloaded'
			  AND d.storage_key IS NOT NULL
			  AND NOT EXISTS (
				SELECT 1
				FROM extractions e
				WHERE e.document_id = d.id
				  AND e.provider = $2
			  )
			ORDER BY d.downloaded_at, d.id
			LIMIT $1
			FOR UPDATE OF d SKIP LOCKED
		)
		INSERT INTO extractions (
			document_id,
			provider,
			status
		)
		SELECT
			c.id,
			$2,
			'pending'
		FROM candidates c
		ON CONFLICT (document_id, provider)
		WHERE status IN (
			'pending',
			'submitting',
			'submitted',
			'processing',
			'submission_unknown'
		)
		DO NOTHING
	`, limit, defaultSOFIProvider)

	if err != nil {
		return 0, fmt.Errorf("create pending extractions: %w", err)
	}

	return tag.RowsAffected(), nil
}

// ClaimPending atomically reserves pending extractions.
func (r *ExtractionRepository) ClaimPending(
	ctx context.Context,
	limit int,
) ([]ExtractionTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("extraction repository is not initialized")
	}

	if limit < 1 || limit > maxExtractionBatch {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	rows, err := r.db.Query(ctx, `
		WITH candidates AS (
			SELECT e.id
			FROM extractions e
			WHERE e.status = 'pending'
			ORDER BY e.created_at, e.id
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE extractions e
		SET status = 'submitting',
		    submission_attempts = e.submission_attempts + 1,
		    submission_started_at = now(),
		    submission_token = gen_random_uuid(),
		    error_message = NULL,
		    updated_at = now()
		FROM candidates c,
		     documents d,
		     matches m,
		     seasons s
		WHERE e.id = c.id
		  AND d.id = e.document_id
		  AND m.id = d.match_id
		  AND s.id = m.season_id
		  AND d.status = 'downloaded'
		  AND d.storage_key IS NOT NULL
		RETURNING
			e.id,
			e.document_id,
			d.storage_key,
			COALESCE(
				NULLIF(d.original_filename, ''),
				'document.pdf'
			),
			s.name,
			e.submission_token,
			e.submission_attempts
	`, limit)

	if err != nil {
		return nil, fmt.Errorf("claim pending extractions: %w", err)
	}
	defer rows.Close()

	tasks := make([]ExtractionTask, 0, limit)

	for rows.Next() {
		var task ExtractionTask

		if err := rows.Scan(
			&task.ID,
			&task.DocumentID,
			&task.StorageKey,
			&task.OriginalFilename,
			&task.Season,
			&task.SubmissionToken,
			&task.SubmissionAttempts,
		); err != nil {
			return nil, fmt.Errorf("scan claimed extraction: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed extractions: %w", err)
	}

	return tasks, nil
}

// MarkSubmitted records the extraction ID returned by SOFI API.
func (r *ExtractionRepository) MarkSubmitted(
	ctx context.Context,
	id string,
	token string,
	externalID string,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" ||
		strings.TrimSpace(externalID) == "" {
		return fmt.Errorf("extraction submission metadata is incomplete")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE extractions
		SET status = 'submitted',
		    external_extraction_id = $3,
		    submitted_at = now(),
		    next_poll_at = now(),
		    submission_started_at = NULL,
		    submission_token = NULL,
		    error_message = NULL,
		    updated_at = now()
		WHERE id = $1
		  AND status = 'submitting'
		  AND submission_token = $2
	`, id, token, externalID)

	if err != nil {
		return fmt.Errorf("mark extraction submitted: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("extraction %s is not in submitting state", id)
	}

	return nil
}

// MarkSubmissionUnknown records an ambiguous submission outcome.
// It must never be automatically retried.
func (r *ExtractionRepository) MarkSubmissionUnknown(
	ctx context.Context,
	id string,
	token string,
	cause error,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" {
		return fmt.Errorf("extraction ID and token are required")
	}

	if cause == nil {
		return fmt.Errorf("submission failure cause is required")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE extractions
		SET status = 'submission_unknown',
		    error_message = $3,
		    submission_started_at = NULL,
		    submission_token = NULL,
		    updated_at = now()
		WHERE id = $1
		  AND status = 'submitting'
		  AND submission_token = $2
	`, id, token, cause.Error())

	if err != nil {
		return fmt.Errorf("mark submission unknown: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("extraction %s is not in submitting state", id)
	}

	return nil
}

// RecoverStale marks abandoned submissions as unknown.
// Reconciliation is required before any retry.
func (r *ExtractionRepository) RecoverStale(
	ctx context.Context,
	lease time.Duration,
) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("extraction repository is not initialized")
	}

	if lease <= 0 {
		lease = defaultExtractionLease
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE extractions
		SET status = 'submission_unknown',
		    submission_started_at = NULL,
		    submission_token = NULL,
		    error_message = 'submission lease expired; reconciliation required',
		    updated_at = now()
		WHERE status = 'submitting'
		  AND submission_started_at <
		      now() - ($1 * interval '1 second')
	`, lease.Seconds())

	if err != nil {
		return 0, fmt.Errorf("recover stale submissions: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *ExtractionRepository) MarkSubmissionFailed(
	ctx context.Context,
	id string,
	token string,
	cause error,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" {
		return fmt.Errorf("extraction ID and token are required")
	}

	if cause == nil {
		return fmt.Errorf("failure cause is required")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE extractions
		SET status = 'failed',
		    error_message = $3,
		    submission_started_at = NULL,
		    submission_token = NULL,
		    updated_at = now()
		WHERE id = $1
		  AND status = 'submitting'
		  AND submission_token = $2
	`, id, token, cause.Error())

	if err != nil {
		return fmt.Errorf("mark submission failed: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"extraction %s is not in submitting state",
			id,
		)
	}

	return nil
}

// ClaimDueForPolling atomically reserves extractions ready for polling.
func (r *ExtractionRepository) ClaimDueForPolling(
	ctx context.Context,
	limit int,
) ([]ExtractionPollTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("extraction repository is not initialized")
	}

	if limit < 1 || limit > maxExtractionBatch {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	rows, err := r.db.Query(ctx, `
        WITH candidates AS (
            SELECT id
            FROM extractions
            WHERE status IN ('submitted', 'processing')
              AND external_extraction_id IS NOT NULL
              AND next_poll_at <= now()
              AND poll_started_at IS NULL
            ORDER BY next_poll_at, id
            LIMIT $1
            FOR UPDATE SKIP LOCKED
        )
        UPDATE extractions e
        SET poll_started_at = now(),
            poll_token = gen_random_uuid(),
            updated_at = now()
        FROM candidates c
        WHERE e.id = c.id
        RETURNING
            e.id,
            e.external_extraction_id,
            e.poll_token
    `, limit)

	if err != nil {
		return nil, fmt.Errorf("claim due extractions: %w", err)
	}
	defer rows.Close()

	tasks := make([]ExtractionPollTask, 0, limit)

	for rows.Next() {
		var task ExtractionPollTask

		if err := rows.Scan(
			&task.ID,
			&task.ExternalID,
			&task.PollToken,
		); err != nil {
			return nil, fmt.Errorf("scan claimed extraction: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed extractions: %w", err)
	}

	return tasks, nil
}

// MarkPollPending schedules another poll for an active extraction.
func (r *ExtractionRepository) MarkPollPending(
	ctx context.Context,
	id string,
	token string,
	status string,
	nextPollAfter time.Duration,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" {
		return fmt.Errorf("extraction ID and poll token are required")
	}

	if status != "submitted" && status != "processing" {
		return fmt.Errorf("invalid polling status: %s", status)
	}

	if nextPollAfter <= 0 {
		return fmt.Errorf("next poll delay must be positive")
	}

	tag, err := r.db.Exec(ctx, `
        UPDATE extractions
        SET status = $3,
            next_poll_at = now() + ($4 * interval '1 second'),
            poll_started_at = NULL,
            poll_token = NULL,
            last_polled_at = now(),
            error_message = NULL,
            updated_at = now()
        WHERE id = $1
          AND status IN ('submitted', 'processing')
          AND poll_token = $2
    `, id, token, status, nextPollAfter.Seconds())

	if err != nil {
		return fmt.Errorf("schedule next extraction poll: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("extraction %s is not owned by this poller", id)
	}

	return nil
}

// MarkPollRetry schedules another attempt after a temporary polling error.
func (r *ExtractionRepository) MarkPollRetry(
	ctx context.Context,
	id string,
	token string,
	cause error,
	retryAfter time.Duration,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" {
		return fmt.Errorf("extraction ID and poll token are required")
	}

	if cause == nil {
		return fmt.Errorf("polling error is required")
	}

	if retryAfter <= 0 {
		return fmt.Errorf("retry delay must be positive")
	}

	tag, err := r.db.Exec(ctx, `
        UPDATE extractions
        SET next_poll_at = now() + ($3 * interval '1 second'),
            poll_started_at = NULL,
            poll_token = NULL,
            last_polled_at = now(),
            error_message = $4,
            updated_at = now()
        WHERE id = $1
          AND status IN ('submitted', 'processing')
          AND poll_token = $2
    `, id, token, retryAfter.Seconds(), cause.Error())

	if err != nil {
		return fmt.Errorf("schedule extraction poll retry: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("extraction %s is not owned by this poller", id)
	}

	return nil
}

// MarkPollCompleted persists the final extraction result.
func (r *ExtractionRepository) MarkPollCompleted(
	ctx context.Context,
	id string,
	token string,
	result json.RawMessage,
	validation json.RawMessage,
	completedAt *time.Time,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" {
		return fmt.Errorf("extraction ID and poll token are required")
	}

	if len(result) == 0 {
		result = json.RawMessage(`null`)
	}

	if len(validation) == 0 {
		validation = json.RawMessage(`null`)
	}

	if !json.Valid(result) {
		return fmt.Errorf("invalid extraction result JSON")
	}

	if !json.Valid(validation) {
		return fmt.Errorf("invalid extraction validation JSON")
	}

	tag, err := r.db.Exec(ctx, `
        UPDATE extractions
        SET status = 'completed',
            raw_result = jsonb_build_object(
                'result', $3::jsonb,
                'validation', $4::jsonb
            ),
            completed_at = COALESCE($5, now()),
            poll_started_at = NULL,
            poll_token = NULL,
            last_polled_at = now(),
            next_poll_at = NULL,
            error_message = NULL,
            updated_at = now()
        WHERE id = $1
          AND status IN ('submitted', 'processing')
          AND poll_token = $2
    `,
		id,
		token,
		string(result),
		string(validation),
		completedAt,
	)

	if err != nil {
		return fmt.Errorf("mark extraction completed: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("extraction %s is not owned by this poller", id)
	}

	return nil
}

// MarkPollFailed records a definitive remote extraction failure.
func (r *ExtractionRepository) MarkPollFailed(
	ctx context.Context,
	id string,
	token string,
	cause error,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("extraction repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(token) == "" {
		return fmt.Errorf("extraction ID and poll token are required")
	}

	if cause == nil {
		return fmt.Errorf("failure cause is required")
	}

	tag, err := r.db.Exec(ctx, `
        UPDATE extractions
        SET status = 'failed',
            error_message = $3,
            completed_at = now(),
            poll_started_at = NULL,
            poll_token = NULL,
            next_poll_at = NULL,
            last_polled_at = now(),
            updated_at = now()
        WHERE id = $1
          AND status IN ('submitted', 'processing')
          AND poll_token = $2
    `, id, token, cause.Error())

	if err != nil {
		return fmt.Errorf("mark extraction failed: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("extraction %s is not owned by this poller", id)
	}

	return nil
}

// RecoverStalePolls releases abandoned polling leases.
func (r *ExtractionRepository) RecoverStalePolls(
	ctx context.Context,
	lease time.Duration,
) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("extraction repository is not initialized")
	}

	if lease <= 0 {
		lease = defaultExtractionLease
	}

	tag, err := r.db.Exec(ctx, `
        UPDATE extractions
        SET poll_started_at = NULL,
            poll_token = NULL,
            next_poll_at = now(),
            error_message = 'poll lease expired; retry scheduled',
            updated_at = now()
        WHERE status IN ('submitted', 'processing')
          AND poll_started_at IS NOT NULL
          AND poll_started_at <
              now() - ($1 * interval '1 second')
    `, lease.Seconds())

	if err != nil {
		return 0, fmt.Errorf("recover stale extraction polls: %w", err)
	}

	return tag.RowsAffected(), nil
}
