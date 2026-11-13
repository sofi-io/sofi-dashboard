package processing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/document"
)

type Repository interface {
	ClaimPending(ctx context.Context, limit int) ([]database.Document, error)

	MarkDownloaded(
		ctx context.Context,
		id string,
		token string,
		contentType string,
		sizeBytes int64,
		sha256 string,
		storageKey string,
	) error

	MarkFailed(
		ctx context.Context,
		id string,
		token string,
		cause error,
	) error

	RecoverStale(
		ctx context.Context,
		lease time.Duration,
	) (int64, error)
}

type ProcessorConfig struct {
	BatchSize int
	Lease     time.Duration
}

type DocumentProcessor struct {
	repository Repository
	downloader *document.Downloader
	config     ProcessorConfig
}

func NewDocumentProcessor(
	repository Repository,
	downloader *document.Downloader,
	config ProcessorConfig,
) (*DocumentProcessor, error) {
	if repository == nil {
		return nil, fmt.Errorf("document repository is required")
	}

	if downloader == nil {
		return nil, fmt.Errorf("document downloader is required")
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 5
	}

	if config.BatchSize > 100 {
		return nil, fmt.Errorf("batch size cannot exceed 100")
	}

	if config.Lease <= 0 {
		config.Lease = 5 * time.Minute
	}

	return &DocumentProcessor{
		repository: repository,
		downloader: downloader,
		config:     config,
	}, nil
}

// ProcessBatch recovers abandoned downloads and processes
// one batch of available documents.
func (p *DocumentProcessor) ProcessBatch(
	ctx context.Context,
) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	_, err := p.repository.RecoverStale(
		ctx,
		p.config.Lease,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale documents: %w", err)
	}

	documents, err := p.repository.ClaimPending(
		ctx,
		p.config.BatchSize,
	)
	if err != nil {
		return 0, fmt.Errorf("claim pending documents: %w", err)
	}

	var (
		processed int
		failures  []error
	)

	for _, doc := range documents {
		if err := ctx.Err(); err != nil {
			return processed, errors.Join(
				errors.Join(failures...),
				err,
			)
		}

		if doc.DownloadToken == nil || *doc.DownloadToken == "" {
			failures = append(
				failures,
				fmt.Errorf(
					"document %s has no download token",
					doc.ID,
				),
			)
			continue
		}

		if err := p.processDocument(ctx, doc); err != nil {
			failures = append(failures, err)
			continue
		}

		processed++
	}

	return processed, errors.Join(failures...)
}

func (p *DocumentProcessor) processDocument(
	ctx context.Context,
	doc database.Document,
) error {
	token := *doc.DownloadToken

	result, err := p.downloader.Download(
		ctx,
		doc.SourceURL,
	)

	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if markErr := p.repository.MarkFailed(
			ctx,
			doc.ID,
			token,
			err,
		); markErr != nil {
			return errors.Join(
				fmt.Errorf(
					"download document %s: %w",
					doc.ID,
					err,
				),
				fmt.Errorf(
					"mark document failed: %w",
					markErr,
				),
			)
		}

		return fmt.Errorf(
			"download document %s: %w",
			doc.ID,
			err,
		)
	}

	if err := p.repository.MarkDownloaded(
		ctx,
		doc.ID,
		token,
		result.ContentType,
		result.SizeBytes,
		result.SHA256,
		result.StorageKey,
	); err != nil {
		return fmt.Errorf(
			"mark document %s downloaded: %w",
			doc.ID,
			err,
		)
	}

	return nil
}
