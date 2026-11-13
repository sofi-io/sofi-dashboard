package cbf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxHTMLSize    = 10 << 20
	maxAttempts    = 4
	initialBackoff = 1 * time.Second
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		baseURL: "https://www.cbf.com.br",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) FetchHTML(
	ctx context.Context,
	targetURL string,
) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		body, err := c.fetchHTMLOnce(ctx, targetURL)
		if err == nil {
			return body, nil
		}

		lastErr = err

		if attempt == maxAttempts {
			break
		}

		delay := initialBackoff * time.Duration(1<<(attempt-1))

		if err := waitContext(ctx, delay); err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf(
		"fetch failed after %d attempts: %w",
		maxAttempts,
		lastErr,
	)
}

func (c *Client) fetchHTMLOnce(
	ctx context.Context,
	targetURL string,
) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		targetURL,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"unexpected HTTP status: %d",
			resp.StatusCode,
		)
	}

	body, err := io.ReadAll(
		io.LimitReader(resp.Body, maxHTMLSize+1),
	)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if len(body) > maxHTMLSize {
		return "", fmt.Errorf("HTML response exceeds size limit")
	}

	return string(body), nil
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
