package http

import (
	"encoding/json"
	"net/http"

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

	result, err := h.useCase.ImportCleanListings(r.Context(), payload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}
