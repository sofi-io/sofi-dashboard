package fpf

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func ParseCompetition(rawURL, rawHTML string) (*Competition, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse competition URL: %w", err)
	}

	id := u.Query().Get("q")
	if id == "" {
		return nil, fmt.Errorf("competition ID not found in URL")
	}

	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("parse competition HTML: %w", err)
	}

	result := &Competition{
		ExternalID: id,
		URL:        rawURL,
	}

	capture := ""
	var currentPhase *Phase

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h3":
				if result.Name == "" &&
					hasClass(n, "head_background_title") {
					capture = "competition"
				}

			case "a":
				if hasClass(n, "fase") {
					if attribute(n, "data-id") == id {
						currentPhase = &Phase{
							ExternalID: attribute(n, "data-fase"),
							Selected:   hasClass(n, "active"),
						}

						currentPhase.InitialRound, _ =
							strconv.Atoi(attribute(n, "data-rodada"))

						if currentPhase.ExternalID != "" {
							capture = "phase"
						} else {
							currentPhase = nil
						}
					}
				}
			}
		}

		if n.Type == html.TextNode {
			value := strings.TrimSpace(n.Data)

			if value != "" {
				switch capture {
				case "competition":
					result.Name += value
				case "phase":
					if currentPhase != nil {
						currentPhase.Name += value
					}
				}
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "h3":
				if capture == "competition" {
					capture = ""
				}
			case "a":
				if capture == "phase" && currentPhase != nil {
					currentPhase.Name = strings.TrimSpace(currentPhase.Name)

					if currentPhase.Name != "" {
						result.Phases = append(result.Phases, *currentPhase)
					}

					currentPhase = nil
					capture = ""
				}
			}
		}
	}

	walk(doc)

	result.Name = strings.TrimSpace(result.Name)

	if result.Name == "" {
		return nil, fmt.Errorf("competition name not found")
	}

	if len(result.Phases) == 0 {
		return nil, fmt.Errorf("no competition phases found")
	}

	parts := strings.Split(result.Name, "/")
	if len(parts) > 1 {
		result.Season = strings.TrimSpace(parts[len(parts)-1])
	}

	return result, nil
}

func attribute(n *html.Node, name string) string {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}

	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key != "class" {
			continue
		}

		for _, value := range strings.Fields(attr.Val) {
			if value == class {
				return true
			}
		}
	}

	return false
}
