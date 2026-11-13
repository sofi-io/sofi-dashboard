package document

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/storage"
)

func newTestDownloader(t *testing.T, server *httptest.Server) *Downloader {
	t.Helper()

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := NewDownloader(store, DownloaderConfig{
		MaxBytes:     1024,
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	downloader.client = server.Client()

	return downloader
}

func TestDownloaderDownloadPDF(t *testing.T) {
	pdf := "%PDF-1.7\nSOFI test document"

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			fmt.Fprint(w, pdf)
		},
	))
	defer server.Close()

	downloader := newTestDownloader(t, server)

	result, err := downloader.Download(
		context.Background(),
		server.URL+"/document.pdf",
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.SizeBytes != int64(len(pdf)) {
		t.Fatalf("unexpected size: %d", result.SizeBytes)
	}

	if result.SHA256 == "" {
		t.Fatal("SHA256 should not be empty")
	}

	if !strings.HasPrefix(result.StorageKey, "documents/") {
		t.Fatalf("unexpected storage key: %s", result.StorageKey)
	}

	if result.ContentType != "application/pdf" {
		t.Fatalf("unexpected content type: %s", result.ContentType)
	}
}

func TestDownloaderRejectsInvalidPDF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			fmt.Fprint(w, "<html>Not a PDF</html>")
		},
	))
	defer server.Close()

	downloader := newTestDownloader(t, server)

	_, err := downloader.Download(
		context.Background(),
		server.URL+"/document.pdf",
	)

	if err == nil {
		t.Fatal("expected invalid PDF error")
	}
}

func TestDownloaderRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		},
	))
	defer server.Close()

	downloader := newTestDownloader(t, server)

	_, err := downloader.Download(
		context.Background(),
		server.URL+"/missing.pdf",
	)

	if err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestDownloaderRejectsUnsupportedContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html>Not a PDF</html>")
		},
	))
	defer server.Close()

	downloader := newTestDownloader(t, server)

	_, err := downloader.Download(
		context.Background(),
		server.URL+"/document.pdf",
	)

	if err == nil {
		t.Fatal("expected content type error")
	}
}

func TestDownloaderRejectsOversizedDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/pdf")
			w.Write([]byte("%PDF-1.7\n"))
			w.Write(make([]byte, 2048))
		},
	))
	defer server.Close()

	downloader := newTestDownloader(t, server)

	_, err := downloader.Download(
		context.Background(),
		server.URL+"/large.pdf",
	)

	if err == nil {
		t.Fatal("expected oversized document error")
	}
}

func TestDownloaderRejectsInvalidURL(t *testing.T) {
	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := NewDownloader(store, DownloaderConfig{
		MaxBytes:     1024,
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	invalidURLs := []string{
		"",
		"ftp://example.com/document.pdf",
		"javascript:alert(1)",
		"/relative/document.pdf",
	}

	for _, rawURL := range invalidURLs {
		t.Run(rawURL, func(t *testing.T) {
			_, err := downloader.Download(
				context.Background(),
				rawURL,
			)

			if err == nil {
				t.Fatalf("expected invalid URL error: %s", rawURL)
			}
		})
	}
}

func TestDownloaderRetry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempts++

			if attempts < 3 {
				http.Error(w, "temporary error", http.StatusServiceUnavailable)
				return
			}

			w.Header().Set("Content-Type", "application/pdf")
			fmt.Fprint(w, "%PDF-1.7\nvalid document")
		},
	))
	defer server.Close()

	store, err := storage.NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	downloader, err := NewDownloader(store, DownloaderConfig{
		MaxBytes:     1024,
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	downloader.client = server.Client()

	_, err = downloader.Download(
		context.Background(),
		server.URL+"/document.pdf",
	)

	if err != nil {
		t.Fatal(err)
	}

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
