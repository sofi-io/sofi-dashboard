package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sofi-io/sofi-dashboard/internal/database"
)

type Handler struct {
	db           *pgxpool.Pool
	extractions  *database.ExtractionRepository
	competitions *database.CompetitionQueryRepository
	competition  *database.CompetitionRepository
}

type competitionMatchListResponse struct {
	Items  []database.CompetitionMatchItem `json:"items"`
	Total  int64                           `json:"total"`
	Limit  int                             `json:"limit"`
	Offset int                             `json:"offset"`
}

func NewHandler(db *pgxpool.Pool) http.Handler {
	h := &Handler{
		db:           db,
		extractions:  database.NewExtractionRepository(db),
		competitions: database.NewCompetitionQueryRepository(db),
		competition:  database.NewCompetitionRepository(db),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", h.live)
	mux.HandleFunc("GET /health/ready", h.ready)

	mux.HandleFunc("GET /api/extractions", h.listExtractions)
	mux.HandleFunc("GET /api/extractions/{id}", h.getExtraction)

	mux.HandleFunc("GET /api/competitions", h.listCompetitions)
	mux.HandleFunc("GET /api/competitions/{id}", h.getCompetition)
	mux.HandleFunc("GET /api/competitions/{id}/matches", h.listCompetitionMatches)

	return mux
}

func (h *Handler) live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  "database unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

type extractionListResponse struct {
	Items  []database.ExtractionListItem `json:"items"`
	Total  int64                         `json:"total"`
	Limit  int                           `json:"limit"`
	Offset int                           `json:"offset"`
}

func (h *Handler) listExtractions(
	w http.ResponseWriter,
	r *http.Request,
) {
	query := r.URL.Query()

	limit := 20
	offset := 0

	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "limit must be between 1 and 100",
			})
			return
		}
		limit = parsed
	}

	if value := query.Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "offset must be a non-negative integer",
			})
			return
		}
		offset = parsed
	}

	filter := database.ExtractionListFilter{
		Status:   strings.TrimSpace(query.Get("status")),
		Season:   strings.TrimSpace(query.Get("season")),
		Provider: strings.TrimSpace(query.Get("provider")),
		Limit:    limit,
		Offset:   offset,
	}

	items, total, err := h.extractions.List(
		r.Context(),
		filter,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list extractions",
		})
		return
	}

	writeJSON(w, http.StatusOK, extractionListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) getExtraction(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := strings.TrimSpace(r.PathValue("id"))

	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "extraction ID is required",
		})
		return
	}

	item, err := h.extractions.GetByID(r.Context(), id)

	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "extraction not found",
		})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not retrieve extraction",
		})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type competitionListResponse struct {
	Items  []database.CompetitionListItem `json:"items"`
	Total  int64                          `json:"total"`
	Limit  int                            `json:"limit"`
	Offset int                            `json:"offset"`
}

func (h *Handler) listCompetitions(
	w http.ResponseWriter,
	r *http.Request,
) {
	query := r.URL.Query()

	limit := 20
	offset := 0

	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "limit must be between 1 and 100",
			})
			return
		}
		limit = parsed
	}

	if value := query.Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "offset must be a non-negative integer",
			})
			return
		}
		offset = parsed
	}

	items, total, err := h.competitions.List(
		r.Context(),
		limit,
		offset,
	)

	if err != nil {
		log.Printf("list competitions failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list competitions",
		})
		return
	}

	writeJSON(w, http.StatusOK, competitionListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) getCompetition(
	w http.ResponseWriter,
	r *http.Request,
) {
	id := strings.TrimSpace(r.PathValue("id"))

	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "competition ID is required",
		})
		return
	}

	item, err := h.competition.GetByID(r.Context(), id)

	if errors.Is(err, database.ErrCompetitionNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "competition not found",
		})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not retrieve competition",
		})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) listCompetitionMatches(
	w http.ResponseWriter,
	r *http.Request,
) {
	competitionID := strings.TrimSpace(r.PathValue("id"))

	if competitionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "competition ID is required",
		})
		return
	}

	query := r.URL.Query()

	limit := 20
	offset := 0

	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "limit must be between 1 and 100",
			})
			return
		}
		limit = parsed
	}

	if value := query.Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "offset must be a non-negative integer",
			})
			return
		}
		offset = parsed
	}

	items, total, err := h.competition.ListMatches(
		r.Context(),
		competitionID,
		strings.TrimSpace(query.Get("season")),
		limit,
		offset,
	)

	if errors.Is(err, database.ErrCompetitionNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "competition not found",
		})
		return
	}

	if err != nil {
		log.Printf("list competition matches failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "could not list competition matches",
		})
		return
	}

	writeJSON(w, http.StatusOK, competitionMatchListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
