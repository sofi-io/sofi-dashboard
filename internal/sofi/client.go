package sofi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxPDFSize      = 20 << 20
	maxResponseSize = 1 << 20
	maxFilenameSize = 255
	maxSeasonLength = 20
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// HTTPError represents an unsuccessful HTTP response from SOFI API.
type HTTPError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *HTTPError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("SOFI API returned HTTP %d", e.StatusCode)
	}

	return fmt.Sprintf(
		"SOFI API returned HTTP %d: %s (%s)",
		e.StatusCode,
		e.Code,
		e.Message,
	)
}

func NewClient(
	baseURL string,
	apiKey string,
	httpClient *http.Client,
) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil ||
		parsed.Scheme == "" ||
		parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid SOFI API base URL")
	}

	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("SOFI API key is required")
	}

	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf(
			"SOFI API base URL cannot contain query or fragment",
		)
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	return &Client{
		baseURL: strings.TrimRight(parsed.String(), "/"),
		apiKey:  apiKey,
		http:    httpClient,
	}, nil
}

func (c *Client) SubmitExtraction(
	ctx context.Context,
	filename string,
	season string,
	pdf io.Reader,
) (Extraction, error) {
	if pdf == nil {
		return Extraction{}, fmt.Errorf("PDF reader is required")
	}

	filename = path.Base(strings.ReplaceAll(filename, `\`, "/"))

	if filename == "." ||
		filename == "/" ||
		strings.TrimSpace(filename) == "" ||
		utf8.RuneCountInString(filename) > maxFilenameSize {
		return Extraction{}, fmt.Errorf("invalid filename")
	}

	season = strings.TrimSpace(season)

	if season == "" {
		return Extraction{}, fmt.Errorf("season is required")
	}

	if utf8.RuneCountInString(season) > maxSeasonLength {
		return Extraction{}, fmt.Errorf(
			"season must contain at most %d characters",
			maxSeasonLength,
		)
	}

	if err := ctx.Err(); err != nil {
		return Extraction{}, err
	}

	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)

	defer reader.Close()

	// Multipart construction happens asynchronously so the PDF
	// is streamed directly into the HTTP request.
	go func() {
		err := writeMultipart(
			form,
			pdf,
			filename,
			season,
		)

		_ = writer.CloseWithError(err)
	}()

	endpoint := c.baseURL + "/v1/extractions"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		reader,
	)
	if err != nil {
		_ = reader.CloseWithError(err)
		return Extraction{}, fmt.Errorf(
			"create submission request: %w",
			err,
		)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", form.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		_ = reader.CloseWithError(err)

		return Extraction{}, fmt.Errorf(
			"submit extraction: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return Extraction{}, decodeAPIError(resp)
	}

	var result Extraction

	if err := decodeJSONResponse(resp.Body, &result); err != nil {
		return Extraction{}, fmt.Errorf(
			"decode submission response: %w",
			err,
		)
	}

	if strings.TrimSpace(result.ID) == "" ||
		strings.TrimSpace(result.Status) == "" {
		return Extraction{}, fmt.Errorf(
			"invalid submission response",
		)
	}

	return result, nil
}

func writeMultipart(
	writer *multipart.Writer,
	pdf io.Reader,
	filename string,
	season string,
) error {
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create PDF form field: %w", err)
	}

	written, err := io.Copy(
		part,
		io.LimitReader(pdf, maxPDFSize+1),
	)
	if err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}

	if written > maxPDFSize {
		return fmt.Errorf(
			"PDF exceeds maximum size of %d bytes",
			maxPDFSize,
		)
	}

	if err := writer.WriteField("season", season); err != nil {
		return fmt.Errorf("write season field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart body: %w", err)
	}

	return nil
}

func (c *Client) GetExtraction(
	ctx context.Context,
	id string,
) (ExtractionStatus, error) {
	id = strings.TrimSpace(id)

	if id == "" {
		return ExtractionStatus{}, fmt.Errorf(
			"extraction ID is required",
		)
	}

	endpoint := c.baseURL + "/v1/extractions/" + url.PathEscape(id)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return ExtractionStatus{}, fmt.Errorf(
			"create status request: %w",
			err,
		)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return ExtractionStatus{}, fmt.Errorf(
			"get extraction status: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ExtractionStatus{}, decodeAPIError(resp)
	}

	var result ExtractionStatus

	if err := decodeJSONResponse(resp.Body, &result); err != nil {
		return ExtractionStatus{}, fmt.Errorf(
			"decode status response: %w",
			err,
		)
	}

	if result.ID != id || strings.TrimSpace(result.Status) == "" {
		return ExtractionStatus{}, fmt.Errorf(
			"invalid status response",
		)
	}

	return result, nil
}

func decodeJSONResponse(
	body io.Reader,
	destination any,
) error {
	limited := io.LimitReader(body, maxResponseSize+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if len(data) > maxResponseSize {
		return fmt.Errorf(
			"response exceeds maximum size of %d bytes",
			maxResponseSize,
		)
	}

	decoder := json.NewDecoder(strings.NewReader(string(data)))

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	var extra any

	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}

		return fmt.Errorf("invalid trailing response data: %w", err)
	}

	return nil
}

func decodeAPIError(resp *http.Response) error {
	var apiError APIError

	err := decodeJSONResponse(resp.Body, &apiError)
	if err != nil {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    "unable to decode API error response",
		}
	}

	return &HTTPError{
		StatusCode: resp.StatusCode,
		Code:       apiError.Error.Code,
		Message:    apiError.Error.Message,
		RequestID:  apiError.Error.RequestID,
	}
}
