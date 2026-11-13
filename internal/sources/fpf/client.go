package fpf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout  = 30 * time.Second
	maxResponseSize = 10 << 20 // 10 MB
	userAgent       = "Mozilla/5.0"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
			Jar:     jar,
		},
	}, nil
}

// GetHTML retrieves an HTML page while preserving the HTTP session.
func (c *Client) GetHTML(
	ctx context.Context,
	targetURL string,
	referer string,
) (string, error) {
	body, err := c.do(
		ctx,
		http.MethodGet,
		targetURL,
		referer,
		nil,
	)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// PostForm submits an application/x-www-form-urlencoded request.
// The response is returned as bytes because FPF endpoints may
// return either HTML or JSON.
func (c *Client) PostForm(
	ctx context.Context,
	targetURL string,
	referer string,
	form url.Values,
) ([]byte, error) {
	var body io.Reader

	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	return c.do(
		ctx,
		http.MethodPost,
		targetURL,
		referer,
		body,
	)
}

func (c *Client) do(
	ctx context.Context,
	method string,
	targetURL string,
	referer string,
	body io.Reader,
) ([]byte, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("FPF client is not initialized")
	}

	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parse request URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return nil, fmt.Errorf("request URL must include a host")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		targetURL,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set(
		"Accept",
		"text/html,application/xhtml+xml,application/json,*/*",
	)

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	if method == http.MethodPost {
		req.Header.Set(
			"Content-Type",
			"application/x-www-form-urlencoded; charset=UTF-8",
		)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("FPF HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"FPF returned HTTP %d for %s",
			resp.StatusCode,
			targetURL,
		)
	}

	responseBody, err := io.ReadAll(
		io.LimitReader(resp.Body, maxResponseSize+1),
	)
	if err != nil {
		return nil, fmt.Errorf("read FPF response: %w", err)
	}

	if len(responseBody) > maxResponseSize {
		return nil, fmt.Errorf(
			"FPF response exceeds maximum size of %d bytes",
			maxResponseSize,
		)
	}

	return responseBody, nil
}
