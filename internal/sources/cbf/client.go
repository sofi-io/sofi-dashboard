package cbf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxHTMLSize = 10 << 20 // 10 MB

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

func (c *Client) FetchHTML(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		targetURL,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "SOFI-Dashboard/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch CBF page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CBF returned HTTP %d for %s",
			resp.StatusCode, targetURL)
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
