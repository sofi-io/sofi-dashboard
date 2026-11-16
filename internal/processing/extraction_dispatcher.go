package processing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/sofi"
	"github.com/sofi-io/sofi-dashboard/internal/storage"
)

type ExtractionRepository interface {
	ClaimPending(
		ctx context.Context,
		limit int,
	) ([]database.ExtractionTask, error)

	MarkSubmitted(
		ctx context.Context,
		id string,
		token string,
		externalID string,
	) error

	MarkSubmissionUnknown(
		ctx context.Context,
		id string,
		token string,
		cause error,
	) error

	MarkSubmissionFailed(
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

type ExtractionAPI interface {
	SubmitExtraction(
		ctx context.Context,
		filename string,
		season string,
		pdf io.Reader,
	) (sofi.Extraction, error)
}

type ExtractionDispatcherConfig struct {
	BatchSize int
	Lease     time.Duration
}

type ExtractionDispatcher struct {
	repository ExtractionRepository
	storage    storage.Storage
	api        ExtractionAPI
	config     ExtractionDispatcherConfig
}

func NewExtractionDispatcher(
	repository ExtractionRepository,
	store storage.Storage,
	api ExtractionAPI,
	config ExtractionDispatcherConfig,
) (*ExtractionDispatcher, error) {
	if repository == nil {
		return nil, fmt.Errorf("extraction repository is required")
	}

	if store == nil {
		return nil, fmt.Errorf("document storage is required")
	}

	if api == nil {
		return nil, fmt.Errorf("SOFI API client is required")
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

	return &ExtractionDispatcher{
		repository: repository,
		storage:    store,
		api:        api,
		config:     config,
	}, nil
}

func (d *ExtractionDispatcher) ProcessBatch(
	ctx context.Context,
) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	_, err := d.repository.RecoverStale(
		ctx,
		d.config.Lease,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale submissions: %w", err)
	}

	tasks, err := d.repository.ClaimPending(
		ctx,
		d.config.BatchSize,
	)
	if err != nil {
		return 0, fmt.Errorf("claim pending extractions: %w", err)
	}

	var (
		dispatched int
		failures   []error
	)

	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return dispatched, errors.Join(
				errors.Join(failures...),
				err,
			)
		}

		if err := d.dispatch(ctx, task); err != nil {
			failures = append(failures, err)
			continue
		}

		dispatched++
	}

	return dispatched, errors.Join(failures...)
}

func (d *ExtractionDispatcher) dispatch(
	ctx context.Context,
	task database.ExtractionTask,
) error {
	if task.ID == "" ||
		task.SubmissionToken == "" ||
		task.StorageKey == "" {
		return fmt.Errorf(
			"extraction %s has incomplete submission metadata",
			task.ID,
		)
	}

	pdf, err := d.storage.Get(ctx, task.StorageKey)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return d.markFailed(
			ctx,
			task,
			fmt.Errorf("retrieve PDF: %w", err),
		)
	}

	result, submitErr := d.api.SubmitExtraction(
		ctx,
		task.OriginalFilename,
		task.Season,
		pdf,
	)

	closeErr := pdf.Close()

	if submitErr != nil {
		cause := fmt.Errorf("submit extraction: %w", submitErr)

		if closeErr != nil {
			cause = errors.Join(
				cause,
				fmt.Errorf("close PDF: %w", closeErr),
			)
		}

		if err := d.markUnknown(ctx, task, cause); err != nil {
			return errors.Join(cause, err)
		}

		return cause
	}

	if result.ID == "" {
		return d.markUnknown(
			ctx,
			task,
			fmt.Errorf("SOFI API returned an empty extraction ID"),
		)
	}

	if err := d.repository.MarkSubmitted(
		ctx,
		task.ID,
		task.SubmissionToken,
		result.ID,
	); err != nil {
		cause := fmt.Errorf("persist submitted extraction: %w", err)

		if markErr := d.markUnknown(ctx, task, cause); markErr != nil {
			return errors.Join(cause, markErr)
		}

		return cause
	}

	return nil
}

func (d *ExtractionDispatcher) markFailed(
	ctx context.Context,
	task database.ExtractionTask,
	cause error,
) error {
	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
	defer cancel()

	if err := d.repository.MarkSubmissionFailed(
		cleanupCtx,
		task.ID,
		task.SubmissionToken,
		cause,
	); err != nil {
		return errors.Join(
			cause,
			fmt.Errorf("mark extraction failed: %w", err),
		)
	}

	return cause
}

func (d *ExtractionDispatcher) markUnknown(
	ctx context.Context,
	task database.ExtractionTask,
	cause error,
) error {
	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
	defer cancel()

	if err := d.repository.MarkSubmissionUnknown(
		cleanupCtx,
		task.ID,
		task.SubmissionToken,
		cause,
	); err != nil {
		return fmt.Errorf("mark submission unknown: %w", err)
	}

	return nil
}
