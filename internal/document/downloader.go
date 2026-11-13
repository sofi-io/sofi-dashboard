package document

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sofi-io/sofi-dashboard/internal/storage"
)

const (
	defaultTimeout  = 30 * time.Second
	defaultMaxBytes = 20 << 20 // 20 MB
	defaultAttempts = 3
	defaultBackoff  = time.Second
)

type DownloaderConfig struct {
	Timeout      time.Duration
	MaxBytes     int64
	MaxAttempts  int
	InitialDelay time.Duration
}

type DownloadedDocument struct {
	URL          string
	StorageKey   string
	ContentType  string
	SizeBytes    int64
	SHA256       string
	DownloadedAt time.Time
}

type Downloader struct {
	client  *http.Client
	storage storage.Storage
	config  DownloaderConfig
}

func NewDownloader(
	store storage.Storage,
	config DownloaderConfig,
) (*Downloader, error) {
	if store == nil {
		return nil, fmt.Errorf("storage is required")
	}

	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}
	if config.MaxBytes <= 0 {
		config.MaxBytes = defaultMaxBytes
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = defaultAttempts
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = defaultBackoff
	}

	if config.MaxAttempts > 5 {
		return nil, fmt.Errorf("max attempts cannot exceed 5")
	}

	return &Downloader{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		storage: store,
		config:  config,
	}, nil
}

func (d *Downloader) Download(
	ctx context.Context,
	rawURL string,
) (DownloadedDocument, error) {
	if err := validateURL(rawURL); err != nil {
		return DownloadedDocument{}, err
	}

	var lastErr error

	for attempt := 1; attempt <= d.config.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return DownloadedDocument{}, err
		}

		result, retry, err := d.downloadOnce(ctx, rawURL)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !retry || attempt == d.config.MaxAttempts {
			break
		}

		delay := d.config.InitialDelay *
			time.Duration(1<<(attempt-1))

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return DownloadedDocument{}, ctx.Err()
		case <-timer.C:
		}
	}

	return DownloadedDocument{}, fmt.Errorf(
		"download failed after %d attempts: %w",
		d.config.MaxAttempts,
		lastErr,
	)
}

func (d *Downloader) downloadOnce(
	ctx context.Context,
	rawURL string,
) (DownloadedDocument, bool, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return DownloadedDocument{}, false, err
	}

	req.Header.Set("User-Agent", "SOFI-DocumentDownloader/1.0")
	req.Header.Set("Accept", "application/pdf,application/octet-stream")

	resp, err := d.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return DownloadedDocument{}, false, ctx.Err()
		}
		return DownloadedDocument{}, true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests ||
		resp.StatusCode >= 500 {
		return DownloadedDocument{}, true, fmt.Errorf(
			"temporary HTTP error: %s",
			resp.Status,
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DownloadedDocument{}, false, fmt.Errorf(
			"unexpected HTTP status: %s",
			resp.Status,
		)
	}

	if resp.ContentLength > d.config.MaxBytes {
		return DownloadedDocument{}, false, fmt.Errorf(
			"document exceeds maximum size: %d bytes",
			resp.ContentLength,
		)
	}

	contentType, err := validateContentType(
		resp.Header.Get("Content-Type"),
	)
	if err != nil {
		return DownloadedDocument{}, false, err
	}

	reader := bufio.NewReader(resp.Body)

	if err := validatePDF(reader); err != nil {
		return DownloadedDocument{}, false, err
	}

	// Hash the URL to generate a deterministic storage key.
	urlHash := sha256.Sum256([]byte(rawURL))
	key := "documents/" + hex.EncodeToString(urlHash[:]) + ".pdf"

	limitedReader := io.LimitReader(
		reader,
		d.config.MaxBytes+1,
	)

	info, err := d.storage.Put(ctx, key, limitedReader)
	if err != nil {
		return DownloadedDocument{}, false, fmt.Errorf(
			"store document: %w",
			err,
		)
	}

	if info.Size > d.config.MaxBytes {
		_ = d.storage.Delete(context.Background(), key)

		return DownloadedDocument{}, false, fmt.Errorf(
			"document exceeds maximum size: %d bytes",
			info.Size,
		)
	}

	if err := ctx.Err(); err != nil {
		_ = d.storage.Delete(context.Background(), key)
		return DownloadedDocument{}, false, err
	}

	return DownloadedDocument{
		URL:          rawURL,
		StorageKey:   info.Key,
		ContentType:  contentType,
		SizeBytes:    info.Size,
		SHA256:       info.SHA256,
		DownloadedAt: time.Now().UTC(),
	}, false, nil
}

func validateURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid document URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	if u.Host == "" || u.User != nil {
		return fmt.Errorf("invalid document URL host or credentials")
	}

	if strings.TrimSpace(rawURL) == "" {
		return fmt.Errorf("document URL is required")
	}

	return nil
}
