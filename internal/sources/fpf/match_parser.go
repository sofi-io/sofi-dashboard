package fpf

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var (
	roundPattern = regexp.MustCompile(`(\d+)`)
	clubPattern  = regexp.MustCompile(`/clubes/(\d+)/escudo`)
)

func ParseMatches(
	baseURL string,
	competitionID string,
	phaseID string,
	phaseName string,
	rawHTML string,
) ([]Match, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("parse matches HTML: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	slides := findElementsByClass(doc, "swiper-slide")
	if len(slides) == 0 {
		return nil, fmt.Errorf("no match rounds found")
	}

	var matches []Match

	for _, slide := range slides {
		round, err := extractRound(slide)
		if err != nil {
			return nil, err
		}

		containers := findElementsByClass(slide, "pr-jogos")

		for i, container := range containers {
			match, err := parseMatchContainer(
				base,
				competitionID,
				phaseID,
				phaseName,
				round,
				container,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"parse match in phase %s, round %d, item %d: %w",
					phaseID,
					round,
					i+1,
					err,
				)
			}

			matches = append(matches, match)
		}
	}

	return matches, nil
}

func extractRound(slide *html.Node) (int, error) {
	title := findFirstByClass(slide, "title-rodada")
	if title == nil {
		return 0, fmt.Errorf("round title not found")
	}

	text := strings.TrimSpace(textContent(title))
	value := roundPattern.FindString(text)

	if value == "" {
		return 0, fmt.Errorf("invalid round title: %q", text)
	}

	round, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse round number: %w", err)
	}

	return round, nil
}

func parseMatchContainer(
	base *url.URL,
	competitionID string,
	phaseID string,
	phaseName string,
	round int,
	container *html.Node,
) (Match, error) {
	var result Match

	result.CompetitionID = competitionID
	result.PhaseID = phaseID
	result.PhaseName = phaseName
	result.Round = round

	// Date, time and venue.
	chronology := findFirstByClass(container, "cronolog-sm")
	if chronology == nil {
		return result, fmt.Errorf("match chronology not found")
	}

	parts := strings.Split(
		textContent(chronology),
		" - ",
	)

	if len(parts) < 3 {
		return result, fmt.Errorf(
			"invalid match chronology: %q",
			textContent(chronology),
		)
	}

	result.DateText = strings.TrimSpace(parts[0])
	result.TimeText = strings.TrimSpace(parts[1])
	result.Venue = strings.TrimSpace(
		strings.Join(parts[2:], " - "),
	)

	// Club IDs from the two shield images.
	images := findElementsByTag(container, "img")

	var clubIDs []string

	for _, img := range images {
		src := attribute(img, "src")
		if src == "" {
			continue
		}

		match := clubPattern.FindStringSubmatch(src)
		if len(match) == 2 {
			clubIDs = append(clubIDs, match[1])
		}
	}

	if len(clubIDs) != 2 {
		return result, fmt.Errorf(
			"expected 2 club shields, found %d",
			len(clubIDs),
		)
	}

	result.HomeClubExternalID = clubIDs[0]
	result.AwayClubExternalID = clubIDs[1]

	// Score.
	scoreContainer := findFirstByClass(container, "result")

	if scoreContainer != nil {
		var scores []int

		for _, span := range findElementsByTag(scoreContainer, "span") {
			if hasClass(span, "versus") {
				continue
			}

			value := strings.TrimSpace(textContent(span))
			score, err := strconv.Atoi(value)

			if err == nil {
				scores = append(scores, score)
			}
		}

		if len(scores) == 2 {
			homeScore := scores[0]
			awayScore := scores[1]

			result.HomeScore = &homeScore
			result.AwayScore = &awayScore
			result.Status = MatchCompleted
		}
	}

	if result.Status == "" {
		result.Status = MatchScheduled
	}

	// Match ID and URL.
	links := findElementsByTag(container, "a")

	for _, link := range links {
		href := attribute(link, "href")
		if href == "" {
			continue
		}

		reference, err := url.Parse(href)
		if err != nil {
			continue
		}

		if !strings.Contains(reference.Path, "jogo.php") {
			continue
		}

		matchID := reference.Query().Get("jogo")
		if matchID == "" {
			continue
		}

		result.ExternalID = matchID
		result.URL = base.ResolveReference(reference).String()
		break
	}

	if result.ExternalID == "" {
		return result, fmt.Errorf("match external ID not found")
	}

	return result, nil
}

func findElementsByClass(root *html.Node, class string) []*html.Node {
	var found []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && hasClass(node, class) {
			found = append(found, node)
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)
	return found
}

func findFirstByClass(root *html.Node, class string) *html.Node {
	var found *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}

		if node.Type == html.ElementNode && hasClass(node, class) {
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

func findElementsByTag(root *html.Node, tag string) []*html.Node {
	var found []*html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			found = append(found, node)
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)
	return found
}

func textContent(node *html.Node) string {
	var builder strings.Builder

	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
			return
		}

		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(node)

	return strings.Join(strings.Fields(builder.String()), " ")
}
