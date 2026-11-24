package fpf

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var clubShieldPattern = regexp.MustCompile(`/clubes/(\d+)/escudo`)

// ParseClubs extracts unique clubs from an FPF classification table.
func ParseClubs(rawHTML string) ([]Club, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("parse classification HTML: %w", err)
	}

	seen := make(map[string]bool)
	clubs := make([]Club, 0)

	rows := findElementsByTag(doc, "tr")

	for _, row := range rows {
		images := findElementsByTag(row, "img")

		var externalID string
		var logoURL string

		for _, img := range images {
			src := attribute(img, "src")
			logoURL = src
			if src == "" {
				continue
			}

			match := clubShieldPattern.FindStringSubmatch(src)
			if len(match) == 2 {
				externalID = match[1]
				break
			}
		}

		if externalID == "" || seen[externalID] {
			continue
		}

		cells := findElementsByTag(row, "td")
		if len(cells) < 3 {
			continue
		}

		name := strings.TrimSpace(textContent(cells[2]))
		if name == "" {
			continue
		}

		seen[externalID] = true

		clubs = append(clubs, Club{
			ExternalID: externalID,
			Name:       name,
			LogoURL:    logoURL,
		})
	}

	return clubs, nil
}

// ClubURL builds the official FPF club URL when available.
// Classification currently exposes the CBF shield URL instead.
func ClubURL(baseURL, externalID string) (string, error) {
	if externalID == "" {
		return "", fmt.Errorf("club external ID is required")
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	return base.ResolveReference(&url.URL{
		Path:     "clube.php",
		RawQuery: "q=" + externalID,
	}).String(), nil
}
