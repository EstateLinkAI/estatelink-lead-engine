package http

import (
	"encoding/json"
	"net/http"

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

	result, err := h.useCase.StartCleanListingsImport(r.Context(), payload)
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