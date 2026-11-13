package processing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/database"
	"github.com/sofi-io/sofi-dashboard/internal/document"
	"github.com/sofi-io/sofi-dashboard/internal/storage"
)

type fakeRepository struct {
	documents []database.Document

	claimed    int
	recovered  int
	downloaded int
	failed     int

	lastError      error
	lastToken      string
	lastStorageKey string
}

func (r *fakeRepository) ClaimPending(
	_ context.Context,
	limit int,
) ([]database.Document, error) {
	r.claimed++

	if len(r.documents) > limit {
		return r.documents[:limit], nil
	}

	return r.documents, nil
}

func (r *fakeRepository) MarkDownloaded(
	_ context.Context,
	id string,
	token string,
	contentType string,
	sizeBytes int64,
	sha256 string,
	storageKey string,
) error {
	r.downloaded++
	r.lastToken = token
	r.lastStorageKey = storageKey
	return nil
}

func (r *fakeRepository) MarkFailed(
	_ context.Context,
	id string,
	token string,
	cause error,
) error {
	r.failed++
	r.lastToken = token
	r.lastError = cause
	return nil
}

func (r *fakeRepository) RecoverStale(
	_ context.Context,
	_ time.Duration,
) (int64, error) {
	r.recovered++
	return 0, nil
}

func newProcessorTest(
	t *testing.T,
	handler http.HandlerFunc,
	docs []database.Document,
) (*DocumentProcessor, *fakeRepository, *storage.FilesystemStorage, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{
			MaxAttempts:  1,
			InitialDelay: time.Millisecond,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	downloaderClient := server.Client()

	repository := &fakeRepository{
		documents: docs,
	}

	processor, err := NewDocumentProcessor(
		repository,
		downloader,
		ProcessorConfig{
			BatchSize: 5,
			Lease:     time.Minute,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Replace the downloader HTTP client through its
	// package-level test helper in the next revision.
	_ = downloaderClient

	return processor, repository, store, server
}

func testDocument(url string) database.Document {
	token := "test-download-token"

	return database.Document{
		ID:            "document-1",
		SourceKey:     "cbf",
		SourceURL:     url,
		DocumentType:  "financial_bulletin",
		Status:        "downloading",
		DownloadToken: &token,
	}
}

func TestProcessBatchDownloadsDocument(t *testing.T) {
	pdf := "%PDF-1.7\nSOFI test document"

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			fmt.Fprint(w, pdf)
		},
	))
	defer server.Close()

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{
			MaxAttempts: 1,
			HTTPClient:  server.Client(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	repository := &fakeRepository{
		documents: []database.Document{
			testDocument(server.URL + "/document.pdf"),
		},
	}

	processor, err := NewDocumentProcessor(
		repository,
		downloader,
		ProcessorConfig{BatchSize: 5},
	)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if processed != 1 {
		t.Fatalf("expected 1 processed document, got %d", processed)
	}

	if repository.downloaded != 1 {
		t.Fatalf("expected 1 successful update, got %d", repository.downloaded)
	}

	if repository.failed != 0 {
		t.Fatalf("expected no failures, got %d", repository.failed)
	}

	if repository.lastToken != "test-download-token" {
		t.Fatalf("unexpected token: %s", repository.lastToken)
	}

	file, err := store.Get(context.Background(), repository.lastStorageKey)
	if err != nil {
		t.Fatalf("stored PDF not found: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, len(pdf))
	if _, err := file.Read(buffer); err != nil {
		t.Fatal(err)
	}

	if string(buffer) != pdf {
		t.Fatal("stored PDF content does not match original")
	}
}

func TestProcessBatchMarksInvalidPDFAsFailedAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			fmt.Fprint(w, "not a PDF")
		},
	))
	defer server.Close()

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{
			MaxAttempts: 1,
			HTTPClient:  server.Client(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	repository := &fakeRepository{
		documents: []database.Document{
			testDocument(server.URL + "/invalid.pdf"),
		},
	}

	processor, err := NewDocumentProcessor(
		repository,
		downloader,
		ProcessorConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.ProcessBatch(context.Background())

	if processed != 0 {
		t.Fatalf("expected 0 processed documents, got %d", processed)
	}

	if err == nil {
		t.Fatal("expected processing error")
	}

	if repository.failed != 1 {
		t.Fatalf("expected 1 failure update, got %d", repository.failed)
	}

	if repository.downloaded != 0 {
		t.Fatal("invalid PDF must not be marked downloaded")
	}
}

func TestProcessBatchContinuesAfterDocumentFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")

			if strings.Contains(r.URL.Path, "invalid") {
				fmt.Fprint(w, "invalid PDF")
				return
			}

			fmt.Fprint(w, "%PDF-1.7\nvalid document")
		},
	))
	defer server.Close()

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{
			MaxAttempts: 1,
			HTTPClient:  server.Client(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	first := testDocument(server.URL + "/invalid.pdf")
	second := testDocument(server.URL + "/valid.pdf")
	second.ID = "document-2"

	repository := &fakeRepository{
		documents: []database.Document{first, second},
	}

	processor, err := NewDocumentProcessor(
		repository,
		downloader,
		ProcessorConfig{BatchSize: 5},
	)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.ProcessBatch(context.Background())

	if processed != 1 {
		t.Fatalf("expected 1 successful document, got %d", processed)
	}

	if err == nil {
		t.Fatal("expected one document failure")
	}

	if repository.failed != 1 || repository.downloaded != 1 {
		t.Fatalf(
			"unexpected state: failed=%d downloaded=%d",
			repository.failed,
			repository.downloaded,
		)
	}
}

func TestProcessBatchRejectsMissingToken(t *testing.T) {
	repository := &fakeRepository{
		documents: []database.Document{
			{
				ID:        "document-1",
				SourceURL: "https://example.com/document.pdf",
			},
		},
	}

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}

	processor, err := NewDocumentProcessor(
		repository,
		downloader,
		ProcessorConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.ProcessBatch(context.Background())

	if processed != 0 {
		t.Fatal("expected no processed documents")
	}

	if err == nil {
		t.Fatal("expected missing token error")
	}

	if repository.downloaded != 0 || repository.failed != 0 {
		t.Fatal("document without token must not be updated")
	}
}

func TestProcessBatchHonorsCancellation(t *testing.T) {
	repository := &fakeRepository{}

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := document.NewDownloader(
		store,
		document.DownloaderConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}

	processor, err := NewDocumentProcessor(
		repository,
		downloader,
		ProcessorConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processed, err := processor.ProcessBatch(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if processed != 0 {
		t.Fatal("expected no processed documents")
	}

	if repository.claimed != 0 {
		t.Fatal("cancelled processor must not claim documents")
	}
}
