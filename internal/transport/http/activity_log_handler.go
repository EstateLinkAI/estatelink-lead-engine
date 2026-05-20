package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/logactivity"
	"github.com/go-chi/chi/v5"
)

type ActivityLogHandler struct {
	ActivityService *logactivity.Service
}

func NewActivityLogHandler(activityService *logactivity.Service) *ActivityLogHandler {
	return &ActivityLogHandler{
		ActivityService: activityService,
	}
}

func (h *ActivityLogHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/activity-logs", h.GetActivityLogs)
	r.Get("/api/activity-logs/{id}", h.GetActivityLogByID)
}

// GET /api/activity-logs
func (h *ActivityLogHandler) GetActivityLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if queryLimit := r.URL.Query().Get("limit"); queryLimit != "" {
		parsedLimit, err := strconv.Atoi(queryLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	if queryOffset := r.URL.Query().Get("offset"); queryOffset != "" {
		parsedOffset, err := strconv.Atoi(queryOffset)
		if err != nil || parsedOffset < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = parsedOffset
	}

	logs, err := h.ActivityService.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, "failed to fetch activity logs", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

// GET /api/activity-logs/{id}
func (h *ActivityLogHandler) GetActivityLogByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid activity log id", http.StatusBadRequest)
		return
	}

	logEntry, err := h.ActivityService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "activity log not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to fetch activity log", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, logEntry)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}