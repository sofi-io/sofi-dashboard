package cbf

import (
	"context"
	"fmt"
	"net/url"
	"sync"
)

func matchDocumentsURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse match URL: %w", err)
	}

	query := u.Query()
	query.Set("view", "documentos")
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// DiscoverDocuments accesses the official match page and
// identifies the documents published for that match.
func (c *Client) DiscoverDocuments(
	ctx context.Context,
	match Match,
) ([]Document, error) {
	targetURL, err := matchDocumentsURL(match.URL)
	if err != nil {
		return nil, err
	}

	html, err := c.FetchHTML(ctx, targetURL)
	if err != nil {
		return nil, fmt.Errorf(
			"fetch documents for match %s: %w",
			match.ExternalID,
			err,
		)
	}

	documents, err := ParseDocuments(targetURL, html)
	if err != nil {
		return nil, fmt.Errorf(
			"parse documents for match %s: %w",
			match.ExternalID,
			err,
		)
	}

	hasFinancialBulletin := false

	for _, document := range documents {
		if document.Type == DocumentFinancialBulletin {
			hasFinancialBulletin = true
			break
		}
	}

	// A successfully inspected page without a financial
	// bulletin means the document has not been published
	// or is not currently listed.
	if !hasFinancialBulletin {
		documents = append(documents, Document{
			Type:   DocumentFinancialBulletin,
			Status: DocumentPending,
		})
	}

	return documents, nil
}

type matchDocumentResult struct {
	index     int
	documents []Document
	warning   *DiscoveryWarning
}

func discoverMatchDocuments(
	ctx context.Context,
	client *Client,
	matches []Match,
	concurrency int,
) ([]Match, []DiscoveryWarning) {
	if concurrency <= 0 {
		concurrency = 3
	}

	if concurrency > len(matches) {
		concurrency = len(matches)
	}

	if len(matches) == 0 {
		return matches, nil
	}

	jobs := make(chan int)
	results := make(chan matchDocumentResult, len(matches))

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for index := range jobs {
				match := matches[index]

				if err := ctx.Err(); err != nil {
					results <- matchDocumentResult{
						index: index,
						warning: &DiscoveryWarning{
							Stage:   "documents",
							MatchID: match.ExternalID,
							URL:     match.URL,
							Error:   err.Error(),
						},
					}
					continue
				}

				documents, err := client.DiscoverDocuments(ctx, match)
				if err != nil {
					results <- matchDocumentResult{
						index: index,
						warning: &DiscoveryWarning{
							Stage:   "documents",
							MatchID: match.ExternalID,
							URL:     match.URL,
							Error:   err.Error(),
						},
					}
					continue
				}

				results <- matchDocumentResult{
					index:     index,
					documents: documents,
				}
			}
		}()
	}

	go func() {
		for index := range matches {
			jobs <- index
		}

		close(jobs)
		wg.Wait()
		close(results)
	}()

	output := make([]Match, len(matches))
	copy(output, matches)

	var warnings []DiscoveryWarning

	for item := range results {
		if item.warning != nil {
			warnings = append(warnings, *item.warning)
			continue
		}

		output[item.index].Documents = item.documents
	}

	return output, warnings
}
