package processing

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/sofi"
	"github.com/sofi-io/sofi-dashboard/internal/storage"
)

type fakeExtractionRepository struct {
	tasks []database.ExtractionTask

	submittedID string
	submitted   int
	unknown     int
	failed      int

	submitErr  error
	unknownErr error
	failedErr  error

	recovered bool
}

func (r *fakeExtractionRepository) ClaimPending(
	_ context.Context,
	_ int,
) ([]database.ExtractionTask, error) {
	return r.tasks, nil
}

func (r *fakeExtractionRepository) MarkSubmitted(
	_ context.Context,
	_ string,
	_ string,
	externalID string,
) error {
	r.submitted++
	r.submittedID = externalID
	return r.submitErr
}

func (r *fakeExtractionRepository) MarkSubmissionUnknown(
	_ context.Context,
	_ string,
	_ string,
	_ error,
) error {
	r.unknown++
	return r.unknownErr
}

func (r *fakeExtractionRepository) MarkSubmissionFailed(
	_ context.Context,
	_ string,
	_ string,
	_ error,
) error {
	r.failed++
	return r.failedErr
}

func (r *fakeExtractionRepository) RecoverStale(
	_ context.Context,
	_ time.Duration,
) (int64, error) {
	r.recovered = true
	return 0, nil
}

type fakeExtractionStorage struct {
	content string
	getErr  error
}

func (s *fakeExtractionStorage) Get(
	_ context.Context,
	_ string,
) (io.ReadCloser, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	return io.NopCloser(strings.NewReader(s.content)), nil
}

func (s *fakeExtractionStorage) Put(
	_ context.Context,
	key string,
	_ io.Reader,
) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{Key: key}, nil
}

func (s *fakeExtractionStorage) Delete(
	_ context.Context,
	_ string,
) error {
	return nil
}

func (s *fakeExtractionStorage) Exists(
	_ context.Context,
	_ string,
) (bool, error) {
	return true, nil
}

type fakeExtractionAPI struct {
	result sofi.Extraction
	err    error

	called   int
	filename string
	season   string
	content  string
}

func (a *fakeExtractionAPI) SubmitExtraction(
	_ context.Context,
	filename string,
	season string,
	pdf io.Reader,
) (sofi.Extraction, error) {
	a.called++
	a.filename = filename
	a.season = season

	data, err := io.ReadAll(pdf)
	if err != nil {
		return sofi.Extraction{}, err
	}

	a.content = string(data)

	return a.result, a.err
}

func newExtractionDispatcherTest(
	t *testing.T,
	repo *fakeExtractionRepository,
	store *fakeExtractionStorage,
	api *fakeExtractionAPI,
) *ExtractionDispatcher {
	t.Helper()

	dispatcher, err := NewExtractionDispatcher(
		repo,
		store,
		api,
		ExtractionDispatcherConfig{
			BatchSize: 5,
			Lease:     time.Minute,
		},
	)
	if err != nil {
		t.Fatalf("NewExtractionDispatcher(): %v", err)
	}

	return dispatcher
}

func testExtractionTask() database.ExtractionTask {
	return database.ExtractionTask{
		ID:                 "task-123",
		DocumentID:         "document-123",
		StorageKey:         "documents/test.pdf",
		OriginalFilename:   "bordero.pdf",
		Season:             "2026",
		SubmissionToken:    "token-123",
		SubmissionAttempts: 1,
	}
}

func TestExtractionDispatcherSuccessfulSubmission(t *testing.T) {
	repo := &fakeExtractionRepository{
		tasks: []database.ExtractionTask{
			testExtractionTask(),
		},
	}

	store := &fakeExtractionStorage{
		content: "%PDF-1.7 test document",
	}

	api := &fakeExtractionAPI{
		result: sofi.Extraction{
			ID:     "external-123",
			Status: "pending",
		},
	}

	dispatcher := newExtractionDispatcherTest(
		t,
		repo,
		store,
		api,
	)

	processed, err := dispatcher.ProcessBatch(
		context.Background(),
	)
	if err != nil {
		t.Fatalf("ProcessBatch(): %v", err)
	}

	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	if !repo.recovered {
		t.Error("expected stale submissions recovery")
	}

	if api.called != 1 {
		t.Errorf("API calls = %d, want 1", api.called)
	}

	if api.filename != "bordero.pdf" {
		t.Errorf("filename = %q", api.filename)
	}

	if api.season != "2026" {
		t.Errorf("season = %q", api.season)
	}

	if api.content != store.content {
		t.Error("PDF content was not forwarded correctly")
	}

	if repo.submitted != 1 {
		t.Errorf("submitted = %d, want 1", repo.submitted)
	}

	if repo.submittedID != "external-123" {
		t.Errorf("submitted ID = %q", repo.submittedID)
	}

	if repo.unknown != 0 || repo.failed != 0 {
		t.Error("unexpected failure state")
	}
}

func TestExtractionDispatcherStorageFailure(t *testing.T) {
	repo := &fakeExtractionRepository{
		tasks: []database.ExtractionTask{
			testExtractionTask(),
		},
	}

	store := &fakeExtractionStorage{
		getErr: errors.New("file not found"),
	}

	api := &fakeExtractionAPI{}

	dispatcher := newExtractionDispatcherTest(
		t,
		repo,
		store,
		api,
	)

	processed, err := dispatcher.ProcessBatch(
		context.Background(),
	)

	if err == nil {
		t.Fatal("expected storage error")
	}

	if processed != 0 {
		t.Errorf("processed = %d, want 0", processed)
	}

	if repo.failed != 1 {
		t.Errorf("failed = %d, want 1", repo.failed)
	}

	if repo.unknown != 0 {
		t.Errorf("unknown = %d, want 0", repo.unknown)
	}

	if api.called != 0 {
		t.Errorf("API calls = %d, want 0", api.called)
	}
}

func TestExtractionDispatcherAPIError(t *testing.T) {
	repo := &fakeExtractionRepository{
		tasks: []database.ExtractionTask{
			testExtractionTask(),
		},
	}

	store := &fakeExtractionStorage{
		content: "%PDF-1.7 test document",
	}

	api := &fakeExtractionAPI{
		err: errors.New("connection timeout"),
	}

	dispatcher := newExtractionDispatcherTest(
		t,
		repo,
		store,
		api,
	)

	processed, err := dispatcher.ProcessBatch(
		context.Background(),
	)

	if err == nil {
		t.Fatal("expected submission error")
	}

	if processed != 0 {
		t.Errorf("processed = %d, want 0", processed)
	}

	if repo.unknown != 1 {
		t.Errorf("unknown = %d, want 1", repo.unknown)
	}

	if repo.failed != 0 {
		t.Errorf("failed = %d, want 0", repo.failed)
	}

	if repo.submitted != 0 {
		t.Errorf("submitted = %d, want 0", repo.submitted)
	}
}

func TestExtractionDispatcherPersistenceFailure(t *testing.T) {
	repo := &fakeExtractionRepository{
		tasks: []database.ExtractionTask{
			testExtractionTask(),
		},
		submitErr: errors.New("database unavailable"),
	}

	store := &fakeExtractionStorage{
		content: "%PDF-1.7 test document",
	}

	api := &fakeExtractionAPI{
		result: sofi.Extraction{
			ID:     "external-123",
			Status: "pending",
		},
	}

	dispatcher := newExtractionDispatcherTest(
		t,
		repo,
		store,
		api,
	)

	processed, err := dispatcher.ProcessBatch(
		context.Background(),
	)

	if err == nil {
		t.Fatal("expected persistence error")
	}

	if processed != 0 {
		t.Errorf("processed = %d, want 0", processed)
	}

	if repo.submitted != 1 {
		t.Errorf("submitted attempts = %d, want 1", repo.submitted)
	}

	if repo.unknown != 1 {
		t.Errorf("unknown = %d, want 1", repo.unknown)
	}

	if repo.failed != 0 {
		t.Errorf("failed = %d, want 0", repo.failed)
	}
}
