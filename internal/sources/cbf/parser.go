package cbf

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	canonical "github.com/sofi-io/sofi-dashboard/internal/competition"
)

const sourceName = "cbf"

type competitionData struct {
	CompetitionID string            `json:"competitionId"`
	Championship  string            `json:"championship"`
	Category      string            `json:"category"`
	Year          string            `json:"year"`
	Phase         phaseData         `json:"phase"`
	PhasesList    []json.RawMessage `json:"phasesList"`
}

type clubData struct {
	ID      string `json:"cod_time"`
	Name    string `json:"time"`
	LogoURL string `json:"escudo"`
}

type phaseData struct {
	ID   string `json:"fase_id"`
	Name string `json:"fase_nome"`
}

type matchClubData struct {
	ID   string  `json:"id"`
	Gols *string `json:"gols"`
}

type matchData struct {
	ID    string        `json:"id_jogo"`
	Round string        `json:"rodada"`
	Home  matchClubData `json:"mandante"`
	Away  matchClubData `json:"visitante"`
	Venue string        `json:"local"`
	Date  string        `json:"data"`
	Time  string        `json:"hora"`
}

var flightPattern = regexp.MustCompile(
	`(?s)self\.__next_f\.push\((\[1,.*?\])\)</script>`,
)

var clubLinkPattern = regexp.MustCompile(
	`href="([^"]*/futebol-brasileiro/times/[^"]+)"`,
)

var matchLinkPattern = regexp.MustCompile(
	`href="([^"]*/futebol-brasileiro/jogos/[^"]+)"`,
)

func extractStructuredData(content string) (string, error) {
	chunks := flightPattern.FindAllStringSubmatch(content, -1)

	if len(chunks) == 0 {
		return "", fmt.Errorf("CBF structured data not found")
	}

	var builder strings.Builder

	for _, chunk := range chunks {
		var payload []json.RawMessage

		if err := json.Unmarshal(
			[]byte(chunk[1]),
			&payload,
		); err != nil {
			return "", fmt.Errorf(
				"decode CBF structured data: %w",
				err,
			)
		}

		if len(payload) != 2 {
			return "", fmt.Errorf("invalid CBF data fragment")
		}

		var fragment string

		if err := json.Unmarshal(payload[1], &fragment); err != nil {
			return "", fmt.Errorf(
				"decode CBF data fragment: %w",
				err,
			)
		}

		builder.WriteString(fragment)
	}

	if builder.Len() == 0 {
		return "", fmt.Errorf("empty CBF structured data")
	}

	return builder.String(), nil
}

func extractClubURLs(
	competitionURL string,
	content string,
) (map[string]string, error) {
	baseURL, err := url.Parse(competitionURL)
	if err != nil {
		return nil, err
	}

	links := make(map[string]string)

	for _, match := range clubLinkPattern.FindAllStringSubmatch(
		content,
		-1,
	) {
		link, err := url.Parse(
			html.UnescapeString(match[1]),
		)
		if err != nil {
			continue
		}

		resolved := baseURL.ResolveReference(link)
		if resolved.Host != baseURL.Host {
			continue
		}

		parts := strings.Split(
			strings.Trim(resolved.Path, "/"),
			"/",
		)
		switch len(parts) {
		case 6:

		case 7:
			parts = append(parts[:5], parts[6])

		default:
			continue
		}

		clubID := parts[len(parts)-1]
		if clubID == "" {
			continue
		}
		resolved.Path = "/" + strings.Join(parts, "/")
		resolved.RawPath = ""

		links[clubID] = resolved.String()
	}

	return links, nil
}

func extractMatchURLs(clubURL string, content string) (map[string]string, error) {
	baseURL, err := url.Parse(clubURL)
	if err != nil {
		return nil, fmt.Errorf("parse club URL: %w", err)
	}

	links := make(map[string]string)

	for _, match := range matchLinkPattern.FindAllStringSubmatch(
		content,
		-1,
	) {
		link, err := url.Parse(
			html.UnescapeString(match[1]),
		)
		if err != nil {
			continue
		}

		resolved := baseURL.ResolveReference(link)

		if resolved.Host != baseURL.Host {
			continue
		}

		parts := strings.Split(
			strings.Trim(resolved.Path, "/"),
			"/",
		)

		if len(parts) < 7 {
			continue
		}

		matchID := parts[len(parts)-1]

		if _, err := strconv.Atoi(matchID); err != nil {
			continue
		}

		links[matchID] = resolved.String()
	}

	return links, nil
}

func decodeCompetitionData(data string) (competitionData, error) {
	const marker = `"competitionData":`

	index := strings.Index(data, marker)
	if index == -1 {
		return competitionData{}, fmt.Errorf(
			"competition data not found",
		)
	}

	payload := data[index+len(marker):]

	var competition competitionData

	decoder := json.NewDecoder(
		strings.NewReader(payload),
	)

	if err := decoder.Decode(&competition); err != nil {
		return competitionData{}, fmt.Errorf(
			"decode competition data: %w",
			err,
		)
	}

	if competition.CompetitionID == "" {
		return competitionData{}, fmt.Errorf(
			"competition external ID not found",
		)
	}

	return competition, nil
}

func ParseClubs(baseURL, rawHTML string) ([]canonical.Club, error) {
	data, err := extractStructuredData(rawHTML)
	if err != nil {
		return nil, fmt.Errorf(
			"extract structured data: %w",
			err,
		)
	}

	clubsURLs, err := extractClubURLs(baseURL, rawHTML)
	if err != nil {
		return nil, fmt.Errorf(
			"extract club URLs: %w",
			err,
		)
	}

	const marker = `{"cod_time":`

	clubs := make([]canonical.Club, 0)
	seen := make(map[string]struct{})
	offset := 0

	for {
		index := strings.Index(data[offset:], marker)
		if index == -1 {
			break
		}

		index += offset

		var club clubData

		decoder := json.NewDecoder(strings.NewReader(data[index:]))
		if err := decoder.Decode(&club); err != nil {
			return nil, fmt.Errorf(
				"decode club data: %w",
				err,
			)
		}

		offset = index + int(decoder.InputOffset())

		if club.ID == "" || club.Name == "" {
			return nil, fmt.Errorf(
				"invalid club data",
			)
		}

		if _, exists := seen[club.ID]; exists {
			continue
		}

		seen[club.ID] = struct{}{}

		clubs = append(clubs, canonical.Club{
			ExternalID: club.ID,
			Name:       club.Name,
			LogoURL:    club.LogoURL,
			URL:        clubsURLs[club.ID],
		})
	}

	if len(clubs) == 0 {
		return nil, fmt.Errorf("no clubs found")
	}

	return clubs, nil
}

func ParseCompetition(rawURL, rawHTML string) (canonical.Competition, error) {
	data, err := extractStructuredData(rawHTML)
	if err != nil {
		return canonical.Competition{}, fmt.Errorf(
			"extract structured data: %w",
			err,
		)
	}

	metadata, err := decodeCompetitionData(data)
	if err != nil {
		return canonical.Competition{}, fmt.Errorf(
			"decode competition data: %w",
			err,
		)
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return canonical.Competition{}, fmt.Errorf(
			"parse competition URL: %w",
			err,
		)
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) < 5 || parts[0] != "futebol-brasileiro" || parts[1] != "tabelas" {
		return canonical.Competition{}, fmt.Errorf(
			"invalid competition URL",
		)
	}

	if metadata.Year == "" || parts[4] != metadata.Year {
		return canonical.Competition{}, fmt.Errorf(
			"competition year not found",
		)
	}

	canonicalKey := strings.Join(parts[2:4], "-")
	name := strings.TrimSpace(metadata.Championship)
	if metadata.Category != "" {
		name += " - " + metadata.Category
	}

	parsedURL.Path = "/" + strings.Join(parts[:5], "/")
	parsedURL.RawPath = ""
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	return canonical.Competition{
		Source:       sourceName,
		ExternalID:   metadata.CompetitionID,
		CanonicalKey: canonicalKey,
		Name:         name,
		Season:       metadata.Year,
		URL:          parsedURL.String(),
	}, nil
}

func ParsePhases(rawURL, rawHTML string) ([]canonical.Phase, error) {
	data, err := extractStructuredData(rawHTML)
	if err != nil {
		return nil, fmt.Errorf(
			"extract structured data: %w",
			err,
		)
	}

	metadata, err := decodeCompetitionData(data)
	if err != nil {
		return nil, fmt.Errorf(
			"decode competition data: %w",
			err,
		)
	}

	phases := make([]canonical.Phase, 0, len(metadata.PhasesList))

	for index, raw := range metadata.PhasesList {
		var phase phaseData
		var reference string

		if err := json.Unmarshal(raw, &reference); err == nil {
			if !strings.HasSuffix(
				reference,
				":props:competitionData:phase",
			) {
				return nil, fmt.Errorf(
					"decode phase data: %w",
					err,
				)
			}

			phase = metadata.Phase
		} else {
			if err := json.Unmarshal(raw, &phase); err != nil {
				return nil, fmt.Errorf(
					"decode phase data: %w",
					err,
				)
			}
		}

		if phase.ID == "" || phase.Name == "" {
			return nil, fmt.Errorf(
				"invalid phase data",
			)
		}

		phases = append(phases, canonical.Phase{
			ExternalID: phase.ID,
			Name:       phase.Name,
			Order:      index + 1,
		})
	}

	if len(phases) == 0 {
		return nil, fmt.Errorf("no phases found")
	}

	return phases, nil
}

func ParseMatches(clubURL string, rawHTML string) ([]canonical.Match, error) {
	data, err := extractStructuredData(rawHTML)
	if err != nil {
		return nil, fmt.Errorf(
			"extract structured data: %w",
			err,
		)
	}

	const marker = `"jogos":[`

	index := strings.Index(data, marker)
	if index == -1 {
		return nil, fmt.Errorf("match data not found")
	}

	payload := data[index+len(`"jogos":`):]
	var items []matchData

	decoder := json.NewDecoder(strings.NewReader(payload))
	if err := decoder.Decode(&items); err != nil {
		return nil, fmt.Errorf(
			"decode match data: %w",
			err,
		)
	}

	matchURLs, err := extractMatchURLs(clubURL, rawHTML)
	if err != nil {
		return nil, fmt.Errorf(
			"extract match URLs: %w",
			err,
		)
	}

	matches := make([]canonical.Match, 0, len(items))

	for _, item := range items {
		if item.ID == "" || item.Home.ID == "" || item.Away.ID == "" {
			return nil, fmt.Errorf(
				"invalid match data",
			)
		}

		matchURL, exists := matchURLs[item.ID]
		if !exists {
			return nil, fmt.Errorf(
				"match URL not found for ID: %s",
				item.ID,
			)
		}

		parsedURL, err := url.Parse(matchURL)
		if err != nil {
			return nil, fmt.Errorf(
				"parse match URL: %w",
				err,
			)
		}

		parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		slug := parts[len(parts)-2]

		round, err := strconv.Atoi(item.Round)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid match round: %q",
				item.Round,
			)
		}

		homeScore, err := parseScore(item.Home.Gols)
		if err != nil {
			return nil, fmt.Errorf(
				"parse home score: %w",
				err,
			)
		}

		awayScore, err := parseScore(item.Away.Gols)
		if err != nil {
			return nil, fmt.Errorf(
				"parse away score: %w",
				err,
			)
		}

		status := canonical.MatchPending
		if homeScore != nil && awayScore != nil {
			status = canonical.MatchCompleted
		}

		matches = append(matches, canonical.Match{
			ExternalID: item.ID,
			URL:        matchURL,
			Slug:       slug,

			Round: round,

			DateText: strings.TrimSpace(item.Date),
			TimeText: strings.TrimSpace(item.Time),
			Venue:    strings.TrimSpace(item.Venue),

			HomeClubExternalID: item.Home.ID,
			AwayClubExternalID: item.Away.ID,

			HomeScore: homeScore,
			AwayScore: awayScore,

			Status: status,
		})
	}

	return matches, nil
}

func ParseMatchDetails(match canonical.Match, phases []canonical.Phase, rawHTML string) (canonical.Match, error) {
	data, err := extractStructuredData(rawHTML)
	if err != nil {
		return canonical.Match{}, fmt.Errorf(
			"extract structured data: %w",
			err,
		)
	}

	const marker = `"jogo":{`
	index := strings.Index(data, marker)
	if index == -1 {
		return canonical.Match{}, fmt.Errorf("match details not found")
	}

	payload := data[index+len(`"jogo":`):]

	var details struct {
		ID        string `json:"id_jogo"`
		PhaseName string `json:"etapa"`

		Documents []struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		} `json:"documentos"`
	}

	decoder := json.NewDecoder(strings.NewReader(payload))
	if err := decoder.Decode(&details); err != nil {
		return canonical.Match{}, fmt.Errorf(
			"decode match details: %w",
			err,
		)
	}

	if details.ID != match.ExternalID {
		return canonical.Match{}, fmt.Errorf(
			"match ID mismatch: expected %s, got %s",
			match.ExternalID,
			details.ID,
		)
	}

	match.PhaseName = strings.TrimSpace(details.PhaseName)

	for _, phase := range phases {
		if strings.EqualFold(strings.TrimSpace(phase.Name), match.PhaseName) {
			match.PhaseExternalID = phase.ExternalID
			break
		}
	}

	documents := make([]canonical.Document, 0)

	for _, doc := range details.Documents {
		documentType, recognized := parseDocumentType(doc.Title)
		if !recognized || strings.TrimSpace(doc.URL) == "" {
			continue
		}

		documents = append(documents, canonical.Document{
			Type:   documentType,
			URL:    doc.URL,
			Status: canonical.DocumentDiscovered,
		})
	}

	match.Documents = documents

	return match, nil
}

func parseScore(value *string) (*int, error) {
	if value == nil {
		return nil, nil
	}

	text := strings.TrimSpace(*value)

	if text == "" {
		return nil, nil
	}

	score, err := strconv.Atoi(text)
	if err != nil || score < 0 {
		return nil, fmt.Errorf(
			"invalid match score: %q",
			text,
		)
	}

	return &score, nil
}

func parseDocumentType(title string) (canonical.DocumentType, bool) {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "súmula", "sumula":
		return canonical.DocumentMatchSummary, true

	case "boletim financeiro":
		return canonical.DocumentFinancialBulletin, true

	case "relatório de jogo", "relatorio de jogo":
		return canonical.DocumentMatchReport, true

	case "plano de ação", "plano de acao":
		return canonical.DocumentActionPlan, true

	default:
		return "", false
	}
}
