package processing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/sofi"
)

type fakeExtractionPollRepository struct {
	tasks []database.ExtractionPollTask

	recovered int
	claimed   int

	pending   int
	retried   int
	completed int
	failed    int

	lastID          string
	lastToken       string
	lastStatus      string
	lastResult      json.RawMessage
	lastValidation  json.RawMessage
	lastCompletedAt *time.Time
	lastCause       error
	lastDelay       time.Duration

	recoverErr   error
	claimErr     error
	pendingErr   error
	retryErr     error
	completedErr error
	failedErr    error
}

func (r *fakeExtractionPollRepository) RecoverStalePolls(
	_ context.Context,
	_ time.Duration,
) (int64, error) {
	r.recovered++
	return 0, r.recoverErr
}

func (r *fakeExtractionPollRepository) ClaimDueForPolling(
	_ context.Context,
	limit int,
) ([]database.ExtractionPollTask, error) {
	r.claimed++
	if limit <= 0 {
		return nil, errors.New("invalid limit")
	}
	return r.tasks, r.claimErr
}

func (r *fakeExtractionPollRepository) MarkPollPending(
	_ context.Context,
	id string,
	token string,
	status string,
	delay time.Duration,
) error {
	r.pending++
	r.lastID = id
	r.lastToken = token
	r.lastStatus = status
	r.lastDelay = delay
	return r.pendingErr
}

func (r *fakeExtractionPollRepository) MarkPollRetry(
	_ context.Context,
	id string,
	token string,
	cause error,
	delay time.Duration,
) error {
	r.retried++
	r.lastID = id
	r.lastToken = token
	r.lastCause = cause
	r.lastDelay = delay
	return r.retryErr
}

func (r *fakeExtractionPollRepository) MarkPollCompleted(
	_ context.Context,
	id string,
	token string,
	result json.RawMessage,
	validation json.RawMessage,
	completedAt *time.Time,
) error {
	r.completed++
	r.lastID = id
	r.lastToken = token
	r.lastResult = result
	r.lastValidation = validation
	r.lastCompletedAt = completedAt
	return r.completedErr
}

func (r *fakeExtractionPollRepository) MarkPollFailed(
	_ context.Context,
	id string,
	token string,
	cause error,
) error {
	r.failed++
	r.lastID = id
	r.lastToken = token
	r.lastCause = cause
	return r.failedErr
}

type fakeExtractionPollAPI struct {
	result sofi.ExtractionStatus
	err    error
	called int
	lastID string
}

func (a *fakeExtractionPollAPI) GetExtraction(
	_ context.Context,
	id string,
) (sofi.ExtractionStatus, error) {
	a.called++
	a.lastID = id
	return a.result, a.err
}

func newExtractionPollerTest(
	t *testing.T,
	repo *fakeExtractionPollRepository,
	api *fakeExtractionPollAPI,
) *ExtractionPoller {
	t.Helper()

	poller, err := NewExtractionPoller(
		repo,
		api,
		ExtractionPollerConfig{
			BatchSize:     5,
			Lease:         time.Minute,
			PollInterval:  15 * time.Second,
			RetryInterval: 30 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf("NewExtractionPoller(): %v", err)
	}

	return poller
}

func testExtractionPollTask() database.ExtractionPollTask {
	return database.ExtractionPollTask{
		ID:         "task-123",
		ExternalID: "external-123",
		PollToken:  "poll-token-123",
	}
}

func TestExtractionPollerRemoteStatuses(t *testing.T) {
	completedAt := time.Date(
		2026, 11, 13, 20, 0, 0, 0, time.UTC,
	)

	tests := []struct {
		name          string
		status        string
		errorCode     *string
		wantPending   int
		wantCompleted int
		wantFailed    int
		wantStatus    string
	}{
		{
			name:        "pending",
			status:      "pending",
			wantPending: 1,
			wantStatus:  "submitted",
		},
		{
			name:        "processing",
			status:      "processing",
			wantPending: 1,
			wantStatus:  "processing",
		},
		{
			name:          "completed",
			status:        "completed",
			wantCompleted: 1,
		},
		{
			name:       "failed",
			status:     "failed",
			errorCode:  stringPointer("invalid_document"),
			wantFailed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeExtractionPollRepository{
				tasks: []database.ExtractionPollTask{
					testExtractionPollTask(),
				},
			}

			api := &fakeExtractionPollAPI{
				result: sofi.ExtractionStatus{
					ID:          "external-123",
					Status:      tt.status,
					ErrorCode:   tt.errorCode,
					Result:      json.RawMessage(`{"amount":1000}`),
					Validation:  json.RawMessage(`{"valid":true}`),
					CompletedAt: &completedAt,
				},
			}

			poller := newExtractionPollerTest(t, repo, api)

			processed, err := poller.ProcessBatch(
				context.Background(),
			)

			if err != nil {
				t.Fatalf("ProcessBatch(): %v", err)
			}

			if processed != 1 {
				t.Fatalf("processed = %d, want 1", processed)
			}

			if repo.recovered != 1 {
				t.Errorf("recovered = %d, want 1", repo.recovered)
			}

			if api.called != 1 {
				t.Errorf("API calls = %d, want 1", api.called)
			}

			if api.lastID != "external-123" {
				t.Errorf("external ID = %q", api.lastID)
			}

			if repo.pending != tt.wantPending {
				t.Errorf(
					"pending = %d, want %d",
					repo.pending,
					tt.wantPending,
				)
			}

			if repo.completed != tt.wantCompleted {
				t.Errorf(
					"completed = %d, want %d",
					repo.completed,
					tt.wantCompleted,
				)
			}

			if repo.failed != tt.wantFailed {
				t.Errorf(
					"failed = %d, want %d",
					repo.failed,
					tt.wantFailed,
				)
			}

			if tt.wantStatus != "" &&
				repo.lastStatus != tt.wantStatus {
				t.Errorf(
					"status = %q, want %q",
					repo.lastStatus,
					tt.wantStatus,
				)
			}

			if tt.status == "completed" {
				if string(repo.lastResult) != `{"amount":1000}` {
					t.Errorf("result = %s", repo.lastResult)
				}

				if string(repo.lastValidation) != `{"valid":true}` {
					t.Errorf("validation = %s", repo.lastValidation)
				}

				if repo.lastCompletedAt == nil ||
					!repo.lastCompletedAt.Equal(completedAt) {
					t.Error("completed_at was not preserved")
				}
			}

			if tt.status == "failed" {
				if repo.lastCause == nil {
					t.Fatal("expected failure cause")
				}

				if !containsError(
					repo.lastCause,
					"invalid_document",
				) {
					t.Errorf("unexpected failure: %v", repo.lastCause)
				}
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func containsError(err error, expected string) bool {
	return err != nil &&
		len(expected) > 0 &&
		fmt.Sprint(err) != "" &&
		contains(fmt.Sprint(err), expected)
}

func contains(value, substring string) bool {
	return len(value) >= len(substring) &&
		func() bool {
			for i := 0; i <= len(value)-len(substring); i++ {
				if value[i:i+len(substring)] == substring {
					return true
				}
			}
			return false
		}()
}

func TestExtractionPollerRetriesAPIError(t *testing.T) {
	repo := &fakeExtractionPollRepository{
		tasks: []database.ExtractionPollTask{
			testExtractionPollTask(),
		},
	}

	api := &fakeExtractionPollAPI{
		err: errors.New("connection timeout"),
	}

	poller := newExtractionPollerTest(t, repo, api)

	processed, err := poller.ProcessBatch(context.Background())

	if err == nil {
		t.Fatal("expected polling error")
	}

	if processed != 0 {
		t.Errorf("processed = %d, want 0", processed)
	}

	if repo.retried != 1 {
		t.Errorf("retried = %d, want 1", repo.retried)
	}

	if repo.failed != 0 {
		t.Errorf("failed = %d, want 0", repo.failed)
	}

	if repo.lastDelay != 30*time.Second {
		t.Errorf("retry delay = %s", repo.lastDelay)
	}

	if !containsError(repo.lastCause, "connection timeout") {
		t.Errorf("unexpected retry cause: %v", repo.lastCause)
	}
}

func TestExtractionPollerUnknownStatusRetries(t *testing.T) {
	repo := &fakeExtractionPollRepository{
		tasks: []database.ExtractionPollTask{
			testExtractionPollTask(),
		},
	}

	api := &fakeExtractionPollAPI{
		result: sofi.ExtractionStatus{
			ID:     "external-123",
			Status: "unexpected",
		},
	}

	poller := newExtractionPollerTest(t, repo, api)

	_, err := poller.ProcessBatch(context.Background())

	if err == nil {
		t.Fatal("expected unknown status error")
	}

	if repo.retried != 1 {
		t.Errorf("retried = %d, want 1", repo.retried)
	}

	if repo.completed != 0 || repo.failed != 0 {
		t.Error("unknown status must not finalize extraction")
	}
}

func TestExtractionPollerPersistenceErrors(t *testing.T) {
	tests := []struct {
		name string
		api  sofi.ExtractionStatus
		set  func(*fakeExtractionPollRepository)
	}{
		{
			name: "pending",
			api: sofi.ExtractionStatus{
				ID:     "external-123",
				Status: "pending",
			},
			set: func(r *fakeExtractionPollRepository) {
				r.pendingErr = errors.New("database unavailable")
			},
		},
		{
			name: "completed",
			api: sofi.ExtractionStatus{
				ID:     "external-123",
				Status: "completed",
			},
			set: func(r *fakeExtractionPollRepository) {
				r.completedErr = errors.New("database unavailable")
			},
		},
		{
			name: "failed",
			api: sofi.ExtractionStatus{
				ID:     "external-123",
				Status: "failed",
			},
			set: func(r *fakeExtractionPollRepository) {
				r.failedErr = errors.New("database unavailable")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeExtractionPollRepository{
				tasks: []database.ExtractionPollTask{
					testExtractionPollTask(),
				},
			}

			tt.set(repo)

			api := &fakeExtractionPollAPI{
				result: tt.api,
			}

			poller := newExtractionPollerTest(t, repo, api)

			_, err := poller.ProcessBatch(context.Background())

			if err == nil {
				t.Fatal("expected persistence error")
			}
		})
	}
}

func TestExtractionPollerRejectsIncompleteTask(t *testing.T) {
	tests := []struct {
		name string
		task database.ExtractionPollTask
	}{
		{
			name: "missing ID",
			task: database.ExtractionPollTask{
				ExternalID: "external-123",
				PollToken:  "token-123",
			},
		},
		{
			name: "missing external ID",
			task: database.ExtractionPollTask{
				ID:        "task-123",
				PollToken: "token-123",
			},
		},
		{
			name: "missing poll token",
			task: database.ExtractionPollTask{
				ID:         "task-123",
				ExternalID: "external-123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeExtractionPollRepository{
				tasks: []database.ExtractionPollTask{tt.task},
			}

			api := &fakeExtractionPollAPI{}

			poller := newExtractionPollerTest(t, repo, api)

			_, err := poller.ProcessBatch(context.Background())

			if err == nil {
				t.Fatal("expected incomplete task error")
			}

			if api.called != 0 {
				t.Error("API should not be called")
			}
		})
	}
}

func TestExtractionPollerCancelledContext(t *testing.T) {
	repo := &fakeExtractionPollRepository{
		tasks: []database.ExtractionPollTask{
			testExtractionPollTask(),
		},
	}

	api := &fakeExtractionPollAPI{}

	poller := newExtractionPollerTest(t, repo, api)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processed, err := poller.ProcessBatch(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}

	if processed != 0 {
		t.Errorf("processed = %d, want 0", processed)
	}

	if repo.recovered != 0 {
		t.Error("should not recover leases after cancellation")
	}

	if api.called != 0 {
		t.Error("API should not be called")
	}
}

func TestExtractionPollerRecoveryError(t *testing.T) {
	repo := &fakeExtractionPollRepository{
		recoverErr: errors.New("database unavailable"),
	}

	api := &fakeExtractionPollAPI{}

	poller := newExtractionPollerTest(t, repo, api)

	_, err := poller.ProcessBatch(context.Background())

	if err == nil {
		t.Fatal("expected recovery error")
	}

	if repo.claimed != 0 {
		t.Error("should not claim tasks after recovery failure")
	}

	if api.called != 0 {
		t.Error("API should not be called")
	}
}

func TestExtractionPollerClaimError(t *testing.T) {
	repo := &fakeExtractionPollRepository{
		claimErr: errors.New("database unavailable"),
	}

	api := &fakeExtractionPollAPI{}

	poller := newExtractionPollerTest(t, repo, api)

	_, err := poller.ProcessBatch(context.Background())

	if err == nil {
		t.Fatal("expected claim error")
	}

	if api.called != 0 {
		t.Error("API should not be called")
	}
}
