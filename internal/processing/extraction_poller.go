package processing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/sofi"
)

type ExtractionPollRepository interface {
	ClaimDueForPolling(
		ctx context.Context,
		limit int,
	) ([]database.ExtractionPollTask, error)

	MarkPollPending(
		ctx context.Context,
		id string,
		token string,
		status string,
		nextPollAfter time.Duration,
	) error

	MarkPollRetry(
		ctx context.Context,
		id string,
		token string,
		cause error,
		retryAfter time.Duration,
	) error

	MarkPollCompleted(
		ctx context.Context,
		id string,
		token string,
		result json.RawMessage,
		validation json.RawMessage,
		completedAt *time.Time,
	) error

	MarkPollFailed(
		ctx context.Context,
		id string,
		token string,
		cause error,
	) error

	RecoverStalePolls(
		ctx context.Context,
		lease time.Duration,
	) (int64, error)
}

type ExtractionPollAPI interface {
	GetExtraction(
		ctx context.Context,
		id string,
	) (sofi.ExtractionStatus, error)
}

type ExtractionPollerConfig struct {
	BatchSize     int
	Lease         time.Duration
	PollInterval  time.Duration
	RetryInterval time.Duration
}

type ExtractionPoller struct {
	repository ExtractionPollRepository
	api        ExtractionPollAPI
	config     ExtractionPollerConfig
}

func NewExtractionPoller(
	repository ExtractionPollRepository,
	api ExtractionPollAPI,
	config ExtractionPollerConfig,
) (*ExtractionPoller, error) {
	if repository == nil {
		return nil, fmt.Errorf("extraction poll repository is required")
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

	if config.PollInterval <= 0 {
		config.PollInterval = 15 * time.Second
	}

	if config.RetryInterval <= 0 {
		config.RetryInterval = 30 * time.Second
	}

	return &ExtractionPoller{
		repository: repository,
		api:        api,
		config:     config,
	}, nil
}

func (p *ExtractionPoller) ProcessBatch(
	ctx context.Context,
) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	_, err := p.repository.RecoverStalePolls(
		ctx,
		p.config.Lease,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale extraction polls: %w", err)
	}

	tasks, err := p.repository.ClaimDueForPolling(
		ctx,
		p.config.BatchSize,
	)
	if err != nil {
		return 0, fmt.Errorf("claim due extraction polls: %w", err)
	}

	var (
		processed int
		failures  []error
	)

	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return processed, errors.Join(
				errors.Join(failures...),
				err,
			)
		}

		if err := p.poll(ctx, task); err != nil {
			failures = append(failures, err)
			continue
		}

		processed++
	}

	return processed, errors.Join(failures...)
}

func (p *ExtractionPoller) poll(
	ctx context.Context,
	task database.ExtractionPollTask,
) error {
	if strings.TrimSpace(task.ID) == "" ||
		strings.TrimSpace(task.ExternalID) == "" ||
		strings.TrimSpace(task.PollToken) == "" {
		return fmt.Errorf(
			"extraction polling task has incomplete metadata: %s",
			task.ID,
		)
	}

	result, err := p.api.GetExtraction(ctx, task.ExternalID)
	if err != nil {
		cause := fmt.Errorf("query SOFI extraction %s: %w", task.ExternalID, err)

		if persistErr := p.markRetry(ctx, task, cause); persistErr != nil {
			return errors.Join(cause, persistErr)
		}

		return cause
	}

	switch result.Status {
	case "pending":
		return p.markPending(ctx, task, "submitted")

	case "processing":
		return p.markPending(ctx, task, "processing")

	case "completed":
		return p.markCompleted(ctx, task, result)

	case "failed":
		cause := fmt.Errorf(
			"SOFI extraction failed: %s",
			errorCode(result.ErrorCode),
		)

		return p.markFailed(ctx, task, cause)

	default:
		cause := fmt.Errorf(
			"SOFI API returned unknown extraction status %q",
			result.Status,
		)

		if persistErr := p.markRetry(ctx, task, cause); persistErr != nil {
			return errors.Join(cause, persistErr)
		}

		return cause
	}
}

func (p *ExtractionPoller) markPending(
	ctx context.Context,
	task database.ExtractionPollTask,
	status string,
) error {
	persistCtx, cancel := p.cleanupContext(ctx)
	defer cancel()

	if err := p.repository.MarkPollPending(
		persistCtx,
		task.ID,
		task.PollToken,
		status,
		p.config.PollInterval,
	); err != nil {
		return fmt.Errorf("schedule next extraction poll: %w", err)
	}

	return nil
}

func (p *ExtractionPoller) markCompleted(
	ctx context.Context,
	task database.ExtractionPollTask,
	result sofi.ExtractionStatus,
) error {
	persistCtx, cancel := p.cleanupContext(ctx)
	defer cancel()

	if err := p.repository.MarkPollCompleted(
		persistCtx,
		task.ID,
		task.PollToken,
		result.Result,
		result.Validation,
		result.CompletedAt,
	); err != nil {
		return fmt.Errorf("persist completed extraction: %w", err)
	}

	return nil
}

func (p *ExtractionPoller) markFailed(
	ctx context.Context,
	task database.ExtractionPollTask,
	cause error,
) error {
	persistCtx, cancel := p.cleanupContext(ctx)
	defer cancel()

	if err := p.repository.MarkPollFailed(
		persistCtx,
		task.ID,
		task.PollToken,
		cause,
	); err != nil {
		return errors.Join(
			cause,
			fmt.Errorf("persist failed extraction: %w", err),
		)
	}

	return nil
}

func (p *ExtractionPoller) markRetry(
	ctx context.Context,
	task database.ExtractionPollTask,
	cause error,
) error {
	persistCtx, cancel := p.cleanupContext(ctx)
	defer cancel()

	if err := p.repository.MarkPollRetry(
		persistCtx,
		task.ID,
		task.PollToken,
		cause,
		p.config.RetryInterval,
	); err != nil {
		return fmt.Errorf("schedule extraction poll retry: %w", err)
	}

	return nil
}

func (p *ExtractionPoller) cleanupContext(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
}

func errorCode(code *string) string {
	if code == nil || strings.TrimSpace(*code) == "" {
		return "unspecified error"
	}

	return *code
}
