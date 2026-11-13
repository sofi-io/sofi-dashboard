package fpf

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClientGetHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}

			if r.Header.Get("User-Agent") != userAgent {
				t.Error("missing expected User-Agent")
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "PHPSESSID",
				Value: "test-session",
				Path:  "/",
			})

			fmt.Fprint(w, "<html>competition</html>")
		},
	))
	defer server.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	body, err := client.GetHTML(
		context.Background(),
		server.URL,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	if body != "<html>competition</html>" {
		t.Fatalf("unexpected response: %s", body)
	}
}

func TestClientPreservesCookiesBetweenRequests(t *testing.T) {
	var sessionCookieReceived bool

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				http.SetCookie(w, &http.Cookie{
					Name:  "PHPSESSID",
					Value: "persistent-session",
					Path:  "/",
				})
				return
			}

			cookie, err := r.Cookie("PHPSESSID")
			if err == nil && cookie.Value == "persistent-session" {
				sessionCookieReceived = true
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"html":"ok"}`)
		},
	))
	defer server.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetHTML(
		context.Background(),
		server.URL,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.PostForm(
		context.Background(),
		server.URL,
		server.URL,
		url.Values{"idjogo": {"51836"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !sessionCookieReceived {
		t.Fatal("PHP session cookie was not preserved")
	}
}

func TestClientPostFormHeadersAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}

			if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
				t.Error("missing X-Requested-With header")
			}

			if r.Header.Get("Referer") != "https://example.com/source" {
				t.Error("unexpected Referer")
			}

			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}

			if r.Form.Get("idjogo") != "51836" {
				t.Errorf("idjogo = %s, want 51836", r.Form.Get("idjogo"))
			}

			fmt.Fprint(w, "<div>match details</div>")
		},
	))
	defer server.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	body, err := client.PostForm(
		context.Background(),
		server.URL,
		"https://example.com/source",
		url.Values{"idjogo": {"51836"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "<div>match details</div>" {
		t.Fatalf("unexpected response: %s", body)
	}
}

func TestClientRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		},
	))
	defer server.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetHTML(
		context.Background(),
		server.URL,
		"",
	)

	if err == nil {
		t.Fatal("expected HTTP error")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientRejectsInvalidURL(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetHTML(
		context.Background(),
		"ftp://example.com/file",
		"",
	)

	if err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestClientRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, strings.Repeat("a", maxResponseSize+1))
		},
	))
	defer server.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.GetHTML(
		context.Background(),
		server.URL,
		"",
	)

	if err == nil {
		t.Fatal("expected oversized response error")
	}
}
