package fpf

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// ParseMatchDocuments extracts the published documents from an FPF match.
func ParseMatchDocuments(
	baseURL string,
	rawHTML string,
) ([]Document, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("parse match documents HTML: %w", err)
	}

	section := findElementByID(doc, "documentos")
	if section == nil {
		return nil, fmt.Errorf("documents section not found")
	}

	links := findElementsByTag(section, "a")

	var documents []Document
	seen := make(map[DocumentType]bool)

	for _, link := range links {
		label := normalizeDocumentLabel(textContent(link))
		documentType := classifyDocument(label)

		if documentType == "" {
			continue
		}

		href := attribute(link, "href")
		if href == "" {
			continue
		}

		reference, err := url.Parse(href)
		if err != nil {
			return nil, fmt.Errorf(
				"parse document URL %q: %w",
				href,
				err,
			)
		}

		if !reference.IsAbs() {
			reference = base.ResolveReference(reference)
		}

		if reference.Scheme != "https" && reference.Scheme != "http" {
			continue
		}

		if seen[documentType] {
			continue
		}

		seen[documentType] = true

		documents = append(documents, Document{
			Type:   documentType,
			URL:    reference.String(),
			Status: DocumentDiscovered,
		})
	}

	return documents, nil
}

func findElementByID(root *html.Node, id string) *html.Node {
	var found *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}

		if node.Type == html.ElementNode &&
			attribute(node, "id") == id {
			found = node
			return
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)
	return found
}

func normalizeDocumentLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	replacer := strings.NewReplacer(
		"á", "a",
		"à", "a",
		"ã", "a",
		"â", "a",
		"é", "e",
		"ê", "e",
		"í", "i",
		"ó", "o",
		"ô", "o",
		"õ", "o",
		"ú", "u",
		"ç", "c",
	)

	return replacer.Replace(value)
}

func classifyDocument(label string) DocumentType {
	switch {
	case strings.Contains(label, "sumula"):
		return DocumentMatchReport

	case strings.Contains(label, "boletim financeiro"):
		return DocumentFinancialBulletin

	case strings.Contains(label, "plano de acao"):
		return DocumentActionPlan

	default:
		return ""
	}
}
