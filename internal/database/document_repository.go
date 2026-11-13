package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDocumentLease = 5 * time.Minute
	maxDocumentAttempts  = 5
)

type Document struct {
	ID                string
	MatchID           *string
	SourceKey         string
	SourceURL         string
	DocumentType      string
	ContentType       *string
	SizeBytes         *int64
	SHA256            *string
	StorageKey        *string
	Status            string
	DownloadAttempts  int
	DownloadStartedAt *time.Time
	LastError         *string
	DiscoveredAt      time.Time
	DownloadedAt      *time.Time
	DownloadToken     *string
}

type DocumentRepository struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// ClaimPending atomically reserves available documents.
func (r *DocumentRepository) ClaimPending(
	ctx context.Context,
	limit int,
) ([]Document, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("document repository is not initialized")
	}

	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin claim transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM documents
			WHERE status IN ('discovered', 'failed')
			  AND source_url IS NOT NULL
			  AND download_attempts < $2
			ORDER BY discovered_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE documents d
		SET status = 'downloading',
			download_attempts = d.download_attempts + 1,
			download_started_at = now(),
			download_token = gen_random_uuid(),
			last_error = NULL
		FROM candidates c
		WHERE d.id = c.id
		RETURNING
			d.id,
			d.match_id,
			d.source_key,
			d.source_url,
			d.document_type,
			d.content_type,
			d.size_bytes,
			d.sha256,
			d.storage_key,
			d.status,
			d.download_attempts,
			d.download_started_at,
			d.download_token,
			d.last_error,
			d.discovered_at,
			d.downloaded_at
	`, limit, maxDocumentAttempts)
	if err != nil {
		return nil, fmt.Errorf("claim documents: %w", err)
	}

	documents := make([]Document, 0, limit)

	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan claimed document: %w", err)
		}
		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate claimed documents: %w", err)
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}

	return documents, nil
}

// MarkDownloaded records successful storage metadata.
func (r *DocumentRepository) MarkDownloaded(
	ctx context.Context,
	id string,
	token string,
	contentType string,
	sizeBytes int64,
	sha256 string,
	storageKey string,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("document repository is not initialized")
	}

	if strings.TrimSpace(id) == "" ||
		strings.TrimSpace(contentType) == "" ||
		strings.TrimSpace(sha256) == "" ||
		strings.TrimSpace(storageKey) == "" {
		return fmt.Errorf("document metadata is incomplete")
	}

	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("download token is required")
	}

	if sizeBytes <= 0 {
		return fmt.Errorf("document size must be positive")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = 'downloaded',
			content_type = $3,
			size_bytes = $4,
			sha256 = $5,
			storage_key = $6,
			downloaded_at = now(),
			download_started_at = NULL,
			download_token = NULL,
			last_error = NULL
		WHERE id = $1
			AND status = 'downloading'
			AND download_token = $2
	`, id, token, contentType, sizeBytes, sha256, storageKey)
	if err != nil {
		return fmt.Errorf("mark document downloaded: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("document %s is not in downloading state", id)
	}

	return nil
}

// MarkFailed records an unsuccessful attempt.
func (r *DocumentRepository) MarkFailed(
	ctx context.Context,
	id string,
	token string,
	cause error,
) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("document repository is not initialized")
	}

	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("document ID is required")
	}

	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("download token is required")
	}

	if cause == nil {
		return fmt.Errorf("failure cause is required")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = CASE
				WHEN download_attempts >= $3 THEN 'failed'
				ELSE 'discovered'
			END,
			download_started_at = NULL,
			download_token = NULL,
			last_error = $4
		WHERE id = $1
			AND status = 'downloading'
			AND download_token = $2
	`, id, token, maxDocumentAttempts, cause.Error())
	if err != nil {
		return fmt.Errorf("mark document failed: %w", err)
	}

	if tag.RowsAffected() != 1 {
		return fmt.Errorf("document %s is not in downloading state", id)
	}

	return nil
}

// RecoverStale returns abandoned downloads to the queue.
func (r *DocumentRepository) RecoverStale(
	ctx context.Context,
	lease time.Duration,
) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("document repository is not initialized")
	}

	if lease <= 0 {
		lease = defaultDocumentLease
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = CASE
				WHEN download_attempts >= $2 THEN 'failed'
				ELSE 'discovered'
			END,
			download_started_at = NULL,
			download_token = NULL,
			last_error = 'download lease expired'
		WHERE status = 'downloading'
			AND download_started_at < now() - ($1 * interval '1 second')
	`, lease.Seconds(), maxDocumentAttempts)
	if err != nil {
		return 0, fmt.Errorf("recover stale documents: %w", err)
	}

	return tag.RowsAffected(), nil
}

// GetByID retrieves a document by its database ID.
func (r *DocumentRepository) GetByID(
	ctx context.Context,
	id string,
) (Document, error) {
	if r == nil || r.db == nil {
		return Document{}, fmt.Errorf("document repository is not initialized")
	}

	if strings.TrimSpace(id) == "" {
		return Document{}, fmt.Errorf("document ID is required")
	}

	row := r.db.QueryRow(ctx, `
		SELECT
			id,
			match_id,
			source_key,
			source_url,
			document_type,
			content_type,
			size_bytes,
			sha256,
			storage_key,
			status,
			download_attempts,
			download_started_at,
			download_token,
			last_error,
			discovered_at,
			downloaded_at
		FROM documents
		WHERE id = $1
	`, id)

	doc, err := scanDocument(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Document{}, fmt.Errorf("document not found: %s", id)
		}
		return Document{}, fmt.Errorf("get document: %w", err)
	}

	return doc, nil
}

type documentScanner interface {
	Scan(dest ...any) error
}

func scanDocument(row documentScanner) (Document, error) {
	var doc Document

	err := row.Scan(
		&doc.ID,
		&doc.MatchID,
		&doc.SourceKey,
		&doc.SourceURL,
		&doc.DocumentType,
		&doc.ContentType,
		&doc.SizeBytes,
		&doc.SHA256,
		&doc.StorageKey,
		&doc.Status,
		&doc.DownloadAttempts,
		&doc.DownloadStartedAt,
		&doc.DownloadToken,
		&doc.LastError,
		&doc.DiscoveredAt,
		&doc.DownloadedAt,
	)

	return doc, err
}
