package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/importlistings"
)

type ImportHandler struct {
	useCase *importlistings.UseCase
}

func NewImportHandler(useCase *importlistings.UseCase) *ImportHandler {
	return &ImportHandler{
		useCase: useCase,
	}
}

func (h *ImportHandler) ImportCleanListings(w http.ResponseWriter, r *http.Request) {
	var payload []json.RawMessage

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON payload",
		})
		return
	}

	currentUser, ok := GetCurrentUser(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "not authenticated",
		})
		return
	}

	ipAddress, userAgent := requestMetadata(r)

	result, err := h.useCase.StartCleanListingsImport(r.Context(), payload, importlistings.ActivityContext{
		ActorUserID: currentUser.ID,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Filename:    r.Header.Get("X-Filename"),
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}

func (h *ImportHandler) GetImportJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "job id is required",
		})
		return
	}

	job, err := h.useCase.GetImportJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (h *ImportHandler) CancelImportJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "job id is required",
		})
		return
	}

	job, err := h.useCase.CancelImportJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (h *ImportHandler) ListImportJobs(w http.ResponseWriter, r *http.Request) {
	limit := 20

	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			limit = parsed
		}
	}

	jobs, err := h.useCase.ListImportJobs(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jobs": jobs,
	})
}
