package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/importlistings"
)

type ImportHandler struct {
	useCase             *importlistings.UseCase
	maxRequestBodyBytes int64
}

func NewImportHandler(useCase *importlistings.UseCase, maxRequestBodyBytes int64) *ImportHandler {
	return &ImportHandler{
		useCase:             useCase,
		maxRequestBodyBytes: maxRequestBodyBytes,
	}
}

func (h *ImportHandler) ImportCleanListings(w http.ResponseWriter, r *http.Request) {
	if h.maxRequestBodyBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxRequestBodyBytes)
	}

	var payload []json.RawMessage

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{
				"error":    "request too large",
				"message":  fmt.Sprintf("Maximum request body size is %d bytes", h.maxRequestBodyBytes),
				"maxBytes": h.maxRequestBodyBytes,
			})
			return
		}

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
		var tooLarge *importlistings.ImportTooLargeError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":        "import too large",
				"message":      fmt.Sprintf("Maximum import size is %d listings per request", tooLarge.MaxRows),
				"maxRows":      tooLarge.MaxRows,
				"receivedRows": tooLarge.ReceivedRows,
			})
			return
		}

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
