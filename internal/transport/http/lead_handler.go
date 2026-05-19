package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/readleads"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
)

type LeadHandler struct {
	useCase *readleads.UseCase
}

func NewLeadHandler(useCase *readleads.UseCase) *LeadHandler {
	return &LeadHandler{
		useCase: useCase,
	}
}

func (h *LeadHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/leads", h.List)
	r.Get("/api/leads/{id}", h.GetByID)
}

func (h *LeadHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	var minScore *int

	if value := query.Get("minScore"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid minScore",
			})
			return
		}

		minScore = &parsed
	}

	filters := lead.ListFilters{
		City:           query.Get("city"),
		PostcodeArea:   query.Get("postcodeArea"),
		PropertyType:   query.Get("propertyType"),
		SourcePlatform: query.Get("sourcePlatform"),
		MinScore:       minScore,
		Limit:          limit,
		Offset:         offset,
	}

	results, err := h.useCase.List(r.Context(), filters)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch leads",
		})
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *LeadHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError

		if err == readleads.ErrLeadNotFound {
			status = http.StatusNotFound
		}

		writeJSON(w, status, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}
