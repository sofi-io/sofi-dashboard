package cbf

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	clubPath     = "/futebol-brasileiro/times/"
	matchPath    = "/futebol-brasileiro/jogos/"
	documentPath = "/sumulas/"
)

func parseHTML(raw string) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	return doc, nil
}

func resolveURL(baseURL, href string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	reference, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return "", err
	}

	return base.ResolveReference(reference).String(), nil
}

func ParseClubs(baseURL, rawHTML string) ([]Club, error) {
	doc, err := parseHTML(rawHTML)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var clubs []Club

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, clubPath) {
			return
		}

		fullURL, err := resolveURL(baseURL, href)
		if err != nil {
			return
		}

		id := lastPathSegment(fullURL)
		if id == "" || seen[id] {
			return
		}

		name := strings.TrimSpace(s.Text())
		if name == "" {
			name, _ = s.Find("img").First().Attr("alt")
		}
		if name == "" {
			name, _ = s.Attr("title")
		}

		seen[id] = true
		clubs = append(clubs, Club{
			ExternalID: id,
			Name:       strings.TrimSpace(name),
			URL:        fullURL,
		})
	})

	return clubs, nil
}

func ParseMatches(baseURL, rawHTML string) ([]Match, error) {
	doc, err := parseHTML(rawHTML)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var matches []Match

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, matchPath) {
			return
		}

		fullURL, err := resolveURL(baseURL, href)
		if err != nil {
			return
		}

		u, err := url.Parse(fullURL)
		if err != nil {
			return
		}

		id := lastPathSegment(u.Path)
		if id == "" || seen[id] {
			return
		}

		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		slug := ""
		if len(parts) >= 2 {
			slug = parts[len(parts)-2]
		}

		seen[id] = true
		matches = append(matches, Match{
			ExternalID: id,
			Slug:       slug,
			URL:        fullURL,
		})
	})

	return matches, nil
}

func ParseDocuments(baseURL, rawHTML string) ([]Document, error) {
	doc, err := parseHTML(rawHTML)
	if err != nil {
		return nil, err
	}

	var documents []Document
	seen := make(map[string]bool)

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, documentPath) {
			return
		}

		fullURL, err := resolveURL(baseURL, href)
		if err != nil || seen[fullURL] {
			return
		}

		label := strings.ToLower(strings.Join(
			strings.Fields(s.Text()), " ",
		))

		var documentType DocumentType

		switch {
		case strings.Contains(label, "boletim financeiro"):
			documentType = DocumentFinancialBulletin

		case strings.Contains(label, "súmula"),
			strings.Contains(label, "sumula"):
			documentType = DocumentSummary

		case strings.Contains(label, "relatório de jogo"),
			strings.Contains(label, "relatorio de jogo"):
			documentType = DocumentMatchReport

		default:
			return
		}

		seen[fullURL] = true
		documents = append(documents, Document{
			Type:   documentType,
			URL:    fullURL,
			Status: DocumentDiscovered,
		})
	})

	return documents, nil
}

func lastPathSegment(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}
