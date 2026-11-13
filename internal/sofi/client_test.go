package sofi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testAPIKey = "sofi_test_api_key"
	testPDF    = "%PDF-1.7 test document"
	testID     = "extraction-123"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	client, err := NewClient(
		server.URL,
		testAPIKey,
		server.Client(),
	)
	if err != nil {
		server.Close()
		t.Fatalf("NewClient(): %v", err)
	}

	return client, server
}

func TestSubmitExtraction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if r.URL.Path != "/v1/extractions" {
			t.Errorf("path = %s, unexpected", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("authorization = %q, unexpected", got)
		}

		mediaType, params, err := mime.ParseMediaType(
			r.Header.Get("Content-Type"),
		)
		if err != nil {
			t.Errorf("parse content type: %v", err)
			return
		}

		if mediaType != "multipart/form-data" {
			t.Errorf("media type = %q", mediaType)
			return
		}

		reader := multipart.NewReader(r.Body, params["boundary"])

		fields := make(map[string]string)

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("read multipart: %v", err)
				return
			}

			content, err := io.ReadAll(part)
			if err != nil {
				t.Errorf("read part: %v", err)
				return
			}

			if part.FormName() == "file" {
				if part.FileName() != "bordero.pdf" {
					t.Errorf("filename = %q", part.FileName())
				}

				if string(content) != testPDF {
					t.Errorf("unexpected PDF content: %q", content)
				}
			} else {
				fields[part.FormName()] = string(content)
			}
		}

		if fields["season"] != "2026" {
			t.Errorf("season = %q, want 2026", fields["season"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)

		_ = json.NewEncoder(w).Encode(Extraction{
			ID:        testID,
			Status:    "pending",
			CreatedAt: time.Date(2026, 11, 13, 22, 0, 0, 0, time.UTC),
		})
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	result, err := client.SubmitExtraction(
		context.Background(),
		"bordero.pdf",
		"2026",
		strings.NewReader(testPDF),
	)
	if err != nil {
		t.Fatalf("SubmitExtraction(): %v", err)
	}

	if result.ID != testID {
		t.Errorf("ID = %q, want %q", result.ID, testID)
	}

	if result.Status != "pending" {
		t.Errorf("status = %q, want pending", result.Status)
	}
}

func TestGetExtraction(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}

		if r.URL.Path != "/v1/extractions/"+testID {
			t.Errorf("path = %s", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("authorization = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")

		_, _ = io.WriteString(w, `{
			"id": "extraction-123",
			"status": "completed",
			"original_filename": "bordero.pdf",
			"created_at": "2026-11-13T22:00:00Z",
			"completed_at": "2026-11-13T22:01:00Z",
			"result": {"matches": []},
			"validation": {"valid": true}
		}`)
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	result, err := client.GetExtraction(
		context.Background(),
		testID,
	)
	if err != nil {
		t.Fatalf("GetExtraction(): %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("status = %q, want completed", result.Status)
	}

	if result.OriginalFilename != "bordero.pdf" {
		t.Errorf("filename = %q", result.OriginalFilename)
	}

	if result.CompletedAt == nil {
		t.Fatal("completed_at should not be nil")
	}

	if len(result.Result) == 0 {
		t.Error("result should not be empty")
	}

	if len(result.Validation) == 0 {
		t.Error("validation should not be empty")
	}
}

func TestSubmitExtractionRejectsInvalidInput(t *testing.T) {
	client, server := newTestClient(t, http.NotFoundHandler())
	defer server.Close()

	tests := []struct {
		name     string
		filename string
		season   string
		pdf      io.Reader
	}{
		{
			name:     "missing filename",
			filename: "",
			season:   "2026",
			pdf:      strings.NewReader(testPDF),
		},
		{
			name:     "missing season",
			filename: "bordero.pdf",
			season:   "",
			pdf:      strings.NewReader(testPDF),
		},
		{
			name:     "missing PDF reader",
			filename: "bordero.pdf",
			season:   "2026",
			pdf:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SubmitExtraction(
				context.Background(),
				tt.filename,
				tt.season,
				tt.pdf,
			)

			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGetExtractionRejectsEmptyID(t *testing.T) {
	client, server := newTestClient(t, http.NotFoundHandler())
	defer server.Close()

	_, err := client.GetExtraction(context.Background(), "")

	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       string
	}{
		{"bad request", http.StatusBadRequest, "invalid_file"},
		{"unauthorized", http.StatusUnauthorized, "unauthorized"},
		{"not found", http.StatusNotFound, "extraction_not_found"},
		{"server error", http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)

				_, _ = io.WriteString(w, `{
					"error": {
						"code": "`+tt.code+`",
						"message": "Test error"
					}
				}`)
			})

			client, server := newTestClient(t, handler)
			defer server.Close()

			_, err := client.GetExtraction(
				context.Background(),
				testID,
			)

			if err == nil {
				t.Fatal("expected API error")
			}

			if !strings.Contains(err.Error(), tt.code) {
				t.Errorf("error %q does not contain %q", err, tt.code)
			}
		})
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{invalid json`)
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	_, err := client.GetExtraction(context.Background(), testID)

	if err == nil {
		t.Fatal("expected JSON decoding error")
	}
}

func TestGetExtractionContextCancellation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetExtraction(ctx, testID)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestHTTPErrorIsTyped(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{
			"error": {
				"code": "unauthorized",
				"message": "Invalid API key",
				"request_id": "request-123"
			}
		}`)
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	_, err := client.GetExtraction(context.Background(), testID)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *HTTPError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d", apiErr.StatusCode)
	}

	if apiErr.Code != "unauthorized" {
		t.Errorf("code = %q", apiErr.Code)
	}

	if apiErr.RequestID != "request-123" {
		t.Errorf("request ID = %q", apiErr.RequestID)
	}
}

func TestGetExtractionRejectsOversizedResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w,
			`{"id":"extraction-123","status":"completed","padding":"`+
				strings.Repeat("a", maxResponseSize)+`"}`,
		)
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	_, err := client.GetExtraction(context.Background(), testID)
	if err == nil {
		t.Fatal("expected oversized response error")
	}

	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetExtractionRejectsTrailingJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w,
			`{"id":"extraction-123","status":"completed"} {}`,
		)
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	_, err := client.GetExtraction(context.Background(), testID)
	if err == nil {
		t.Fatal("expected trailing JSON error")
	}
}

func TestSubmitExtractionRejectsOversizedPDF(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusAccepted)
	})

	client, server := newTestClient(t, handler)
	defer server.Close()

	pdf := strings.NewReader(
		strings.Repeat("A", maxPDFSize+1),
	)

	_, err := client.SubmitExtraction(
		context.Background(),
		"bordero.pdf",
		"2026",
		pdf,
	)

	if err == nil {
		t.Fatal("expected oversized PDF error")
	}
}

func TestSubmitExtractionRejectsLongSeason(t *testing.T) {
	client, server := newTestClient(t, http.NotFoundHandler())
	defer server.Close()

	_, err := client.SubmitExtraction(
		context.Background(),
		"bordero.pdf",
		strings.Repeat("2", maxSeasonLength+1),
		strings.NewReader(testPDF),
	)

	if err == nil {
		t.Fatal("expected season validation error")
	}
}
