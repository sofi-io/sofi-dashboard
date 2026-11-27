package httpclient

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
	defaultTimeout         = 30 * time.Second
	defaultMaxResponseSize = 10 << 20 // 10 MB
	defaultUserAgent       = "Mozilla/5.0"
)

func NewClient(config ClientConfig) (*Client, error) {
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}

	if config.MaxResponseSize <= 0 {
		config.MaxResponseSize = defaultMaxResponseSize
	}

	if config.UserAgent == "" {
		config.UserAgent = defaultUserAgent
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	headers := make(http.Header)
	headers.Set("User-Agent", config.UserAgent)
	headers.Set("Accept", "text/html,application/xhtml+xml,application/json,*/*")

	for key, values := range config.Headers {
		headers[http.CanonicalHeaderKey(key)] = append([]string(nil), values...)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Jar:     jar,
		},
		maxResponseSize: config.MaxResponseSize,
		headers:         headers,
	}, nil
}

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
		nil,
	)

	if err != nil {
		return "", fmt.Errorf("fetch HTML: %w", err)
	}

	return string(body), nil
}

func (c *Client) PostForm(
	ctx context.Context,
	targetURL string,
	referer string,
	formData url.Values,
) ([]byte, error) {
	var body io.Reader

	if formData != nil {
		body = strings.NewReader(formData.Encode())
	}

	headers := make(http.Header)
	headers.Set(
		"Content-Type",
		"application/x-www-form-urlencoded; charset=UTF-8",
	)

	return c.do(
		ctx,
		http.MethodPost,
		targetURL,
		referer,
		body,
		headers,
	)
}

func (c *Client) PostFormAJAX(
	ctx context.Context,
	targetURL string,
	referer string,
	formData url.Values,
) ([]byte, error) {
	headers := make(http.Header)
	headers.Set(
		"Content-Type",
		"application/x-www-form-urlencoded; charset=UTF-8",
	)
	headers.Set(
		"X-Requested-With",
		"XMLHttpRequest",
	)

	return c.do(
		ctx,
		http.MethodPost,
		targetURL,
		referer,
		strings.NewReader(formData.Encode()),
		headers,
	)
}

func (c *Client) do(
	ctx context.Context,
	method string,
	targetURL string,
	referer string,
	body io.Reader,
	headers http.Header,
) ([]byte, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("client is not initialized")
	}

	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return nil, fmt.Errorf("URL must have a host: %s", targetURL)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		targetURL,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header = c.headers.Clone()

	for key, values := range headers {
		req.Header[http.CanonicalHeaderKey(key)] = append([]string(nil), values...)
	}

	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"unexpected status code: %d for %s",
			resp.StatusCode,
			targetURL,
		)
	}

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if int64(len(responseBody)) > c.maxResponseSize {
		return nil, fmt.Errorf(
			"response body exceeds maximum size of %d bytes",
			c.maxResponseSize,
		)
	}

	return responseBody, nil

}
